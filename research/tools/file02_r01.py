#!/usr/bin/env python3
"""Locked R01 primitive benchmark producer and immutable evidence auditor."""

import argparse
import csv
import hashlib
import json
import os
from pathlib import Path
import platform
import random
import re
import statistics
import subprocess
import sys
import time
from datetime import datetime, timezone

import file02_common as common

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
BENCH = re.compile(r"^BenchmarkFile02R01/(secp256k1|mldsa44)-\d+\s+(\d+)\s+([0-9.]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op\s*$")
COMMAND = ["go", "test", "-run", "^$", "-bench", "^BenchmarkFile02R01$",
           "-benchtime=200ms", "-benchmem", "-cpu=1", "-count=1",
           "./internal/experiment/benchmark"]
SEED = 20261113
REPETITIONS = 12


def require(condition, message):
    if not condition:
        raise ValueError(message)


def write_new(path, obj):
    with Path(path).open("x", encoding="utf-8") as out:
        json.dump(obj, out, sort_keys=True, indent=2)
        out.write("\n")


def git(*args):
    return subprocess.check_output(["git", *args], cwd=ROOT, text=True).strip()


def cpu_inventory():
    info = Path("/proc/cpuinfo").read_text()
    def field(name):
        match = re.search(r"^" + re.escape(name) + r"\s*:\s*(.+)$", info, re.M)
        return match.group(1).strip() if match else "UNAVAILABLE"
    return {"model": field("model name"), "flags": field("flags").split(),
            "logical_cpus": os.cpu_count(), "architecture": platform.machine(),
            "os": platform.platform()}


def orders():
    result = ["secp_first"] * 6 + ["ml_first"] * 6
    random.Random(SEED).shuffle(result)
    return result


def parse_benchmark(stdout, expected_order):
    rows = []
    for line in stdout.splitlines():
        match = BENCH.match(line.strip())
        if match:
            name, iterations, ns, bytes_op, allocs = match.groups()
            rows.append({"primitive": name, "iterations": int(iterations),
                         "ns_per_op": float(ns), "B_per_op": int(bytes_op),
                         "allocs_per_op": int(allocs)})
    require(len(rows) == 2 and {r["primitive"] for r in rows} == {"secp256k1", "mldsa44"},
            "missing or duplicate benchmark case")
    expected = ["secp256k1", "mldsa44"] if expected_order == "secp_first" else ["mldsa44", "secp256k1"]
    require([r["primitive"] for r in rows] == expected, "benchmark order mismatch")
    require(all(r["iterations"] > 0 and r["ns_per_op"] > 0 and
                r["B_per_op"] >= 0 and r["allocs_per_op"] >= 0 for r in rows),
            "invalid benchmark value")
    return {r["primitive"]: r for r in rows}


def percentile(values, q):
    ordered = sorted(values)
    position = (len(ordered) - 1) * q
    lo = int(position)
    hi = min(lo + 1, len(ordered) - 1)
    return ordered[lo] + (ordered[hi] - ordered[lo]) * (position - lo)


def paired_ci(deltas):
    require(len(deltas) == REPETITIONS, "insufficient independent process runs")
    rng = random.Random(SEED)
    n = len(deltas)
    draws = [statistics.mean(deltas[rng.randrange(n)] for _ in range(n))
             for _ in range(20000)]
    return [percentile(draws, 0.025), percentile(draws, 0.975)]


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve(), "session must be directly under ignored File 02 artifacts")
    require(not git("status", "--porcelain"), "source tree must be clean before lock")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT,
                           check=False).returncode == 0, "session is not Git-ignored")
    session.mkdir(mode=0o700)
    plan = {"schema": "pluto.file02.r01.plan.v1", "status": "LOCKED", "case_id": "R01",
            "source_commit": git("rev-parse", "HEAD"), "source_tree": git("rev-parse", "HEAD^{tree}"),
            "module_sha256": hashlib.sha256((ROOT / "go.mod").read_bytes()).hexdigest(),
            "safety_config_sha256": hashlib.sha256((ROOT / "research/config/s06_pilot.json").read_bytes()).hexdigest(),
            "created_utc": datetime.now(timezone.utc).isoformat(), "host": cpu_inventory(),
            "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
            "command": COMMAND, "environment": {"GOMAXPROCS": "1"},
            "master_seed": SEED, "orders": orders(), "independent_process_runs": REPETITIONS,
            "subprocess_timeout_seconds": 30, "benchtime": "200ms", "ci_method": "paired_percentile_bootstrap_20000_seed_20261113",
            "primary_units": ["ns/op", "B/op", "allocs/op"],
            "cycles_per_op": {"value": None, "status": "UNAVAILABLE", "reason": "no verified cycle counter"}}
    write_new(session / "plan.json", plan)
    print(session / "plan.json")


