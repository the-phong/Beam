#!/usr/bin/env python3
"""Audit bounded equal-tx/s and equal-bytes/s four-profile QA runs."""

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


def validate_run(run, source, tree, regime, profile):
    meta, plan, result, oracle = (common.read_json(run / name) for name in
                                  ("metadata.json", "plan.json", "result.json", "oracle.json"))
    require(meta.get("schema") == "pluto.file02.common04_run.v1" and meta.get("scope") == "QA_ONLY" and
            meta.get("profile") == profile and meta.get("regime") == regime and
            meta.get("source_commit") == source and meta.get("source_tree") == tree and
            meta.get("network") == "fresh_four_validator_single_host_loopback", "run metadata mismatch")
    require(meta["run_id"] == plan["run_id"] == result["run_id"] == oracle["run_id"] and
            result["plan_sha256"] == common.digest(run / "plan.json") and
            plan["genesis_sha256"] == result["genesis_sha256"] == common.digest(run / "genesis.bin") and
            plan["config_sha256"] == result["config_sha256"] == meta["config_sha256"], "run binding mismatch")
    require(result["status"] == oracle["status"] == "OK" and oracle["common_height"] >= 8 and
            oracle["block_id"] and oracle["app_hash"] and
            result["actual_send_count"] == result["planned_count"] == len(plan["events"]) == len(result["events"]),
            "run execution or oracle failed")
    counts, byte_windows = [0] * 4, [0] * 4
    hashes, slots = set(), set()
    for expected, actual in zip(plan["events"], result["events"]):
        raw = bytes.fromhex(expected["bytes_hex"])
        sent, target = actual["actual_offset_ns"], expected["offset_ns"]
        require(hashlib.sha256(raw).hexdigest() == expected["sha256"] and
                expected["sha256"] not in hashes and expected["account_slot"] not in slots and
                expected["nonce"] == 0 and expected["traffic"] == actual["traffic"] == "legitimate" and
                expected["expected_code"] == actual["actual_code"] == 0 and not actual.get("submission_error") and
                expected["id"] == actual["id"] and target == actual["planned_offset_ns"] and
                isinstance(sent, int) and 0 <= sent < plan["warmup_ns"] + plan["steady_ns"] and
                (target < plan["warmup_ns"]) == (sent < plan["warmup_ns"]), "wire, account or send mismatch")
        hashes.add(expected["sha256"])
        slots.add(expected["account_slot"])
        if target >= plan["warmup_ns"]:
            window = (sent - plan["warmup_ns"]) // plan["window_ns"]
            require(window < 4, "steady window index")
            counts[window] += 1
            byte_windows[window] += len(raw)
    require(oracle["valid_committed"] == len(plan["events"]) and oracle["invalid_rejected"] == 0,
            "valid commit oracle mismatch")
    return plan, {"run_id": plan["run_id"], "regime": regime, "profile": profile,
                  "target_tx_s": plan["target_tx_per_second"], "target_bytes_s": plan["target_bytes_per_second"],
                  "achieved_tx_s": sum(counts) / 4, "achieved_bytes_s": sum(byte_windows) / 4,
                  "steady_tx_windows": counts, "steady_bytes_windows": byte_windows,
                  "valid_committed": oracle["valid_committed"], "distinct_hashes": len(hashes),
                  "distinct_account_slots": len(slots), "plan_sha256": common.digest(run / "plan.json")}


def audit(session, source, tree):
    rows = []
    for regime in ("equal-tx", "equal-byte"):
        plans = []
        items = []
        for profile in PROFILES:
            plan, item = validate_run(session / regime / profile, source, tree, regime, profile)
            plans.append(plan)
            items.append(item)
            rows.append(item)
        require(len({p["genesis_sha256"] for p in plans}) == 1 and
                len({p["config_sha256"] for p in plans}) == 1, "cross-profile fixture/config mismatch")
        if regime == "equal-tx":
            require(all(p["regime"] == "equal_tx_s" and p["target_tx_per_second"] == 2 for p in plans) and
                    all(x["steady_tx_windows"] == [2] * 4 for x in items), "equal tx/s not achieved")
        else:
            target = plans[0]["target_bytes_per_second"]
            require(all(p["regime"] == "equal_bytes_s" and p["target_bytes_per_second"] == target for p in plans) and
                    all(all(abs(value - target) / target <= 0.05 for value in x["steady_bytes_windows"])
                        for x in items), "equal bytes/s not achieved within 5%")
    require(len({row["run_id"] for row in rows}) == 8, "reused run ID")
    return {"schema": "pluto.file02.common04_audit.v1", "scope": "QA_ONLY", "source_commit": source,
            "source_tree": tree, "fresh_networks": 8, "runs": rows}


def smoke():
    require(not git_value("status", "--porcelain"), "commit source and reports before running QA")
    source, tree = git_value("rev-parse", "HEAD"), git_value("rev-parse", "HEAD^{tree}")
    ARTIFACTS.mkdir(parents=True, exist_ok=True, mode=0o700)
    session = ARTIFACTS / ("common04-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8])
    session.mkdir(mode=0o700)
    with (session / "build.log").open("wb") as log:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT, stdout=log, stderr=subprocess.STDOUT)
    require(build.returncode == 0, "build failed; inspect private build.log")
    env = os.environ.copy()
    env.update(FILE02_COMMON04_ROOT=str(session), FILE02_SOURCE_COMMIT=source, FILE02_SOURCE_TREE=tree)
    with (session / "integration.log").open("wb") as log:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02Common04MatchedRates$", "-timeout", "900s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=1000)
    require(run.returncode == 0, "integration failed; inspect private integration.log")
    summary = audit(session, source, tree)
    common.write_once(session / "summary.json", json.dumps(summary, indent=2, sort_keys=True) + "\n")
    manifest = common.freeze_raw(session)
    print(f"COMMON04_STATUS=OK networks=8 session={session} manifest_sha256={manifest}")


def verify(path):
    session = private_session(path)
    manifest = common.verify_raw(session)
    prior = common.read_json(session / "summary.json")
    require(prior.get("schema") == "pluto.file02.common04_audit.v1", "summary schema")
    require(audit(session, prior["source_commit"], prior["source_tree"]) == prior, "summary differs from raw")
    print(f"COMMON04_VERIFY=OK networks=8 manifest_sha256={manifest}")


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
        print(f"COMMON04_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__": sys.exit(main())
