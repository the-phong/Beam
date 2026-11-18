#!/usr/bin/env python3
"""Independent raw/QC and paired resource analysis for locked X16 B/I runs."""

import argparse
import csv
import hashlib
import io
import json
import os
from pathlib import Path
import random
import statistics
import subprocess
import sys

import file02_common as common
import file02_x07_trace as x07
import file02_x16_confirmatory as locked
import file02_x16_stage as stage

ROOT = Path(__file__).resolve().parents[2]
MEASURES = ("app_cpu_s", "comet_cpu_s", "observer_cpu_s", "generator_cpu_s",
            "relay_send_v0_bytes", "accepted_mempool_v0_peak_bytes",
            "socket_queue_v0_cumulative_s", "accepted_residence_v0_mean_ms",
            "fixed_valid_mean_observation_upper_ms", "fixed_valid_timely_fraction")


def require(condition, reason):
    if not condition:
        raise ValueError(reason)


def counter_delta(samples, field):
    values = [row.get(field) for row in samples if row.get(field) is not None]
    require(len(values) >= 2 and all(isinstance(v, (int, float)) for v in values) and
            values[-1] >= values[0], f"missing/reset counter {field}")
    return values[-1] - values[0]


def percentile(values, fraction):
    ordered = sorted(values)
    position = (len(ordered)-1)*fraction
    lower = int(position)
    weight = position-lower
    return ordered[lower]*(1-weight)+ordered[min(lower+1,len(ordered)-1)]*weight


def paired_ci(values):
    require(len(values) >= 2, "paired CI needs at least two blocks")
    rng = random.Random(locked.SEED)
    draws = sorted(statistics.mean(rng.choice(values) for _ in values) for _ in range(20000))
    return [percentile(draws, .025), percentile(draws, .975)]


def process_delta(samples, field, pid):
    return stage.process_delta(samples, field, pid)