def validate_plan(plan):
    require(plan.get("schema") == "pluto.file02.r01.plan.v1" and plan.get("status") == "LOCKED"
            and plan.get("case_id") == "R01", "R01 plan schema/status")
    require(plan.get("source_commit") == git("rev-parse", "HEAD") and
            plan.get("source_tree") == git("rev-parse", "HEAD^{tree}"), "source revision mismatch")
    require(plan.get("module_sha256") == hashlib.sha256((ROOT / "go.mod").read_bytes()).hexdigest(), "module changed")
    require(plan.get("safety_config_sha256") == hashlib.sha256((ROOT / "research/config/s06_pilot.json").read_bytes()).hexdigest(), "safety config changed")
    require(plan.get("command") == COMMAND and plan.get("orders") == orders() and
            plan.get("independent_process_runs") == REPETITIONS and plan.get("master_seed") == SEED,
            "R01 preregistration changed")
    require(plan.get("host") == cpu_inventory(), "host inventory changed")
    require(not git("status", "--porcelain"), "source tree changed after plan lock")


def derive(session, plan):
    raw = session / "raw"
    common.verify_raw(raw)
    per_run = []
    failures = []
    for index, order in enumerate(plan["orders"], 1):
        path = raw / f"process-{index:02d}.json"
        if not path.exists():
            failures.append({"index": index, "reason": "process not executed after earlier failure"})
            per_run.append({"index": index, "order": order, "qc_status": "INVALID_FOR_ANALYSIS",
                            "reason": "process not executed after earlier failure", "secp_ns_per_op": None,
                            "ml_ns_per_op": None, "delta_ns_per_op": None, "ratio": None})
            continue
        item = common.read_json(path)
        reason = None
        if item.get("index") != index or item.get("order") != order or item.get("command") != COMMAND:
            reason = "provenance mismatch"
        elif item.get("returncode") != 0:
            reason = f"benchmark exit {item.get('returncode')}"
        else:
            try:
                values = parse_benchmark(item["stdout"], order)
            except (KeyError, ValueError) as exc:
                reason = str(exc)
        if reason:
            failures.append({"index": index, "reason": reason})
            per_run.append({"index": index, "order": order, "qc_status": "INVALID_FOR_ANALYSIS",
                            "reason": reason, "secp_ns_per_op": None, "ml_ns_per_op": None,
                            "delta_ns_per_op": None, "ratio": None})
            continue
        secp, ml = values["secp256k1"], values["mldsa44"]
        per_run.append({"index": index, "order": order, "qc_status": "PASS", "reason": None,
                        "secp_ns_per_op": secp["ns_per_op"], "ml_ns_per_op": ml["ns_per_op"],
                        "delta_ns_per_op": ml["ns_per_op"] - secp["ns_per_op"],
                        "ratio": ml["ns_per_op"] / secp["ns_per_op"],
                        "secp_B_per_op": secp["B_per_op"], "ml_B_per_op": ml["B_per_op"],
                        "secp_allocs_per_op": secp["allocs_per_op"], "ml_allocs_per_op": ml["allocs_per_op"],
                        "secp_iterations": secp["iterations"], "ml_iterations": ml["iterations"]})
    valid = [r for r in per_run if r["qc_status"] == "PASS"]
    qc = {"schema": "pluto.file02.r01.qc.v1", "case_id": "R01",
          "status": "PASS" if len(valid) == REPETITIONS else "INVALID_FOR_INFERENCE",
          "n_planned": REPETITIONS, "n_valid": len(valid), "failed": failures,
          "plan_sha256": hashlib.sha256((session / "plan.json").read_bytes()).hexdigest(),
          "raw_manifest_sha256": hashlib.sha256((raw / "manifest.json").read_bytes()).hexdigest()}
    aggregate = {"schema": "pluto.file02.r01.aggregate.v1", "case_id": "R01",
                 "status": "ANALYZED" if len(valid) == REPETITIONS else "INSUFFICIENT_RUNS",
                 "n_independent_process_runs": len(valid), "cycles_per_op": plan["cycles_per_op"],
                 "mean_secp_ns_per_op": None, "mean_ml_ns_per_op": None,
                 "mean_delta_ns_per_op": None, "delta_ci95_ns_per_op": None,
                 "mean_paired_ratio": None, "plan_sha256": qc["plan_sha256"],
                 "raw_manifest_sha256": qc["raw_manifest_sha256"]}
    if len(valid) == REPETITIONS:
        deltas = [r["delta_ns_per_op"] for r in valid]
        aggregate.update(mean_secp_ns_per_op=statistics.mean(r["secp_ns_per_op"] for r in valid),
                         mean_ml_ns_per_op=statistics.mean(r["ml_ns_per_op"] for r in valid),
                         mean_delta_ns_per_op=statistics.mean(deltas),
                         delta_ci95_ns_per_op=paired_ci(deltas),
                         mean_paired_ratio=statistics.mean(r["ratio"] for r in valid))
    return per_run, qc, aggregate


