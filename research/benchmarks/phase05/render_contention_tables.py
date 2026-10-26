#!/usr/bin/env python3
"""Render publication-safe tables from verified contention analysis JSON."""
import argparse
import json
from pathlib import Path
import statistics


def value(number, digits=2):
    return "UNAVAILABLE" if number is None or isinstance(number, str) else f"{number:.{digits}f}"


def render(analysis):
    rows = json.loads((analysis / "per_run.json").read_text())
    grouped = json.loads((analysis / "aggregate.json").read_text())
    excluded = json.loads((analysis / "excluded_runs.json").read_text())
    if len(rows) != 78 or len(grouped) != 18 or len(excluded) != 1:
        raise ValueError("incomplete contention analysis")
    if any(r["valid_confirmed"] != 300 for r in rows if r["series"] != "invalid_only"):
        raise ValueError("valid sample count too small")
    if any(r["valid_confirmed"] != 0 for r in rows if r["series"] == "invalid_only"):
        raise ValueError("invalid-only row has valid inclusion")
    lines = ["# Phase 05 matched-rate contention: raw-derived result tables", "",
             "78 usable independent four-validator runs: 77 original plus one clock-repaired replacement. The original anomalous run remains in private raw evidence and is excluded from measurement aggregates. Every valid-bearing run confirmed 300/300 valid transactions; every invalid submission was rejected. Means below are means of **per-run** summaries. p99 is descriptive even with 300 samples/run. CPU is summed process CPU as a percentage of one logical core across four Pluto/Comet processes; RSS is the sum of sampled process peaks. Client admission and block-observation inclusion are distinct timings.", ""]
    sources = sorted({r["source_tree"] for r in rows})
    lines += [f"Tested source trees: `{', '.join(sources)}`. Public corpus SHA-256: `{rows[0]['corpus_sha256']}`.", ""]
    for series, title in (("primary", "Primary: fixed 1.5 legitimate tx/s"),
                          ("secondary", "Secondary: fixed 4 legitimate tx/s")):
        lines += [f"## {title}", "",
                  "| Profile | Invalid % | Runs | Target valid / invalid tx/s | Achieved valid / invalid tx/s | Admission p50 / p95 / p99 (µs) | Inclusion p50 / p95 / p99 (ms) | Goodput tx/s | p99 amplification | Goodput degradation | CPU % | Sampled RSS MiB |",
                  "|---|---:|---:|---|---|---|---|---:|---:|---:|---:|---:|"]
        for g in grouped:
            if g["series"] != series:
                continue
            part = [r for r in rows if r["series"] == series and r["profile"] == g["profile"] and r["ratio"] == g["ratio"]]
            def mean(path):
                values = [path(r) for r in part]
                return statistics.mean(values) if all(isinstance(v, (int, float)) for v in values) else None
            admission = " / ".join(value(mean(lambda r, k=k: r["valid_admission_us"][k])) for k in ("p50", "p95", "p99"))
            inclusion = " / ".join(value(mean(lambda r, k=k: r["inclusion_ms"][k])) for k in ("p50", "p95", "p99"))
            lines.append(f"| {g['profile'].upper()} | {g['ratio']} | {g['repetitions']} | {value(part[0]['target_valid_tps'],1)} / {value(part[0]['target_invalid_tps'],1)} | {value(mean(lambda r:r['achieved_valid_tps']),2)} / {value(mean(lambda r:r['achieved_invalid_tps']),2)} | {admission} | {inclusion} | {value(g['goodput_tps']['mean'],3)} | {value(g['inclusion_p99_amplification']['mean'],3)}× | {value(100*g['goodput_degradation_fraction']['mean'],2)}% | {value(g['cpu_percent']['mean'],2)} | {value(g['peak_sampled_rss_bytes']['mean']/1048576,1)} |")
        lines += ["", "Uncertainty intervals and each validator's CPU/RSS are in `CONTENTION_AGGREGATE.json` and `CONTENTION_PER_RUN.csv`.", ""]
    lines += ["## Invalid-only and rejection classes", "",
              "| Profile | Runs | Invalid offered / achieved tx/s | Invalid rejected per run | CPU % | Sampled RSS MiB | Valid inclusion latency |",
              "|---|---:|---|---:|---:|---:|---|"]
    for g in grouped:
        if g["series"] != "invalid_only":
            continue
        part = [r for r in rows if r["series"] == "invalid_only" and r["profile"] == g["profile"]]
        lines.append(f"| {g['profile'].upper()} | {g['repetitions']} | 15 / {value(statistics.mean(r['achieved_invalid_tps'] for r in part),2)} | 3000 | {value(g['cpu_percent']['mean'],2)} | {value(g['peak_sampled_rss_bytes']['mean']/1048576,1)} | UNAVAILABLE (zero valid submitted) |")
    lines += ["", "| Profile | Case ID | Rejected samples per run | Mean of per-run client rejection p50 (µs) |",
              "|---|---|---:|---:|"]
    for profile in ("b0", "b1"):
        part = [r for r in rows if r["series"] == "invalid_only" and r["profile"] == profile]
        cases = sorted(set().union(*(r["rejection_cost_by_case_us"] for r in part)))
        for case in cases:
            counts = [r["rejection_cost_by_case_us"][case]["count"] for r in part]
            medians = [r["rejection_cost_by_case_us"][case]["p50"] for r in part]
            lines.append(f"| {profile.upper()} | {case} | {min(counts)}–{max(counts)} | {value(statistics.mean(medians),2)} |")
    lines += ["", "Structural S04 is constructible by capability A. Proof mutation P04/P03 needs an observed signed transaction (B); fully signed A03 requires a registered malicious account (C) or a prior state change. Repeated identical public bytes may interact with mempool caching, and client RPC cost includes transport and scheduling; direct Go microbenchmarks isolate application verification more closely. The B0 mixed class repeats a classical-proof slot where B1 uses binding.", "",
              "## Interpretation boundary", "",
              "At 90% invalid, the means of paired within-profile p99 amplification are B0 " + value(next(g for g in grouped if (g["series"],g["profile"],g["ratio"]) == ("primary","b0",90))["inclusion_p99_amplification"]["mean"],3) + "× and B1 " + value(next(g for g in grouped if (g["series"],g["profile"],g["ratio"]) == ("primary","b1",90))["inclusion_p99_amplification"]["mean"],3) + "×. These bounded single-host rates did not locate a saturation threshold or establish DoS resistance. The 2× p99 and 20% goodput screening signals are provisional research heuristics, not standards.", "",
              "The superseded B0 secondary baseline run `034-secondary-b0-p0-r2` had a wall-clock discontinuity: its outer duration was 2463.8 s while Go monotonic test duration was 83.85 s. It remains hashed in the raw ledger. The replacement used monotonic duration and was measured in a fresh network under the same workload configuration.", ""]
    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("analysis", type=Path)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    args.output.write_text(render(args.analysis), encoding="utf-8")
    print(args.output)


if __name__ == "__main__":
    main()