def audit_run(raw, job, plan):
    run = raw / "runs" / job["run_id"]
    meta, workload, result, oracle, bound, commits, nodes = (
        common.read_json(run / name) for name in
        ("metadata.json", "plan.json", "result.json", "oracle.json",
         "research_plan/plan.json", "commit_observations.json", "nodes.json"))
    arm = job["condition"]
    expected_valid = 241 if arm == "B" else 73
    expected_invalid = 0 if arm == "B" else 168
    require(meta.get("schema") == "pluto.file02.x16_run.v1" and
            meta.get("scope") == "X_CONFIRMATORY" and meta.get("case_id") == "X16" and
            meta.get("run_id") == job["run_id"] and
            all(meta.get(k) == job[k] for k in ("condition", "block", "order")) and
            meta.get("source_commit") == plan["source_commit"] and
            meta.get("source_tree") == plan["source_tree"] and
            meta.get("trace_mode") == meta.get("stage_cpu_mode") == "ON" and
            result.get("status") == "OK" and result.get("actual_send_count") == 241 and
            result.get("actual_send_bytes") == 241*locked.WIRE_BYTES and
            oracle.get("status") == "OK" and oracle.get("valid_committed") == expected_valid and
            oracle.get("invalid_rejected") == expected_invalid and
            bound.get("schema") == "pluto.file02.x16_run_plan.v1" and
            bound.get("status") == "LOCKED" and bound.get("run_id") == job["run_id"] and
            bound.get("source_commit") == plan["source_commit"] and
            bound.get("source_tree") == plan["source_tree"] and
            bound.get("expected_valid_committed") == expected_valid and
            bound.get("expected_invalid_rejected") == expected_invalid and
            bound.get("workload_sha256") == common.digest(run / "plan.json") and
            bound.get("genesis_sha256") == common.digest(run / "genesis.bin") and
            result.get("genesis_sha256") == common.digest(run / "genesis.bin") and
            bound.get("config_sha256") == plan["config_sha256"] and
            common.digest(run / "config.json") == plan["config_sha256"] and
            len(nodes) == 4, f"{job['run_id']} identity/oracle/provenance QC")
    require(workload["profile"] == 4 and workload["steady_ns"] == 12_000_000_000 and
            len(workload["events"]) == 241 and all(e["source"] == "rpc" and
            len(e["bytes_hex"]) == 2*locked.WIRE_BYTES for e in workload["events"]),
            f"{job['run_id']} workload shape")
    expected_codes = {e["id"]: int(e["expected_code"]) for e in workload["events"]}
    require(len(result["events"]) == 241 and all(e["actual_code"] == expected_codes[e["id"]]
            for e in result["events"]), f"{job['run_id']} per-event result code")
    app = [x07.read_lines(run / f"app-{n}-events.jsonl") for n in range(4)]
    app_resources = [x07.read_lines(run / f"app-{n}-resources.jsonl") for n in range(4)]
    observers = [x07.read_lines(run / f"observer-{n}.jsonl") for n in range(4)]
    traces = [x07.trace_by_tx(run / f"comet-{n}-trace.jsonl", nodes[n]["comet_pid"])
              for n in range(4)]
    intervals = x07.validate_origin(workload, traces[0][1], app[0], require_all_relays=False)
    require(len([x for x in intervals if x["code"] == 0]) == expected_valid and
            len([x for x in intervals if x["code"] != 0]) == expected_invalid,
            f"{job['run_id']} linked path code counts")
    require(all(all(row["recorder_drops"] == 0 and row["pid"] == nodes[n]["app_pid"]
                    for row in app_resources[n]) for n in range(4)) and
            all(all(row["dropped_samples"] == 0 for row in observers[n]) for n in range(4)),
            f"{job['run_id']} recorder/observer drops or app PID")
    for n in range(1, 4):
        require(len([e for e in traces[n][0] if e["event"] == "mempool_accepted"]) == expected_valid,
                f"{job['run_id']} validator {n} acceptance coverage")
    stage_stats = stage.stage_statistics(app[0], "ON", expected_valid=expected_valid)
    fixed = {event["id"]: event for event in workload["events"] if
             event["scenario"] == "fixed_legitimate"}
    require(len(fixed) == 73 and commits.get("poll_interval_ms") == 100 and
            len(commits["events"]) == expected_valid, f"{job['run_id']} fixed observation coverage")
    observed = {event["id"]: event for event in commits["events"]}
    submitted = {event["id"]: event for event in result["events"]}
    latency = [(observed[event_id]["all_four_seen_monotonic_ns"]-
                submitted[event_id]["actual_send_monotonic_ns"])/1e6 for event_id in fixed]
    require(all(0 <= value < 60000 for value in latency), f"{job['run_id']} observation order")
    app_cpu = [app_resources[n][-1]["user_cpu_ns"]+app_resources[n][-1]["system_cpu_ns"]-
               app_resources[n][0]["user_cpu_ns"]-app_resources[n][0]["system_cpu_ns"]
               for n in range(4)]
    require(all(value >= 0 for value in app_cpu), f"{job['run_id']} app CPU reset")
    comet_cpu = [process_delta(observers[n], "comet_process", nodes[n]["comet_pid"])
                 for n in range(4)]
    observer_cpu = [process_delta(observers[n], "observer_process", nodes[n]["observer_pid"])
                    for n in range(4)]
    generator = common.read_json(run / "generator_resources.json")
    require(generator["run_id"] == job["run_id"] and generator["wall_ns"] > 0,
            f"{job['run_id']} generator resource identity")
    v0 = observers[0]
    for sample in v0:
        require(sample.get("comet_process") is not None and
                sample.get("application_process") is not None and
                sample.get("observer_process") is not None and
                isinstance(sample.get("accepted_mempool_bytes"), int),
                f"{job['run_id']} missing aligned resource curve sample")
    resource_curve = [{"sample_index": index,
        "accepted_mempool_bytes": sample["accepted_mempool_bytes"],
        "comet_cpu_cumulative_s": (sample["comet_process"]["cpu_ns"]-
                                    v0[0]["comet_process"]["cpu_ns"])/1e9,
        "app_cpu_cumulative_s": (sample["application_process"]["cpu_ns"]-
                                  v0[0]["application_process"]["cpu_ns"])/1e9,
        "observer_cpu_cumulative_s": (sample["observer_process"]["cpu_ns"]-
                                       v0[0]["observer_process"]["cpu_ns"])/1e9}
        for index, sample in enumerate(v0)]
    accepted_residence = [x["accepted_mempool_residence_ns"]/1e6 for x in intervals
                          if x["accepted_mempool_residence_ns"] is not None]
    return {"run_id": job["run_id"], "block": job["block"], "order": job["order"],
        "condition": arm, "scope": meta["scope"], "genesis_sha256": result["genesis_sha256"],
        "fixed_stream": [[e["sha256"], e["offset_ns"]] for e in fixed.values()],
        "valid_committed": expected_valid, "invalid_rejected": expected_invalid,
        "app_cpu_s": sum(app_cpu)/1e9, "comet_cpu_s": sum(comet_cpu)/1e9,
        "observer_cpu_s": sum(observer_cpu)/1e9,
        "generator_cpu_s": (generator["user_cpu_ns"]+generator["system_cpu_ns"])/1e9,
        "relay_send_v0_bytes": counter_delta(v0, "relay_send_envelope_bytes"),
        "accepted_mempool_v0_peak_bytes": max(s["accepted_mempool_bytes"] for s in v0),
        "socket_queue_v0_cumulative_s": counter_delta(v0, "checktx_socket_queue_seconds"),
        "accepted_residence_v0_mean_ms": statistics.mean(accepted_residence),
        "fixed_valid_mean_observation_upper_ms": statistics.mean(latency),
        "fixed_valid_timely_fraction": sum(x <= 3000 for x in latency)/73,
        "stage_wall_ns_mean": stage_stats["wall_ns_mean"],
        "stage_thread_cpu_ns_mean": stage_stats["instrumented_thread_cpu_ns_mean"],
        "trace_events_by_node": [len(t[0]) for t in traces],
        "validator0_resource_curve": resource_curve,
        "v0_sample_count": len(v0), "source_commit": plan["source_commit"],
        "raw_manifest_sha256": None}


