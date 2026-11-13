#!/usr/bin/env python3
"""R04 CP/P/H direct validation contrasts with explicit size confounding."""

import argparse
import csv
import json
import os
from pathlib import Path
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
PAIRS = (("CP", "P"), ("P", "H"), ("CP", "H"))
CASE = "R04"


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def plan_check(plan, strict=True):
    require(plan.get("schema") == "pluto.file02.r04.plan.v1" and
            plan.get("status") == "LOCKED" and plan.get("case_id") == CASE,
            "R04 plan schema/status")
    source = plan.get("source_commit")
    require(isinstance(source, str) and len(source) == 40 and
            plan.get("source_tree") == r01.git("rev-parse", source + "^{tree}"),
            "source/tree mismatch")
    if strict:
        require(source == r01.git("rev-parse", "HEAD"), "source changed after lock")
    require(plan.get("host") == r01.cpu_inventory() and
            plan.get("go_version") == subprocess.check_output(["go", "version"], text=True).strip() and
            plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("safety_config_sha256") == common.digest(ROOT / "research/config/s06_pilot.json"),
            "host/toolchain/module/safety mismatch")
    require(plan.get("command") == r02.COMMAND and plan.get("orders") == r02.orders() and
            plan.get("pairs") == [list(pair) for pair in PAIRS] and
            plan.get("n_process_runs") == r02.RUNS and
            plan.get("samples_per_profile") == r02.SAMPLES and
            plan.get("chain_id") == "pluto-file02-r02" and
            plan.get("subprocess_timeout_seconds") == 55 and
            plan.get("master_seed") == r02.SEED and
            plan.get("ci_method") == "paired_percentile_bootstrap_20000_seed_20261113" and
            plan.get("P_byte_match_status") == "UNSUPPORTED_CANONICAL_CODEC",
            "R04 preregistration changed")
    require(not r01.git("status", "--porcelain"), "source tree dirty")


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve(), "private session directory")
    require(not r01.git("status", "--porcelain"), "source dirty before lock")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT,
                           check=False).returncode == 0, "session not ignored")
    session.mkdir(mode=0o700)
    r01.write_new(session / "plan.json", {
        "schema": "pluto.file02.r04.plan.v1", "status": "LOCKED", "case_id": CASE,
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
        "pairs": [list(pair) for pair in PAIRS], "chain_id": "pluto-file02-r02",
        "wire_bytes": r02.WIRE, "P_byte_match_status": "UNSUPPORTED_CANONICAL_CODEC",
        "subprocess_timeout_seconds": 55,
        "ci_method": "paired_percentile_bootstrap_20000_seed_20261113"})
    print(session / "plan.json")


