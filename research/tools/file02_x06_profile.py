#!/usr/bin/env python3
"""Isolated primitive CPU sampling; not a per-stage CheckTx CPU estimator."""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import sys

import file02_common as common

ROOT = Path(__file__).resolve().parents[2]
BENCH = re.compile(r"^BenchmarkFile02R01/(secp256k1|mldsa44)\s+(\d+)\s+([0-9.]+) ns/op", re.M)
SAMPLES = re.compile(r"Total samples = ([0-9.]+)s")


def run(session):
    session = Path(session).resolve()
    raw = session / "raw"
    if not session.is_relative_to(ROOT / "research/artifacts/file02") or not raw.is_dir():
        raise ValueError("use a private File02 session/raw directory")
    if any(raw.iterdir()):
        raise ValueError("raw directory must start empty")
    env = os.environ.copy()
    env["GOMAXPROCS"] = "1"
    rows = []
    for index in range(4):
        for primitive in (("secp256k1", "mldsa44") if index % 2 == 0 else ("mldsa44", "secp256k1")):
            name = f"{index:02d}-{primitive}"
            profile = raw / f"{name}.pprof"
            cmd = ["go", "test", "./internal/experiment/benchmark", "-run", "^$",
                   "-bench", f"^BenchmarkFile02R01$/^{primitive}$", "-benchtime=2s",
                   "-cpu=1", "-count=1", f"-cpuprofile={profile}"]
            done = subprocess.run(cmd, cwd=ROOT, env=env, capture_output=True, text=True)
            (raw / f"{name}.bench.txt").write_text(done.stdout + done.stderr)
            if done.returncode:
                raise RuntimeError(f"{name} benchmark exit {done.returncode}")
            matched = BENCH.search(done.stdout)
            if not matched or matched.group(1) != primitive:
                raise RuntimeError(f"{name} benchmark parser mismatch")
            baseline_cmd = ["go", "test", "./internal/experiment/benchmark", "-run", "^$",
                            "-bench", f"^BenchmarkFile02R01$/^{primitive}$", "-benchtime=2s",
                            "-cpu=1", "-count=1"]
            baseline = subprocess.run(baseline_cmd, cwd=ROOT, env=env, capture_output=True, text=True)
            (raw / f"{name}.baseline.txt").write_text(baseline.stdout + baseline.stderr)
            baseline_match = BENCH.search(baseline.stdout)
            if baseline.returncode or not baseline_match or baseline_match.group(1) != primitive:
                raise RuntimeError(f"{name} unprofiled baseline failed")
            pprof = subprocess.run(["go", "tool", "pprof", "-top", "-nodecount=40", str(profile)],
                                   cwd=ROOT, capture_output=True, text=True)
            (raw / f"{name}.top.txt").write_text(pprof.stdout + pprof.stderr)
            if pprof.returncode or not SAMPLES.search(pprof.stdout):
                raise RuntimeError(f"{name} CPU profile unreadable")
            rows.append({"index": index, "primitive": primitive,
                         "iterations": int(matched.group(2)), "wall_ns_op": float(matched.group(3)),
                         "unprofiled_wall_ns_op": float(baseline_match.group(3)),
                         "unprofiled_iterations": int(baseline_match.group(2)),
                         "sampled_cpu_seconds": float(SAMPLES.search(pprof.stdout).group(1)),
                         "benchmark_exit": done.returncode, "baseline_exit": baseline.returncode,
                         "profile_exit": pprof.returncode})
    (raw / "summary.json").write_text(json.dumps({"schema": "pluto.file02.x06_profile.v1",
        "scope": "isolated valid signature primitives only; CPU_crypto per CheckTx unavailable",
        "rows": rows}, indent=2, sort_keys=True) + "\n")
    print(common.freeze_raw(raw))


def verify(session):
    raw = Path(session).resolve() / "raw"
    digest = common.verify_raw(raw)
    summary = json.loads((raw / "summary.json").read_text())
    if summary.get("schema") != "pluto.file02.x06_profile.v1" or len(summary.get("rows", [])) != 8:
        raise ValueError("summary schema/count")
    for row in summary["rows"]:
        name = f"{row['index']:02d}-{row['primitive']}"
        bench = (raw / f"{name}.bench.txt").read_text()
        baseline = (raw / f"{name}.baseline.txt").read_text()
        top = (raw / f"{name}.top.txt").read_text()
        match = BENCH.search(bench)
        baseline_match = BENCH.search(baseline)
        samples = SAMPLES.search(top)
        if (not match or not baseline_match or not samples or row["primitive"] != match.group(1)
                or row["iterations"] != int(match.group(2))
                or row["wall_ns_op"] != float(match.group(3))
                or row["unprofiled_iterations"] != int(baseline_match.group(2))
                or row["unprofiled_wall_ns_op"] != float(baseline_match.group(3))
                or row["sampled_cpu_seconds"] != float(samples.group(1))
                or row["sampled_cpu_seconds"] < 1.0
                or not (raw / f"{name}.pprof").is_file()):
            raise ValueError(f"profile result mismatch: {name}")
    print(digest)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("run", "verify"))
    parser.add_argument("session")
    args = parser.parse_args()
    try:
        (run if args.command == "run" else verify)(args.session)
    except (OSError, ValueError, RuntimeError) as exc:
        print(exc, file=sys.stderr)
        sys.exit(2)
