#!/usr/bin/env python3
"""Run and audit authorized bounded T06 generator calibration on loopback."""

import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
from uuid import uuid4

import file02_common as common
from file02_common03 import private_session, git_value, require

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
PROFILES = ("C", "CP", "P", "H")
RATES = (2, 6, 12, 20)


def validate_run(run, source, tree, profile, rate, *, schema="pluto.file02.t06_calibration.v1", scope="QA_ONLY"):
    meta, plan, result, oracle, resources = (common.read_json(run / name) for name in
                                            ("metadata.json", "plan.json", "result.json", "oracle.json",
                                             "generator_resources.json"))
    require(meta.get("schema") == schema and meta.get("scope") == scope and
            meta.get("capture_commit_observations") is False and meta.get("profile") == profile and
            meta.get("target_tx_s") == rate and meta.get("source_commit") == source and
            meta.get("source_tree") == tree and meta.get("network") == "fresh_four_validator_single_host_loopback",
            "metadata mismatch")
    require(meta["run_id"] == plan["run_id"] == result["run_id"] == oracle["run_id"] == resources["run_id"] and
            result["plan_sha256"] == common.digest(run / "plan.json") and
            plan["genesis_sha256"] == result["genesis_sha256"] == common.digest(run / "genesis.bin") and
            plan["config_sha256"] == result["config_sha256"] == meta["config_sha256"], "source/run binding mismatch")
    require(plan["regime"] == "equal_tx_s" and plan["target_tx_per_second"] == rate and
            result["status"] == oracle["status"] == "OK" and oracle["common_height"] >= 8 and
            oracle["block_id"] and oracle["app_hash"] and oracle["invalid_rejected"] == 0 and
            oracle["valid_committed"] == result["planned_count"] == result["actual_send_count"] == len(plan["events"]) == 1+4*rate,
            "generator, rate or independent oracle failed")
    require(resources["schema"] == "pluto.file02.generator_resources.v1" and
            all(isinstance(resources[key], int) and resources[key] >= 0 for key in
                ("wall_ns", "user_cpu_ns", "system_cpu_ns", "peak_rss_bytes")) and
            resources["wall_ns"] > 0, "generator resource sample missing")
    counts, byte_counts = [0]*4, [0]*4
    hashes, slots = set(), set()
    for expected, actual in zip(plan["events"], result["events"]):
        raw = bytes.fromhex(expected["bytes_hex"])
        target, sent = expected["offset_ns"], actual["actual_offset_ns"]
        require(hashlib.sha256(raw).hexdigest() == expected["sha256"] and expected["sha256"] not in hashes and
                expected["account_slot"] not in slots and expected["nonce"] == 0 and
                expected["id"] == actual["id"] and expected["traffic"] == actual["traffic"] == "legitimate" and
                actual["actual_code"] == expected["expected_code"] == 0 and not actual.get("submission_error") and
                isinstance(sent, int) and target == actual["planned_offset_ns"] and
                0 <= sent < plan["warmup_ns"] + plan["steady_ns"] and
                (target < plan["warmup_ns"]) == (sent < plan["warmup_ns"]), "event/send mismatch")
        hashes.add(expected["sha256"])
        slots.add(expected["account_slot"])
        if target >= plan["warmup_ns"]:
            window = (sent-plan["warmup_ns"]) // plan["window_ns"]
            require(0 <= window < 4, "send outside steady window")
            counts[window] += 1
            byte_counts[window] += len(raw)
    require(all(count >= .9*rate for count in counts) and abs(sum(counts)/4-rate)/rate <= .05,
            "generator did not achieve preregistered tx/s")
    require(max(byte_counts) <= 65536 and max(counts) <= 25, "safety cap exceeded")
    return {"run_id": plan["run_id"], "profile": profile, "target_tx_s": rate,
            "achieved_tx_s": sum(counts)/4, "target_bytes_s": sum(len(bytes.fromhex(e["bytes_hex"]))
                                                             for e in plan["events"] if e["offset_ns"] >= plan["warmup_ns"])/4,
            "achieved_bytes_s": sum(byte_counts)/4, "steady_tx_windows": counts,
            "steady_byte_windows": byte_counts, "genesis_sha256": plan["genesis_sha256"],
            "generator_cpu_core_fraction": (resources["user_cpu_ns"]+resources["system_cpu_ns"])/resources["wall_ns"],
            "generator_peak_rss_bytes": resources["peak_rss_bytes"],
            "dropped_or_unsent_requests": result["planned_count"]-result["actual_send_count"],
            "oracle_valid_committed": oracle["valid_committed"]}


def audit(session, source, tree):
    rows = [validate_run(session / f"rate-{rate:02d}" / profile, source, tree, profile, rate)
            for rate in RATES for profile in PROFILES]
    require(len({r["run_id"] for r in rows}) == len({r["genesis_sha256"] for r in rows}) == 16,
            "run or genesis reused")
    return {"schema": "pluto.file02.t06_calibration_audit.v1", "scope": "PILOT_ONLY",
            "source_commit": source, "source_tree": tree, "fresh_networks": 16,
            "testbed": "four_validators_single_host_loopback", "rate_sequence_tx_s": list(RATES),
            "high_rate_result": "GENERATOR_ACHIEVED_WITHIN_S06_CAPS" if all(r["achieved_tx_s"] == r["target_tx_s"] for r in rows)
                                else "RATE_MISS", "runs": rows}


def smoke():
    require(not git_value("status", "--porcelain"), "commit source before pilot")
    source, tree = git_value("rev-parse", "HEAD"), git_value("rev-parse", "HEAD^{tree}")
    ARTIFACTS.mkdir(parents=True, exist_ok=True, mode=0o700)
    session = ARTIFACTS / ("t06-calibration-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8])
    session.mkdir(mode=0o700)
    with (session / "build.log").open("wb") as log:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT, stdout=log, stderr=subprocess.STDOUT)
    require(build.returncode == 0, "build failed; inspect private build.log")
    env = os.environ.copy()
    env.update(FILE02_T06_ROOT=str(session), FILE02_SOURCE_COMMIT=source, FILE02_SOURCE_TREE=tree)
    with (session / "integration.log").open("wb") as log:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02T06BoundedCalibration$", "-timeout", "1200s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=1300)
    require(run.returncode == 0, "pilot integration failed; inspect private integration.log")
    summary = audit(session, source, tree)
    common.write_once(session / "summary.json", json.dumps(summary, indent=2, sort_keys=True) + "\n")
    manifest = common.freeze_raw(session)
    print(f"T06_CALIBRATION=OK runs=16 session={session} manifest_sha256={manifest}")


def verify(path):
    session = private_session(path)
    manifest = common.verify_raw(session)
    prior = common.read_json(session / "summary.json")
    require(prior.get("schema") == "pluto.file02.t06_calibration_audit.v1", "summary schema")
    require(audit(session, prior["source_commit"], prior["source_tree"]) == prior, "summary differs from raw")
    print(f"T06_VERIFY=OK runs=16 manifest_sha256={manifest}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("smoke")
    p = sub.add_parser("verify")
    p.add_argument("session")
    args = parser.parse_args()
    try:
        if args.command == "smoke": smoke()
        else: verify(args.session)
        return 0
    except (OSError, ValueError, KeyError, TypeError, subprocess.TimeoutExpired) as exc:
        print(f"T06_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__": sys.exit(main())
