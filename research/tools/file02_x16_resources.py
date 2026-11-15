#!/usr/bin/env python3
"""Independent descriptive X10 app/Comet/producer/relay resource reanalysis."""

import argparse
import json
from pathlib import Path
import statistics
import sys

import file02_common as common
import file02_x10 as x10

ROOT = Path(__file__).resolve().parents[2]
SESSION = ROOT / "research/artifacts/file02/x10-20261115T051140Z-3d69d6eb"
OUT = ROOT / "research/evidence/file02/mechanism/X16_RESOURCE_DECOMPOSITION.json"


def counter_delta(samples, key):
    values = [s[key] for s in samples if s.get(key) is not None]
    if len(values) < 2 or values[-1] < values[0]:
        return None
    return values[-1]-values[0]


def run_resources(raw, row):
    run = raw / "runs" / row["run_id"]
    nodes = common.read_json(run / "nodes.json")
    comet_cpu = 0
    for node, identity in enumerate(nodes):
        series = x10.read_jsonl(run / f"observer-{node}.jsonl")
        procs = [s.get("comet_process") for s in series if s.get("comet_process")]
        if (len(procs) < 2 or any(p["pid"] != identity["comet_pid"] or
                                  p["start_ticks"] != procs[0]["start_ticks"] for p in procs)
                or procs[-1]["cpu_ns"] < procs[0]["cpu_ns"]):
            raise ValueError(f"{row['run_id']} Comet process identity/CPU")
        comet_cpu += procs[-1]["cpu_ns"]-procs[0]["cpu_ns"]
    v0 = x10.read_jsonl(run / "observer-0.jsonl")
    generator = common.read_json(run / "generator_resources.json")
    oracle = common.read_json(run / "oracle.json")
    relay = counter_delta(v0, "relay_send_envelope_bytes")
    socket = counter_delta(v0, "checktx_socket_queue_seconds")
    if relay is None or socket is None or generator["run_id"] != row["run_id"]:
        raise ValueError(f"{row['run_id']} missing producer/relay/socket samples")
    return {"run_id": row["run_id"], "block": row["block"], "condition": row["condition"],
        "app_cpu_s": sum(row["app_process_cpu_delta_ns"])/1e9,
        "comet_cpu_s": comet_cpu/1e9,
        "generator_cpu_s": (generator["user_cpu_ns"]+generator["system_cpu_ns"])/1e9,
        "relay_send_v0_delta_bytes": relay,
        "accepted_mempool_v0_peak_bytes": max(s["accepted_mempool_bytes"] for s in v0),
        "checktx_socket_queue_v0_delta_s": socket,
        "valid_committed": oracle["valid_committed"], "invalid_rejected": oracle["invalid_rejected"],
        "v0_sample_count": len(v0)}


def derive():
    plan = common.read_json(SESSION / "plan.json")
    x10.validate_plan(SESSION, plan)
    rows, qc, _ = x10.derive(SESSION, plan)
    if qc["status"] != "PASS" or len(rows) != 72:
        raise ValueError("X10 raw QC not PASS")
    raw = SESSION / "raw"
    resources = [run_resources(raw, row) for row in rows]
    by = {block: {} for block in range(24)}
    for r in resources:
        by[r["block"]][r["condition"]] = r
    fields = ("app_cpu_s", "comet_cpu_s", "generator_cpu_s", "relay_send_v0_delta_bytes",
              "accepted_mempool_v0_peak_bytes", "checktx_socket_queue_v0_delta_s")
    arm_mean = {arm: {field: statistics.mean(by[b][arm][field] for b in range(24))
                      for field in fields} for arm in "LBI"}
    b_minus_i = {}
    for field in fields:
        values = [by[b]["B"][field]-by[b]["I"][field] for b in range(24)]
        b_minus_i[field] = {"mean": statistics.mean(values), "ci95": x10.paired_ci(values)}
    curves = {}
    for arm in "LBI":
        series = [x10.read_jsonl(raw / "runs" / f"X10-b{b:02d}-{arm}" / "observer-0.jsonl")
                  for b in range(24)]
        count = min(map(len, series))
        curves[arm] = [{"sample_index": i, "n_networks": 24,
            "mean_mempool_bytes": statistics.mean(s[i]["accepted_mempool_bytes"] for s in series),
            "mean_comet_cpu_cumulative_s": statistics.mean(
                (s[i]["comet_process"]["cpu_ns"]-s[0]["comet_process"]["cpu_ns"])/1e9
                for s in series)} for i in range(count)]
    return {"schema": "pluto.file02.x16_resources.v1", "status": "DESCRIPTIVE",
        "source": "X10 successful 24-block session", "raw_manifest_sha256": qc["raw_manifest_sha256"],
        "n_networks": 72, "n_blocks": 24, "arm_means": arm_mean,
        "paired_B_minus_I": b_minus_i, "per_run": resources,
        "validator0_sample_index_curves": curves,
        "observer_cpu_s": None, "observer_cpu_reason": "UNAVAILABLE_OBSERVER_SELF_CPU_IN_X10",
        "stage_cpu_ns": None, "stage_cpu_reason": "UNAVAILABLE_NETWORK_STAGE_CPU_ATTRIBUTION",
        "full_queue_residence_ms": None, "full_queue_reason": "UNAVAILABLE_COMET_TO_APP_QUEUE_LINKAGE",
        "note": "CPU intervals include all process work; relay counter includes envelope bytes, not transaction-only bytes"}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("write", "check"))
    args = parser.parse_args()
    try:
        result = derive()
        if args.command == "write":
            common.write_once(OUT, json.dumps(result, indent=2, sort_keys=True) + "\n")
        elif common.read_json(OUT) != result:
            raise ValueError("X16 resource decomposition drift")
        print(f"X16_{args.command.upper()}=OK")
        return 0
    except (OSError, ValueError, KeyError, TypeError) as exc:
        print(f"X16_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
