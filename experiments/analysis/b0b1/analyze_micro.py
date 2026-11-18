#!/usr/bin/env python3
"""Parse real Go benchmark output; never invent missing samples."""
import argparse
import json
from pathlib import Path
import re
import statistics

NAME = re.compile(r"(BenchmarkPhase05[^\s]+-\d+)")
VALUES = re.compile(r"\b(\d+)\s+([\d.]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op")


def parse(text):
    groups={}
    current=None
    for line in text.splitlines():
        named=NAME.search(line)
        if named:current=re.sub(r"-\d+$","",named.group(1))
        measured=VALUES.search(line)
        if current and measured:
            groups.setdefault(current,[]).append(dict(iterations=int(measured.group(1)),
                                                   ns_per_op=float(measured.group(2)),
                                                   bytes_per_op=int(measured.group(3)),
                                                   allocs_per_op=int(measured.group(4))))
    if not groups:raise ValueError("no Phase 05 benchmark samples")
    return groups


def summarize(groups):
    return {name:dict(sample_count=len(samples),raw=samples,
                      median_ns_per_op=statistics.median(s["ns_per_op"] for s in samples),
                      median_bytes_per_op=statistics.median(s["bytes_per_op"] for s in samples),
                      median_allocs_per_op=statistics.median(s["allocs_per_op"] for s in samples))
            for name,samples in sorted(groups.items())}


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument("input",type=Path)
    parser.add_argument("output",type=Path)
    args=parser.parse_args()
    text=args.input.read_text(errors="replace")
    result=summarize(parse(text))
    args.output.parent.mkdir(parents=True,exist_ok=True)
    args.output.write_text(json.dumps(result,indent=2)+"\n")
    print(f"benchmarks={len(result)} min_samples={min(x['sample_count'] for x in result.values())}")


if __name__=="__main__":main()
