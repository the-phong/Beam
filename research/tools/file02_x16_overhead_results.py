#!/usr/bin/env python3
"""Independent raw/QC and paired ON/OFF probe-overhead analysis."""

import argparse
import json
from pathlib import Path
import statistics
import subprocess
import sys

import file02_common as common
import file02_x07_trace as x07
import file02_x16_overhead as locked
import file02_x16_results as x16
import file02_x16_stage as stage

ROOT = Path(__file__).resolve().parents[2]
FIELDS = ("app_cpu_s", "comet_cpu_s", "observer_cpu_s", "generator_cpu_s",
          "producer_rpc_median_ms", "fixed_valid_observation_upper_mean_ms")


def require(condition, reason):
    if not condition:
        raise ValueError(reason)


def audit_run(raw, job, plan):
    run = raw / "runs" / job["run_id"]
    meta, workload, result, oracle, commits, nodes = (
        common.read_json(run / name) for name in
        ("metadata.json", "plan.json", "result.json", "oracle.json",
         "commit_observations.json", "nodes.json"))
    mode = job["mode"]
    require(meta.get("schema") == "pluto.file02.x16_overhead_run.v1" and
            meta.get("scope") == "QA_ONLY" and meta.get("case_id") == "X16" and
            meta.get("block") == job["block"] and meta.get("order") == job["order"] and
            meta.get("run_id") == job["run_id"] and meta.get("stage_cpu_mode") == mode and
            meta.get("trace_mode") == "ON" and
            meta.get("source_commit") == plan["source_commit"] and
            meta.get("source_tree") == plan["source_tree"] and
            common.digest(run / "config.json") == plan["config_sha256"] and
            result.get("status") == "OK" and result.get("run_id") == job["run_id"] and
            result.get("actual_send_count") == 241 and
            result.get("actual_send_bytes") == 241*2568 and
            result.get("genesis_sha256") == common.digest(run / "genesis.bin") and
            oracle.get("status") == "OK" and oracle.get("valid_committed") == 73 and
            oracle.get("invalid_rejected") == 168 and len(nodes) == 4 and
            workload.get("profile") == 4 and workload.get("steady_ns") == 12_000_000_000 and
            len(workload.get("events", [])) == 241 and
            all(e["source"] == "rpc" and len(e["bytes_hex"]) == 5136
                for e in workload["events"]), f"{job['run_id']} provenance/shape/oracle")
    codes = {e["id"]: int(e["expected_code"]) for e in workload["events"]}
    require(len(codes) == 241 and len(result["events"]) == 241 and
            all(e["actual_code"] == codes[e["id"]] for e in result["events"]),
            f"{job['run_id']} result code coverage")
    app = [x07.read_lines(run / f"app-{n}-events.jsonl") for n in range(4)]
    app_resource = [x07.read_lines(run / f"app-{n}-resources.jsonl") for n in range(4)]
    observers = [x07.read_lines(run / f"observer-{n}.jsonl") for n in range(4)]
    traces = [x07.trace_by_tx(run / f"comet-{n}-trace.jsonl", nodes[n]["comet_pid"])
              for n in range(4)]
    intervals = x07.validate_origin(workload, traces[0][1], app[0])
    require(len(intervals) == 241 and
            all(all(sample["recorder_drops"] == 0 and sample["pid"] == nodes[n]["app_pid"]
                    for sample in app_resource[n]) for n in range(4)) and
            all(all(sample["dropped_samples"] == 0 for sample in observers[n])
                for n in range(4)), f"{job['run_id']} trace/PID/drop QC")
    for n in range(1, 4):
        require(len([row for row in traces[n][0] if row["event"] == "mempool_accepted"]) == 73,
                f"{job['run_id']} validator {n} accepted trace")
    stage_stats = stage.stage_statistics(app[0], mode)
    app_cpu = sum(series[-1]["user_cpu_ns"]+series[-1]["system_cpu_ns"]-
                  series[0]["user_cpu_ns"]-series[0]["system_cpu_ns"]
                  for series in app_resource)/1e9
    require(app_cpu >= 0, f"{job['run_id']} app CPU reset")
    comet_cpu = sum(stage.process_delta(observers[n], "comet_process", nodes[n]["comet_pid"])
                    for n in range(4))/1e9
    observer_cpu = sum(stage.process_delta(observers[n], "observer_process", nodes[n]["observer_pid"])
                       for n in range(4))/1e9
    generator = common.read_json(run / "generator_resources.json")
    require(generator.get("run_id") == job["run_id"] and generator.get("wall_ns", 0) > 0,
            f"{job['run_id']} generator identity")
    rpc = [(x07.utc_ns(e["completed_utc"])-x07.utc_ns(e["actual_send_utc"]))/1e6
           for e in result["events"]]
    require(all(0 <= value < 60000 for value in rpc), f"{job['run_id']} RPC clock order")
    fixed = {e["id"]: e for e in workload["events"] if e["scenario"] == "fixed_legitimate"}
    observed = {e["id"]: e for e in commits["events"]}
    submitted = {e["id"]: e for e in result["events"]}
    require(len(fixed) == 73 and len(commits["events"]) == 73 and
            commits.get("poll_interval_ms") == 100, f"{job['run_id']} fixed service coverage")
    latency = [(observed[eid]["all_four_seen_monotonic_ns"]-
                submitted[eid]["actual_send_monotonic_ns"])/1e6 for eid in fixed]
    require(all(0 <= value < 60000 for value in latency), f"{job['run_id']} service clock order")
    return {"run_id": job["run_id"], "block": job["block"], "order": job["order"],
            "mode": mode, "source_commit": plan["source_commit"],
            "genesis_sha256": result["genesis_sha256"],
            "planned_stream": [[e["sha256"], e["offset_ns"], e["expected_code"]]
                               for e in workload["events"]],
            "app_hash": oracle["app_hash"],
            "fixed_stream": [[e["sha256"], e["offset_ns"]] for e in fixed.values()],
            "app_cpu_s": app_cpu, "comet_cpu_s": comet_cpu,
            "observer_cpu_s": observer_cpu,
            "generator_cpu_s": (generator["user_cpu_ns"]+generator["system_cpu_ns"])/1e9,
            "producer_rpc_median_ms": statistics.median(rpc),
            "fixed_valid_observation_upper_mean_ms": statistics.mean(latency),
            "stage_wall_ns_mean": stage_stats["wall_ns_mean"],
            "instrumented_stage_thread_cpu_ns_mean": stage_stats["instrumented_thread_cpu_ns_mean"],
            "trace_events_by_node": [len(trace[0]) for trace in traces]}


