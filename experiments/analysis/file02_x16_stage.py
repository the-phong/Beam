#!/usr/bin/env python3
"""Independent X16 stage/observer CPU engineering QA verifier."""

import argparse
from collections import defaultdict
import json
from pathlib import Path
import statistics
import sys

import file02_common as common
import file02_x07_trace as x07

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"


def require(condition, reason):
    if not condition:
        raise ValueError(reason)


def process_delta(samples, key, pid):
    values = [row.get(key) for row in samples]
    require(len(values) >= 2 and all(p is not None and p["pid"] == pid and
            p["start_ticks"] == values[0]["start_ticks"] for p in values) and
            values[-1]["cpu_ns"] >= values[0]["cpu_ns"], f"{key} process identity/CPU")
    return values[-1]["cpu_ns"] - values[0]["cpu_ns"]


def stage_statistics(events, mode, expected_valid=73):
    by_stage_wall = defaultdict(list)
    by_stage_cpu = defaultdict(list)
    new_count = 0
    for event in events:
        require(event["stage_cpu_status"] == ("CLOCK_THREAD_CPUTIME_ID_INSTRUMENTED" if
                mode == "ON" else "UNAVAILABLE_PROFILER_OFF"), "stage CPU status")
        if event["check_type"] != "new":
            continue
        new_count += 1
        for span in event["spans"]:
            require(span["duration_ns"] >= 0, "negative wall duration")
            by_stage_wall[span["stage"]].append(span["duration_ns"])
            if mode == "ON":
                require(isinstance(span.get("cpu_ns"), int) and span["cpu_ns"] > 0,
                        f"missing stage CPU: {span['stage']}")
                by_stage_cpu[span["stage"]].append(span["cpu_ns"])
            else:
                require("cpu_ns" not in span, "OFF mode fabricated stage CPU")
    require(new_count == 241 and set(by_stage_wall) ==
            {"structural", "binding", "digest", "secp", "ml", "state", "state_copy"} and
            all(len(by_stage_wall[stage]) == (expected_valid if stage in ("state", "state_copy") else 241)
                for stage in by_stage_wall), "stage path coverage")
    return {"wall_ns_mean": {stage: statistics.mean(values) for stage, values in by_stage_wall.items()},
            "instrumented_thread_cpu_ns_mean":
                {stage: statistics.mean(values) for stage, values in by_stage_cpu.items()} if mode == "ON" else None,
            "n_new": new_count}


def audit(session):
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve() and session.is_dir(), "private session path")
    manifest = common.verify_raw(session / "raw")
    plan = common.read_json(session / "plan.json")
    require(plan.get("schema") == "pluto.file02.x16_qa_plan.v1" and
            plan.get("status") == "LOCKED" and plan.get("scope") == "QA_ONLY" and
            plan.get("stage_cpu_modes") == ["OFF", "ON"] and plan.get("trace_mode") == "ON" and
            plan.get("rate_total_tx_s") == 20 and plan.get("safety_caps") ==
            {"max_tx_per_second": 25, "max_bytes_per_second": 65536,
             "max_outstanding": 16, "max_duration_seconds": 60}, "X16 locked QA plan drift")
    arms = {}
    for mode in plan["stage_cpu_modes"]:
        run = session / "raw/runs" / f"X16-QA-I-{mode}"
        meta, workload, oracle, result = (common.read_json(run / name) for name in
                                         ("metadata.json", "plan.json", "oracle.json", "result.json"))
        nodes = common.read_json(run / "nodes.json")
        require(meta["source_commit"] == plan["source_commit"] and
                meta["source_tree"] == plan["source_tree"] and
                meta["stage_cpu_mode"] == mode and meta["trace_mode"] == "ON" and
                meta["case_id"] == "X16" and len(nodes) == 4 and
                oracle["status"] == "OK" and oracle["valid_committed"] == 73 and
                oracle["invalid_rejected"] == 168 and result["status"] == "OK" and
                result["actual_send_count"] == 241 and
                common.digest(run / "config.json") == plan["config_sha256"],
                f"{mode} metadata/oracle/config QC")
        app = [x07.read_lines(run / f"app-{n}-events.jsonl") for n in range(4)]
        app_resources = [x07.read_lines(run / f"app-{n}-resources.jsonl") for n in range(4)]
        observers = [x07.read_lines(run / f"observer-{n}.jsonl") for n in range(4)]
        traces = [x07.trace_by_tx(run / f"comet-{n}-trace.jsonl", nodes[n]["comet_pid"])
                  for n in range(4)]
        x07.validate_origin(workload, traces[0][1], app[0])
        require(all(all(row["recorder_drops"] == 0 for row in series) for series in app_resources) and
                all(all(row["dropped_samples"] == 0 for row in series) for series in observers),
                f"{mode} recorder/observer drop")
        for n in range(4):
            require(all(row["pid"] == nodes[n]["app_pid"] for row in app_resources[n]),
                    f"{mode} app {n} PID")
        arms[mode] = {"genesis_sha256": result["genesis_sha256"],
            "planned_tx_sha256": [event["sha256"] for event in workload["events"]],
            "app_hash": oracle["app_hash"], "stage": stage_statistics(app[0], mode),
            "app_cpu_ns_by_node": [series[-1]["user_cpu_ns"]+series[-1]["system_cpu_ns"]-
                series[0]["user_cpu_ns"]-series[0]["system_cpu_ns"] for series in app_resources],
            "comet_cpu_ns_by_node": [process_delta(observers[n], "comet_process", nodes[n]["comet_pid"])
                for n in range(4)],
            "observer_cpu_ns_by_node": [process_delta(observers[n], "observer_process", nodes[n]["observer_pid"])
                for n in range(4)]}
    require(arms["OFF"]["genesis_sha256"] == arms["ON"]["genesis_sha256"] and
            arms["OFF"]["planned_tx_sha256"] == arms["ON"]["planned_tx_sha256"] and
            arms["OFF"]["app_hash"] == arms["ON"]["app_hash"], "X16 ON/OFF fixture/AppHash mismatch")
    for arm in arms.values():
        arm.pop("planned_tx_sha256")
    return {"schema": "pluto.file02.x16_qa_result.v1", "status": "PASS_QA_ONLY",
            "raw_manifest_sha256": manifest, "plan_sha256": common.digest(session / "plan.json"),
            "arms": arms, "stage_cpu_scope": "instrumented Linux thread CPU on CheckTx only",
            "stage_cpu_unperturbed": None,
            "stage_cpu_unperturbed_reason": "UNAVAILABLE_SINGLE_ON_OFF_PAIR_AND_PROBE_OVERHEAD",
            "bottleneck_location": None,
            "bottleneck_reason": "NOT_OBSERVED_WITHIN_SAFE_RANGE"}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("analyze", "verify"))
    parser.add_argument("session")
    args = parser.parse_args()
    try:
        result = audit(args.session)
        target = Path(args.session) / "x16_qa_result.json"
        if args.command == "analyze":
            common.write_once(target, json.dumps(result, indent=2, sort_keys=True) + "\n")
        else:
            require(common.read_json(target) == result, "X16 derived result drift")
        print(f"X16_{args.command.upper()}=OK status={result['status']} raw={result['raw_manifest_sha256']}")
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError, AssertionError) as exc:
        print(f"X16_ERROR: {exc}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
