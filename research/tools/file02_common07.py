#!/usr/bin/env python3
"""Derive public per-run identity/rate records from a QC-verified T06 pilot."""

import argparse
import json
from pathlib import Path
import sys

import file02_common as common
import file02_t06_precision as precision


def record(metadata, plan, result):
    """Produce a public run record only from verified benign T06 input."""
    common.require(metadata["run_id"] == plan["run_id"] == result["run_id"] and
                   metadata["profile"] in precision.PROFILES and
                   metadata["scope"] == "QA_ONLY" and
                   plan["regime"] == "equal_tx_s", "run identity")
    warmup = plan["warmup_ns"]
    steady = plan["steady_ns"] / 1_000_000_000
    common.require(steady > 0 and len(plan["events"]) == len(result["events"]), "run duration/events")
    planned_count = achieved_count = planned_bytes = achieved_bytes = 0
    for expected, actual in zip(plan["events"], result["events"]):
        common.require(expected["id"] == actual["id"] and
                       expected["capability"] == "C_registered_key_holder" and
                       expected["traffic"] == actual["traffic"] == "legitimate" and
                       expected["source"] == actual["source"] == "rpc", "unsupported case/capability")
        wire_bytes = len(bytes.fromhex(expected["bytes_hex"]))
        common.require(actual["wire_bytes"] == wire_bytes, "wire size mismatch")
        if expected["offset_ns"] >= warmup:
            planned_count += 1
            planned_bytes += wire_bytes
        if actual["actual_offset_ns"] >= warmup:
            achieved_count += 1
            achieved_bytes += actual["wire_bytes"]
    common.require(planned_count == achieved_count == precision.RATE * precision.STEADY_SECONDS,
                   "steady count mismatch")
    return {"schema": "pluto.file02.common07_run_record.v1",
            "run_id": metadata["run_id"], "source_commit": metadata["source_commit"],
            "source_tree": metadata["source_tree"], "profile": metadata["profile"],
            "case_id": "T06_BOUNDED_PRECISION_PILOT", "actor_capability": "C",
            "traffic_source": "RPC", "scope": "QA_ONLY", "qc_status": "USABLE_FOR_PILOT_ONLY",
            "planned_tx_s": planned_count / steady, "achieved_tx_s": achieved_count / steady,
            "planned_bytes_s": planned_bytes / steady,
            "achieved_bytes_s": achieved_bytes / steady}


def audit(session):
    session = precision.private_session(session)
    manifest = common.verify_raw(session)
    summary = common.read_json(session / "summary.json")
    verified = precision.audit(session, summary["source_commit"], summary["source_tree"])
    common.require(verified == summary and
                   all(item["status"] == "PRECISION_PASS" for item in summary["precision"].values()),
                   "QC/precision gate failed")
    jobs = common.read_json(session / "schedule.json")["jobs"]
    rows = []
    for job in jobs:
        run = session / "runs" / job["run_id"]
        rows.append(record(*(common.read_json(run / name) for name in
                             ("metadata.json", "plan.json", "result.json"))))
    common.require(len(rows) == 48 and len({r["run_id"] for r in rows}) == 48,
                   "run inventory incomplete")
    return {"schema": "pluto.file02.common07_audit.v1", "scope": "QA_ONLY",
            "raw_manifest_sha256": manifest, "source_commit": summary["source_commit"],
            "source_tree": summary["source_tree"], "n_runs": len(rows), "runs": rows}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("session")
    parser.add_argument("output")
    args = parser.parse_args()
    try:
        output = audit(Path(args.session))
        common.write_once(args.output, json.dumps(output, indent=2, sort_keys=True) + "\n")
        print(f"COMMON07=PASS runs={output['n_runs']} output={args.output}")
        return 0
    except (ValueError, KeyError, TypeError, OSError) as exc:
        print(f"COMMON07_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
