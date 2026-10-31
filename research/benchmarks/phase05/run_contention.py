#!/usr/bin/env python3
"""Bounded matched-legitimate-rate four-validator experiments.

Each run uses a fresh network and public signed corpus. Failures and timeouts
remain in execution.jsonl; no failed run is replaced or silently omitted.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import sys
import time

from run_matrix import ROOT, RAW, run_bounded


def conditions(mode):
    if mode == "smoke":
        yield dict(profile="b1", valid_rate_tps=1, invalid_rate_tps=1,
                   valid_count=12, invalid_count=12, class_="ml", seed=5,
                   series="smoke", ratio=50, repetition=1)
        return
    for profile in ("b0", "b1"):
        for repetition in range(1, 6):
            for ratio, invalid_rate in ((0, 0), (25, .5), (50, 1.5),
                                        (75, 4.5), (90, 13.5)):
                yield dict(profile=profile, valid_rate_tps=1.5,
                           invalid_rate_tps=invalid_rate, valid_count=300,
                           invalid_count=round(200 * invalid_rate),
                           class_="mixed", seed=5000 + repetition,
                           series="primary", ratio=ratio,
                           repetition=repetition)
            yield dict(profile=profile, valid_rate_tps=0,
                       invalid_rate_tps=15, valid_count=0,
                       invalid_count=3000, class_="mixed",
                       seed=5000 + repetition, series="invalid_only",
                       ratio=100, repetition=repetition)
        for repetition in range(1, 4):
            for ratio, invalid_rate in ((0, 0), (50, 4), (75, 12)):
                yield dict(profile=profile, valid_rate_tps=4,
                           invalid_rate_tps=invalid_rate, valid_count=300,
                           invalid_count=round(75 * invalid_rate),
                           class_="mixed", seed=6000 + repetition,
                           series="secondary", ratio=ratio,
                           repetition=repetition)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--mode", choices=("smoke", "full", "repair34"), required=True)
    parser.add_argument("--output-root", type=Path)
    args = parser.parse_args()
    stamp = time.strftime("%Y%m%dT%H%M%SZ", time.gmtime())
    root = args.output_root or RAW / f"{stamp}-contention-{args.mode}"
    root.mkdir(mode=0o700, parents=True, exist_ok=False)
    for name in ("runs", "logs", "node-logs"):
        (root / name).mkdir(mode=0o700)
    manifest = root / "execution.jsonl"
    failures = 0
    count = 0
    completed = 0
    selected = ([(34, list(conditions("full"))[33])]
                if args.mode == "repair34" else enumerate(conditions(args.mode), 1))
    for count, spec in selected:
        completed += 1
        klass = spec.pop("class_")
        series = spec.pop("series")
        ratio = spec.pop("ratio")
        repetition = spec.pop("repetition")
        run_id = f"{count:03d}-{series}-{spec['profile']}-p{ratio}-r{repetition}"
        config = {**spec, "class": klass, "run_id": run_id}
        env = os.environ.copy()
        env["PLUTO_PHASE05_CONTENTION_CONFIG"] = json.dumps(config, separators=(",", ":"))
        env["PLUTO_PHASE05_RAW_DIR"] = str(root / "runs")
        env["PLUTO_4V_ARTIFACTS"] = str(root / "node-logs" / run_id)
        command = ["go", "test", "-tags", "integration fourval",
                   "./test/integration", "-run", "^TestPhase05Contention$",
                   "-timeout", "360s", "-count=1", "-v"]
        begun = time.time()
        path = root / "logs" / (run_id + ".log")
        with path.open("wb") as log:
            code, stopped = run_bounded(command, log, env, 390)
        data = root / "runs" / (run_id + ".json")
        record = dict(run_id=run_id, series=series, ratio=ratio,
                      repetition=repetition, command=command, config=config,
                      exit_code=code, wall_seconds=time.time() - begun,
                      data_present=data.is_file(), timeout_terminated_pids=stopped,
                      log_sha256=hashlib.sha256(path.read_bytes()).hexdigest(),
                      data_sha256=hashlib.sha256(data.read_bytes()).hexdigest()
                      if data.is_file() else None)
        with manifest.open("a", encoding="utf-8") as stream:
            stream.write(json.dumps(record, sort_keys=True) + "\n")
            stream.flush()
            os.fsync(stream.fileno())
        print(f"{run_id}: exit={code} data={data.is_file()} "
              f"elapsed={record['wall_seconds']:.1f}s", flush=True)
        if code != 0 or not data.is_file():
            failures += 1
    print(f"root={root} runs={completed} failures={failures}")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
