#!/usr/bin/env python3
"""Exploratory paired upper-bound latency analysis of frozen A04/A05/A08 runs."""

from __future__ import annotations

import argparse
import json
import math
import random
import statistics
import subprocess
from pathlib import Path

import file02_common as common


ROOT = Path(__file__).resolve().parents[2]
CASES = ("A04", "A05", "A08")
BOOTSTRAP = 20_000
SEED = 20261115


def percentile(values: list[float], fraction: float) -> float:
    if not values or not 0 <= fraction <= 1 or any(not math.isfinite(x) for x in values):
        raise ValueError("invalid percentile input")
    ordered = sorted(values)
    index = (len(ordered) - 1) * fraction
    low = math.floor(index)
    high = math.ceil(index)
    return ordered[low] + (ordered[high] - ordered[low]) * (index - low)


def pair_mean_ci(differences: list[float], seed: int) -> list[float]:
    if len(differences) < 2:
        raise ValueError("insufficient independent pairs")
    rng = random.Random(seed)
    n = len(differences)
    means = sorted(statistics.mean(differences[rng.randrange(n)] for _ in range(n))
                   for _ in range(BOOTSTRAP))
    return [percentile(means, 0.025), percentile(means, 0.975)]


def verify_case(session: Path) -> None:
    result = subprocess.run(["python3", "experiments/analysis/file02_attack.py", "verify", str(session)],
                            cwd=ROOT, capture_output=True, text=True)
    if result.returncode:
        raise ValueError(f"source case verifier failed for {session.name}: {result.stderr.strip()}")


def analyze_case(case: str, session: Path, ordinal: int) -> dict:
    verify_case(session)
    qc = common.read_json(session / "qc_report.json")
    if qc.get("status") != "PASS":
        raise ValueError(f"source QC failed: {case}")
    source_manifest = common.verify_raw(session / "raw")
    if source_manifest != qc.get("raw_manifest_sha256"):
        raise ValueError(f"source manifest drift: {case}")
    rows = common.read_json(session / "result_per_run.json")["rows"]
    if len(rows) != 24:
        raise ValueError(f"unexpected case run count: {case}")
    grouped: dict[int, dict[str, dict]] = {}
    for row in rows:
        if row.get("qc_status") != "PASS" or row.get("profile") != "H":
            raise ValueError(f"invalid row QC/profile: {case}")
        grouped.setdefault(row["pair"], {})[row["condition"]] = row
    if sorted(grouped) != list(range(12)):
        raise ValueError(f"missing pair: {case}")
    pairs = []
    arm_metrics = {"benign": {"mean": [], "p50": [], "p95": []},
                   "attack": {"mean": [], "p50": [], "p95": []}}
    for pair, arm in sorted(grouped.items()):
        if set(arm) != {"benign", "attack"}:
            raise ValueError(f"unmatched arm: {case} pair {pair}")
        benign, attack = arm["benign"], arm["attack"]
        if benign["seed"] != attack["seed"] or benign["genesis_sha256"] != attack["genesis_sha256"] or benign["fixed_hashes"] != attack["fixed_hashes"]:
            raise ValueError(f"fixture mismatch: {case} pair {pair}")
        metrics = {}
        for label, row in arm.items():
            samples = row["fixed_inclusion_upper_ms"]
            if len(samples) != 72 or any(not isinstance(x, (int, float)) or not math.isfinite(x) or x < 0 for x in samples):
                raise ValueError(f"invalid latency samples: {case} pair {pair}")
            if row["fixed_timely_fraction"] != 1:
                raise ValueError(f"original 3-second endpoint mismatch: {case} pair {pair}")
            metrics[label] = {"mean": statistics.mean(samples),
                              "p50": percentile(samples, 0.50),
                              "p95": percentile(samples, 0.95)}
            for key, value in metrics[label].items():
                arm_metrics[label][key].append(value)
        pairs.append({"pair": pair, "benign_run_id": benign["run_id"],
                      "attack_run_id": attack["run_id"], "seed": benign["seed"],
                      "samples_per_arm": 72,
                      "attack_minus_benign_ms": {
                          key: metrics["attack"][key] - metrics["benign"][key]
                          for key in ("mean", "p50", "p95")}})
    summaries = {}
    for key in ("mean", "p50", "p95"):
        diffs = [pair["attack_minus_benign_ms"][key] for pair in pairs]
        summaries[key] = {"attack_minus_benign_pair_mean_ms": statistics.mean(diffs),
                          "exploratory_bootstrap_CI95_ms": pair_mean_ci(diffs, SEED + ordinal * 10 + ("mean", "p50", "p95").index(key)),
                          "benign_mean_of_run_metric_ms": statistics.mean(arm_metrics["benign"][key]),
                          "attack_mean_of_run_metric_ms": statistics.mean(arm_metrics["attack"][key])}
    return {"case_id": case, "source_session": session.name,
            "source_raw_manifest_sha256": source_manifest,
            "n_independent_pairs": 12, "source_qc": "PASS",
            "original_3s_timely_effect": 0,
            "metrics": summaries, "pairs": pairs}


def analyze() -> dict:
    root = ROOT / "research/artifacts/file02"
    cases = []
    for ordinal, case in enumerate(CASES):
        sessions = list(root.glob(case.lower() + "-*/plan.json"))
        if len(sessions) != 1:
            raise ValueError(f"expected one frozen {case} session")
        cases.append(analyze_case(case, sessions[0].parent, ordinal))
    return {"schema": "pluto.file02.x03_exploratory_latency.v1",
            "status": "EXPLORATORY_NOT_CONFIRMATORY",
            "estimand": "attack-minus-benign difference in per-run first-all-four inclusion observation upper-bound metrics",
            "unit": "milliseconds", "independent_unit": "matched fresh-network pair",
            "bootstrap_resamples": BOOTSTRAP, "bootstrap_seed": SEED,
            "measurement_limit": "100-ms block scanner gives observation upper bound, not exact commit timestamp; scanner scheduling adds uncertainty",
            "multiple_testing": "unadjusted exploratory intervals across three cases and three metrics",
            "cases": cases}


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("write", "check"))
    parser.add_argument("--output", type=Path,
                        default=ROOT / "research/evidence/file02/mechanism/EXPLORATORY_LATENCY_A04_A05_A08.json")
    args = parser.parse_args()
    expected = json.dumps(analyze(), sort_keys=True, indent=2) + "\n"
    if args.mode == "write":
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(expected, encoding="utf-8")
    elif args.output.read_text(encoding="utf-8") != expected:
        raise SystemExit("exploratory latency output drift")


if __name__ == "__main__":
    main()