def execute(session):
    plan = common.read_json(session / "plan.json")
    validate_plan(plan)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    for index, order in enumerate(plan["orders"], 1):
        env = os.environ.copy()
        env.update(GOMAXPROCS="1", FILE02_R01_ORDER=order)
        started = time.monotonic_ns()
        try:
            completed = subprocess.run(COMMAND, cwd=ROOT, env=env, text=True,
                                       stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                       timeout=plan["subprocess_timeout_seconds"])
            returncode, output = completed.returncode, completed.stdout
        except subprocess.TimeoutExpired as exc:
            returncode = 124
            output = (exc.stdout or b"").decode(errors="replace") if isinstance(exc.stdout, bytes) else (exc.stdout or "")
            output += "\nTIMEOUT\n"
        write_new(raw / f"process-{index:02d}.json", {"index": index, "order": order,
                  "command": COMMAND, "returncode": returncode, "stdout": output,
                  "elapsed_ns": time.monotonic_ns() - started})
        if returncode != 0:
            break
    common.freeze_raw(raw)
    per_run, qc, aggregate = derive(session, plan)
    write_new(session / "result_per_run.json", {"schema": "pluto.file02.r01.per_run.v1", "rows": per_run})
    with (session / "result_per_run.csv").open("x", newline="", encoding="utf-8") as out:
        fields = ["index", "order", "qc_status", "reason", "secp_ns_per_op", "ml_ns_per_op",
                  "delta_ns_per_op", "ratio", "secp_B_per_op", "ml_B_per_op",
                  "secp_allocs_per_op", "ml_allocs_per_op", "secp_iterations", "ml_iterations"]
        writer = csv.DictWriter(out, fieldnames=fields)
        writer.writeheader()
        writer.writerows(per_run)
    write_new(session / "qc_report.json", qc)
    write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir()
    write_new(session / "figures/status.json", {"status": "NOT_GENERATED", "reason": "tabular R01 primary; no scientific plot required"})
    decision = ("# R01 decision\n\n" +
                f"QC: {qc['status']}; independent processes: {qc['n_valid']}/{REPETITIONS}.\n\n" +
                (f"ML minus secp mean: {aggregate['mean_delta_ns_per_op']:.3f} ns/op; "
                 f"paired-bootstrap 95% CI: {aggregate['delta_ci95_ns_per_op']}.\n\n"
                 if aggregate["status"] == "ANALYZED" else
                 "No inferential contrast: insufficient valid process runs.\n\n") +
                "This is primitive verification cost on one host, not full CheckTx, throughput or DoS. "
                "cycles/op is UNAVAILABLE without a verified cycle counter.\n")
    (session / "decision.md").write_text(decision)
    verify(session)
    print(json.dumps({"session": str(session), "qc": qc["status"], "aggregate": aggregate["status"],
                      "raw_manifest_sha256": qc["raw_manifest_sha256"]}, sort_keys=True))


def verify(session):
    plan = common.read_json(session / "plan.json")
    validate_plan(plan)
    per_run, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.r01.per_run.v1", "rows": per_run}, "per-run results changed")
    require(common.read_json(session / "qc_report.json") == qc, "QC report changed")
    require(common.read_json(session / "result_aggregate.json") == aggregate, "aggregate changed")
    require((session / "decision.md").is_file() and (session / "figures/status.json").is_file(), "missing output")
    print("R01_VERIFY=OK")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=("prepare", "run", "verify"))
    parser.add_argument("session", type=Path)
    args = parser.parse_args()
    session = args.session.resolve()
    if args.action == "prepare":
        prepare(session)
    elif args.action == "run":
        execute(session)
    else:
        verify(session)


if __name__ == "__main__":
    try:
        main()
    except (ValueError, OSError, subprocess.CalledProcessError) as exc:
        print(f"R01_ERROR: {exc}", file=sys.stderr)
        sys.exit(2)