def derive(session):
    session = locked.session_path(session)
    plan = common.read_json(session / "plan.json")
    locked.validate_plan(session, plan)
    source_tree = subprocess.run(["git", "rev-parse", f"{plan['source_commit']}^{{tree}}"],
        cwd=ROOT, capture_output=True, text=True, check=True).stdout.strip()
    require(source_tree == plan["source_tree"], "X16 source commit/tree provenance")
    manifest = common.verify_raw(session / "raw")
    rows, failures = [], []
    raw = session / "raw"
    for job in locked.schedule()["jobs"]:
        try:
            rows.append(audit_run(raw, job, plan))
        except (ValueError, OSError, KeyError, TypeError, json.JSONDecodeError,
                AssertionError, ZeroDivisionError) as exc:
            failures.append({"run_id": job["run_id"], "reason": str(exc)})
    blocks = {}
    for row in rows:
        blocks.setdefault(row["block"], {})[row["condition"]] = row
    complete = {block: pair for block, pair in blocks.items() if set(pair) == {"B", "I"} and
                pair["B"]["genesis_sha256"] == pair["I"]["genesis_sha256"] and
                pair["B"]["fixed_stream"] == pair["I"]["fixed_stream"]}
    for block in range(24):
        if block not in complete:
            failures.append({"block": block, "reason": "missing/unmatched pair; no replacement"})
    qc_status = "PASS" if len(rows) == 48 and len(complete) == 24 and not failures else "INVALID_FOR_INFERENCE"
    qc = {"schema": "pluto.file02.x16_qc.v1", "status": qc_status,
        "n_planned": 48, "n_valid": len(rows), "n_pairs": len(complete),
        "failures": failures, "raw_manifest_sha256": manifest,
        "plan_sha256": common.digest(session / "plan.json")}
    aggregate = {"schema": "pluto.file02.x16_aggregate.v1",
        "status": "ANALYZED" if qc_status == "PASS" else "WITHHELD_QC_FAILURE",
        "n_pairs": len(complete), "raw_manifest_sha256": manifest,
        "plan_sha256": qc["plan_sha256"], "arm_means": None,
        "paired_B_minus_I": None, "stage_thread_cpu_ns_by_arm": None,
        "stage_thread_cpu_paired_B_minus_I": None,
        "validator0_resource_curves": None,
        "stage_cpu_unperturbed": None,
        "stage_cpu_unperturbed_reason": "INSTRUMENTED_THREAD_CLOCK_ONLY_OVERHEAD_NOT_PRECISELY_BOUNDED",
        "full_transaction_queue_ns": None,
        "full_transaction_queue_reason": "UNAVAILABLE_P2P_AND_CONSENSUS_QUEUE_SUBINTERVALS",
        "bottleneck_location": None, "bottleneck_reason": "NOT_OBSERVED_WITHIN_SAFE_RANGE"}
    if qc_status == "PASS":
        aggregate["arm_means"] = {arm: {field: statistics.mean(complete[b][arm][field]
            for b in range(24)) for field in MEASURES} for arm in ("B", "I")}
        aggregate["paired_B_minus_I"] = {field: {"mean": statistics.mean(values),
            "ci95": paired_ci(values)} for field in MEASURES for values in [[
                complete[b]["B"][field]-complete[b]["I"][field] for b in range(24)]]}
        aggregate["stage_thread_cpu_ns_by_arm"] = {arm: {stage_name: {
            "mean": statistics.mean(values), "ci95": paired_ci(values)}
            for stage_name in complete[0][arm]["stage_thread_cpu_ns_mean"]
            for values in [[complete[b][arm]["stage_thread_cpu_ns_mean"][stage_name]
                            for b in range(24)]]} for arm in ("B", "I")}
        common_stages = ("structural", "binding", "digest", "secp", "ml")
        aggregate["stage_thread_cpu_paired_B_minus_I"] = {stage_name: {
            "mean": statistics.mean(values), "ci95": paired_ci(values)}
            for stage_name in common_stages for values in [[
                complete[b]["B"]["stage_thread_cpu_ns_mean"][stage_name]-
                complete[b]["I"]["stage_thread_cpu_ns_mean"][stage_name]
                for b in range(24)]]}
        curves = {}
        for arm in ("B", "I"):
            count = min(len(complete[b][arm]["validator0_resource_curve"]) for b in range(24))
            curves[arm] = [{"sample_index": index, "n_networks": 24,
                **{field: statistics.mean(complete[b][arm]["validator0_resource_curve"][index][field]
                      for b in range(24)) for field in
                   ("accepted_mempool_bytes", "comet_cpu_cumulative_s",
                    "app_cpu_cumulative_s", "observer_cpu_cumulative_s")}}
                for index in range(count)]
        aggregate["validator0_resource_curves"] = curves
    for row in rows:
        row["raw_manifest_sha256"] = manifest
    return rows, qc, aggregate