def derive(session, plan):
    rawdir = session / "raw"
    common.verify_raw(rawdir)
    rows, failures, geneses = [], [], set()
    for index, order in enumerate(plan["orders"], 1):
        path = rawdir / f"process-{index:02d}.json"
        if not path.exists():
            failures.append({"index": index, "reason": "not executed after earlier failure"})
            continue
        item = common.read_json(path)
        try:
            require(item.get("index") == index and item.get("order") == order and
                    item.get("command") == r02.COMMAND and item.get("returncode") == 0,
                    "process provenance/exit")
            data = json.loads(item["stdout"])
            require(data.get("go_version") == plan["go_version"].split()[2] and
                    data.get("goos") == "linux" and
                    data.get("goarch") == r02.go_arch(plan["host"]["architecture"]),
                    "toolchain/platform mismatch")
            measured, _ = r02.parse_process(data, order)
            genesis = data["profiles"][0]["genesis_sha256"]
            require(genesis not in geneses, "genesis reused")
            geneses.add(genesis)
            by = {p["profile"]: p for p in measured}
            row = {"run_index": index, "genesis_sha256": genesis, "qc_status": "PASS"}
            for name in ("CP", "P", "H"):
                p = by[name]
                row[f"{name}_wire_bytes"] = p["wire_bytes"]
                row[f"{name}_service_ns"] = p["mean_service_ns"]
                row[f"{name}_cpu_ns_per_tx"] = p["process_cpu_ns_per_tx"]
                row[f"{name}_alloc_B_per_tx"] = p["alloc_B_per_tx"]
                row[f"{name}_allocs_per_tx"] = p["allocs_per_tx"]
                row[f"{name}_secp_span_ns"] = p["mean_stage_ns"].get("secp")
                row[f"{name}_ml_span_ns"] = p["mean_stage_ns"].get("ml")
            for left, right in PAIRS:
                row[f"{right}_minus_{left}_service_ns"] = row[f"{right}_service_ns"] - row[f"{left}_service_ns"]
            rows.append(row)
        except (ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
            failures.append({"index": index, "reason": str(exc)})
    qc = {"schema": "pluto.file02.r04.qc.v1", "case_id": CASE,
          "status": "PASS" if len(rows) == r02.RUNS else "INVALID_FOR_INFERENCE",
          "n_planned": r02.RUNS, "n_valid": len(rows), "failed": failures,
          "plan_sha256": common.digest(session / "plan.json"),
          "raw_manifest_sha256": common.digest(rawdir / "manifest.json")}
    aggregate = {"schema": "pluto.file02.r04.aggregate.v1", "case_id": CASE,
                 "status": "ANALYZED_CONFOUNDED_P_CONTRASTS" if len(rows) == r02.RUNS else "INSUFFICIENT_RUNS",
                 "n_process_runs": len(rows), "P_byte_match_status": plan["P_byte_match_status"],
                 "contrasts": {}, "mean_profile_stage_ns": {},
                 "plan_sha256": qc["plan_sha256"],
                 "raw_manifest_sha256": qc["raw_manifest_sha256"]}
    if len(rows) == r02.RUNS:
        for name in ("CP", "P", "H"):
            aggregate["mean_profile_stage_ns"][name] = {
                "secp": statistics.mean(r[f"{name}_secp_span_ns"] for r in rows) if name != "P" else None,
                "ml": statistics.mean(r[f"{name}_ml_span_ns"] for r in rows) if name != "CP" else None}
        for left, right in PAIRS:
            label = f"{right}_minus_{left}"
            values = [r[f"{label}_service_ns"] for r in rows]
            aggregate["contrasts"][label] = {
                "mean_service_ns": statistics.mean(values),
                "ci95_ns": r01.paired_ci(values),
                "wire_delta_bytes": r02.WIRE[right] - r02.WIRE[left],
                "exact_byte_matched": r02.WIRE[right] == r02.WIRE[left]}
    return rows, qc, aggregate


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.r04.per_run.v1", "rows": rows})
    with (session / "result_per_run.csv").open("x", newline="", encoding="utf-8") as out:
        fields = ["run_index", "genesis_sha256", "qc_status"]
        for name in ("CP", "P", "H"):
            fields += [f"{name}_{suffix}" for suffix in ("wire_bytes", "service_ns", "cpu_ns_per_tx",
                       "alloc_B_per_tx", "allocs_per_tx", "secp_span_ns", "ml_span_ns")]
        fields += [f"{right}_minus_{left}_service_ns" for left, right in PAIRS]
        writer = csv.DictWriter(out, fieldnames=fields)
        writer.writeheader()
        writer.writerows(rows)
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir()
    r01.write_new(session / "figures/status.json", {"status": "NOT_GENERATED", "reason": "tabular R04 mechanism contrasts"})
    (session / "decision.md").write_text(
        "# R04 decision\n\n" + f"QC: {qc['status']}; process runs: {qc['n_valid']}/{r02.RUNS}.\n\n" +
        ("Contrasts and CIs: " + json.dumps(aggregate["contrasts"], sort_keys=True) + "\n\n"
         if aggregate["contrasts"] else "No inferential contrast.\n\n") +
        "P is 64 bytes shorter than CP/H by canonical codec, so P−CP and H−P are confounded. "
        "CP−H is exact-size. Stage spans are wall time, not stage CPU.\n")
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
            {"schema": "pluto.file02.r04.per_run.v1", "rows": rows}, "per-run result changed")
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
    print("R04_VERIFY=OK")


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
        print(f"R04_ERROR: {exc}", file=sys.stderr)
        sys.exit(2)
