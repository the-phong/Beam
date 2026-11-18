#!/usr/bin/env python3
"""Analyze real Phase 05 JSON records with independent runs as replicates."""
import argparse
import csv
import hashlib
import json
import math
import os
from pathlib import Path
import random
import statistics as stats
import sys


def quantile(values, q):
    if not values:
        return None
    ordered = sorted(values)
    position = (len(ordered)-1)*q
    low = math.floor(position)
    high = math.ceil(position)
    return ordered[low]+(ordered[high]-ordered[low])*(position-low)


def summary(values):
    if not values:
        return {"count": 0, "min": None, "mean": None, "p50": None,
                "p90": None, "p95": None, "p99": None, "max": None, "stdev": None}
    return dict(count=len(values), min=min(values), mean=stats.mean(values),
                p50=quantile(values, .5), p90=quantile(values, .9),
                p95=quantile(values, .95),
                p99=quantile(values, .99) if len(values)>=100 else "INSUFFICIENT SAMPLES",
                max=max(values), stdev=stats.stdev(values) if len(values)>1 else None)


def bootstrap_mean_ci(values, seed=3505):
    if len(values)<3:
        return None
    rng=random.Random(seed)
    means=[]
    for _ in range(3000):
        means.append(stats.mean(rng.choice(values) for _ in values))
    return [quantile(means,.025),quantile(means,.975)]


def process_metrics(record):
    first={}
    last={}
    peak={}
    selected=[sample for sample in record["process_samples"]
              if record["measurement_start_unix_nano"]<=sample["unix_nano"]<=record["measurement_end_unix_nano"]]
    for sample in selected:
        key=(sample["validator"],sample["role"],sample["pid"])
        first.setdefault(key,sample)
        last[key]=sample
        peak[key]=max(peak.get(key,0),sample["rss_bytes"])
    ticks=sum(max(0,last[key]["cpu_ticks"]-start["cpu_ticks"]) for key,start in first.items())
    split_available=bool(first) and all("user_cpu_ticks" in start and "system_cpu_ticks" in start
                                            and "user_cpu_ticks" in last[key] and "system_cpu_ticks" in last[key]
                                            for key,start in first.items())
    user_ticks=sum(max(0,last[key]["user_cpu_ticks"]-start["user_cpu_ticks"]) for key,start in first.items()) if split_available else None
    system_ticks=sum(max(0,last[key]["system_cpu_ticks"]-start["system_cpu_ticks"]) for key,start in first.items()) if split_available else None
    role_ticks={}
    role_peak={}
    for key,start in first.items():
        role=key[1]
        role_ticks[role]=role_ticks.get(role,0)+max(0,last[key]["cpu_ticks"]-start["cpu_ticks"])
        role_peak[role]=role_peak.get(role,0)+peak[key]
    wall=(record.get("measurement_duration_ns") or
          record["measurement_end_unix_nano"]-record["measurement_start_unix_nano"])/1e9
    tick_rate=os.sysconf("SC_CLK_TCK")
    return dict(cpu_seconds=ticks/tick_rate, user_cpu_seconds=user_ticks/tick_rate if user_ticks is not None else None,
                system_cpu_seconds=system_ticks/tick_rate if system_ticks is not None else None,
                cpu_percent=100*ticks/tick_rate/wall if wall>0 else None,
                peak_sampled_rss_bytes=sum(peak.values()),sample_count=len(selected),
                cpu_seconds_by_role={role:value/tick_rate for role,value in role_ticks.items()},
                peak_sampled_rss_bytes_by_role=role_peak,
                clock_ticks_per_second=tick_rate)