def csv_text(rows):
    output = io.StringIO()
    fields = ("run_id", "block", "order", "condition", "valid_committed",
              "invalid_rejected") + MEASURES + ("source_commit", "raw_manifest_sha256")
    writer = csv.DictWriter(output, fieldnames=fields, lineterminator="\n")
    writer.writeheader()
    writer.writerows({field: row.get(field) for field in fields} for row in rows)
    return output.getvalue()


def decision_text(qc, aggregate):
    if qc["status"] != "PASS":
        return "# X16 matched resource decision\n\nQC failed; all effects withheld. See qc_report.json.\n"
    lines = ["# X16 matched resource decision", "",
        f"QC {qc['status']}: {qc['n_valid']}/48 runs and {qc['n_pairs']}/24 pairs.", "",
        "B and I each offered 20 H-sized tx/s (51,360 B/s) with the same six fixed legitimate H/s. ",
        "B adds fourteen valid H/s; I adds fourteen late-bad-ML H/s. Each line below is B−I ",
        "with a paired 95% block-bootstrap interval; the fresh four-validator network is the unit.", "",
        "| Measure | B−I mean | 95% CI | Unit |", "| --- | ---: | ---: | --- |"]
    units = {"app_cpu_s": "s/run", "comet_cpu_s": "s/run", "observer_cpu_s": "s/run",
             "generator_cpu_s": "s/run", "relay_send_v0_bytes": "B/run",
             "accepted_mempool_v0_peak_bytes": "B", "socket_queue_v0_cumulative_s": "s/run",
             "accepted_residence_v0_mean_ms": "ms/accepted tx",
             "fixed_valid_mean_observation_upper_ms": "ms/fixed valid tx",
             "fixed_valid_timely_fraction": "fraction"}
    for field in MEASURES:
        effect = aggregate["paired_B_minus_I"][field]
        lines.append(f"| `{field}` | {effect['mean']:.6g} | "
                     f"[{effect['ci95'][0]:.6g}, {effect['ci95'][1]:.6g}] | {units[field]} |")
    lines.extend(["", "Instrumented Linux thread-clock stage CPU and per-run resource curves are "
        "in result_aggregate.json and figures/. The stage probe's unperturbed CPU bias is "
        "unavailable; its values must not be used as exact native crypto CPU shares. "
        "The socket queue measure is only the ABCI socket subqueue. Accepted mempool "
        "residence includes proposal/commit scheduling and is not pure queue wait; "
        "the B and I residence averages cover different accepted transaction sets. "
        "Relay bytes are envelope counters, not transaction-only bytes. "
        "These are single-host RPC-origin workload effects; no exact bottleneck, "
        "capacity knee, DoS or distributed effect is established.", ""])
    return "\n".join(lines)


