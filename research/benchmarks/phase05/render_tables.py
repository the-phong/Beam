#!/usr/bin/env python3
"""Render publication-safe tables from verified raw-derived summaries."""
import argparse
import json
from pathlib import Path


def clean(value, digits=3):
    if value is None:return "UNAVAILABLE"
    if isinstance(value,str):return value
    return f"{value:.{digits}f}"


def render(aggregates, runs, micro):
    lines=["# Phase 05 raw-derived result tables", "",
           "Each network row averages independent run summaries; it does not pool requests as independent trials.",
           "`admission` is client RPC wall time. `confirmed/s` counts only valid included transactions.",
           "`CPU s` is sampled process user+system time for all Pluto/Comet processes; it is not wall time.",
           "", "## Primary network matrix", "",
           "| Profile | Invalid target | Offered tx/s | Runs | Admission p50 mean (µs) | Admission p95 mean (µs) | p99 | 95% run bootstrap interval for p50 (µs) | Valid inclusion p50 mean (ms) | Valid confirmed/s mean | CPU s mean | Four-process RSS sum (MiB) |",
           "|---|---:|---:|---:|---:|---:|---|---|---:|---:|---:|---:|"]
    primary=[x for x in aggregates if x["invalid_percent"] in (0,50,90,100) and x["rate_tps"] in (2,8) and x["invalid_class"] in ("classical","ml")]
    for row in sorted(primary,key=lambda x:(x["profile"],x["invalid_percent"],x["rate_tps"])):
        interval=row["median_admission_us_ci95"]
        ci="UNAVAILABLE" if interval is None else f"[{clean(interval[0])}, {clean(interval[1])}]"
        subset=[r for r in runs if r["run_id"] in row["run_ids"]]
        p95=sum(r["admission_us"]["p95"] for r in subset)/len(subset)
        inclusion=[r["inclusion_ms"]["p50"] for r in subset if r["inclusion_ms"]["p50"] is not None]
        inclusion_mean=sum(inclusion)/len(inclusion) if inclusion else None
        rss=sum(r["process"]["peak_sampled_rss_bytes"] for r in subset)/len(subset)/1048576
        p99="INSUFFICIENT SAMPLES" if any(r["admission_us"]["p99"]=="INSUFFICIENT SAMPLES" for r in subset) else "See raw"
        lines.append(f"| {row['profile'].upper()} | {row['invalid_percent']}% | {row['rate_tps']} | {row['repetitions']} | {clean(row['median_admission_us_mean'])} | {clean(p95)} | {p99} | {ci} | {clean(inclusion_mean)} | {clean(row['valid_committed_tps_mean'])} | {clean(row['cpu_seconds_mean'])} | {clean(rss)} |")
    lines += ["", "## Secondary moderate-rate conditions", "",
              "| Profile | Invalid target | Offered tx/s | Runs | Admission p50 mean (µs) | Valid confirmed/s mean | CPU s mean |",
              "|---|---:|---:|---:|---:|---:|---:|"]
    for row in sorted((x for x in aggregates if x["invalid_percent"] in (25,75)
                       and x["rate_tps"]==4), key=lambda x:(x["profile"],x["invalid_percent"])):
        lines.append(f"| {row['profile'].upper()} | {row['invalid_percent']}% | {row['rate_tps']} | {row['repetitions']} | {clean(row['median_admission_us_mean'])} | {clean(row['valid_committed_tps_mean'])} | {clean(row['cpu_seconds_mean'])} |")
    lines += ["", "## Primary traffic rates", "",
              "Rates are means of independent run rates over each run's observed window; offered pacing targets are listed above.", "",
              "| Profile | Invalid target | Target tx/s | Incoming tx/s | Incoming KiB/s | Accepted admission tx/s | Rejected tx/s | Confirmed valid tx/s | Client errors/timeouts |",
              "|---|---:|---:|---:|---:|---:|---:|---:|---:|"]
    for row in sorted(primary,key=lambda x:(x["profile"],x["invalid_percent"],x["rate_tps"])):
        subset=[r for r in runs if r["run_id"] in row["run_ids"]]
        avg=lambda key:sum(r[key] for r in subset)/len(subset)
        errors=sum(r["error_count"] for r in subset)
        timeouts=sum(r["timeout_count"] for r in subset)
        lines.append(f"| {row['profile'].upper()} | {row['invalid_percent']}% | {row['rate_tps']} | {clean(avg('incoming_tps'))} | {clean(avg('incoming_bytes_per_second')/1024)} | {clean(avg('accepted_admission_tps'))} | {clean(avg('rejected_tps'))} | {clean(avg('valid_committed_tps'))} | {errors}/{timeouts} |")
    lines += ["", "## Focused invalid classes", "",
              "| Profile | Invalid class | Runs | Admission median mean (µs) | CPU s mean |",
              "|---|---|---:|---:|---:|"]
    for row in sorted((x for x in aggregates if x["rate_tps"]==4 and x["invalid_percent"]==100),key=lambda x:(x["profile"],x["invalid_class"])):
        lines.append(f"| {row['profile'].upper()} | {row['invalid_class']} | {row['repetitions']} | {clean(row['median_admission_us_mean'])} | {clean(row['cpu_seconds_mean'])} |")
    lines += ["", "## Direct and callback microbenchmarks", "",
              "| Benchmark | Command repetitions | Median ns/op | Median B/op | Median allocs/op |",
              "|---|---:|---:|---:|---:|"]
    for name,row in sorted(micro.items()):
        lines.append(f"| `{name}` | {row['sample_count']} | {clean(row['median_ns_per_op'],1)} | {clean(row['median_bytes_per_op'],0)} | {clean(row['median_allocs_per_op'],0)} |")
    p99_counts=sum(1 for row in runs if row["admission_us"]["p99"]!="INSUFFICIENT SAMPLES")
    lines += ["", "## Sample sufficiency and raw link", "",
              f"Complete independent network runs: **{len(runs)}**. Runs with sufficient per-run admission samples for p99: **{p99_counts}**.",
              "For short 12-request runs, p99 is `INSUFFICIENT SAMPLES`. Per-run p50/p90/p95, counts, standard deviation, inclusion and resource observations are in `per_run.json`.",
              "The full raw records and execution logs remain local; `RAW_SHA256.txt` identifies them without publishing their contents.",""]
    return "\n".join(lines)


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument("analysis_dir",type=Path)
    parser.add_argument("micro_json",type=Path)
    parser.add_argument("output",type=Path)
    args=parser.parse_args()
    aggregates=json.loads((args.analysis_dir/"aggregate.json").read_text())
    runs=json.loads((args.analysis_dir/"per_run.json").read_text())
    micro=json.loads(args.micro_json.read_text())
    args.output.parent.mkdir(parents=True,exist_ok=True)
    args.output.write_text(render(aggregates,runs,micro))
    print(f"wrote {args.output} from {len(runs)} independent runs")


if __name__=="__main__":main()
