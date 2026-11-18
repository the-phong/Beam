#!/usr/bin/env python3
"""Lock and execute the balanced X07 ABCI/P2P queue-trace QA."""

import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import subprocess
import sys
from uuid import uuid4

import file02_common as common
import file02_r01 as r01

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"


def schedule():
    return [{"block": block, "order": order, "mode": mode,
             "run_id": f"X07-V2-b{block:02d}-{mode}"}
            for block in range(8)
            for order, mode in enumerate(("OFF", "ON") if block % 2 == 0 else ("ON", "OFF"))]


def prepare(session):
    if session.parent.resolve() != ARTIFACTS.resolve() or r01.git("status", "--porcelain"):
        raise ValueError("X07 v2 QA requires clean source and private session path")
    if subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT).returncode != 0:
        raise ValueError("X07 v2 QA session is not ignored")
    session.mkdir(mode=0o700)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    common.write_once(raw / "config.json", (ROOT / "configs/experiments/s06_pilot.json").read_text())
    common.write_once(raw / "schedule.json", json.dumps(schedule(), indent=2, sort_keys=True) + "\n")
    plan = {"schema": "pluto.file02.x07_v2_plan.v1", "status": "LOCKED",
            "scope": "QA_ONLY", "case_id": "X07", "n_pairs": 8, "n_networks": 16,
            "source_commit": r01.git("rev-parse", "HEAD"),
            "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
            "created_utc": datetime.now(timezone.utc).isoformat(),
            "protocol": "research/protocol/FILE02_X07_QUEUE_V2_LOCKED_PLAN.md",
            "config_sha256": common.digest(raw / "config.json"),
            "schedule_sha256": common.digest(raw / "schedule.json"),
            "topology": "four_validator_single_physical_host_loopback_RPC_validator0",
            "workload": "6 fixed valid H/s + 14 late-bad-ML H/s; 20 tx/s; 51360 B/s",
            "trace_modes": ["OFF", "ON"], "stage_cpu_mode": "OFF",
            "safety_caps": [25, 65536, 16, 60],
            "ci_method": "paired_block_percentile_bootstrap_20000_seed_20261117",
            "exclusion_policy": "failed_arm_excludes_block_no_replacement"}
    common.write_once(session / "plan.json", json.dumps(plan, indent=2, sort_keys=True) + "\n")
    return plan


def execute():
    session = ARTIFACTS / ("x07-v2-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") +
                           "-" + uuid4().hex[:8])
    plan = prepare(session)
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT,
                               stdout=stream, stderr=subprocess.STDOUT)
        if build.returncode == 0:
            build = subprocess.run(["go", "build", "-modfile=build/s05-comet.mod",
                                    "-o", "build/profile-app-x07", "./cmd/research/profile-app"],
                                   cwd=ROOT, stdout=stream, stderr=subprocess.STDOUT)
    if build.returncode != 0:
        raise ValueError(f"X07 v2 QA build failed: {session}")
    env = os.environ.copy()
    env.update(FILE02_X07_V2_ROOT=str(raw), FILE02_SOURCE_COMMIT=plan["source_commit"],
               FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        completed = subprocess.run(["go", "test", "-tags", "integration fourval",
                                    "./tests/integration", "-run", "^TestFile02X07QueueV2QA$",
                                    "-timeout", "3600s", "-count=1", "-v"],
                                   cwd=ROOT, env=env, stdout=stream,
                                   stderr=subprocess.STDOUT, timeout=3700)
    manifest = common.freeze_raw(raw)
    print(f"X07_V2_SESSION={session} INTEGRATION_EXIT={completed.returncode} RAW_MANIFEST={manifest}")
    return completed.returncode


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("execute",))
    parser.parse_args()
    try:
        return execute()
    except (OSError, ValueError, subprocess.TimeoutExpired) as exc:
        print(f"X07_V2_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