def figure_bytes(aggregate):
    """Render only QC-passing observed data; keep labels faithful to source."""
    if aggregate["status"] != "ANALYZED":
        return {}
    import matplotlib
    matplotlib.use("Agg")
    import matplotlib.pyplot as plt

    curves = aggregate["validator0_resource_curves"]
    fig, axes = plt.subplots(3, 1, figsize=(8, 9), constrained_layout=True)
    fields = (("accepted_mempool_bytes", "Accepted mempool bytes"),
              ("app_cpu_cumulative_s", "Validator-0 app CPU, cumulative s"),
              ("comet_cpu_cumulative_s", "Validator-0 Comet CPU, cumulative s"))
    for axis, (field, label) in zip(axes, fields):
        for arm, color in (("B", "#1f77b4"), ("I", "#d62728")):
            axis.plot([row["sample_index"] for row in curves[arm]],
                      [row[field] for row in curves[arm]], label=arm, color=color)
        axis.set_ylabel(label)
        axis.grid(alpha=.2)
    axes[0].legend(title="20 tx/s arm")
    axes[-1].set_xlabel("Observer sample index (approximately 1 s apart)")
    fig.suptitle("X16 validator-0 resource curves, 24 networks per arm")
    resource = io.BytesIO()
    fig.savefig(resource, format="png", dpi=150, metadata={"Software": "file02_x16_results.py"})
    plt.close(fig)

    stages = ("structural", "binding", "digest", "secp", "ml", "state", "state_copy")
    fig, axis = plt.subplots(figsize=(9, 5), constrained_layout=True)
    for arm, shift, color in (("B", -.16, "#1f77b4"), ("I", .16, "#d62728")):
        values = [aggregate["stage_thread_cpu_ns_by_arm"][arm][name] for name in stages]
        x = [index+shift for index in range(len(stages))]
        mean = [value["mean"]/1000 for value in values]
        lower = [(value["mean"]-value["ci95"][0])/1000 for value in values]
        upper = [(value["ci95"][1]-value["mean"])/1000 for value in values]
        axis.errorbar(x, mean, yerr=[lower, upper], fmt="o", capsize=3,
                      color=color, label=arm)
    axis.set_xticks(range(len(stages)), stages, rotation=30)
    axis.set_yscale("log")
    axis.set_ylabel("Instrumented thread CPU per observed stage (µs)")
    axis.set_title("X16 CheckTx stage CPU, run-level 95% bootstrap CI")
    axis.grid(axis="y", alpha=.2)
    axis.legend(title="20 tx/s arm")
    stage_image = io.BytesIO()
    fig.savefig(stage_image, format="png", dpi=150, metadata={"Software": "file02_x16_results.py"})
    plt.close(fig)
    return {"resource_curve.png": resource.getvalue(), "stage_thread_cpu.png": stage_image.getvalue()}


