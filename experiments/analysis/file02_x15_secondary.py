#!/usr/bin/env python3
"""Recompute preregistered secondary p50/p95 intervals from frozen X raw."""

import argparse
import json
from pathlib import Path
import statistics
import sys

import file02_common as common
import file02_x10 as x10
import file02_x13 as x13

ROOT = Path(__file__).resolve().parents[2]
X10 = ROOT / "research/artifacts/file02/x10-20261115T051140Z-3d69d6eb"
X13 = ROOT / "research/artifacts/file02/x13-20261115T135447Z-a960b885"
OUT = ROOT / "research/evidence/file02/mechanism/X15_SECONDARY_CI.json"


def derive():
    p10 = common.read_json(X10 / "plan.json")
    x10.validate_plan(X10, p10)
    rows10, qc10, _ = x10.derive(X10, p10)
    p13 = common.read_json(X13 / "plan.json")
    x13.validate_plan(X13, p13)
    rows13, qc13, _ = x13.derive(X13, p13)
    if qc10["status"] != "PASS" or qc13["status"] != "PASS":
        raise ValueError("X10/X13 QC not PASS")
    x10_blocks = {block: {} for block in range(24)}
    for row in rows10:
        x10_blocks[row["block"]][row["condition"]] = row
    x13_blocks = {block: {} for block in range(12)}
    for row in rows13:
        x13_blocks[row["block"]][row["rate"]] = row
    result = {"schema": "pluto.file02.x15_secondary_ci.v1",
              "method": "network-level statistic, matched-block percentile bootstrap, 20000 draws seed 20261115",
              "x10_raw_manifest_sha256": qc10["raw_manifest_sha256"],
              "x13_raw_manifest_sha256": qc13["raw_manifest_sha256"],
              "x10_secondary": {}, "x13_secondary": {}}
    for name in ("p50", "p95"):
        key = f"fixed_{name}_upper_ms"
        for comparison in ("I_minus_B", "I_minus_L"):
            left, right = comparison[0], comparison[-1]
            values = [x10_blocks[b][left][key] - x10_blocks[b][right][key]
                      for b in range(24)]
            result["x10_secondary"][f"{comparison}_{name}"] = {
                "n_blocks": 24, "effect_ms": statistics.mean(values),
                "ci95_ms": x10.paired_ci(values)}
        for rate in x13.RATES:
            values = [x13_blocks[b][rate][key] for b in range(12)]
            result["x13_secondary"][f"rate_{rate}_{name}"] = {
                "n_networks": 12, "mean_ms": statistics.mean(values),
                "ci95_ms": x10.paired_ci(values)}
    result["interpretation"] = "secondary exploratory intervals; upper-bound observation, not exact commit"
    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("write", "check"))
    args = parser.parse_args()
    try:
        result = derive()
        if args.command == "write":
            common.write_once(OUT, json.dumps(result, indent=2, sort_keys=True) + "\n")
        elif common.read_json(OUT) != result:
            raise ValueError("X15 secondary CI drift")
        print(f"X15_{args.command.upper()}=OK")
        return 0
    except (OSError, ValueError, KeyError, TypeError) as exc:
        print(f"X15_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
