#!/usr/bin/env python3
"""Run bounded localhost Phase 05 experiments sequentially, retaining failures.

No signing keys are loaded: the public corpus supplies signed transactions.
Each Go test creates and reaps its own four validator processes and homes.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[3]
RAW = ROOT / "research/evidence/phase_05/raw"


def process_starttime(pid):
    """Read a Linux PID birth marker to avoid signalling a reused PID."""
    try:
        raw = Path(f"/proc/{pid}/stat").read_text()
        fields = raw[raw.rfind(")") + 2:].split()
        return int(fields[19])  # field 22 after state (field 3)
    except (OSError, ValueError, IndexError):
        return None


def owned_descendants(pid):
    """Snapshot only descendants of the process we launched."""
    result = []
    seen = {pid}
    def visit(parent):
        try:
            children = Path(f"/proc/{parent}/task/{parent}/children").read_text().split()
        except OSError:
            return
        for child in children:
            candidate = int(child)
            if candidate in seen:
                continue
            seen.add(candidate)
            marker = process_starttime(candidate)
            if marker is None:
                continue
            result.append((candidate, marker))
            visit(candidate)
    visit(pid)
    return result


def stop_owned_tree(pid):
    """Stop exact captured PIDs after a timeout; never scan by process name."""
    targets = owned_descendants(pid) + [(pid, process_starttime(pid))]
    targets = [(child, marker) for child, marker in targets if marker is not None]
    for sig in (signal.SIGTERM, signal.SIGKILL):
        for child, marker in reversed(targets):
            if process_starttime(child) == marker:
                try:
                    os.kill(child, sig)
                except ProcessLookupError:
                    pass
        if sig == signal.SIGTERM:
            time.sleep(1)
    return [child for child, _ in targets]


def run_bounded(command, log, env, timeout):
    process = subprocess.Popen(command, cwd=ROOT, env=env, stdout=log,
                               stderr=subprocess.STDOUT, start_new_session=True)
    try:
        return process.wait(timeout=timeout), []
    except subprocess.TimeoutExpired:
        terminated = stop_owned_tree(process.pid)
        process.wait(timeout=10)
        return 124, terminated


def conditions(mode):
    if mode == "smoke":
        yield dict(profile="b1", invalid_percent=50, rate_tps=2, count=12,
                   seed=5, class_="ml", repetition=1)
        return
    profiles = ("b0", "b1") if mode == "full" else ("b1",)
    for profile in profiles:
        for ratio in (0, 50, 90, 100):
            for load, rate in (("low", 2), ("high", 8)):
                for rep in range(1, 6):
                    yield dict(profile=profile, invalid_percent=ratio, rate_tps=rate,
                               count=12, seed=1000+rep, class_="classical" if profile == "b0" else "ml",
                               repetition=rep, load=load)
        for ratio in (25, 75):
            for rep in range(1, 4):
                yield dict(profile=profile, invalid_percent=ratio, rate_tps=4,
                           count=12, seed=2000+rep, class_="mixed", repetition=rep, load="moderate")
        for klass in (("structural", "classical", "state", "mixed") if profile == "b0"
                      else ("structural", "binding", "classical", "ml", "state", "mixed")):
            for rep in range(1, 4):
                yield dict(profile=profile, invalid_percent=100, rate_tps=4,
                           count=12, seed=3000+rep, class_=klass, repetition=rep, load="focused")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--mode", choices=("smoke", "b1", "full"), required=True)
    parser.add_argument("--output-root", type=Path)
    args = parser.parse_args()
    stamp = time.strftime("%Y%m%dT%H%M%SZ", time.gmtime())
    root = args.output_root or RAW / (f"{stamp}-{args.mode}")
    root.mkdir(mode=0o700, parents=True, exist_ok=False)
    runs = root / "runs"
    logs = root / "logs"
    nodes = root / "node-logs"
    for path in (runs, logs, nodes):
        path.mkdir(mode=0o700)
    manifest = root / "execution.jsonl"
    failures = 0
    for index, spec in enumerate(conditions(args.mode), 1):
        klass = spec.pop("class_")
        repetition = spec.pop("repetition")
        load = spec.pop("load", "smoke")
        run_id = f"{index:03d}-{spec['profile']}-p{spec['invalid_percent']}-{load}-{klass}-r{repetition}"
        config = {**spec, "class": klass, "run_id": run_id}
        env = os.environ.copy()
        env["PLUTO_PHASE05_CONFIG"] = json.dumps(config, separators=(",", ":"))
        env["PLUTO_PHASE05_RAW_DIR"] = str(runs)
        env["PLUTO_4V_ARTIFACTS"] = str(nodes / run_id)
        command = ["go", "test", "-tags", "integration fourval", "./test/integration",
                   "-run", "^TestPhase05Workload$", "-timeout", "180s", "-count=1", "-v"]
        begun = time.time()
        with (logs / f"{run_id}.log").open("wb") as log:
            code, timeout_pids = run_bounded(command, log, env, 210)
        data_path = runs / (run_id + ".json")
        record = dict(run_id=run_id, command=command, config=config, exit_code=code,
                      wall_seconds=time.time()-begun, data_present=data_path.is_file(),
                      timeout_terminated_pids=timeout_pids,
                      log_sha256=hashlib.sha256((logs / f"{run_id}.log").read_bytes()).hexdigest())
        with manifest.open("a", encoding="utf-8") as stream:
            stream.write(json.dumps(record, sort_keys=True) + "\n")
            stream.flush()
            os.fsync(stream.fileno())
        print(f"{run_id}: exit={code} data={record['data_present']} elapsed={record['wall_seconds']:.1f}s", flush=True)
        if code != 0 or not data_path.is_file():
            failures += 1
            # Preserve every failed run. Continuing makes omissions visible,
            # but no failed condition is silently replaced or marked PASS.
    print(f"root={root} runs={index} failures={failures}")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
