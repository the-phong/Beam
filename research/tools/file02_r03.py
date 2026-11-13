#!/usr/bin/env python3
"""R03 C/CP size-control producer using the verified direct CheckTx harness."""

import argparse
import csv
import json
import os
from pathlib import Path
import shutil
import statistics
import subprocess
import sys
import time
from datetime import datetime, timezone

import file02_common as common
import file02_r01 as r01
import file02_r02 as r02

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
CASE = "R03"
METRICS = {"wire_bytes": "wire_bytes", "parse_ns": "mean_stage_ns.structural",
           "checktx_ns": "mean_service_ns", "process_cpu_ns_per_tx": "process_cpu_ns_per_tx",
           "alloc_B_per_tx": "alloc_B_per_tx", "allocs_per_tx": "allocs_per_tx"}


def require(ok, message):
    if not ok:
        raise ValueError(message)


def field(row, name):
    value = row
    for part in name.split("."):
        value = value[part]
    return value


def plan_check(plan, strict=True):
    require(plan.get("schema") == "pluto.file02.r03.plan.v1" and
            plan.get("status") == "LOCKED" and plan.get("case_id") == CASE,
            "R03 locked plan schema")
    source = plan.get("source_commit")
    require(isinstance(source, str) and len(source) == 40 and
            plan.get("source_tree") == r01.git("rev-parse", source + "^{tree}"),
            "source/tree provenance")
    if strict:
        require(source == r01.git("rev-parse", "HEAD"), "source changed after lock")
    require(plan.get("host") == r01.cpu_inventory() and
            plan.get("go_version") == subprocess.check_output(["go", "version"], text=True).strip(),
            "host/toolchain changed")
    require(plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("safety_config_sha256") == common.digest(ROOT / "research/config/s06_pilot.json"),
            "module/safety changed")
    require(plan.get("command") == r02.COMMAND and plan.get("orders") == r02.orders() and
            plan.get("n_process_runs") == r02.RUNS and
            plan.get("samples_per_profile") == r02.SAMPLES and
            plan.get("profiles_compared") == ["C", "CP"] and
            plan.get("master_seed") == r02.SEED and
            plan.get("subprocess_timeout_seconds") == 55 and
            plan.get("ci_method") == "paired_percentile_bootstrap_20000_seed_20261113",
            "R03 preregistration changed")
    require(plan.get("relay_propagation_latency") == {"value": None, "status": "UNAVAILABLE",
            "reason": "no matched P2P peer send/receive timestamps"},
            "relay availability declaration changed")
    require(not r01.git("status", "--porcelain"), "source tree dirty")


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve(), "private session location")
    require(not r01.git("status", "--porcelain"), "dirty source before lock")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT,
                           check=False).returncode == 0, "session not Git-ignored")
    session.mkdir(mode=0o700)
    r01.write_new(session / "plan.json", {
        "schema": "pluto.file02.r03.plan.v1", "status": "LOCKED", "case_id": CASE,
        "source_commit": r01.git("rev-parse", "HEAD"),
        "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
        "host": r01.cpu_inventory(),
        "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
        "module_sha256": common.digest(ROOT / "go.mod"),
        "safety_config_sha256": common.digest(ROOT / "research/config/s06_pilot.json"),
        "created_utc": datetime.now(timezone.utc).isoformat(),
        "command": r02.COMMAND, "environment": {"GOMAXPROCS": "1"},
        "orders": r02.orders(), "master_seed": r02.SEED,
        "n_process_runs": r02.RUNS, "samples_per_profile": r02.SAMPLES,
        "profiles_compared": ["C", "CP"], "chain_id": "pluto-file02-r02",
        "subprocess_timeout_seconds": 55,
        "ci_method": "paired_percentile_bootstrap_20000_seed_20261113",
        "relay_propagation_latency": {"value": None, "status": "UNAVAILABLE",
                                      "reason": "no matched P2P peer send/receive timestamps"}})
    print(session / "plan.json")


