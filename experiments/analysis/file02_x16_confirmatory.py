#!/usr/bin/env python3
"""Lock and execute matched B/I X16 stage-resource networks."""

import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import random
import subprocess
import sys
from uuid import uuid4

import file02_common as common
import file02_r01 as r01

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
SEED = 20261116
WIRE_BYTES = 2568


def require(condition, reason):
    if not condition:
        raise ValueError(reason)


def schedule():
    order = ["BI", "IB"] * 12
    random.Random(SEED).shuffle(order)
    return {"schema": "pluto.file02.x16_schedule.v1", "seed": SEED,
        "permutations": order,
        "jobs": [{"block": block, "order": position, "condition": arm,
                  "run_id": f"X16-b{block:02d}-{arm}"}
                 for block, permutation in enumerate(order)
                 for position, arm in enumerate(permutation)]}


def session_path(path):
    session = Path(path).resolve()
    require(session.parent == ARTIFACTS.resolve() and session.is_dir(), "private X16 session path")
    return session


def validate_plan(session, plan):
    raw = session / "raw"
    require(plan.get("schema") == "pluto.file02.x16_plan.v1" and
            plan.get("status") == "LOCKED" and plan.get("case_id") == "X16" and
            plan.get("n_blocks") == 24 and plan.get("n_networks") == 48 and
            plan.get("arms") == ["B", "I"] and plan.get("wire_bytes") == WIRE_BYTES and
            plan.get("steady_seconds") == 12 and plan.get("source_commit") and
            plan.get("source_tree") and plan.get("topology") ==
            "four_validator_single_physical_host_loopback_RPC_validator0" and
            plan.get("safety_caps") == [25, 65536, 16, 60] and
            plan.get("stage_cpu_mode") == "ON" and plan.get("trace_mode") == "ON" and
            plan.get("ci_method") == "paired_block_percentile_bootstrap_20000_seed_20261116" and
            plan.get("exclusion_policy") == "failed_arm_excludes_block_no_replacement",
            "X16 outer plan drift")
    require(common.read_json(raw / "schedule.json") == schedule() and
            common.digest(raw / "schedule.json") == plan["schedule_sha256"] and
            common.digest(raw / "config.json") == plan["config_sha256"],
            "X16 schedule/config drift")


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve() and
            not r01.git("status", "--porcelain"), "X16 source must be clean before plan lock")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT).returncode == 0,
            "X16 session not ignored")
    session.mkdir(mode=0o700)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    common.write_once(raw / "schedule.json", json.dumps(schedule(), indent=2, sort_keys=True) + "\n")
    common.write_once(raw / "config.json", (ROOT / "configs/experiments/s06_pilot.json").read_text())
    plan = {"schema": "pluto.file02.x16_plan.v1", "status": "LOCKED", "case_id": "X16",
        "source_commit": r01.git("rev-parse", "HEAD"),
        "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
        "created_utc": datetime.now(timezone.utc).isoformat(),
        "protocol": "research/protocol/FILE02_X07_X16_REWORK_LOCKED_PLAN.md",
        "host": r01.cpu_inventory(), "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
        "module_sha256": common.digest(ROOT / "go.mod"),
        "config_sha256": common.digest(raw / "config.json"),
        "schedule_sha256": common.digest(raw / "schedule.json"),
        "n_blocks": 24, "n_networks": 48, "arms": ["B", "I"],
        "arm_semantics": {"B": "6 fixed valid H/s + 14 valid H/s",
                          "I": "6 fixed valid H/s + 14 late-bad-ML H/s"},
        "wire_bytes": WIRE_BYTES, "offered_tx_s": 20, "offered_bytes_s": 51360,
        "warmup_seconds": 1, "steady_seconds": 12, "drain_seconds": 5,
        "topology": "four_validator_single_physical_host_loopback_RPC_validator0",
        "safety_caps": [25, 65536, 16, 60], "stage_cpu_mode": "ON", "trace_mode": "ON",
        "independent_unit": "fresh_four_validator_network_paired_by_block",
        "primary_descriptive_contrasts": ["app_cpu_B_minus_I", "comet_cpu_B_minus_I",
                                           "relay_send_B_minus_I", "mempool_residence_B_minus_I"],
        "ci_method": "paired_block_percentile_bootstrap_20000_seed_20261116",
        "exclusion_policy": "failed_arm_excludes_block_no_replacement",
        "bottleneck_rule": "only locate if repeated queue/resource accumulation or saturation observed within safe range",
        "stage_cpu_interpretation": "instrumented thread CPU; clock/probe overhead separately reported",
        "original_rq1b_endpoint": "untouched historical 3s result"}
    r01.write_new(session / "plan.json", plan)
    validate_plan(session, plan)
    return plan


