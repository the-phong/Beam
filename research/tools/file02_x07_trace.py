#!/usr/bin/env python3
"""Independently verify bounded X07 engineering trace and derive intervals."""

import argparse
from collections import Counter, defaultdict
from datetime import datetime, timezone
import json
from pathlib import Path
import re
import statistics
import sys

import file02_common as common

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
SCHEMA = "pluto.file02.x07_comet_trace.v1"
UTC_RE = re.compile(r"^(\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d)\.(\d{1,9})Z$")


def require(condition, reason):
    if not condition:
        raise ValueError(reason)


def read_lines(path):
    raw = path.read_bytes()
    require(raw and raw.endswith(b"\n"), f"partial/empty JSONL: {path.name}")
    return [json.loads(line) for line in raw.splitlines()]


def utc_ns(value):
    match = UTC_RE.fullmatch(value)
    require(match is not None, "invalid UTC nanosecond timestamp")
    return (int(datetime.fromisoformat(match.group(1)).replace(tzinfo=timezone.utc).timestamp()) * 1_000_000_000 +
            int(match.group(2).ljust(9, "0")))


def trace_by_tx(path, pid):
    rows = read_lines(path)
    grouped = defaultdict(list)
    for index, item in enumerate(rows, 1):
        require(item.get("schema") == SCHEMA and item.get("pid") == pid and
                item.get("sequence") == index and isinstance(item.get("monotonic_ns"), int) and
                item["monotonic_ns"] >= 0 and len(item.get("tx_sha256", "")) == 64 and
                item.get("wire_bytes", 0) > 0, f"trace schema/order/PID: {path.name}:{index}")
        if "host_monotonic_ns" in item:
            require(isinstance(item["host_monotonic_ns"], int) and
                    item["host_monotonic_ns"] > 0, f"host clock: {path.name}:{index}")
        utc_ns(item["utc"])
        grouped[item["tx_sha256"]].append(item)
    return rows, grouped


def one_stage(items, name):
    found = [item for item in items if item["event"] == name]
    require(len(found) == 1, f"{name}: expected exactly one, got {len(found)}")
    return found[0]["monotonic_ns"]


def first_stage(items, name):
    found = [item for item in items if item["event"] == name]
    require(found, f"missing {name}")
    return found[0]["monotonic_ns"]


