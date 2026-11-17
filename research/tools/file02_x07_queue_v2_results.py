#!/usr/bin/env python3
"""Independent X07 queue-v2 raw/QC and balanced trace-overhead verifier."""

import argparse
from collections import defaultdict
import json
from pathlib import Path
import random
import statistics
import subprocess
import sys

import file02_common as common
import file02_x07_queue_v2 as locked
import file02_x07_trace as x07
import file02_x16_stage as stage

ROOT = Path(__file__).resolve().parents[2]
OVERHEAD_FIELDS = ("app_cpu_s", "comet_cpu_s", "observer_cpu_s",
                   "producer_rpc_median_ms", "fixed_valid_observation_upper_mean_ms")
TRACE_FIELDS = ("socket_client_attempt_to_dequeue_ns_mean",
                "client_dequeue_to_server_read_ns_mean", "server_mutex_wait_ns_mean",
                "app_lock_wait_ns_mean", "app_service_ns_mean",
                "server_return_to_client_callback_ns_mean",
                "accepted_mempool_residence_ns_mean",
                "p2p_queue_attempt_to_dequeue_ns_mean")


def require(condition, reason):
    if not condition:
        raise ValueError(reason)


def percentile(values, fraction):
    values = sorted(values)
    position = (len(values)-1)*fraction
    lower = int(position)
    weight = position-lower
    return values[lower]*(1-weight)+values[min(lower+1,len(values)-1)]*weight


def ci(values):
    require(len(values) == 8, "paired overhead CI requires all eight blocks")
    rng = random.Random(20261117)
    draws = sorted(statistics.mean(rng.choice(values) for _ in values) for _ in range(20000))
    return [percentile(draws, .025), percentile(draws, .975)]


def first_event(events, name):
    found = [event for event in events if event["event"] == name]
    require(found, f"missing {name}")
    return found[0]


