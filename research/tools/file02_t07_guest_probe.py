#!/usr/bin/env python3
"""Standalone read-only clock/host probe copied to approved T07 guest VMs."""

import hashlib
import json
import os
from pathlib import Path
import platform
import socket
import subprocess
import sys
import time


def optional(path):
    try:
        return Path(path).read_text().strip()
    except OSError:
        return None


def run(*argv):
    try:
        result = subprocess.run(argv, capture_output=True, text=True, timeout=5, check=False)
        return result.stdout.strip() if result.returncode == 0 else None
    except (OSError, subprocess.TimeoutExpired):
        return None


def throttle():
    paths = sorted(Path("/sys/devices/system/cpu").glob("cpu*/thermal_throttle/*_throttle_count"))
    try:
        return sum(int(optional(p)) for p in paths) if paths else None
    except (TypeError, ValueError):
        return None


def inventory():
    cpuinfo = optional("/proc/cpuinfo") or ""
    model = next((line.split(":", 1)[1].strip() for line in cpuinfo.splitlines()
                  if line.startswith("model name")), None)
    flags = set(next((line.split(":", 1)[1].strip() for line in cpuinfo.splitlines()
                      if line.startswith("flags")), "").split())
    stat = os.statvfs("/")
    sync = run("timedatectl", "show", "-p", "NTP", "-p", "NTPSynchronized")
    links = {p.name: optional(p / "mtu") for p in Path("/sys/class/net").iterdir()}
    machine_id = optional("/etc/machine-id")
    product_uuid = optional("/sys/class/dmi/id/product_uuid")
    return {"guest_hostname": socket.gethostname(), "kernel": platform.release(),
            "machine": platform.machine(), "cpu_model": model, "logical_cpus": os.cpu_count(),
            "product_name": optional("/sys/class/dmi/id/product_name"),
            "machine_id_sha256": hashlib.sha256(machine_id.encode()).hexdigest() if machine_id else None,
            "product_uuid_sha256": hashlib.sha256(product_uuid.encode()).hexdigest() if product_uuid else None,
            "timing_cpu_features": sorted(flags.intersection(
                {"constant_tsc", "nonstop_tsc", "tsc_reliable", "rdtscp", "hypervisor"})),
            "filesystem_type": run("findmnt", "-no", "FSTYPE", "-T", "/"),
            "filesystem_free_fraction": stat.f_bavail / stat.f_blocks,
            "network_interfaces_mtu": links,
            "clock_sync_status": dict(line.split("=", 1) for line in sync.splitlines() if "=" in line)
            if sync else None,
            "thermal_sensor_status": "EXPOSED" if list(Path("/sys/class/thermal").glob("thermal_zone*/temp"))
                                     else "UNAVAILABLE",
            "thermal_throttle_counter_status": "EXPOSED" if throttle() is not None else "UNAVAILABLE"}


def sample(index):
    first = time.clock_gettime_ns(time.CLOCK_MONOTONIC)
    realtime = time.clock_gettime_ns(time.CLOCK_REALTIME)
    last = time.clock_gettime_ns(time.CLOCK_MONOTONIC)
    return {"index": index, "mono_before_ns": first, "realtime_ns": realtime,
            "mono_after_ns": last}


def survey():
    warmup_start = time.clock_gettime_ns(time.CLOCK_MONOTONIC)
    time.sleep(1)
    warmup_end = time.clock_gettime_ns(time.CLOCK_MONOTONIC)
    before = throttle()
    readings = []
    for index in range(200):
        readings.append(sample(index))
        if index + 1 < 200:
            time.sleep(.05)
    steady_end = time.clock_gettime_ns(time.CLOCK_MONOTONIC)
    time.sleep(3)
    drain_end = time.clock_gettime_ns(time.CLOCK_MONOTONIC)
    return {"schema": "pluto.file02.t07_guest_survey.v1", "inventory": inventory(),
            "warmup_start_ns": warmup_start, "warmup_end_ns": warmup_end,
            "steady_end_ns": steady_end, "drain_end_ns": drain_end,
            "throttle_before": before, "throttle_after": throttle(), "samples": readings}


def main():
    if len(sys.argv) != 2 or sys.argv[1] not in ("clock", "survey", "serve"):
        raise SystemExit("usage: file02_t07_guest_probe.py clock|survey|serve")

    def answer(command):
        if command == "clock":
            before = time.clock_gettime_ns(time.CLOCK_REALTIME)
            after = time.clock_gettime_ns(time.CLOCK_REALTIME)
            return {"remote_before_ns": before, "remote_after_ns": after}
        if command == "survey":
            return survey()
        raise ValueError("unknown probe command")

    if sys.argv[1] == "serve":
        for line in sys.stdin:
            command = line.strip()
            if command == "quit":
                return
            print(json.dumps(answer(command), sort_keys=True, separators=(",", ":")), flush=True)
    else:
        print(json.dumps(answer(sys.argv[1]), sort_keys=True, separators=(",", ":")))


if __name__ == "__main__":
    main()
