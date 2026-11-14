#!/usr/bin/env python3
"""Q03 valid-included p99 audit from immutable A/Q02 sessions."""

import argparse
import json
import math
from pathlib import Path
import random
import sys

import file02_a13 as a13
import file02_attack as attack
import file02_common as common
from file02_common06 import rank_coverage
import file02_q01 as q01
import file02_q02 as q02

ROOT = Path(__file__).resolve().parents[2]
OUTPUT = ROOT / "research/evidence/file02/quality/Q03_VALID_INCLUDED_P99_V2.json"
Q02_SESSION = "q02-20261114T044327Z-e7f236e2"
Q02_MANIFEST = "58265e00a8c0fc08a37cb4e2935633d85304a1e70d32142df5d9eaa4d1180f3f"


def require(value, reason):
    if not value:
        raise ValueError(reason)


def nearest_rank(values):
    require(values and all(isinstance(x, (int, float)) and math.isfinite(x) and x >= 0
                           for x in values), "nonempty finite included latencies required")
    return sorted(values)[math.ceil(.99 * len(values)) - 1]


def precision(groups, expected_per_run, seed=20261114):
    require(groups and all(len(group) == expected_per_run for group in groups),
            "complete valid-included sample clusters required")
    values = [value for group in groups for value in group]
    point, coverage = nearest_rank(values), rank_coverage(len(values))
    result = {"n_independent_networks": len(groups), "n_valid_included_samples": len(values),
              "descriptive_p99_upper_ms": point, "rank_band_coverage_iid_optimistic": coverage,
              "inferential_p99_upper_ms": None, "ci95_ms": None,
              "relative_ci_radius": None, "status": "INSUFFICIENT_SAMPLES",
              "failed_gate": "rank_band_coverage_below_0.95"}
    if coverage < .95:
        return result
    rng = random.Random(seed)
    draws = []
    for _ in range(2000):
        sample = []
        for _ in groups:
            group = groups[rng.randrange(len(groups))]
            sample.extend(group[rng.randrange(len(group))] for _ in group)
        draws.append(nearest_rank(sample))
    draws.sort()
    ci = [draws[math.floor(.025 * len(draws))], draws[math.ceil(.975 * len(draws)) - 1]]
    radius = max(point-ci[0], ci[1]-point)/point if point > 0 else math.inf
    result["relative_ci_radius"] = radius
    if radius <= .20:
        result.update(inferential_p99_upper_ms=point, ci95_ms=ci,
                      status="PRECISION_PASS", failed_gate=None)
    else:
        result.update(status="INSUFFICIENT_PRECISION", failed_gate="relative_ci_radius_above_0.20")
    return result


def source_groups():
    groups = []
    for case, (session_name, expected_manifest) in q01.INPUTS.items():
        session = q01.ARTIFACTS / session_name
        plan = common.read_json(session / "plan.json")
        if case == "A13":
            a13.verify(session)
            rows, qc, _ = a13.derive(session, plan)
            expected_count, per_run = 8, 36
        else:
            attack.verify(session)
            rows, qc, _ = attack.derive(session, plan)
            expected_count, per_run = 24, 72
        require(qc["status"] == "PASS" and qc["raw_manifest_sha256"] == expected_manifest and
                len(rows) == expected_count, "A source QC or manifest")
        for condition in ("benign", "attack"):
            selected = [row for row in rows if row["condition"] == condition]
            selected.sort(key=lambda row: row.get("pair", row.get("block")))
            arrays = [row["phase_fixed_latency_upper_ms"]["on"] if case == "A13" else
                      row["fixed_inclusion_upper_ms"] for row in selected]
            require(all(row["qc_status"] == "PASS" for row in selected) and
                    len(selected) == expected_count//2 and
                    all(len(group) == per_run for group in arrays), "A included-only groups")
            groups.append({"case_id": case, "profile": "H", "condition": condition,
                           "run_ids": [row["run_id"] for row in selected],
                           "source_raw_manifest_sha256": expected_manifest,
                           **precision(arrays, per_run)})
    session = q02.ARTIFACTS / Q02_SESSION
    q02.verify(session)
    plan = common.read_json(session / "plan.json")
    rows, _, qc, _ = q02.derive(session, plan)
    require(qc["status"] == "PASS" and qc["raw_manifest_sha256"] == Q02_MANIFEST and
            len(rows) == 96, "Q02 source QC or manifest")
    for profile in q02.PROFILES:
        for condition in ("benign", "attack"):
            selected = sorted((row for row in rows if row["profile"] == profile and
                               row["condition"] == condition), key=lambda row: row["block"])
            arrays = [row["fixed_inclusion_upper_ms"] for row in selected]
            require(len(selected) == 12 and all(row["qc_status"] == "PASS" and len(a) == 72
                                                for row, a in zip(selected, arrays)),
                    "Q02 included-only groups")
            groups.append({"case_id": "Q02", "profile": profile, "condition": condition,
                           "run_ids": [row["run_id"] for row in selected],
                           "source_raw_manifest_sha256": Q02_MANIFEST,
                           **precision(arrays, 72)})
    return groups


def derive():
    groups = source_groups()
    require(len(groups) == 22, "Q03 group coverage")
    return {"schema": "pluto.file02.q03.valid_included_p99.v1", "status": "PASS_AUDIT",
            "metric": "socket_write_to_first_all_four_observation_upper_bound_ms",
            "precision_gate": "rank_band_98.5_99.5_coverage>=0.95_then_two_stage_2000_CI_radius<=0.20",
            "groups": groups, "inferential_case_pooling": None,
            "limit": "descriptive p99 is an inclusion upper bound; no exact commit time"}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("run", "verify"))
    args = parser.parse_args()
    try:
        result = derive()
        rendered = json.dumps(result, sort_keys=True, indent=2) + "\n"
        if args.command == "run":
            require(not OUTPUT.exists(), "Q03 output exists")
            OUTPUT.write_text(rendered)
        else:
            require(OUTPUT.read_text() == rendered, "Q03 output drift")
        counts = {status: sum(g["status"] == status for g in result["groups"])
                  for status in sorted({g["status"] for g in result["groups"]})}
        print(f"Q03_{args.command.upper()}=PASS groups={len(result['groups'])} precision={counts}")
        return 0
    except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError) as exc:
        print(f"Q03_ERROR={exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