def validate_origin(plan, grouped, app_events):
    planned = {item["sha256"]: item for item in plan["events"]}
    require(len(planned) == len(plan["events"]) == 241, "duplicate/missing workload hash")
    require({tx for tx, events in grouped.items() if any(e["event"] == "rpc_sync_entry" for e in events)} ==
            set(planned), "RPC trace/workload hash mismatch")
    new_app = {item["tx_sha256"]: item for item in app_events if item["check_type"] == "new"}
    require(set(new_app) == set(planned), "app CheckTx/workload hash mismatch")
    intervals = []
    for txid, item in planned.items():
        events = grouped[txid]
        names = ("rpc_sync_entry", "mempool_checktx_entry", "precheck_pass",
                 "abci_call", "socket_queue_attempt", "socket_dequeue",
                 "checktx_callback", "rpc_sync_return")
        points = [one_stage(events, name) if name in ("rpc_sync_entry", "abci_call",
                  "checktx_callback", "rpc_sync_return") else first_stage(events, name) for name in names]
        require(points == sorted(points), f"nonmonotonic path: {txid}")
        callback = next(e for e in events if e["event"] == "checktx_callback")
        require(callback["code"] == int(item["expected_code"]) == new_app[txid]["code"],
                f"CheckTx code mismatch: {txid}")
        for stamp in ("app_ingress_utc", "app_lock_acquired_utc", "app_result_utc"):
            require(stamp in new_app[txid], f"app timestamp missing: {txid}")
        app_points = [utc_ns(new_app[txid][stamp])
                      for stamp in ("app_ingress_utc", "app_lock_acquired_utc", "app_result_utc")]
        require(app_points == sorted(app_points), f"app timestamp order: {txid}")
        accepted = [e for e in events if e["event"] == "mempool_accepted"]
        removed = [e for e in events if e["event"] == "mempool_removed"]
        relays = [e for e in events if e["event"] == "p2p_send_handoff"]
        if callback["code"] == 0:
            require(len(accepted) == len(removed) == 1 and len(relays) == 3 and
                    len([e for e in events if e["event"] == "committed_update"]) == 1 and
                    points[6] <= accepted[0]["monotonic_ns"] <= removed[0]["monotonic_ns"],
                    f"accepted path incomplete: {txid}")
            residence = removed[0]["monotonic_ns"] - accepted[0]["monotonic_ns"]
        else:
            require(not accepted and not removed and not relays and
                    len([e for e in events if e["event"] == "mempool_app_reject"]) == 1,
                    f"rejected path contamination: {txid}")
            residence = None
        intervals.append({"tx_sha256": txid, "code": callback["code"],
            "rpc_to_precheck_ns": points[2]-points[0],
            "socket_queue_ns": points[5]-points[4],
            "socket_dequeue_to_callback_ns": points[6]-points[5],
            "rpc_roundtrip_comet_ns": points[7]-points[0],
            "app_lock_wait_ns": new_app[txid]["app_lock_wait_ns"],
            "app_service_ns": new_app[txid]["service_ns"],
            "accepted_mempool_residence_ns": residence,
            "p2p_send_handoffs": len(relays)})
        if all("host_monotonic_ns" in e for e in events) and all(
                new_app[txid].get(name) is not None for name in
                ("app_ingress_host_monotonic_ns", "app_lock_host_monotonic_ns",
                 "app_result_host_monotonic_ns")):
            host = {}
            for e in events:
                if e["event"] in ("rpc_sync_entry", "socket_dequeue", "checktx_callback"):
                    host.setdefault(e["event"], e["host_monotonic_ns"])
            ingress = new_app[txid]["app_ingress_host_monotonic_ns"]
            locked = new_app[txid]["app_lock_host_monotonic_ns"]
            result = new_app[txid]["app_result_host_monotonic_ns"]
            require(host["rpc_sync_entry"] <= host["socket_dequeue"] <= ingress <=
                    locked <= result <= host["checktx_callback"],
                    f"cross-process host clock path: {txid}")
            intervals[-1].update({"rpc_to_app_ingress_ns": ingress-host["rpc_sync_entry"],
                "socket_dequeue_to_app_ingress_ns": ingress-host["socket_dequeue"],
                "app_result_to_comet_callback_ns": host["checktx_callback"]-result,
                "app_lock_host_ns": locked-ingress})
    return intervals