def trace_intervals(workload, comet, server, app):
    origin = x07.validate_origin(workload, comet[0][1], app[0])
    planned = {event["sha256"]: event for event in workload["events"]}
    require(len(origin) == 241, "origin path count")
    server_events = server[0][1]
    require(set(planned).issubset(server_events), "app server tx coverage")
    intervals = []
    for path in origin:
        tx = path["tx_sha256"]
        s = server_events[tx]
        c = comet[0][1][tx]
        read, attempt, acquired, app_return, response = (
            first_event(s, name) for name in ("server_socket_read", "server_mutex_attempt",
                                                "server_mutex_acquired", "server_app_return",
                                                "server_response_handoff"))
        client_dequeue = first_event(c, "socket_dequeue")
        callback = first_event(c, "checktx_callback")
        app_event = next(event for event in app[0] if event["tx_sha256"] == tx and
                         event["check_type"] == "new")
        require(all(event["pid"] == server[0][0][0]["pid"] for event in
                    (read, attempt, acquired, app_return, response)) and
                read["monotonic_ns"] <= attempt["monotonic_ns"] <=
                acquired["monotonic_ns"] <= app_return["monotonic_ns"] <=
                response["monotonic_ns"] and
                client_dequeue["host_monotonic_ns"] <= read["host_monotonic_ns"] <=
                acquired["host_monotonic_ns"] <=
                app_event["app_ingress_host_monotonic_ns"] <=
                app_event["app_result_host_monotonic_ns"] <=
                app_return["host_monotonic_ns"] <=
                callback["host_monotonic_ns"] and
                app_return["code"] == path["code"], f"ABCI server/client path {tx}")
        intervals.append({"tx_sha256": tx, "code": path["code"],
            "socket_client_attempt_to_dequeue_ns": path["socket_queue_ns"],
            "client_dequeue_to_server_read_ns": read["host_monotonic_ns"]-
                                                client_dequeue["host_monotonic_ns"],
            "server_mutex_wait_ns": acquired["monotonic_ns"]-attempt["monotonic_ns"],
            "app_lock_wait_ns": path["app_lock_wait_ns"],
            "app_service_ns": path["app_service_ns"],
            "server_return_to_client_callback_ns": callback["host_monotonic_ns"]-
                                                    app_return["host_monotonic_ns"],
            "accepted_mempool_residence_ns": path["accepted_mempool_residence_ns"]})
    # The envelope map records a tx hash before peer.Send. The connection
    # queue records only the wrapped envelope hash and per-channel token.
    mappings = defaultdict(set)
    for event in comet[0][0]:
        if event["event"] == "p2p_envelope_map":
            require(event.get("envelope_sha256") and event["tx_sha256"] in planned,
                    "unmatched P2P envelope map")
            mappings[event["tx_sha256"]].add(event["envelope_sha256"])
    queue = defaultdict(lambda: defaultdict(list))
    for event in comet[0][0]:
        if event["event"] in ("p2p_queue_attempt", "p2p_queue_accepted", "p2p_queue_dequeue"):
            envelope, channel = event.get("envelope_sha256"), event.get("channel_id")
            require(envelope and isinstance(channel, int) and channel > 0 and
                    event["tx_sha256"] == envelope, "P2P envelope/channel schema")
            queue[(envelope, channel)][event["event"]].append(event)
    p2p_bounds, accepted_races, success_count = [], 0, 0
    for path in intervals:
        tx = path["tx_sha256"]
        if path["code"] != 0:
            require(tx not in mappings, f"invalid tx entered P2P map: {tx}")
            continue
        require(len(mappings[tx]) == 1, f"valid tx envelope map: {tx}")
        envelope = next(iter(mappings[tx]))
        channels = [events for (digest, _), events in queue.items() if digest == envelope and
                    events.get("p2p_queue_accepted")]
        require(len(channels) == 3, f"valid tx peer queue coverage: {tx}")
        for events in channels:
            attempt = events["p2p_queue_attempt"]
            accepted = events["p2p_queue_accepted"]
            dequeue = events["p2p_queue_dequeue"]
            require(len(attempt) == len(accepted) == len(dequeue) == 1 and
                    attempt[0]["monotonic_ns"] <= dequeue[0]["monotonic_ns"],
                    f"P2P queue attempt/dequeue: {tx}")
            if accepted[0]["monotonic_ns"] > dequeue[0]["monotonic_ns"]:
                accepted_races += 1
            p2p_bounds.append(dequeue[0]["monotonic_ns"]-attempt[0]["monotonic_ns"])
            success_count += 1
    require(success_count == 219, "P2P queue total success coverage")
    for n in range(1, 4):
        require(len([event for event in comet[n][0] if event["event"] == "p2p_receive"]) >= 73,
                f"validator {n} P2P receiver coverage")
    return {name+"_mean": statistics.mean(row[name] for row in intervals if row[name] is not None)
            for name in ("socket_client_attempt_to_dequeue_ns",
                         "client_dequeue_to_server_read_ns", "server_mutex_wait_ns",
                         "app_lock_wait_ns", "app_service_ns",
                         "server_return_to_client_callback_ns",
                         "accepted_mempool_residence_ns")} | {
            "p2p_queue_attempt_to_dequeue_ns_mean": statistics.mean(p2p_bounds),
            "p2p_accepted_after_dequeue_race_count": accepted_races,
            "p2p_queue_success_count": success_count,
            "linked_new_paths": 241}


