#!/usr/bin/env python3
"""Raw-bound secondary matched-rate and matched-byte A09/A10 analyses."""

import argparse
import csv
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import statistics
import subprocess
import sys
from uuid import uuid4

import file02_attack as attack
import file02_common as common
import file02_r01 as r01
import file02_r06 as r06

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
UPSTREAM = {
    "A01": "a01-20261113T170536Z-9bdf0a45",
    "A02": "a02-20261113T171854Z-5021cdef",
    "A03": "a03-20261113T172941Z-5a42d176",
    "A04": "a04-20261113T174032Z-edd720e5",
    "A05": "a05-20261113T175129Z-edc80e07",
    "A08": "a08-20261113T182741Z-0dff8ab6",
}


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def input_record(case, name):
    session = ARTIFACTS / name
    attack.verify(session)
    qc = common.read_json(session / "qc_report.json")
    agg = common.read_json(session / "result_aggregate.json")
    require(qc["status"] == "PASS" and qc["n_valid"] == 24 and agg["n_pairs"] == 12 and
            agg["status"] == "ANALYZED" and qc["case_id"] == agg["case_id"] == case,
            "upstream case QC")
    return {"case_id": case, "session": name,
            "raw_manifest_sha256": common.verify_raw(session / "raw"),
            "plan_sha256": common.digest(session / "plan.json"),
            "per_run_sha256": common.digest(session / "result_per_run.json"),
            "aggregate_sha256": common.digest(session / "result_aggregate.json"),
            "qc_sha256": common.digest(session / "qc_report.json")}


def prepare(session, case):
    require(case in ("A09", "A10") and session.parent.resolve() == ARTIFACTS.resolve() and
            not r01.git("status", "--porcelain"), "clean source/private analysis session")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT).returncode == 0,
            "analysis session not ignored")
    inputs = [input_record(c, name) for c, name in UPSTREAM.items()]
    session.mkdir(mode=0o700)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    common.write_once(raw / "inputs.json", json.dumps({"schema": "pluto.file02.matched_inputs.v1",
        "case_id": case, "inputs": inputs}, sort_keys=True, indent=2) + "\n")
    plan = {"schema": "pluto.file02.matched_analysis_plan.v1", "status": "LOCKED",
            "case_id": case, "source_commit": r01.git("rev-parse", "HEAD"),
            "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
            "created_utc": datetime.now(timezone.utc).isoformat(),
            "protocol": f"research/protocol/FILE02_{case}_LOCKED_ANALYSIS.md",
            "input_sha256": common.digest(raw / "inputs.json"),
            "input_cases": list(UPSTREAM), "n_cases": 6, "n_pairs": 72, "n_arms": 144,
            "rate_tx_s": 20, "wire_bytes": 2568, "bytes_s": 51360,
            "analysis_type": "SECONDARY_OF_PREREGISTERED_CASE_PAIRS",
            "ci_rule": "retain_per_case_original_paired_CI; no pooled CI",
            "fpr_status": "UNAVAILABLE_NO_DETECTOR"}
    r01.write_new(session / "plan.json", plan)
    return plan


def validate_plan(session, plan):
    require(plan.get("schema") == "pluto.file02.matched_analysis_plan.v1" and
            plan.get("status") == "LOCKED" and plan.get("case_id") in ("A09", "A10") and
            plan.get("source_tree") == r01.git("rev-parse", plan["source_commit"] + "^{tree}") and
            plan.get("input_sha256") == common.digest(session / "raw/inputs.json") and
            plan.get("input_cases") == list(UPSTREAM) and plan.get("n_pairs") == 72 and
            plan.get("n_arms") == 144 and plan.get("rate_tx_s") == 20 and
            plan.get("wire_bytes") == 2568 and plan.get("bytes_s") == 51360,
            "A09/A10 locked plan drift")


def queue_wait_us(path):
    samples = r06.samples(path)
    require(len(samples) >= 2 and all(s.get("kind") == "node_sample" and
            s.get("dropped_samples") == 0 for s in samples), "observer sample/drop")
    first, last = samples[0], samples[-1]
    a, b = first.get("checktx_socket_queue_count"), last.get("checktx_socket_queue_count")
    x, y = first.get("checktx_socket_queue_seconds"), last.get("checktx_socket_queue_seconds")
    if None in (a, b, x, y) or b <= a:
        return None
    require(y >= x, "negative socket-queue counter")
    return (y-x)/(b-a)*1_000_000