def write_bytes_once(path, raw):
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(descriptor, "wb") as stream:
        stream.write(raw)


def figure_manifest(aggregate, pictures):
    import matplotlib
    return {"schema": "pluto.file02.x16_figures.v1",
        "raw_manifest_sha256": aggregate["raw_manifest_sha256"],
        "matplotlib_version": matplotlib.__version__,
        "files": {name: hashlib.sha256(raw).hexdigest() for name, raw in pictures.items()}}


def write_outputs(session):
    rows, qc, aggregate = derive(session)
    common.write_once(session / "result_per_run.json", json.dumps({
        "schema": "pluto.file02.x16_per_run.v1", "rows": rows}, indent=2, sort_keys=True) + "\n")
    common.write_once(session / "result_per_run.csv", csv_text(rows))
    common.write_once(session / "qc_report.json", json.dumps(qc, indent=2, sort_keys=True) + "\n")
    common.write_once(session / "result_aggregate.json", json.dumps(aggregate, indent=2, sort_keys=True) + "\n")
    figures = session / "figures"
    figures.mkdir(mode=0o700)
    pictures = figure_bytes(aggregate)
    if pictures:
        for name, raw in pictures.items():
            write_bytes_once(figures / name, raw)
        common.write_once(figures / "manifest.json", json.dumps(
            figure_manifest(aggregate, pictures), indent=2, sort_keys=True) + "\n")
    else:
        common.write_once(figures / "status.json", json.dumps({"status": "WITHHELD_QC_FAILURE"}) + "\n")
    common.write_once(session / "decision.md", decision_text(qc, aggregate))
    return qc


def verify_outputs(session):
    rows, qc, aggregate = derive(session)
    require(common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.x16_per_run.v1", "rows": rows}, "X16 per-run drift")
    require((session / "result_per_run.csv").read_text() == csv_text(rows), "X16 CSV drift")
    require(common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate and
            (session / "decision.md").read_text() == decision_text(qc, aggregate), "X16 output drift")
    pictures = figure_bytes(aggregate)
    figures = session / "figures"
    if pictures:
        require(common.read_json(figures / "manifest.json") == figure_manifest(aggregate, pictures) and
                all((figures / name).read_bytes() == raw for name, raw in pictures.items()),
                "X16 figure drift")
    else:
        require(common.read_json(figures / "status.json")["status"] == "WITHHELD_QC_FAILURE",
                "X16 QC-failed figures must be withheld")
    return qc


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("analyze", "verify"))
    parser.add_argument("session")
    args = parser.parse_args()
    try:
        session = locked.session_path(args.session)
        qc = write_outputs(session) if args.command == "analyze" else verify_outputs(session)
        print(f"X16_{args.command.upper()}=OK QC={qc['status']} runs={qc['n_valid']} pairs={qc['n_pairs']} raw={qc['raw_manifest_sha256']}")
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError,
            AssertionError, ZeroDivisionError, subprocess.CalledProcessError) as exc:
        print(f"X16_RESULT_ERROR: {exc}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