def audit_run(raw, job, plan):
    run = raw / "runs" / job["run_id"]
    meta, workload, result, oracle, commits, nodes = (
        common.read_json(run / name) for name in
        ("metadata.json", "plan.json", "result.json", "oracle.json",
         "commit_observations.json", "nodes.json"))
    mode = job["mode"]
    require(meta.get("schema") == "pluto.file02.x07_v2_run.v1" and
            meta.get("scope") == "QA_ONLY" and meta.get("case_id") == "X07" and
            meta.get("x07_queue_v2") is True and meta.get("block") == job["block"] and
            meta.get("order") == job["order"] and meta.get("trace_mode") == mode and
            meta.get("run_id") == job["run_id"] and
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
            all(event["source"] == "rpc" and len(event["bytes_hex"]) == 5136
                for event in workload["events"]), f"{job['run_id']} provenance/shape/oracle")
    expected = {event["id"]: int(event["expected_code"]) for event in workload["events"]}
    require(len(expected) == len(result["events"]) == 241 and
            all(event["actual_code"] == expected[event["id"]] for event in result["events"]),
            f"{job['run_id']} result codes")
    app = [x07.read_lines(run / f"app-{n}-events.jsonl") for n in range(4)]
    app_resource = [x07.read_lines(run / f"app-{n}-resources.jsonl") for n in range(4)]
    observers = [x07.read_lines(run / f"observer-{n}.jsonl") for n in range(4)]
    require(all(all(row["recorder_drops"] == 0 and row["pid"] == nodes[n]["app_pid"]
                    for row in app_resource[n]) for n in range(4)) and
            all(all(row["dropped_samples"] == 0 for row in observers[n])
                for n in range(4)), f"{job['run_id']} PID/drop QC")
    app_cpu = sum(series[-1]["user_cpu_ns"]+series[-1]["system_cpu_ns"]-
                  series[0]["user_cpu_ns"]-series[0]["system_cpu_ns"]
                  for series in app_resource)/1e9
    require(app_cpu >= 0, f"{job['run_id']} app CPU reset")
    comet_cpu = sum(stage.process_delta(observers[n], "comet_process", nodes[n]["comet_pid"])
                    for n in range(4))/1e9
    observer_cpu = sum(stage.process_delta(observers[n], "observer_process", nodes[n]["observer_pid"])
                       for n in range(4))/1e9
    rpc = [(x07.utc_ns(event["completed_utc"])-x07.utc_ns(event["actual_send_utc"]))/1e6
           for event in result["events"]]
    require(all(0 <= value < 60000 for value in rpc), f"{job['run_id']} RPC clock order")
    fixed = {event["id"]: event for event in workload["events"] if
             event["scenario"] == "fixed_legitimate"}
    observed = {event["id"]: event for event in commits["events"]}
    submitted = {event["id"]: event for event in result["events"]}
    require(len(fixed) == len(commits["events"]) == 73 and
            commits.get("poll_interval_ms") == 100, f"{job['run_id']} fixed coverage")
    latency = [(observed[eid]["all_four_seen_monotonic_ns"]-
                submitted[eid]["actual_send_monotonic_ns"])/1e6 for eid in fixed]
    require(all(0 <= value < 60000 for value in latency), f"{job['run_id']} service clock order")
    traces = None
    if mode == "ON":
        comet = [x07.trace_by_tx(run / f"comet-{n}-trace.jsonl", nodes[n]["comet_pid"])
                 for n in range(4)]
        server = [x07.trace_by_tx(run / f"app-server-{n}-trace.jsonl", nodes[n]["app_pid"])
                  for n in range(4)]
        traces = trace_intervals(workload, comet, server, app)
    else:
        require(all(not (run / f"comet-{n}-trace.jsonl").exists() and
                    not (run / f"app-server-{n}-trace.jsonl").exists()
                    for n in range(4)), f"{job['run_id']} OFF trace presence")
    return {"run_id": job["run_id"], "block": job["block"], "order": job["order"],
            "mode": mode, "genesis_sha256": result["genesis_sha256"],
            "app_hash": oracle["app_hash"],
            "planned_stream": [[event["sha256"], event["offset_ns"], event["expected_code"]]
                               for event in workload["events"]],
            "app_cpu_s": app_cpu, "comet_cpu_s": comet_cpu, "observer_cpu_s": observer_cpu,
            "producer_rpc_median_ms": statistics.median(rpc),
            "fixed_valid_observation_upper_mean_ms": statistics.mean(latency),
            "trace": traces}