def per_run(record):
    events=record["events"]
    valid=[e for e in events if e["expected_code"]==0]
    invalid=[e for e in events if e["expected_code"]!=0]
    duration=(record["measurement_end_unix_nano"]-record["measurement_start_unix_nano"])/1e9
    if duration<=0: raise ValueError("nonpositive duration")
    counts={}
    for e in events: counts[str(e["code"])]=counts.get(str(e["code"]),0)+1
    return dict(run_id=record["config"]["run_id"],source_commit=record["source_commit"],
                source_tree=record["source_tree"],seed=record["config"]["seed"],
                profile=record["config"]["profile"],invalid_percent=record["config"]["invalid_percent"],
                rate_tps=record["config"]["rate_tps"],invalid_class=record["config"]["class"],
                generated=len(events),submitted=len(events),valid_submitted=len(valid),
                invalid_rejected=record["invalid_rejected"],valid_confirmed=record["valid_included"],
                duration_seconds=duration,incoming_tps=len(events)/duration,
                incoming_bytes_per_second=sum(e["bytes"] for e in events)/duration,
                accepted_admission_tps=len(valid)/duration,
                valid_committed_tps=record["valid_included"]/duration,
                rejected_tps=record["invalid_rejected"]/duration,
                error_count=sum(bool(e.get("error")) for e in events),
                timeout_count=sum("timeout" in e.get("error", "").lower() or "deadline" in e.get("error", "").lower() for e in events),
                input_size_bytes=summary([e["bytes"] for e in events]),
                admission_us=summary([e["admission_latency_ns"]/1000 for e in events]),
                invalid_admission_us=summary([e["admission_latency_ns"]/1000 for e in invalid]),
                valid_admission_us=summary([e["admission_latency_ns"]/1000 for e in valid]),
                inclusion_ms=summary([e["inclusion_latency_ns"]/1e6 for e in valid if e.get("included_height")]),
                rejection_codes=counts,process=process_metrics(record),
                matched_height=record["matched_height"],app_hash=record["app_hashes"][0],
                block_hash=record["block_hashes"][0],total_supply=record["total_supplies"][0])


def aggregate(rows):
    groups={}
    for row in rows:
        key=(row["profile"],row["invalid_percent"],row["rate_tps"],row["invalid_class"])
        groups.setdefault(key,[]).append(row)
    output=[]
    for key,items in sorted(groups.items()):
        admission=[x["admission_us"]["p50"] for x in items]
        throughput=[x["valid_committed_tps"] for x in items]
        cpu=[x["process"]["cpu_seconds"] for x in items]
        output.append(dict(profile=key[0],invalid_percent=key[1],rate_tps=key[2],invalid_class=key[3],
                           repetitions=len(items),run_ids=[x["run_id"] for x in items],
                           median_admission_us_mean=stats.mean(admission),
                           median_admission_us_ci95=bootstrap_mean_ci(admission),
                           valid_committed_tps_mean=stats.mean(throughput),
                           valid_committed_tps_ci95=bootstrap_mean_ci(throughput),
                           cpu_seconds_mean=stats.mean(cpu),cpu_seconds_ci95=bootstrap_mean_ci(cpu)))
    return output


def paired(rows):
    pairs={}
    for row in rows:
        if row["invalid_percent"] not in (0,50,90,100) or row["rate_tps"] not in (2,8):continue
        key=(row["invalid_percent"],row["rate_tps"],row["seed"])
        pairs.setdefault(key,{})[row["profile"]]=row
    output=[]
    for (ratio,rate,seed),profiles in sorted(pairs.items()):
        if set(profiles)!={"b0","b1"}:continue
        b0,b1=profiles["b0"],profiles["b1"]
        output.append(dict(invalid_percent=ratio,rate_tps=rate,seed=seed,
                           b0_run=b0["run_id"],b1_run=b1["run_id"],
                           median_admission_us_difference=b1["admission_us"]["p50"]-b0["admission_us"]["p50"],
                           median_admission_us_ratio=b1["admission_us"]["p50"]/b0["admission_us"]["p50"] if b0["admission_us"]["p50"] else None,
                           valid_committed_tps_difference=b1["valid_committed_tps"]-b0["valid_committed_tps"]))
    return output