def derive(session):
    session = Path(session).resolve()
    require(session.parent == locked.ARTIFACTS.resolve() and session.is_dir(),
            "private overhead QA session path")
    manifest = common.verify_raw(session / "raw")
    plan = common.read_json(session / "plan.json")
    require(plan.get("schema") == "pluto.file02.x16_overhead_plan.v1" and
            plan.get("status") == "LOCKED" and plan.get("scope") == "QA_ONLY" and
            plan.get("n_pairs") == 8 and plan.get("n_networks") == 16 and
            plan.get("protocol_correction") ==
                "research/protocol/FILE02_X16_PROBE_OVERHEAD_CORRECTION_20261117.md" and
            plan.get("quarantined_prior_raw_manifest_sha256") ==
                "d6f38452707aff536aac9cdad4e1e50c1e70a344ae0dadab278a9a2d15c4ec43" and
            plan.get("safety_caps") == [25, 65536, 16, 60] and
            plan.get("trace_mode") == "ON" and
            plan.get("ci_method") == "paired_block_percentile_bootstrap_20000_seed_20261116" and
            common.digest(session / "raw/config.json") == plan["config_sha256"] and
            common.read_json(session / "raw/schedule.json") == locked.schedule() and
            common.digest(session / "raw/schedule.json") == plan["schedule_sha256"],
            "overhead QA plan/schedule/config drift")
    tree = subprocess.run(["git", "rev-parse", f"{plan['source_commit']}^{{tree}}"],
                          cwd=ROOT, capture_output=True, text=True, check=True).stdout.strip()
    require(tree == plan["source_tree"], "overhead QA source commit/tree")
    rows, failures = [], []
    for job in locked.schedule():
        try:
            rows.append(audit_run(session / "raw", job, plan))
        except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError, AssertionError) as exc:
            failures.append({"run_id": job["run_id"], "reason": str(exc)})
    blocks = {}
    for row in rows:
        blocks.setdefault(row["block"], {})[row["mode"]] = row
    complete = {block: pair for block, pair in blocks.items() if set(pair) == {"OFF", "ON"} and
                pair["OFF"]["genesis_sha256"] == pair["ON"]["genesis_sha256"] and
                pair["OFF"]["planned_stream"] == pair["ON"]["planned_stream"] and
                pair["OFF"]["fixed_stream"] == pair["ON"]["fixed_stream"] and
                pair["OFF"]["app_hash"] == pair["ON"]["app_hash"]}
    for block in range(8):
        if block not in complete:
            failures.append({"block": block, "reason": "missing/unmatched pair; no replacement"})
    status = "PASS" if len(rows) == 16 and len(complete) == 8 and not failures else "INVALID_FOR_INFERENCE"
    qc = {"schema": "pluto.file02.x16_overhead_qc.v1", "status": status,
          "n_valid": len(rows), "n_pairs": len(complete), "n_planned": 16,
          "failures": failures, "raw_manifest_sha256": manifest,
          "plan_sha256": common.digest(session / "plan.json")}
    aggregate = {"schema": "pluto.file02.x16_overhead_aggregate.v1",
                 "status": "ANALYZED" if status == "PASS" else "WITHHELD_QC_FAILURE",
                 "raw_manifest_sha256": manifest, "n_pairs": len(complete),
                 "paired_ON_minus_OFF": None, "stage_wall_paired_ON_minus_OFF_ns": None,
                 "stage_cpu_unperturbed": None,
                 "stage_cpu_unperturbed_reason": "NO_VALIDATED_PER_STAGE_PROBE_BIAS_CORRECTION"}
    if status == "PASS":
        aggregate["paired_ON_minus_OFF"] = {field: {"mean": statistics.mean(values),
            "ci95": x16.paired_ci(values)} for field in FIELDS for values in [[
                complete[b]["ON"][field]-complete[b]["OFF"][field] for b in range(8)]]}
        stages = ("structural", "binding", "digest", "secp", "ml", "state", "state_copy")
        aggregate["stage_wall_paired_ON_minus_OFF_ns"] = {
            name: {"mean": statistics.mean(values), "ci95": x16.paired_ci(values)}
            for name in stages for values in [[
                complete[b]["ON"]["stage_wall_ns_mean"][name]-
                complete[b]["OFF"]["stage_wall_ns_mean"][name] for b in range(8)]]}
    return rows, qc, aggregate


def output(session, command):
    session = Path(session).resolve()
    rows, qc, aggregate = derive(session)
    paths = {"result_per_run.json": {"schema": "pluto.file02.x16_overhead_per_run.v1", "rows": rows},
             "qc_report.json": qc, "result_aggregate.json": aggregate}
    if command == "analyze":
        for name, value in paths.items():
            common.write_once(session / name, json.dumps(value, indent=2, sort_keys=True) + "\n")
    else:
        require(all(common.read_json(session / name) == value for name, value in paths.items()),
                "overhead QA derived drift")
    return qc


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("analyze", "verify"))
    parser.add_argument("session")
    args = parser.parse_args()
    try:
        qc = output(args.session, args.command)
        print(f"X16_OVERHEAD_{args.command.upper()}=OK QC={qc['status']} "
              f"runs={qc['n_valid']} pairs={qc['n_pairs']} raw={qc['raw_manifest_sha256']}")
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError,
            AssertionError, subprocess.CalledProcessError) as exc:
        print(f"X16_OVERHEAD_RESULT_ERROR: {exc}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