def bind_run_plan(run_path, network_root):
    run = Path(run_path).resolve()
    session = run.parents[2]
    network = Path(network_root).resolve()
    require(session_path(session) == session and network.is_dir() and run.is_dir() and
            not (run / "research_plan/plan.json").exists(), "X16 run/network path")
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    meta, workload, nodes = (common.read_json(run / name) for name in
                             ("metadata.json", "plan.json", "nodes.json"))
    job = next((j for j in schedule()["jobs"] if j["run_id"] == run.name), None)
    require(job is not None and meta.get("scope") == "X_CONFIRMATORY" and
            meta.get("case_id") == "X16" and meta.get("condition") == job["condition"] and
            meta.get("block") == job["block"] and meta.get("order") == job["order"] and
            meta.get("run_id") == workload.get("run_id") == run.name and
            meta.get("source_commit") == plan["source_commit"] and
            meta.get("source_tree") == plan["source_tree"] and
            meta.get("trace_mode") == meta.get("stage_cpu_mode") == "ON" and
            len(nodes) == 4 and all(node["rpc"].startswith("http://127.0.0.1:") and
                node["app_pid"] > 0 and node["observer_pid"] > 0 for node in nodes),
            "X16 run identity/topology")
    require(common.digest(run / "config.json") == plan["config_sha256"] and
            workload.get("profile") == 4 and len(workload.get("events", [])) == 241 and
            workload.get("warmup_ns") == 1_000_000_000 and
            workload.get("steady_ns") == 12_000_000_000 and
            workload.get("drain_ns") == 5_000_000_000 and
            all(item["source"] == "rpc" and len(item["bytes_hex"]) == 2*WIRE_BYTES
                for item in workload["events"]), "X16 workload/config")
    config = common.read_json(run / "config.json")
    safety = config["safety"]
    require([safety[k] for k in ("max_tx_per_second", "max_bytes_per_second",
            "max_outstanding", "max_duration_seconds")] == plan["safety_caps"],
            "X16 S06 safety drift")
    bound = {"schema": "pluto.file02.x16_run_plan.v1", "status": "LOCKED",
        "case_id": "X16", "run_id": run.name, "condition": job["condition"],
        "block": job["block"], "order": job["order"],
        "source_commit": plan["source_commit"], "source_tree": plan["source_tree"],
        "config_sha256": plan["config_sha256"],
        "workload_sha256": common.digest(run / "plan.json"),
        "genesis_sha256": common.digest(run / "genesis.bin"),
        "topology": plan["topology"], "safety_caps": plan["safety_caps"],
        "trace_mode": "ON", "stage_cpu_mode": "ON",
        "clock_semantics": "same-host CLOCK_MONOTONIC + locked-thread CLOCK_THREAD_CPUTIME_ID",
        "network_config_sha256": [common.digest(network / f"validator-{i}/config/config.toml")
                                  for i in range(4)],
        "expected_valid_committed": 241 if job["condition"] == "B" else 73,
        "expected_invalid_rejected": 0 if job["condition"] == "B" else 168}
    directory = run / "research_plan"
    directory.mkdir(mode=0o700)
    r01.write_new(directory / "plan.json", bound)


def execute():
    session = ARTIFACTS / ("x16-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") +
                           "-" + uuid4().hex[:8])
    plan = prepare(session)
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT,
                               stdout=stream, stderr=subprocess.STDOUT)
    require(build.returncode == 0, f"X16 build failed; private session {session}")
    env = os.environ.copy()
    env.update(FILE02_X16_ROOT=str(raw), FILE02_SOURCE_COMMIT=plan["source_commit"],
               FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        completed = subprocess.run(["go", "test", "-tags", "integration fourval",
                                    "./tests/integration", "-run", "^TestFile02X16MatchedBI$",
                                    "-timeout", "7200s", "-count=1", "-v"],
                                   cwd=ROOT, env=env, stdout=stream,
                                   stderr=subprocess.STDOUT, timeout=7300)
    manifest = common.freeze_raw(raw)
    print(f"X16_SESSION={session} INTEGRATION_EXIT={completed.returncode} RAW_MANIFEST={manifest}", flush=True)
    return completed.returncode


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("execute", "bind-run-plan"))
    parser.add_argument("run_dir", nargs="?")
    parser.add_argument("network_root", nargs="?")
    args = parser.parse_args()
    try:
        if args.command == "execute":
            return execute()
        require(args.run_dir and args.network_root, "bind paths required")
        bind_run_plan(args.run_dir, args.network_root)
        return 0
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError, subprocess.TimeoutExpired) as exc:
        print(f"X16_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