def svg_plot(rows,path):
    # Plain standalone SVG; descriptive axes and no invented interpolation.
    primary=[r for r in rows if r["invalid_percent"] in (0,50,90,100) and r["rate_tps"]==8
             and r["invalid_class"] in ("classical","ml")]
    width,height=740,420
    peak=max((r["median_admission_us_mean"] for r in primary),default=1)*1.15
    lines=[f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">',
           '<rect width="100%" height="100%" fill="white"/>',
           '<text x="30" y="30" font-family="sans-serif" font-size="16">High offered rate: median admission latency by run</text>',
           '<line x1="70" y1="350" x2="690" y2="350" stroke="black"/>',
           '<line x1="70" y1="60" x2="70" y2="350" stroke="black"/>',
           '<text x="290" y="395" font-family="sans-serif">Invalid input percentage</text>',
           '<text x="10" y="55" font-family="sans-serif">µs</text>']
    for pct in (0,25,50,75,90,100):
        x=70+620*pct/100
        lines.extend((f'<line x1="{x:.1f}" y1="350" x2="{x:.1f}" y2="356" stroke="black"/>',
                      f'<text x="{x-10:.1f}" y="374" font-family="sans-serif" font-size="12">{pct}</text>'))
    for value in (0,500,1000):
        if value>peak:continue
        y=350-290*value/peak
        lines.extend((f'<line x1="64" y1="{y:.1f}" x2="70" y2="{y:.1f}" stroke="black"/>',
                      f'<text x="25" y="{y+4:.1f}" font-family="sans-serif" font-size="12">{value}</text>'))
    for profile,color in (("b0","#2763aa"),("b1","#b54b24")):
        points=[]
        for row in primary:
            if row["profile"]!=profile:continue
            x=70+620*row["invalid_percent"]/100
            y=350-290*row["median_admission_us_mean"]/peak
            points.append((x,y))
            lines.append(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="5" fill="{color}"/>')
        points.sort()
        if points:lines.append(f'<polyline points="{" ".join(f"{x:.1f},{y:.1f}" for x,y in points)}" fill="none" stroke="{color}" stroke-width="2"/>')
        lines.append(f'<text x="{510 if profile=="b0" else 600}" y="50" fill="{color}" font-family="sans-serif">{profile.upper()}</text>')
    lines.append('</svg>')
    path.write_text("\n".join(lines)+"\n",encoding="utf-8")


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument("raw_dir",type=Path)
    parser.add_argument("output_dir",type=Path)
    parser.add_argument("--require-complete",action="store_true")
    args=parser.parse_args()
    args.output_dir.mkdir(parents=True,exist_ok=True)
    records=[]
    failures=[]
    ledger_path=args.raw_dir/"execution.jsonl"
    ledger=[json.loads(line) for line in ledger_path.read_text().splitlines()] if ledger_path.exists() else []
    expected=122 if args.raw_dir.name.endswith("-full") else 64 if args.raw_dir.name.endswith("-b1") else 1 if args.raw_dir.name.endswith("-smoke") else None
    for entry in ledger:
        if entry["exit_code"]!=0 or not entry["data_present"]:failures.append(entry["run_id"])
    for path in sorted((args.raw_dir/"runs").glob("*.json")):
        raw=path.read_bytes()
        item=json.loads(raw)
        if not item.get("complete"):
            failures.append(path.name)
            continue
        if len(item["events"])!=item["config"]["count"]:raise ValueError(f"truncated event list {path}")
        if len(set(item["app_hashes"]))!=1 or len(set(item["block_hashes"]))!=1 or len(set(item["ordered_tx_digests"]))!=1:
            raise ValueError(f"same-height divergence {path}")
        records.append((item,hashlib.sha256(raw).hexdigest(),path.name))
    if not records:raise ValueError("no complete real runs")
    rows=[per_run(x[0]) for x in records]
    groups=aggregate(rows)
    pairs=paired(rows)
    (args.output_dir/"per_run.json").write_text(json.dumps(rows,indent=2)+"\n")
    (args.output_dir/"aggregate.json").write_text(json.dumps(groups,indent=2)+"\n")
    (args.output_dir/"paired.json").write_text(json.dumps(pairs,indent=2)+"\n")
    svg_plot(groups,args.output_dir/"admission_latency.svg")
    with (args.output_dir/"per_run.csv").open("w",newline="") as stream:
        writer=csv.DictWriter(stream,fieldnames=("run_id","source_commit","source_tree","seed","profile","invalid_percent","rate_tps","invalid_class","generated","submitted","valid_submitted","invalid_rejected","valid_confirmed","duration_seconds","incoming_tps","incoming_bytes_per_second","accepted_admission_tps","valid_committed_tps","rejected_tps","error_count","timeout_count","matched_height","app_hash","block_hash","total_supply"))
        writer.writeheader()
        for row in rows:writer.writerow({key:row[key] for key in writer.fieldnames})
    digest_manifest={str(path.relative_to(args.raw_dir)):hashlib.sha256(path.read_bytes()).hexdigest()
                     for path in sorted(args.raw_dir.rglob("*")) if path.is_file() and args.output_dir not in path.parents}
    (args.output_dir/"raw_sha256.json").write_text(json.dumps(digest_manifest,indent=2)+"\n")
    print(json.dumps(dict(complete_runs=len(rows),expected_runs=expected,ledger_runs=len(ledger),failed_runs=failures,groups=len(groups),paired_runs=len(pairs),source_trees=sorted({r["source_tree"] for r in rows})),indent=2))
    return 1 if args.require_complete and (failures or expected is None or len(ledger)!=expected or len(rows)!=expected) else 0


if __name__=="__main__":sys.exit(main())
