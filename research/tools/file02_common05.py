#!/usr/bin/env python3
"""Audit matched total tx/s and bytes/s in bounded benign/attack QA pairs."""

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


def validate_run(run, source, tree, profile, condition):
    meta, plan, result, oracle = (common.read_json(run / name) for name in
                                  ("metadata.json", "plan.json", "result.json", "oracle.json"))
    require(meta.get("schema") == "pluto.file02.common05_run.v1" and meta.get("scope") == "QA_ONLY" and
            meta.get("profile") == profile and meta.get("condition") == condition and
            meta.get("source_commit") == source and meta.get("source_tree") == tree and
            meta.get("network") == "fresh_four_validator_single_host_loopback", "metadata mismatch")
    require(meta["run_id"] == plan["run_id"] == result["run_id"] == oracle["run_id"] and
            result["plan_sha256"] == common.digest(run / "plan.json") and
            plan["genesis_sha256"] == result["genesis_sha256"] == common.digest(run / "genesis.bin") and
            plan["config_sha256"] == result["config_sha256"] == meta["config_sha256"], "identity/hash mismatch")
    require(result["status"] == oracle["status"] == "OK" and oracle["common_height"] >= 8 and
            oracle["block_id"] and oracle["app_hash"] and
            result["actual_send_count"] == result["planned_count"] == len(plan["events"]) == len(result["events"]),
            "execution/oracle mismatch")
    windows = {traffic: [[0, 0] for _ in range(4)] for traffic in ("legitimate", "adversarial")}
    hashes, slots = set(), set()
    for expected, actual in zip(plan["events"], result["events"]):
        raw = bytes.fromhex(expected["bytes_hex"])
        target, sent = expected["offset_ns"], actual["actual_offset_ns"]
        traffic = expected["traffic"]
        require(hashlib.sha256(raw).hexdigest() == expected["sha256"] and expected["sha256"] not in hashes and
                expected["account_slot"] not in slots and expected["nonce"] == 0 and
                actual["id"] == expected["id"] and actual["traffic"] == traffic and
                actual["planned_offset_ns"] == target and actual["actual_code"] == expected["expected_code"] and
                not actual.get("submission_error") and isinstance(sent, int) and
                0 <= sent < plan["warmup_ns"] + plan["steady_ns"] and
                (target < plan["warmup_ns"]) == (sent < plan["warmup_ns"]), "event mismatch")
        hashes.add(expected["sha256"])
        slots.add(expected["account_slot"])
        if target >= plan["warmup_ns"]:
            index = (sent - plan["warmup_ns"]) // plan["window_ns"]
            require(index < 4, "steady window overflow")
            windows[traffic][index][0] += 1
            windows[traffic][index][1] += len(raw)
    require(oracle["valid_committed"] == (9 if condition == "benign" else 5) and
            oracle["invalid_rejected"] == (0 if condition == "benign" else 4), "commit/reject mismatch")
    require([v[0] for v in windows["legitimate"]] == ([2]*4 if condition == "benign" else [1]*4) and
            [v[0] for v in windows["adversarial"]] == ([0]*4 if condition == "benign" else [1]*4),
            "traffic split mismatch")
    total_count = [sum(windows[traffic][i][0] for traffic in windows) for i in range(4)]
    total_bytes = [sum(windows[traffic][i][1] for traffic in windows) for i in range(4)]
    return plan, {"run_id": plan["run_id"], "profile": profile, "condition": condition,
                  "total_tx_s_windows": total_count, "total_bytes_s_windows": total_bytes,
                  "legitimate_tx_s_windows": [v[0] for v in windows["legitimate"]],
                  "adversarial_tx_s_windows": [v[0] for v in windows["adversarial"]],
                  "valid_committed": oracle["valid_committed"], "invalid_rejected": oracle["invalid_rejected"],
                  "distinct_wire_hashes": len(hashes), "distinct_account_slots": len(slots)}