def derive(session):
    session = Path(session).resolve()
    require(session.parent == locked.ARTIFACTS.resolve() and session.is_dir(),
            "private X07 v2 session path")
    manifest = common.verify_raw(session / "raw")
    plan = common.read_json(session / "plan.json")
    require(plan.get("schema") == "pluto.file02.x07_v2_plan.v1" and
            plan.get("status") == "LOCKED" and plan.get("n_pairs") == 8 and
            plan.get("n_networks") == 16 and plan.get("safety_caps") == [25, 65536, 16, 60] and
            plan.get("ci_method") == "paired_block_percentile_bootstrap_20000_seed_20261117" and
            common.digest(session / "raw/config.json") == plan["config_sha256"] and
            common.read_json(session / "raw/schedule.json") == locked.schedule() and
            common.digest(session / "raw/schedule.json") == plan["schedule_sha256"],
            "X07 v2 locked plan/config/schedule")
    tree = subprocess.run(["git", "rev-parse", f"{plan['source_commit']}^{{tree}}"],
                          cwd=ROOT, capture_output=True, text=True, check=True).stdout.strip()
    require(tree == plan["source_tree"], "X07 v2 source commit/tree")
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
                pair["OFF"]["app_hash"] == pair["ON"]["app_hash"] and
                pair["OFF"]["planned_stream"] == pair["ON"]["planned_stream"]}
    for block in range(8):
        if block not in complete:
            failures.append({"block": block, "reason": "missing/unmatched pair; no replacement"})
    status = "PASS" if len(rows) == 16 and len(complete) == 8 and not failures else "INVALID_FOR_INFERENCE"
    qc = {"schema": "pluto.file02.x07_v2_qc.v1", "status": status,
          "n_valid": len(rows), "n_pairs": len(complete), "n_planned": 16,
          "failures": failures, "raw_manifest_sha256": manifest,
          "plan_sha256": common.digest(session / "plan.json")}
    aggregate = {"schema": "pluto.file02.x07_v2_aggregate.v1",
                 "status": "ANALYZED" if status == "PASS" else "WITHHELD_QC_FAILURE",
                 "raw_manifest_sha256": manifest, "n_pairs": len(complete),
                 "paired_ON_minus_OFF": None, "trace_on_segment_means_ns": None,
                 "full_transaction_queue_ns": None,
                 "full_transaction_queue_reason":
                    "PARALLEL_RELAY_AND_CONSENSUS_NO_SINGLE_ADDITIVE_QUEUE_SCALAR"}
    if status == "PASS":
        aggregate["paired_ON_minus_OFF"] = {field: {"mean": statistics.mean(values),
            "ci95": ci(values)} for field in OVERHEAD_FIELDS for values in [[
                complete[b]["ON"][field]-complete[b]["OFF"][field] for b in range(8)]]}
        aggregate["trace_on_segment_means_ns"] = {field: statistics.mean(
            complete[b]["ON"]["trace"][field] for b in range(8)) for field in TRACE_FIELDS}
        aggregate["p2p_queue_accepted_after_dequeue_race_total"] = sum(
            complete[b]["ON"]["trace"]["p2p_accepted_after_dequeue_race_count"]
            for b in range(8))
        aggregate["p2p_queue_success_total"] = sum(
            complete[b]["ON"]["trace"]["p2p_queue_success_count"] for b in range(8))
    return rows, qc, aggregate


def output(session, command):
    session = Path(session).resolve()
    rows, qc, aggregate = derive(session)
    paths = {"result_per_run.json": {"schema": "pluto.file02.x07_v2_per_run.v1", "rows": rows},
             "qc_report.json": qc, "result_aggregate.json": aggregate}
    if command == "analyze":
        for name, value in paths.items():
            common.write_once(session / name, json.dumps(value, indent=2, sort_keys=True) + "\n")
    else:
        require(all(common.read_json(session / name) == value for name, value in paths.items()),
                "X07 v2 derived output drift")
    return qc


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("analyze", "verify"))
    parser.add_argument("session")
    args = parser.parse_args()
    try:
        qc = output(args.session, args.command)
        print(f"X07_V2_{args.command.upper()}=OK QC={qc['status']} "
              f"runs={qc['n_valid']} pairs={qc['n_pairs']} raw={qc['raw_manifest_sha256']}")
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError,
            AssertionError, subprocess.CalledProcessError) as exc:
        print(f"X07_V2_RESULT_ERROR: {exc}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