def derive(session, plan):
    manifest = common.verify_raw(session / "raw")
    input_doc = common.read_json(session / "raw/inputs.json")
    require(input_doc.get("schema") == "pluto.file02.matched_inputs.v1" and
            input_doc.get("case_id") == plan["case_id"] and
            [item["case_id"] for item in input_doc["inputs"]] == list(UPSTREAM),
            "analysis input registry")
    rows, summaries, failed = [], [], []
    for item in input_doc["inputs"]:
        case, name = item["case_id"], item["session"]
        require(name == UPSTREAM[case] and item == input_record(case, name),
                "upstream input drift")
        upstream = ARTIFACTS / name
        source_rows = common.read_json(upstream / "result_per_run.json")["rows"]
        source_aggregate = common.read_json(upstream / "result_aggregate.json")
        require(len(source_rows) == 24 and source_aggregate["n_pairs"] == 12 and
                source_aggregate["ci95_fraction"] is not None, "upstream case coverage")
        case_rows = []
        for row in source_rows:
            try:
                require(row["qc_status"] == "PASS" and len(row["windows"]) == 12 and
                        all(window == {"fixed": 6, "background": 14, "bytes": 51360}
                            for window in row["windows"]), "upstream rate/bytes")
                observer = row["observer_validator0"]
                cpu = observer["application_process_cpu_delta_ns"]
                require(isinstance(cpu, int) and cpu >= 0, "shared process CPU")
                out = {"case_id": case, "run_id": row["run_id"], "pair": row["pair"],
                       "condition": row["condition"], "offered_tx_s": 20,
                       "wire_bytes": 2568, "offered_bytes_s": 51360,
                       "fixed_timely_fraction": row["fixed_timely_fraction"],
                       "shared_app_cpu_us_per_offered_attempt": cpu/241/1000,
                       "max_accepted_mempool_bytes": observer["max_accepted_mempool_bytes"],
                       "checktx_socket_queue_mean_wait_us": queue_wait_us(
                           upstream / "raw/runs" / row["run_id"] / "observer-0.jsonl"),
                       "relay_send_envelope_bytes": observer["relay_send_envelope_bytes_delta"],
                       "valid_committed": row["valid_committed"],
                       "invalid_rejected": row["invalid_rejected"], "qc_status": "PASS"}
                rows.append(out)
                case_rows.append(out)
            except (ValueError, KeyError, TypeError, OSError) as exc:
                failed.append({"run_id": row.get("run_id"), "reason": str(exc)})
        if len(case_rows) != 24:
            continue
        by = {condition: [r for r in case_rows if r["condition"] == condition]
              for condition in ("benign", "attack")}
        require(all(len(group) == 12 for group in by.values()), "condition balance")
        summary = {"case_id": case,
            "primary_benign_minus_attack_fraction": source_aggregate["primary_benign_minus_attack_fraction"],
            "original_ci95_fraction": source_aggregate["ci95_fraction"],
            "mean_shared_app_cpu_us_per_attempt": {c: statistics.mean(
                r["shared_app_cpu_us_per_offered_attempt"] for r in group) for c, group in by.items()},
            "median_mempool_peak_bytes": {c: statistics.median(
                r["max_accepted_mempool_bytes"] for r in group) for c, group in by.items()},
            "mean_queue_wait_us": {c: statistics.mean(
                r["checktx_socket_queue_mean_wait_us"] for r in group
                if r["checktx_socket_queue_mean_wait_us"] is not None) for c, group in by.items()},
            "mean_relay_send_envelope_bytes": {c: statistics.mean(
                r["relay_send_envelope_bytes"] for r in group
                if r["relay_send_envelope_bytes"] is not None) for c, group in by.items()},
            "mean_valid_committed": {c: statistics.mean(r["valid_committed"] for r in group)
                                     for c, group in by.items()},
            "mean_invalid_rejected": {c: statistics.mean(r["invalid_rejected"] for r in group)
                                       for c, group in by.items()}}
        summaries.append(summary)
    require(len(rows) == 144 and len(summaries) == 6 and not failed, "A09/A10 complete inputs")
    status = "PASS"
    qc = {"schema": "pluto.file02.matched_qc.v1", "case_id": plan["case_id"],
          "status": status, "n_cases": 6, "n_pairs": 72, "n_arms": 144,
          "failed": failed, "raw_manifest_sha256": manifest,
          "plan_sha256": common.digest(session / "plan.json")}
    benign_queue = [r["checktx_socket_queue_mean_wait_us"] for r in rows
                    if r["condition"] == "benign" and r["checktx_socket_queue_mean_wait_us"] is not None]
    attack_queue = [r["checktx_socket_queue_mean_wait_us"] for r in rows
                    if r["condition"] == "attack" and r["checktx_socket_queue_mean_wait_us"] is not None]
    queue_overlap = [max(min(benign_queue), min(attack_queue)),
                     min(max(benign_queue), max(attack_queue))]
    aggregate = {"schema": "pluto.file02.matched_aggregate.v1", "case_id": plan["case_id"],
        "status": "ANALYZED", "n_cases": 6, "n_pairs": 72, "case_summaries": summaries,
        "raw_manifest_sha256": manifest, "plan_sha256": qc["plan_sha256"],
        "class_specific_cpu_per_tx": None,
        "class_specific_cpu_status": "UNAVAILABLE_SHARED_INTEGRATION_PROCESS",
        "fpr": None, "fpr_status": "UNAVAILABLE_NO_DETECTOR",
        "pooled_ci95": None, "pooled_ci_status": "NOT_ESTIMATED_CASE_HETEROGENEITY"}
    if plan["case_id"] == "A09":
        aggregate["queue_wait_range_overlap_us"] = queue_overlap if queue_overlap[0] <= queue_overlap[1] else None
        aggregate["queue_wait_overlap_interpretation"] = "FEATURE_CONFUSABLE_CANDIDATE_NOT_FPR"
    else:
        aggregate["exact_offered_byte_match_all_72_pairs"] = True
        aggregate["size_only_explanation"] = "NO_POSITIVE_SERVICE_EFFECT_TO_EXPLAIN; STAGE_COST_DIFFERS_AT_EQUAL_WIRE_BYTES"
    return rows, qc, aggregate


