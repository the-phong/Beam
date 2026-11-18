#!/usr/bin/env python3
"""Capture and verify a bounded T07 single-host clock/inventory survey."""

import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import platform
import subprocess
import sys
import time
from uuid import uuid4

import file02_common as common
from file02_common03 import ARTIFACTS, ROOT, git_value, private_session, require

SCHEMA = "pluto.file02.t07_single_host.v1"
SAMPLE_COUNT = 200
STEP_LIMIT_NS = 10_000_000


def read_optional(path):
    try:
        return Path(path).read_text().strip()
    except OSError:
        return None


def command_output(*args):
    try:
        run = subprocess.run(args, capture_output=True, text=True, timeout=5, check=False)
        return run.stdout.strip() if run.returncode == 0 else None
    except (OSError, subprocess.TimeoutExpired):
        return None


def throttle_count():
    paths = sorted(Path("/sys/devices/system/cpu").glob("cpu*/thermal_throttle/*_throttle_count"))
    if not paths:
        return None
    values = [read_optional(path) for path in paths]
    try:
        return sum(int(value) for value in values)
    except (TypeError, ValueError):
        return None


def inventory():
    cpuinfo = read_optional("/proc/cpuinfo") or ""
    model = next((line.split(":", 1)[1].strip() for line in cpuinfo.splitlines()
                  if line.startswith("model name")), None)
    flags_line = next((line.split(":", 1)[1].strip() for line in cpuinfo.splitlines()
                       if line.startswith("flags")), "")
    flags = set(flags_line.split())
    sync = command_output("timedatectl", "show", "-p", "NTP", "-p", "NTPSynchronized")
    sync_state = dict(line.split("=", 1) for line in sync.splitlines() if "=" in line) if sync else None
    fs = command_output("findmnt", "-no", "FSTYPE", "-T", str(ROOT))
    stat = os.statvfs(ROOT)
    mtu = read_optional("/sys/class/net/lo/mtu")
    return {"kernel": platform.release(), "machine": platform.machine(),
            "cpu_model": model, "logical_cpus": os.cpu_count(),
            "timing_cpu_features": sorted(flags.intersection(
                {"constant_tsc", "nonstop_tsc", "tsc_reliable", "rdtscp", "hypervisor"})),
            "filesystem_type": fs, "filesystem_free_fraction": stat.f_bavail / stat.f_blocks,
            "network_profile": "loopback_only", "loopback_mtu": int(mtu) if mtu else None,
            "clock_sync_status": sync_state,
            "thermal_sensor": "UNAVAILABLE" if not list(Path("/sys/class/thermal").glob("thermal_zone*/temp"))
                              else "EXPOSED",
            "thermal_throttle_counter_status": "UNAVAILABLE" if throttle_count() is None
                                               else "EXPOSED"}


def summarize(samples, before, after):
    require(len(samples) == SAMPLE_COUNT, "clock sample count")
    mids, offsets, widths = [], [], []
    for sample in samples:
        first, realtime, last = (sample[key] for key in ("mono_before_ns", "realtime_ns", "mono_after_ns"))
        require(all(isinstance(x, int) and x > 0 for x in (first, realtime, last)) and last >= first,
                "invalid clock sample")
        mid = (first + last) // 2
        mids.append(mid)
        offsets.append(realtime - mid)
        widths.append(last - first)
    require(all(a < b for a, b in zip(mids, mids[1:])), "monotonic clock discontinuity")
    steps = [abs(b - a) for a, b in zip(offsets, offsets[1:])]
    maximum = max(steps)
    return {"sample_count": len(samples), "sample_interval_ms": 50,
            "elapsed_monotonic_ns": mids[-1] - mids[0],
            "max_adjacent_wall_monotonic_offset_step_ns": maximum,
            "max_read_bracket_ns": max(widths),
            "same_host_mapping_uncertainty_ns": (max(widths) + 1) // 2,
            "clock_continuous_within_10ms": maximum <= STEP_LIMIT_NS,
            "throttle_count_before": before, "throttle_count_after": after,
            "throttle_count_delta": after - before if before is not None and after is not None else None,
            "throttle_unavailable_reason": "guest exposes no readable thermal throttle counters"
            if before is None or after is None else None,
            "cross_host_offset_ns": None,
            "cross_host_uncertainty_reason": "no independent second host available for measurement"}


def capture():
    require(not git_value("status", "--porcelain"), "commit source before T07 survey")
    source, tree = git_value("rev-parse", "HEAD"), git_value("rev-parse", "HEAD^{tree}")
    ARTIFACTS.mkdir(parents=True, exist_ok=True, mode=0o700)
    session = ARTIFACTS / ("t07-host-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
                           + "-" + uuid4().hex[:8])
    session.mkdir(mode=0o700)
    info = inventory()
    before = throttle_count()
    time.sleep(1)
    samples = []
    for index in range(SAMPLE_COUNT):
        first = time.clock_gettime_ns(time.CLOCK_MONOTONIC)
        realtime = time.clock_gettime_ns(time.CLOCK_REALTIME)
        last = time.clock_gettime_ns(time.CLOCK_MONOTONIC)
        samples.append({"index": index, "mono_before_ns": first,
                        "realtime_ns": realtime, "mono_after_ns": last})
        if index + 1 < SAMPLE_COUNT:
            time.sleep(.05)
    after = throttle_count()
    summary = {"schema": SCHEMA, "scope": "T_PREFLIGHT_PARTIAL",
               "source_commit": source, "source_tree": tree,
               "placement": "single_host_wsl2", "inventory": info,
               "clock_resolution_ns": {
                   "monotonic": round(1e9 * time.clock_getres(time.CLOCK_MONOTONIC)),
                   "realtime": round(1e9 * time.clock_getres(time.CLOCK_REALTIME))},
               "survey": summarize(samples, before, after)}
    common.write_once(session / "samples.json", json.dumps(samples, indent=2) + "\n")
    common.write_once(session / "summary.json", json.dumps(summary, indent=2, sort_keys=True) + "\n")
    manifest = common.freeze_raw(session)
    print(f"T07_SINGLE_HOST={'PASS' if summary['survey']['clock_continuous_within_10ms'] else 'FAIL'} "
          f"session={session} manifest_sha256={manifest}")


def verify(path):
    session = private_session(path)
    manifest = common.verify_raw(session)
    summary = common.read_json(session / "summary.json")
    samples = common.read_json(session / "samples.json")
    survey = summary["survey"]
    require(summary["schema"] == SCHEMA and summary["scope"] == "T_PREFLIGHT_PARTIAL" and
            summarize(samples, survey["throttle_count_before"], survey["throttle_count_after"]) == survey,
            "T07 survey differs from raw")
    print(f"T07_SINGLE_HOST_VERIFY=OK manifest_sha256={manifest}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("capture")
    p = sub.add_parser("verify")
    p.add_argument("session")
    args = parser.parse_args()
    try:
        if args.command == "capture":
            capture()
        else:
            verify(args.session)
        return 0
    except (OSError, ValueError, TypeError, KeyError) as exc:
        print(f"T07_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
