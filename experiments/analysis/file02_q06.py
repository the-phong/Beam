#!/usr/bin/env python3
"""Q06 raw-bound seed/order and cross-session replication audit."""

import argparse
import json
from pathlib import Path
import statistics
import sys

import file02_common as common
import file02_q01 as q01
import file02_q02 as q02

ROOT = Path(__file__).resolve().parents[2]
OUTPUT = ROOT / "research/evidence/file02/quality/Q06_REPLICATION.json"
Q02_SESSION = "q02-20261114T044327Z-e7f236e2"
Q02_SUPERSEDED = "q02-20261114T025508Z-a4b258cb"
Q02_ABORTED = "q02-20261114T024019Z-8ccd9928"
Q02_MANIFEST = "58265e00a8c0fc08a37cb4e2935633d85304a1e70d32142df5d9eaa4d1180f3f"
SUPERSEDED_MANIFEST = "9098dc2f0c3700e11c6477ae50a17c2cd45f5e6ebe2a7c063af49be695705f95"
ABORTED_MANIFEST = "0308674aaf4695c5e8261bf6be0034004f6adca1376f67d4a8a264ecf29cdb0b"


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def stability(effects, first_conditions):
    require(len(effects) == len(first_conditions) >= 4 and
            set(first_conditions) == {"benign", "attack"}, "Q06 order/effect coverage")
    n = len(effects)
    require(first_conditions.count("benign") == first_conditions.count("attack") == n//2,
            "Q06 arm order unbalanced")
    return {"n_independent_pairs": n, "mean_effect_fraction": statistics.mean(effects),
            "leave_one_pair_out_mean_range_fraction": [min(statistics.mean(effects[:i]+effects[i+1:])
                for i in range(n)), max(statistics.mean(effects[:i]+effects[i+1:]) for i in range(n))],
            "first_condition_counts": {condition: first_conditions.count(condition)
                                       for condition in ("benign", "attack")},
            "mean_by_first_condition_fraction": {condition: statistics.mean(effect for effect, first
                in zip(effects, first_conditions) if first == condition)
                for condition in ("benign", "attack")}}