def csv_text(rows):
    from io import StringIO
    buffer = StringIO()
    keys = ("case_id", "run_id", "pair", "condition", "offered_tx_s", "wire_bytes",
            "offered_bytes_s", "fixed_timely_fraction",
            "shared_app_cpu_us_per_offered_attempt", "max_accepted_mempool_bytes",
            "checktx_socket_queue_mean_wait_us", "relay_send_envelope_bytes",
            "valid_committed", "invalid_rejected", "qc_status")
    writer = csv.DictWriter(buffer, fieldnames=keys, lineterminator="\n")
    writer.writeheader()
    writer.writerows({key: row[key] for key in keys} for row in rows)
    return buffer.getvalue()


def decision_text(qc, aggregate):
    return (f"# {qc['case_id']} matched secondary synthesis\n\nQC: {qc['status']}; "
            f"{qc['n_cases']} cases, {qc['n_pairs']} pairs, {qc['n_arms']} arms. "
            "Retain each upstream case's original paired CI; no pooled CI. "
            "Classifier FPR and class-specific CPU are unavailable. "
            "No new network run, DoS, knee, P2P-origin or multi-host claim.\n")


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.matched_per_run.v1", "rows": rows})
    common.write_once(session / "result_per_run.csv", csv_text(rows))
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir(mode=0o700)
    r01.write_new(session / "figures/status.json", {"status": "PENDING_CASE_REVIEW",
        "reason": "secondary numeric synthesis without scientific figure"})
    common.write_once(session / "decision.md", decision_text(qc, aggregate))
    return qc


def execute(case):
    session = ARTIFACTS / (case.lower() + "-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") +
                           "-" + uuid4().hex[:8])
    plan = prepare(session, case)
    manifest = common.freeze_raw(session / "raw")
    qc = write_outputs(session, plan)
    print(f"MATCHED_CASE={case} session={session} qc={qc['status']} "
          f"raw_manifest_sha256={manifest}")
    return 0 if qc["status"] == "PASS" else 2


def verify(session):
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve() and session.is_dir(), "private matched session")
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    rows, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.matched_per_run.v1", "rows": rows} and
            (session / "result_per_run.csv").read_text() == csv_text(rows) and
            common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate and
            (session / "decision.md").read_text() == decision_text(qc, aggregate) and
            common.read_json(session / "figures/status.json")["status"] == "PENDING_CASE_REVIEW",
            "matched derived artifact mismatch")
    print(f"MATCHED_VERIFY=OK case={plan['case_id']} qc={qc['status']} "
          f"raw_manifest_sha256={qc['raw_manifest_sha256']}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    run = sub.add_parser("run")
    run.add_argument("case", choices=("A09", "A10"))
    check = sub.add_parser("verify")
    check.add_argument("session")
    args = parser.parse_args()
    try:
        return execute(args.case) if args.command == "run" else (verify(args.session) or 0)
    except (ValueError, OSError, subprocess.SubprocessError) as exc:
        print(f"MATCHED_ERROR={exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
