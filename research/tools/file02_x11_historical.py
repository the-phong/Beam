#!/usr/bin/env python3
"""Independent class-by-class X11 reanalysis of frozen A01/A03/A04/A05/A08."""

import argparse
import json
from pathlib import Path
import statistics
import sys

import file02_attack as attack
import file02_common as common
import file02_x10 as x10

ROOT = Path(__file__).resolve().parents[2]
DEST = ROOT / "research/evidence/file02/mechanism/X11_CLASS_CONTRASTS.json"
SESSIONS = {
    "A01": "a01-20261113T170536Z-9bdf0a45",
    "A03": "a03-20261113T172941Z-5a42d176",
    "A04": "a04-20261113T174032Z-edd720e5",
    "A05": "a05-20261113T175129Z-edc80e07",
    "A08": "a08-20261113T182741Z-0dff8ab6",
}


def derive():
    cases = []
    for case, name in SESSIONS.items():
        session = ROOT / "research/artifacts/file02" / name
        plan = common.read_json(session / "plan.json")
        attack.validate_plan(session, plan)
        if (plan["case_id"] != case or plan["profile"] != "H" or
                plan["fixed_legitimate_tx_s"] != 6 or
                plan["background_tx_s"] != 14 or
                plan["total_tx_s"] != 20 or
                plan["wire_bytes"] != 2568 or
                plan["total_bytes_s"] != 51360 or
                plan["n_pairs"] != 12):
            raise ValueError(f"{case} locked regimen mismatch")
        rows, qc, aggregate = attack.derive(session, plan)
        if qc["status"] != "PASS" or len(rows) != 24 or aggregate["n_pairs"] != 12:
            raise ValueError(f"{case} historical QC unavailable")
        by_pair = {index: {} for index in range(12)}
        for row in rows:
            by_pair[row["pair"]][row["condition"]] = row
        cost = []
        mean_deltas, p50_deltas, p95_deltas = [], [], []
        for index, arms in by_pair.items():
            if set(arms) != {"attack", "benign"}:
                raise ValueError(f"{case} pair {index} incomplete")
            invalid = arms["attack"]
            benign = arms["benign"]
            if invalid["fixed_hashes"] != benign["fixed_hashes"] or \
                    invalid["genesis_sha256"] != benign["genesis_sha256"]:
                raise ValueError(f"{case} pair {index} fixed stream mismatch")
            cost.append(invalid["invalid_mean_checktx_service_ns"])
            a, b = invalid["fixed_inclusion_upper_ms"], benign["fixed_inclusion_upper_ms"]
            if len(a) != len(b) or len(a) != 72:
                raise ValueError(f"{case} pair {index} fixed observations incomplete")
            mean_deltas.append(statistics.mean(a) - statistics.mean(b))
            p50_deltas.append(x10.percentile(a, .5) - x10.percentile(b, .5))
            p95_deltas.append(x10.percentile(a, .95) - x10.percentile(b, .95))
        cases.append({"case_id": case, "n_pairs": 12, "n_networks": 24,
            "raw_manifest_sha256": qc["raw_manifest_sha256"],
            "source_commit": plan["source_commit"],
            "invalid_mean_checktx_wall_ns": statistics.mean(cost),
            "invalid_mean_checktx_wall_ci95_ns_exploratory": x10.paired_ci(cost),
            "original_3s_benign_minus_attack_fraction": aggregate["primary_benign_minus_attack_fraction"],
            "original_3s_ci95_fraction": aggregate["ci95_fraction"],
            "exploratory_attack_minus_benign_mean_upper_ms": statistics.mean(mean_deltas),
            "exploratory_mean_ci95_ms": x10.paired_ci(mean_deltas),
            "exploratory_p50_ci95_ms": x10.paired_ci(p50_deltas),
            "exploratory_p95_ci95_ms": x10.paired_ci(p95_deltas),
            "run_ids": [row["run_id"] for row in rows],
            "stage_cpu_ns": None, "stage_cpu_reason": "UNAVAILABLE_STAGE_CPU_ATTRIBUTION"})
    return {"schema": "pluto.file02.x11_historical_class_contrasts.v1",
            "status": "HISTORICAL_PREREGISTERED_3S_WITH_EXPLORATORY_CONTINUOUS_REANALYSIS",
            "independent_unit": "paired fresh four-validator network fixture, 12 pairs per class",
            "no_cross_class_pooling": True,
            "continuous_measurement": "100-ms first-all-four block scan observation upper bound, not exact commit",
            "cases": cases}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("write", "check"))
    args = parser.parse_args()
    try:
        expected = derive()
        if args.command == "write":
            common.write_once(DEST, json.dumps(expected, indent=2, sort_keys=True) + "\n")
        elif common.read_json(DEST) != expected:
            raise ValueError("X11 derived evidence drift")
        print(f"X11_{args.command.upper()}=OK cases={len(expected['cases'])}")
        return 0
    except (OSError, ValueError, KeyError, TypeError) as exc:
        print(f"X11_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