def derive():
    pairs, q01_summary = q01.derive()
    by_case = []
    for case, (name, manifest) in q01.INPUTS.items():
        session = q01.ARTIFACTS / name
        selected = sorted((pair for pair in pairs if pair["case_id"] == case),
                          key=lambda pair: pair["pair"])
        expected_count = 4 if case == "A13" else 12
        require(len(selected) == expected_count and len({p["seed"] for p in selected}) ==
                expected_count, "Q06 pair seed independence")
        first_jobs = common.read_json(session / "raw/schedule.json")["jobs"]
        first = [next(job["condition"] for job in first_jobs if
                      job.get("pair", job.get("block")) == index and job["order"] == 0)
                 for index in range(expected_count)]
        summary = next(item for item in q01_summary["case_summaries"] if item["case_id"] == case)
        require(summary["source_raw_manifest_sha256"] == manifest and
                not common.read_json(session / "qc_report.json")["failed"], "Q06 A source exclusions")
        effects = [p["benign_minus_attack_fraction"] for p in selected]
        result = stability(effects, first)
        require(result["mean_effect_fraction"] == summary["effect_fraction"], "Q06 A mean drift")
        by_case.append({"case_id": case, "source_raw_manifest_sha256": manifest,
                        "seed_count": len({p["seed"] for p in selected}),
                        "pair_run_ids": [[p["benign_run_id"], p["attack_run_id"]] for p in selected],
                        "ci95_fraction": summary["ci95_fraction"], **result})
    session = q02.ARTIFACTS / Q02_SESSION
    q02.verify(session)
    plan = common.read_json(session / "plan.json")
    rows, blocks, qc, aggregate = q02.derive(session, plan)
    require(qc["status"] == "PASS" and qc["raw_manifest_sha256"] == Q02_MANIFEST and
            len(rows) == 96 and len(blocks) == 12 and not qc["failed"], "Q06 Q02 source")
    h = sorted((row for row in rows if row["profile"] == "H"),
               key=lambda row: (row["block"], row["order"]))
    require(len(h) == 24 and len({r["seed"] for r in h}) == 12 and
            len({r["genesis_sha256"] for r in h}) == 12, "Q06 independent H blocks")
    h_effects, h_first, h_pairs = [], [], []
    for block in range(12):
        group = [row for row in h if row["block"] == block]
        require(len(group) == 2 and {r["condition"] for r in group} ==
                {"benign", "attack"}, "Q06 H arm membership")
        b, a = (next(row for row in group if row["condition"] == condition)
                for condition in ("benign", "attack"))
        h_effects.append(b["fixed_timely_fraction"]-a["fixed_timely_fraction"])
        h_first.append(group[0]["condition"])
        h_pairs.append([b["run_id"], a["run_id"]])
    q02_stability = stability(h_effects, h_first)
    require(q02_stability["mean_effect_fraction"] ==
            aggregate["profile_effects_fraction"]["H"]["mean"], "Q06 Q02 H effect drift")
    aborted = q02.ARTIFACTS / Q02_ABORTED
    q02.verify(aborted)
    failed_qc = common.read_json(aborted / "qc_report.json")
    require(failed_qc["status"] == "INVALID_FOR_INFERENCE" and
            failed_qc["raw_manifest_sha256"] == ABORTED_MANIFEST and
            any(item.get("run_id") == "Q02-b00-CP-attack" for item in failed_qc["failed"]) and
            "stale observer sample" in (aborted / "raw/runs/Q02-b00-CP-attack/load.log").read_text(),
            "Q06 failed-attempt provenance")
    superseded = q02.ARTIFACTS / Q02_SUPERSEDED
    q02.verify(superseded)
    previous = common.read_json(superseded / "qc_report.json")
    old_jobs = common.read_json(superseded / "raw/schedule.json")["jobs"]
    require(previous["status"] == "PASS" and
            previous["raw_manifest_sha256"] == SUPERSEDED_MANIFEST and
            all(len({next(job["condition"] for job in old_jobs if job["block"] == b and
                          job["profile"] == profile) for b in range(12)}) == 1
                for profile in q02.PROFILES), "Q06 order-confounded v1 provenance")
    a01 = by_case[0]
    return {"schema": "pluto.file02.q06.replication.v1", "status": "PARTIAL_HOST_REPLICATION",
            "endpoint": "fixed_legitimate_all_four_observed_within_3000ms_fraction",
            "a_case_results": by_case,
            "q02_h_structural_replication": {"source_raw_manifest_sha256": Q02_MANIFEST,
                "pair_run_ids": h_pairs, "seed_count": 12,
                "ci95_fraction": aggregate["profile_effects_fraction"]["H"]["ci95"],
                **q02_stability},
            "cross_session_A01_vs_Q02_H": {"comparison": "same_20_tx_s_structural_invalid_H_endpoint",
                "a01_effect_fraction": a01["mean_effect_fraction"],
                "q02_h_effect_fraction": q02_stability["mean_effect_fraction"],
                "pooled_ci95": None, "limit": "different fixture seeds and sessions on same physical host"},
            "failed_excluded_rerun_log": [{"session": Q02_ABORTED,
                "raw_manifest_sha256": ABORTED_MANIFEST,
                "status": "INVALID_FOR_INFERENCE_PRE_SEND_STALE_OBSERVER",
                "used_in_inference": False},
                {"session": Q02_SUPERSEDED, "raw_manifest_sha256": SUPERSEDED_MANIFEST,
                 "status": "TECHNICAL_QC_PASS_SUPERSEDED_PROFILE_ORDER_CONFOUND",
                 "used_in_inference": False},
                {"session": Q02_SESSION, "raw_manifest_sha256": Q02_MANIFEST,
                 "status": "COMPLETE_NEW_FULL_SCHEDULE", "used_in_inference": True}],
            "host_replication_status": "UNAVAILABLE_NO_SECOND_PHYSICAL_HOST",
            "equivalence_status": "NOT_ESTABLISHED_CEILING_LIMITED_ENDPOINT"}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("run", "verify"))
    args = parser.parse_args()
    try:
        report = derive()
        rendered = json.dumps(report, sort_keys=True, indent=2) + "\n"
        if args.command == "run":
            require(not OUTPUT.exists(), "Q06 output exists")
            OUTPUT.write_text(rendered)
        else:
            require(OUTPUT.read_text() == rendered, "Q06 output drift")
        print(f"Q06_{args.command.upper()}={report['status']} "
              f"families={len(report['a_case_results'])} Q02_H_pairs=12")
        return 0
    except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError) as exc:
        print(f"Q06_ERROR={exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
