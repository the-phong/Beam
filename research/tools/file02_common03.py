#!/usr/bin/env python3
"""Run and audit bounded same-legitimate benign/attack four-validator QA pairs."""

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

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
PROFILES = ("C", "CP", "P", "H")


def require(condition, reason):
    if not condition:
        raise ValueError(reason)


def private_session(path):
    target = Path(path).resolve(strict=True)
    require(target.parent == ARTIFACTS.resolve() and target.is_dir(), "session outside ignored File 02 artifacts")
    return target


def steady_counts(plan, result):
    counts = {"legitimate": [0] * 4, "adversarial": [0] * 4}
    for planned, actual in zip(plan["events"], result["events"]):
        target, sent = planned["offset_ns"], actual["actual_offset_ns"]
        require(isinstance(sent, int) and planned["id"] == actual["id"] and
                actual["planned_offset_ns"] == target and actual["traffic"] == planned["traffic"] and
                actual["actual_code"] == planned["expected_code"] and not actual.get("submission_error"),
                "event send or code mismatch")
        require(0 <= sent < plan["warmup_ns"] + plan["steady_ns"] and
                (target < plan["warmup_ns"]) == (sent < plan["warmup_ns"]), "phase-crossing send")
        if target >= plan["warmup_ns"]:
            window = (sent - plan["warmup_ns"]) // plan["window_ns"]
            require(window < 4, "steady window index")
            counts[planned["traffic"]][window] += 1
    return counts


def validate_run(run, source, tree, profile, condition):
    meta, plan, result, oracle = (common.read_json(run / name) for name in
                                  ("metadata.json", "plan.json", "result.json", "oracle.json"))
    require(meta.get("schema") == "pluto.file02.common03_run.v1" and meta.get("scope") == "QA_ONLY" and
            meta.get("profile") == profile and meta.get("condition") == condition and
            meta.get("source_commit") == source and meta.get("source_tree") == tree and
            meta.get("network") == "fresh_four_validator_single_host_loopback", "run metadata mismatch")
    require(meta["run_id"] == plan["run_id"] == result["run_id"] == oracle["run_id"] and
            result["plan_sha256"] == common.digest(run / "plan.json") and
            plan["genesis_sha256"] == result["genesis_sha256"] == common.digest(run / "genesis.bin") and
            plan["config_sha256"] == result["config_sha256"] == meta["config_sha256"], "run binding mismatch")
    require(result["status"] == oracle["status"] == "OK" and oracle["common_height"] >= 8 and
            oracle["block_id"] and oracle["app_hash"] and
            result["actual_send_count"] == result["planned_count"] == len(plan["events"]) == len(result["events"]),
            "run execution or four-validator oracle failed")
    hashes, slots = set(), set()
    for event in plan["events"]:
        raw = bytes.fromhex(event["bytes_hex"])
        require(hashlib.sha256(raw).hexdigest() == event["sha256"] and event["sha256"] not in hashes and
                event["account_slot"] not in slots and event["nonce"] == 0, "duplicate wire/account/nonce")
        hashes.add(event["sha256"])
        slots.add(event["account_slot"])
    counts = steady_counts(plan, result)
    require(counts["legitimate"] == [1] * 4 and
            counts["adversarial"] == ([0] * 4 if condition == "benign" else [1] * 4),
            "achieved steady offered rate differs")
    require(oracle["valid_committed"] == 5 and oracle["invalid_rejected"] == (0 if condition == "benign" else 4),
            "commit/rejection oracle mismatch")
    return plan, result, {"run_id": plan["run_id"], "profile": profile, "condition": condition,
                          "legitimate_steady_tx_s": counts["legitimate"],
                          "adversarial_steady_tx_s": counts["adversarial"],
                          "legitimate_committed": oracle["valid_committed"],
                          "invalid_rejected": oracle["invalid_rejected"],
                          "distinct_wire_hashes": len(hashes), "distinct_account_slots": len(slots),
                          "plan_sha256": common.digest(run / "plan.json"),
                          "result_sha256": common.digest(run / "result.json")}


def audit(session, source, tree):
    rows = []
    for profile in PROFILES:
        benign, _, b = validate_run(session / f"profile-{profile}/benign", source, tree, profile, "benign")
        attack, _, a = validate_run(session / f"profile-{profile}/attack", source, tree, profile, "attack")
        legitimate = lambda plan: [(e["offset_ns"], e["sha256"], e["account_slot"], e["nonce"], e["bytes_hex"])
                                   for e in plan["events"] if e["traffic"] == "legitimate"]
        require(legitimate(benign) == legitimate(attack) and
                benign["genesis_sha256"] == attack["genesis_sha256"] and
                benign["seed"] == attack["seed"] and benign["profile"] == attack["profile"],
                "legitimate stream or fixture changed between pair")
        require(benign["target_tx_per_second"] == attack["target_legitimate_tx_per_second"] == 1 and
                attack["target_invalid_tx_per_second"] == 1, "planned offered-rate pair mismatch")
        rows.extend((b, a))
    require(len({row["run_id"] for row in rows}) == 8, "reused run ID")
    return {"schema": "pluto.file02.common03_audit.v1", "scope": "QA_ONLY", "source_commit": source,
            "source_tree": tree, "pairs": 4, "fresh_networks": 8, "runs": rows}


def git_value(*args):
    return subprocess.check_output(["git", *args], cwd=ROOT, text=True).strip()


def smoke():
    require(not git_value("status", "--porcelain"), "commit source and reports before running QA")
    source, tree = git_value("rev-parse", "HEAD"), git_value("rev-parse", "HEAD^{tree}")
    ARTIFACTS.mkdir(parents=True, exist_ok=True, mode=0o700)
    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    session = ARTIFACTS / f"common03-{stamp}-{uuid4().hex[:8]}"
    session.mkdir(mode=0o700)
    build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT,
                           stdout=(session / "build.log").open("wb"), stderr=subprocess.STDOUT)
    require(build.returncode == 0, "build failed; inspect private build.log")
    env = os.environ.copy()
    env.update(FILE02_COMMON03_ROOT=str(session), FILE02_SOURCE_COMMIT=source, FILE02_SOURCE_TREE=tree)
    with (session / "integration.log").open("wb") as log:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02Common03FixedLegitimate$", "-timeout", "900s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=1000)
    require(run.returncode == 0, "integration failed; inspect private integration.log")
    summary = audit(session, source, tree)
    common.write_once(session / "summary.json", json.dumps(summary, indent=2, sort_keys=True) + "\n")
    manifest = common.freeze_raw(session)
    print(f"COMMON03_STATUS=OK pairs=4 networks=8 session={session} manifest_sha256={manifest}")


def verify(path):
    session = private_session(path)
    manifest = common.verify_raw(session)
    prior = common.read_json(session / "summary.json")
    require(prior.get("schema") == "pluto.file02.common03_audit.v1", "summary schema")
    require(audit(session, prior["source_commit"], prior["source_tree"]) == prior, "summary differs from raw")
    print(f"COMMON03_VERIFY=OK pairs=4 manifest_sha256={manifest}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("smoke")
    p = sub.add_parser("verify")
    p.add_argument("session")
    args = parser.parse_args()
    try:
        if args.command == "smoke":
            smoke()
        else:
            verify(args.session)
        return 0
    except (OSError, ValueError, KeyError, TypeError, subprocess.TimeoutExpired) as exc:
        print(f"COMMON03_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