def derive(session, plan):
    rawdir = session / "raw"
    common.verify_raw(rawdir)
    rows, failures, genesis_seen = [], [], set()
    for index, order in enumerate(plan["orders"], 1):
        path = rawdir / f"process-{index:02d}.json"
        if not path.exists():
            failures.append({"index": index, "reason": "not executed after earlier failure"})
            continue
        envelope = common.read_json(path)
        try:
            require(envelope.get("index") == index and envelope.get("order") == order and
                    envelope.get("command") == r02.COMMAND and envelope.get("returncode") == 0,
                    "process provenance/exit")
            data = json.loads(envelope["stdout"])
            require(data.get("go_version") == plan["go_version"].split()[2] and
                    data.get("goos") == "linux" and
                    data.get("goarch") == r02.go_arch(plan["host"]["architecture"]),
                    "toolchain/platform mismatch")
            measured, _ = r02.parse_process(data, order)
            genesis = data["profiles"][0]["genesis_sha256"]
            require(genesis not in genesis_seen, "fresh genesis reused")
            genesis_seen.add(genesis)
            by_profile = {item["profile"]: item for item in measured}
            row = {"run_index": index, "genesis_sha256": genesis, "qc_status": "PASS"}
            for metric, path_name in METRICS.items():
                c = field(by_profile["C"], path_name)
                cp = field(by_profile["CP"], path_name)
                row[f"C_{metric}"] = c
                row[f"CP_{metric}"] = cp
                row[f"delta_{metric}"] = cp - c
            rows.append(row)
        except (ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
            failures.append({"index": index, "reason": str(exc)})
    qc = {"schema": "pluto.file02.r03.qc.v1", "case_id": CASE,
          "status": "PASS" if len(rows) == r02.RUNS else "INVALID_FOR_INFERENCE",
          "n_planned": r02.RUNS, "n_valid": len(rows), "failed": failures,
          "plan_sha256": common.digest(session / "plan.json"),
          "raw_manifest_sha256": common.digest(rawdir / "manifest.json")}
    aggregate = {"schema": "pluto.file02.r03.aggregate.v1", "case_id": CASE,
                 "status": "PARTIAL_RELAY_UNAVAILABLE" if len(rows) == r02.RUNS else "INSUFFICIENT_RUNS",
                 "n_independent_process_runs": len(rows),
                 "relay_propagation_latency": plan["relay_propagation_latency"],
                 "deltas": {}, "plan_sha256": qc["plan_sha256"],
                 "raw_manifest_sha256": qc["raw_manifest_sha256"]}
    if len(rows) == r02.RUNS:
        for metric in METRICS:
            values = [row[f"delta_{metric}"] for row in rows]
            aggregate["deltas"][metric] = {"mean": statistics.mean(values),
                "min": min(values), "max": max(values),
                "ci95": r01.paired_ci(values) if metric in
                ("parse_ns", "checktx_ns", "process_cpu_ns_per_tx") else None}
    return rows, qc, aggregate


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.r03.per_run.v1", "rows": rows})
    with (session / "result_per_run.csv").open("x", newline="", encoding="utf-8") as out:
        fields = ["run_index", "genesis_sha256", "qc_status"]
        for metric in METRICS:
            fields.extend([f"C_{metric}", f"CP_{metric}", f"delta_{metric}"])
        writer = csv.DictWriter(out, fieldnames=fields)
        writer.writeheader()
        writer.writerows(rows)
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir()
    r01.write_new(session / "figures/status.json", {"status": "NOT_GENERATED", "reason": "tabular partial R03"})
    (session / "decision.md").write_text(
        "# R03 decision\n\n" + f"QC: {qc['status']}; independent runs: {qc['n_valid']}/{r02.RUNS}.\n\n" +
        (f"CP-C CheckTx mean ns: {aggregate['deltas']['checktx_ns']['mean']:.3f}; "
         f"95% paired CI: {aggregate['deltas']['checktx_ns']['ci95']}.\n\n"
         if aggregate["deltas"] else "No inferential effect.\n\n") +
        "Relay propagation latency is UNAVAILABLE. Direct application size/parse contrast only; "
        "the full R03 checklist output is PARTIAL, not PASS.\n")
    r01.write_new(session / "analysis_provenance.json", {"schema": "pluto.file02.analysis_provenance.v1",
                  "measurement_source_commit": plan["source_commit"],
                  "analyzer_commit": r01.git("rev-parse", "HEAD")})
    verify(session)
    print(json.dumps({"session": str(session), "qc": qc["status"], "aggregate": aggregate["status"],
                      "raw_manifest_sha256": qc["raw_manifest_sha256"]}, sort_keys=True))


def execute(session):
    plan = common.read_json(session / "plan.json")
    plan_check(plan)
    rawdir = session / "raw"
    rawdir.mkdir(mode=0o700)
    for index, order in enumerate(plan["orders"], 1):
        env = os.environ.copy()
        env.update(GOMAXPROCS="1", FILE02_R02_ORDER=",".join(order))
        started = time.monotonic_ns()
        try:
            run = subprocess.run(r02.COMMAND, cwd=ROOT, env=env, text=True,
                                 stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=55)
            code, stdout, stderr = run.returncode, run.stdout, run.stderr
        except subprocess.TimeoutExpired as exc:
            code = 124
            stdout = exc.stdout.decode(errors="replace") if isinstance(exc.stdout, bytes) else (exc.stdout or "")
            stderr = "TIMEOUT"
        r01.write_new(rawdir / f"process-{index:02d}.json", {"index": index,
                      "order": order, "command": r02.COMMAND, "returncode": code,
                      "stdout": stdout, "stderr": stderr,
                      "elapsed_ns": time.monotonic_ns() - started})
        if code != 0:
            break
    common.freeze_raw(rawdir)
    write_outputs(session, plan)


def verify(session):
    plan = common.read_json(session / "plan.json")
    plan_check(plan, strict=False)
    rows, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.r03.per_run.v1", "rows": rows}, "per-run result changed")
    require(common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate,
            "QC/aggregate changed")
    require((session / "result_per_run.csv").is_file() and
            (session / "figures/status.json").is_file() and
            (session / "decision.md").is_file(), "missing output artifact")
    provenance = common.read_json(session / "analysis_provenance.json")
    require(provenance.get("measurement_source_commit") == plan["source_commit"] and
            r01.git("cat-file", "-t", provenance.get("analyzer_commit", "")) == "commit",
            "analysis provenance")
    print("R03_VERIFY=OK")


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
        print(f"R03_ERROR: {exc}", file=sys.stderr)
        sys.exit(2)
