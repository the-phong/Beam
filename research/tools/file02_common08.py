#!/usr/bin/env python3
"""COMMON-08 fault QC from a frozen four-validator QA run and recorder raw."""

import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import subprocess
import sys
from uuid import uuid4

import file02_common as common
from file02_common03 import git_value, private_session
from file02_t06_precision import parse_go_rfc3339

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
FAULTS = ("validator_lost", "apphash_divergence", "wall_clock_discontinuity",
          "failed_target_load", "monitoring_drops", "generator_saturation")


def classify(signals):
    common.require(set(signals) == set(FAULTS) and
                   all(isinstance(value, bool) for value in signals.values()), "incomplete fault signals")
    reasons = [key for key in FAULTS if signals[key]]
    return {"status": "INVALID_FOR_ANALYSIS" if reasons else "USABLE_FOR_QA_ONLY",
            "exclude_reasons": reasons}


def inspect_run(run, source, tree):
    meta = common.read_json(run / "metadata.json")
    plan = common.read_json(run / "plan.json")
    result = common.read_json(run / "result.json")
    oracle = common.read_json(run / "oracle.json")
    resources = common.read_json(run / "generator_resources.json")
    config = common.read_json(ROOT / "research/config/s06_pilot.json")
    common.require(meta["run_id"] == plan["run_id"] == result["run_id"] == oracle["run_id"] ==
                   resources["run_id"] and meta["source_commit"] == source and meta["source_tree"] == tree and
                   result["plan_sha256"] == common.digest(run / "plan.json"), "run binding mismatch")
    signals = dict.fromkeys(FAULTS, False)
    signals["validator_lost"] = oracle["status"] != "OK" or oracle["common_height"] < 8
    signals["apphash_divergence"] = not oracle["block_id"] or not oracle["app_hash"]
    signals["failed_target_load"] = (result["status"] != "OK" or
                                     result["actual_send_count"] != result["planned_count"] or
                                     len(result["events"]) != len(plan["events"]))
    warmup, window, steady = plan["warmup_ns"], plan["window_ns"], plan["steady_ns"]
    windows = [0] * (steady // window)
    first_utc = parse_go_rfc3339(result["started_utc"])
    for expected, actual in zip(plan["events"], result["events"]):
        common.require(expected["id"] == actual["id"], "event identity")
        offset = actual.get("actual_offset_ns")
        if not isinstance(offset, int):
            signals["failed_target_load"] = True
            continue
        sent_utc = parse_go_rfc3339(actual["actual_send_utc"])
        utc_offset_ns = (sent_utc-first_utc).total_seconds() * 1e9
        if abs(utc_offset_ns-offset) > 100_000_000:
            signals["wall_clock_discontinuity"] = True
        if expected["offset_ns"] >= warmup:
            index = (offset-warmup) // window
            if 0 <= index < len(windows):
                windows[index] += 1
            else:
                signals["failed_target_load"] = True
    if any(count < 0.9 * plan["target_tx_per_second"] for count in windows):
        signals["failed_target_load"] = True
    max_drops = 0
    for i in range(4):
        lines = (run / f"observer-{i}.jsonl").read_text().splitlines()
        if not lines:
            signals["monitoring_drops"] = True
            continue
        last_time = None
        for line in lines:
            sample = json.loads(line)
            if sample.get("kind") != "node_sample" or sample.get("comet_process") is None or sample.get("application_process") is None:
                signals["monitoring_drops"] = True
            drops = sample.get("dropped_samples")
            if not isinstance(drops, int) or drops < 0:
                signals["monitoring_drops"] = True
            else:
                max_drops = max(max_drops, drops)
                if drops > 0 or sample.get("unavailable"):
                    signals["monitoring_drops"] = True
            observed = parse_go_rfc3339(sample["observed_utc"])
            if last_time is not None and observed < last_time:
                signals["wall_clock_discontinuity"] = True
            last_time = observed
    cpu_fraction = (resources["user_cpu_ns"] + resources["system_cpu_ns"]) / resources["wall_ns"]
    if cpu_fraction >= config["safety"]["max_cpu_fraction"]:
        signals["generator_saturation"] = True
    decision = classify(signals)
    return {"schema": "pluto.file02.common08_qc.v1", "run_id": meta["run_id"],
            "source_commit": source, "source_tree": tree, "signals": signals,
            "status": decision["status"], "exclude_reasons": decision["exclude_reasons"],
            "observer_max_dropped_samples": max_drops, "generator_cpu_core_fraction": cpu_fraction,
            "steady_tx_s_windows": windows, "raw_preserved": True}


def audit(session, source, tree):
    report = inspect_run(session / "run", source, tree)
    return {"schema": "pluto.file02.common08_audit.v1", "scope": "QA_ONLY",
            "source_commit": source, "source_tree": tree, "qc": report}


def smoke():
    common.require(not git_value("status", "--porcelain"), "commit source before QA")
    source, tree = git_value("rev-parse", "HEAD"), git_value("rev-parse", "HEAD^{tree}")
    ARTIFACTS.mkdir(parents=True, exist_ok=True, mode=0o700)
    session = ARTIFACTS / ("common08-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8])
    session.mkdir(mode=0o700)
    with (session / "build.log").open("wb") as log:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT, stdout=log, stderr=subprocess.STDOUT)
    common.require(build.returncode == 0, "build failed")
    env = os.environ.copy()
    env.update(FILE02_COMMON08_ROOT=str(session), FILE02_SOURCE_COMMIT=source, FILE02_SOURCE_TREE=tree)
    with (session / "integration.log").open("wb") as log:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02Common08QCEvidence$", "-timeout", "180s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=200)
    common.require(run.returncode == 0, "integration failed")
    summary = audit(session, source, tree)
    common.require(summary["qc"]["status"] == "USABLE_FOR_QA_ONLY", "QA QC failed")
    common.write_once(session / "summary.json", json.dumps(summary, indent=2, sort_keys=True) + "\n")
    manifest = common.freeze_raw(session)
    print(f"COMMON08=PASS run=1 session={session} manifest_sha256={manifest}")


def verify(path):
    session = private_session(path)
    manifest = common.verify_raw(session)
    prior = common.read_json(session / "summary.json")
    common.require(audit(session, prior["source_commit"], prior["source_tree"]) == prior,
                   "QC differs from frozen raw")
    print(f"COMMON08_VERIFY=OK status={prior['qc']['status']} manifest_sha256={manifest}")


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
    except (ValueError, KeyError, TypeError, OSError, subprocess.TimeoutExpired) as exc:
        print(f"COMMON08_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
