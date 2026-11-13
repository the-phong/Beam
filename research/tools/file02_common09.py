#!/usr/bin/env python3
"""Audit an intentional private safety stop on one fresh four-validator QA net."""

import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import subprocess
import sys
from urllib.parse import urlsplit
from uuid import uuid4

import file02_common as common
import file02_common08 as qc
from file02_common03 import git_value, private_session

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"


def audit(session, source, tree):
    run = session / "run"
    meta = common.read_json(run / "metadata.json")
    plan = common.read_json(run / "plan.json")
    result = common.read_json(run / "result.json")
    resources = common.read_json(run / "generator_resources.json")
    nodes = common.read_json(run / "nodes.json")
    common.require(meta["source_commit"] == source and meta["source_tree"] == tree and
                   meta["run_id"] == plan["run_id"] == result["run_id"] == resources["run_id"] and
                   meta["inject_stop_before_load"] is True and meta["network"] == "fresh_four_validator_single_host_loopback" and
                   len(nodes) == 4 and len({n["rpc"] for n in nodes}) == 4 and
                   all(urlsplit(n["rpc"]).hostname == "127.0.0.1" for n in nodes), "identity/network scope")
    common.require((run / "STOP").is_file() and
                   result["status"] == "SAFETY_ABORT" and "manual kill switch" in result["reason"] and
                   result["actual_send_count"] == 0 and len(result.get("events") or []) == 0 and
                   resources["wall_ns"] < 2_000_000_000 and
                   all((run / f"observer-{i}.jsonl").is_file() for i in range(4)),
                   "guard did not stop before sending")
    decision = qc.inspect_run(run, source, tree)
    common.require(decision["status"] == "INVALID_FOR_ANALYSIS" and
                   "INCOMPLETE_EVIDENCE" in decision["exclude_reasons"],
                   "aborted run entered analysis")
    return {"schema": "pluto.file02.common09_audit.v1", "scope": "QA_ONLY",
            "source_commit": source, "source_tree": tree, "run_id": result["run_id"],
            "status": "SAFETY_ABORT", "sent_count": 0,
            "generator_wall_ms": resources["wall_ns"] / 1_000_000,
            "qc_status": decision["status"], "exclude_reasons": decision["exclude_reasons"],
            "endpoint_hosts": [urlsplit(n["rpc"]).hostname for n in nodes],
            "observer_streams_preserved": 4}


def smoke():
    common.require(not git_value("status", "--porcelain"), "commit source before QA")
    source, tree = git_value("rev-parse", "HEAD"), git_value("rev-parse", "HEAD^{tree}")
    ARTIFACTS.mkdir(parents=True, exist_ok=True, mode=0o700)
    session = ARTIFACTS / ("common09-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8])
    session.mkdir(mode=0o700)
    with (session / "build.log").open("wb") as log:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT, stdout=log, stderr=subprocess.STDOUT)
    common.require(build.returncode == 0, "build failed")
    env = os.environ.copy()
    env.update(FILE02_COMMON09_ROOT=str(session), FILE02_SOURCE_COMMIT=source, FILE02_SOURCE_TREE=tree)
    with (session / "integration.log").open("wb") as log:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02Common09ForcedStop$", "-timeout", "120s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=140)
    common.require(run.returncode != 0, "intentional safety stop unexpectedly passed")
    summary = audit(session, source, tree)
    common.write_once(session / "summary.json", json.dumps(summary, indent=2, sort_keys=True) + "\n")
    manifest = common.freeze_raw(session)
    print(f"COMMON09=PASS sent=0 wall_ms={summary['generator_wall_ms']:.1f} session={session} manifest_sha256={manifest}")


def verify(path):
    session = private_session(path)
    manifest = common.verify_raw(session)
    prior = common.read_json(session / "summary.json")
    common.require(audit(session, prior["source_commit"], prior["source_tree"]) == prior,
                   "summary differs from frozen raw")
    print(f"COMMON09_VERIFY=OK sent=0 manifest_sha256={manifest}")


def finalize(path):
    session = private_session(path)
    common.require(not (session / "summary.json").exists() and not (session / "manifest.json").exists(),
                   "session already finalized")
    meta = common.read_json(session / "run/metadata.json")
    summary = audit(session, meta["source_commit"], meta["source_tree"])
    common.write_once(session / "summary.json", json.dumps(summary, indent=2, sort_keys=True) + "\n")
    manifest = common.freeze_raw(session)
    print(f"COMMON09=PASS sent=0 wall_ms={summary['generator_wall_ms']:.1f} session={session} manifest_sha256={manifest}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("smoke")
    p = sub.add_parser("verify")
    p.add_argument("session")
    p = sub.add_parser("finalize")
    p.add_argument("session")
    args = parser.parse_args()
    try:
        if args.command == "smoke": smoke()
        elif args.command == "verify": verify(args.session)
        else: finalize(args.session)
        return 0
    except (ValueError, KeyError, TypeError, OSError, subprocess.TimeoutExpired) as exc:
        print(f"COMMON09_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
