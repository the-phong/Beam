#!/usr/bin/env python3
"""Nearest-rank diagnostic percentiles from logged Pebble sync samples."""
import math
import re
import sys
from collections import defaultdict
from pathlib import Path

groups = defaultdict(list)
for line in Path(sys.argv[1]).read_text().splitlines():
    match = re.search(r"sync_sample k=(\d+) trial=(\d+) elapsed_ns=(\d+)", line)
    if match:
        groups[int(match.group(1))].append(int(match.group(3)))
for k in (1, 64, 4096, 65536):
    values = sorted(groups[k])
    assert len(values) == 30, (k, len(values))
    def rank(p):
        return values[math.ceil(p * len(values)) - 1]
    print(f"K={k} n={len(values)} median_ns={rank(.5)} p95_ns={rank(.95)} p99_ns={rank(.99)} min_ns={values[0]} max_ns={values[-1]}")