def audit(session):
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve() and session.is_dir(), "private session path")
    manifest = common.verify_raw(session / "raw")
    top = common.read_json(session / "plan.json")
    require(top.get("schema") == "pluto.file02.x07_qa_plan.v1" and
            top.get("status") == "LOCKED" and top.get("scope") == "QA_ONLY" and
            top.get("conditions") == ["OFF", "ON"] and
            top.get("rate_total_tx_s") == 20 and top.get("safety_caps") ==
            {"max_tx_per_second": 25, "max_bytes_per_second": 65536,
             "max_outstanding": 16, "max_duration_seconds": 60}, "X07 locked plan drift")
    results = {}
    for mode in top["conditions"]:
        run = session / "raw/runs" / f"X07-QA-I-{mode}"
        meta, workload, oracle, result = (common.read_json(run / name) for name in
                                         ("metadata.json", "plan.json", "oracle.json", "result.json"))
        nodes = common.read_json(run / "nodes.json")
        require(meta["source_commit"] == top["source_commit"] and
                meta["source_tree"] == top["source_tree"] and
                meta["trace_mode"] == mode and meta["scope"] == "QA_ONLY" and
                meta["case_id"] == "X07" and len(nodes) == 4 and
                oracle["status"] == "OK" and oracle["valid_committed"] == 73 and
                oracle["invalid_rejected"] == 168 and result["status"] == "OK" and
                result["actual_send_count"] == 241 and
                common.digest(run / "config.json") == top["config_sha256"],
                f"{mode} metadata/oracle/config QC")
        app = [read_lines(run / f"app-{n}-events.jsonl") for n in range(4)]
        resources = [read_lines(run / f"app-{n}-resources.jsonl") for n in range(4)]
        require(all(all(sample["recorder_drops"] == 0 for sample in series) for series in resources),
                f"{mode} app recorder drop")
        require(all(not (run / f"comet-{n}-trace.jsonl").exists() for n in range(4)) if mode == "OFF"
                else all((run / f"comet-{n}-trace.jsonl").exists() for n in range(4)),
                f"{mode} trace file mode")
        if mode == "ON":
            traces = [trace_by_tx(run / f"comet-{n}-trace.jsonl", nodes[n]["comet_pid"])
                      for n in range(4)]
            intervals = validate_origin(workload, traces[0][1], app[0])
            require(len([x for x in intervals if x["code"] == 0]) == 73 and
                    len([x for x in intervals if x["code"] == 13]) == 168,
                    "X07 path count")
            for n in range(1, 4):
                require(len([e for e in traces[n][0] if e["event"] == "mempool_accepted"]) == 73 and
                        len([e for e in traces[n][0] if e["event"] == "mempool_removed"]) == 73,
                        f"validator {n} accepted/residence coverage")
            results[mode] = {"run_id": workload["run_id"], "trace_events_by_node":
                             [len(x[0]) for x in traces], "intervals": intervals}
        else:
            results[mode] = {"run_id": workload["run_id"]}
        results[mode]["app_cpu_s_by_node"] = [
            (series[-1]["user_cpu_ns"]+series[-1]["system_cpu_ns"]-
             series[0]["user_cpu_ns"]-series[0]["system_cpu_ns"])/1e9 for series in resources]
        results[mode]["oracle"] = oracle
        results[mode]["genesis_sha256"] = result["genesis_sha256"]
        results[mode]["planned_tx_sha256"] = [event["sha256"] for event in workload["events"]]
    require(results["OFF"]["genesis_sha256"] == results["ON"]["genesis_sha256"] and
            results["OFF"]["planned_tx_sha256"] == results["ON"]["planned_tx_sha256"] and
            results["OFF"]["oracle"]["app_hash"] == results["ON"]["oracle"]["app_hash"],
            "ON/OFF fixture/genesis/AppHash mismatch")
    on = results["ON"]["intervals"]
    summary = {"schema": "pluto.file02.x07_qa_result.v1", "status": "PASS_QA_ONLY",
        "raw_manifest_sha256": manifest, "plan_sha256": common.digest(session / "plan.json"),
        "run_ids": [results[x]["run_id"] for x in ("OFF", "ON")],
        "trace_events_by_node": results["ON"]["trace_events_by_node"],
        "valid_paths": 73, "invalid_paths": 168,
        "socket_queue_ns_mean": statistics.mean(x["socket_queue_ns"] for x in on),
        "accepted_residence_ns_mean": statistics.mean(x["accepted_mempool_residence_ns"]
                                                 for x in on if x["code"] == 0),
        "app_cpu_s_by_mode": {mode: results[mode]["app_cpu_s_by_node"] for mode in ("OFF", "ON")},
        "cross_process_precise_queue_ns": None,
        "cross_process_reason": "UNAVAILABLE_CALIBRATED_APP_COMET_CLOCK_BOUND",
        "note": "single ON/OFF QA pair; no inferential observer-overhead claim"}
    if all("rpc_to_app_ingress_ns" in x for x in on):
        # This interval includes precheck and transport work; it is not a
        # pure queue residence and must never be labelled as one.
        summary["schema"] = "pluto.file02.x07_qa_result.v2"
        summary.pop("cross_process_precise_queue_ns")
        summary.pop("cross_process_reason")
        summary["rpc_to_app_ingress_ns_mean"] = statistics.mean(
            x["rpc_to_app_ingress_ns"] for x in on)
        summary["full_transaction_queue_ns"] = None
        summary["full_transaction_queue_reason"] = "UNAVAILABLE_UNINSTRUMENTED_P2P_AND_CONSENSUS_QUEUE_SUBINTERVALS"
        summary["supersedes"] = "x07_qa_result.json; prior derived label misidentified RPC-to-app interval as queue"
        summary["clock_semantics"] = "same-host Linux CLOCK_MONOTONIC; timestamp placement and tracer overhead remain measurement uncertainty"
    return summary


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("analyze", "verify"))
    parser.add_argument("session")
    args = parser.parse_args()
    try:
        result = audit(args.session)
        target = Path(args.session) / ("x07_qa_result_v2.json" if
            result["schema"].endswith(".v2") else "x07_qa_result.json")
        if args.command == "analyze":
            common.write_once(target, json.dumps(result, indent=2, sort_keys=True) + "\n")
        else:
            require(common.read_json(target) == result, "X07 derived result drift")
        print(f"X07_{args.command.upper()}=OK status={result['status']} raw={result['raw_manifest_sha256']}")
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError, AssertionError) as exc:
        print(f"X07_ERROR: {exc}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