def audit(session, source, tree):
    pairs = []
    run_ids = set()
    for profile in PROFILES:
        benign, b = validate_run(session / f"profile-{profile}/benign", source, tree, profile, "benign")
        attack, a = validate_run(session / f"profile-{profile}/attack", source, tree, profile, "attack")
        legit = lambda plan: [(e["offset_ns"], e["sha256"], e["account_slot"], e["nonce"], e["bytes_hex"])
                              for e in plan["events"] if e["traffic"] == "legitimate"]
        require(len(benign["events"]) == len(attack["events"]) == 9 and
                len(legit(attack)) == 5 and
                all(be["offset_ns"] == ae["offset_ns"] and be["account_slot"] == ae["account_slot"] and
                    (be["sha256"] == ae["sha256"] and be["bytes_hex"] == ae["bytes_hex"]
                     if ae["traffic"] == "legitimate" else len(be["bytes_hex"]) == len(ae["bytes_hex"]))
                    for be, ae in zip(benign["events"], attack["events"])),
                "fixed legitimate stream or byte-matched replacement changed")
        require(benign["genesis_sha256"] == attack["genesis_sha256"] and
                benign["config_sha256"] == attack["config_sha256"] and
                benign["seed"] == attack["seed"] and benign["profile"] == attack["profile"], "pair context mismatch")
        require(b["total_tx_s_windows"] == a["total_tx_s_windows"] == [2]*4 and
                b["total_bytes_s_windows"] == a["total_bytes_s_windows"], "total offered pressure not matched")
        require(benign["target_tx_per_second"] == 2 and attack["target_legitimate_tx_per_second"] == 1 and
                attack["target_invalid_tx_per_second"] == 1, "planned rate mismatch")
        run_ids.update((b["run_id"], a["run_id"]))
        pairs.append({"profile": profile, "matched": ["total_tx_s", "total_bytes_s", "genesis", "profile", "schedule"],
                      "unmatched": ["accepted_tx_s", "mempool_occupancy", "verification_path", "valid_commit_count"],
                      "benign": b, "attack": a})
    require(len(run_ids) == 8, "duplicate run ID")
    return {"schema": "pluto.file02.common05_audit.v1", "scope": "QA_ONLY", "source_commit": source,
            "source_tree": tree, "fresh_networks": 8, "pairs": pairs}


def smoke():
    require(not git_value("status", "--porcelain"), "commit source before QA")
    source, tree = git_value("rev-parse", "HEAD"), git_value("rev-parse", "HEAD^{tree}")
    ARTIFACTS.mkdir(parents=True, exist_ok=True, mode=0o700)
    session = ARTIFACTS / ("common05-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8])
    session.mkdir(mode=0o700)
    with (session / "build.log").open("wb") as log:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT, stdout=log, stderr=subprocess.STDOUT)
    require(build.returncode == 0, "build failed; inspect private build.log")
    env = os.environ.copy()
    env.update(FILE02_COMMON05_ROOT=str(session), FILE02_SOURCE_COMMIT=source, FILE02_SOURCE_TREE=tree)
    with (session / "integration.log").open("wb") as log:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02Common05MatchedPressure$", "-timeout", "900s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=1000)
    require(run.returncode == 0, "integration failed; inspect private integration.log")
    summary = audit(session, source, tree)
    common.write_once(session / "summary.json", json.dumps(summary, indent=2, sort_keys=True) + "\n")
    manifest = common.freeze_raw(session)
    print(f"COMMON05_STATUS=OK pairs=4 networks=8 session={session} manifest_sha256={manifest}")


def verify(path):
    session = private_session(path)
    manifest = common.verify_raw(session)
    prior = common.read_json(session / "summary.json")
    require(prior.get("schema") == "pluto.file02.common05_audit.v1", "summary schema")
    require(audit(session, prior["source_commit"], prior["source_tree"]) == prior, "summary differs from raw")
    print(f"COMMON05_VERIFY=OK pairs=4 manifest_sha256={manifest}")


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
        print(f"COMMON05_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__": sys.exit(main())
