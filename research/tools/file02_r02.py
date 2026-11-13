#!/usr/bin/env python3
"""R02 direct CheckTx producer, QC and immutable raw verifier."""

import argparse
import csv
import hashlib
import json
import math
import os
from pathlib import Path
import random
import shutil
import statistics
import subprocess
import sys
import time
from datetime import datetime, timezone

import file02_common as common
import file02_common06
import file02_r01 as r01

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
PROFILES = ("C", "CP", "P", "H")
WIRE = {"C": 148, "CP": 2568, "P": 2504, "H": 2568}
STAGES = {"C": {"structural", "binding", "digest", "secp", "state", "state_copy"},
          "CP": {"structural", "binding", "digest", "secp", "state", "state_copy"},
          "P": {"structural", "binding", "digest", "ml", "state", "state_copy"},
          "H": {"structural", "binding", "digest", "secp", "ml", "state", "state_copy"}}
COMMAND = ["go", "run", "./cmd/research/r02"]
SEED = 20261113
RUNS = 12
SAMPLES = 240


def require(ok, message):
    if not ok:
        raise ValueError(message)


def orders():
    first = list(PROFILES)
    random.Random(SEED).shuffle(first)
    return [first[i % 4:] + first[:i % 4] for i in range(RUNS)]


def go_arch(host_arch):
    return {"x86_64": "amd64", "aarch64": "arm64"}.get(host_arch, host_arch)


def validate_plan(plan, strict_source=True):
    require(plan.get("schema") == "pluto.file02.r02.plan.v1" and plan.get("status") == "LOCKED"
            and plan.get("case_id") == "R02", "R02 plan schema/status")
    source = plan.get("source_commit")
    require(isinstance(source, str) and len(source) == 40 and
            plan.get("source_tree") == r01.git("rev-parse", source + "^{tree}"), "source identity")
    if strict_source:
        require(source == r01.git("rev-parse", "HEAD"), "source changed since lock")
    require(plan.get("host") == r01.cpu_inventory() and
            plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("safety_config_sha256") == common.digest(ROOT / "research/config/s06_pilot.json"),
            "host/module/safety changed")
    require(plan.get("command") == COMMAND and plan.get("orders") == orders() and
            plan.get("n_process_runs") == RUNS and plan.get("samples_per_profile") == SAMPLES and
            plan.get("chain_id") == "pluto-file02-r02" and
            plan.get("subprocess_timeout_seconds") == 55 and
            plan.get("ci_method") == "paired_percentile_bootstrap_20000_seed_20261113",
            "R02 preregistration changed")
    require(not r01.git("status", "--porcelain"), "source tree dirty")


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve(), "private session directory")
    require(not r01.git("status", "--porcelain"), "source tree dirty before lock")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT,
                           check=False).returncode == 0, "session not ignored")
    session.mkdir(mode=0o700)
    plan = {"schema": "pluto.file02.r02.plan.v1", "status": "LOCKED", "case_id": "R02",
            "source_commit": r01.git("rev-parse", "HEAD"), "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
            "host": r01.cpu_inventory(), "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
            "module_sha256": common.digest(ROOT / "go.mod"),
            "safety_config_sha256": common.digest(ROOT / "research/config/s06_pilot.json"),
            "created_utc": datetime.now(timezone.utc).isoformat(), "command": COMMAND,
            "environment": {"GOMAXPROCS": "1"}, "master_seed": SEED,
            "orders": orders(), "n_process_runs": RUNS, "samples_per_profile": SAMPLES,
            "chain_id": "pluto-file02-r02", "wire_bytes": WIRE,
            "subprocess_timeout_seconds": 55,
            "ci_method": "paired_percentile_bootstrap_20000_seed_20261113",
            "p99_precision": {"rank_band": [0.985, 0.995], "coverage_floor": 0.95,
                              "cluster_draws": 2000, "relative_ci_radius_max": 0.20}}
    r01.write_new(session / "plan.json", plan)
    print(session / "plan.json")


def quantile(values, q):
    require(values, "empty quantile input")
    return r01.percentile(values, q)


def quantile99(values):
    return sorted(values)[math.ceil(0.99 * len(values)) - 1]


def p99_cluster(groups):
    require(len(groups) == RUNS and all(len(g) == SAMPLES for g in groups), "p99 sample clusters")
    point = quantile99([x for g in groups for x in g])
    rng = random.Random(SEED)
    estimates = []
    for _ in range(2000):
        draw = []
        for _ in range(RUNS):
            group = groups[rng.randrange(RUNS)]
            draw.extend(group[rng.randrange(SAMPLES)] for _ in range(SAMPLES))
        estimates.append(quantile99(draw))
    ci = [quantile(estimates, 0.025), quantile(estimates, 0.975)]
    radius = max(point - ci[0], ci[1] - point) / point
    coverage = file02_common06.rank_coverage(RUNS * SAMPLES)
    passed = coverage >= 0.95 and radius <= 0.20
    return {"status": "PRECISION_PASS" if passed else "INSUFFICIENT_PRECISION",
            "n_process_runs": RUNS, "n_samples": RUNS * SAMPLES,
            "p99_service_ns": point if passed else None,
            "ci95_ns": ci if passed else None,
            "relative_ci_radius": radius, "rank_coverage_iid_optimistic": coverage}


def parse_process(raw, expected_order):
    require(raw.get("schema") == "pluto.file02.r02.raw.v1" and raw.get("order") == expected_order,
            "raw schema/order")
    profiles = raw.get("profiles")
    require(isinstance(profiles, list) and len(profiles) == 4 and
            [x.get("profile") for x in profiles] == expected_order, "raw profiles")
    genesis = {x.get("genesis_sha256") for x in profiles}
    require(len(genesis) == 1 and all(common.sha(x) for x in genesis), "genesis mismatch")
    rows = []
    groups = {}
    for item in profiles:
        profile = item["profile"]
        require(item.get("code") == {"C": 1, "CP": 2, "P": 3, "H": 4}[profile],
                "profile code mismatch")
        samples = item.get("samples")
        require(item.get("expected_count") == SAMPLES and isinstance(samples, list) and
                len(samples) == SAMPLES and item.get("recorder_drops") == 0, "count/drop QC")
        values = []
        stage_values = {stage: [] for stage in sorted(STAGES[profile])}
        for index, sample in enumerate(samples, 1):
            require(sample.get("slot") == index and sample.get("wire_bytes") == WIRE[profile] and
                    isinstance(sample.get("service_ns"), int) and sample["service_ns"] > 0 and
                    isinstance(sample.get("app_lock_wait_ns"), int) and sample["app_lock_wait_ns"] >= 0,
                    "sample identity/duration/size")
            spans = sample.get("spans")
            require(isinstance(spans, list) and {x.get("stage") for x in spans} == STAGES[profile] and
                    len(spans) == len(STAGES[profile]) and
                    all(isinstance(x.get("duration_ns"), int) and x["duration_ns"] > 0 for x in spans),
                    "stage span QC")
            values.append(sample["service_ns"])
            for span in spans:
                stage_values[span["stage"]].append(span["duration_ns"])
        cpu, alloc, mallocs = (item.get(k) for k in ("process_cpu_ns", "total_alloc_bytes", "total_mallocs"))
        require(all(isinstance(x, int) and x >= 0 for x in (cpu, alloc, mallocs)) and
                item.get("elapsed_ns", 0) > 0, "CPU/allocation QC")
        mean_stage = {stage: statistics.mean(v) for stage, v in stage_values.items()}
        residual = statistics.mean(values) - sum(mean_stage.values())
        require(residual >= -0.05 * statistics.mean(values), "overlapping stage clocks")
        rows.append({"profile": profile, "n": SAMPLES, "mean_service_ns": statistics.mean(values),
                     "p50_service_ns": quantile(values, 0.5), "p95_service_ns": quantile(values, 0.95),
                     "p99_service_ns_descriptive": quantile99(values),
                     "mean_stage_ns": mean_stage, "mean_residual_ns": residual,
                     "process_cpu_ns_per_tx": cpu / SAMPLES, "alloc_B_per_tx": alloc / SAMPLES,
                     "allocs_per_tx": mallocs / SAMPLES, "wire_bytes": WIRE[profile]})
        groups[profile] = values
    return rows, groups


def derive(session, plan):
    rawdir = session / "raw"
    common.verify_raw(rawdir)
    rows, failures = [], []
    groups = {p: [] for p in PROFILES}
    genesis_seen = set()
    for index, order in enumerate(plan["orders"], 1):
        path = rawdir / f"process-{index:02d}.json"
        if not path.exists():
            failures.append({"index": index, "reason": "not executed after earlier failure"})
            continue
        envelope = common.read_json(path)
        try:
            require(envelope.get("index") == index and envelope.get("order") == order and
                    envelope.get("command") == COMMAND and envelope.get("returncode") == 0,
                    "process provenance/exit")
            data = json.loads(envelope["stdout"])
            require(data.get("go_version") == plan["go_version"].split()[2] and
                    data.get("goos") == "linux" and data.get("goarch") == go_arch(plan["host"]["architecture"]),
                    "toolchain/platform mismatch")
            parsed, samples = parse_process(data, order)
            genesis = data["profiles"][0]["genesis_sha256"]
            require(genesis not in genesis_seen, "fresh genesis reused")
            genesis_seen.add(genesis)
            for item in parsed:
                item.update(run_index=index, qc_status="PASS", genesis_sha256=genesis)
                rows.append(item)
            for profile in PROFILES:
                groups[profile].append(samples[profile])
        except (ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
            failures.append({"index": index, "reason": str(exc)})
    valid_runs = RUNS - len(failures)
    qc = {"schema": "pluto.file02.r02.qc.v1", "case_id": "R02",
          "status": "PASS" if valid_runs == RUNS else "INVALID_FOR_INFERENCE",
          "n_planned": RUNS, "n_valid": valid_runs, "failed": failures,
          "plan_sha256": common.digest(session / "plan.json"),
          "raw_manifest_sha256": common.digest(rawdir / "manifest.json")}
    agg = {"schema": "pluto.file02.r02.aggregate.v1", "case_id": "R02",
           "status": "ANALYZED" if valid_runs == RUNS else "INSUFFICIENT_RUNS",
           "n_process_runs": valid_runs, "profiles": {}, "H_minus_CP_mean_service_ns": None,
           "H_minus_CP_ci95_ns": None, "plan_sha256": qc["plan_sha256"],
           "raw_manifest_sha256": qc["raw_manifest_sha256"]}
    if valid_runs == RUNS:
        by_profile = {p: sorted((r for r in rows if r["profile"] == p), key=lambda r: r["run_index"]) for p in PROFILES}
        for profile in PROFILES:
            subset = by_profile[profile]
            agg["profiles"][profile] = {
                "mean_service_ns": statistics.mean(r["mean_service_ns"] for r in subset),
                "mean_cpu_ns_per_tx": statistics.mean(r["process_cpu_ns_per_tx"] for r in subset),
                "mean_alloc_B_per_tx": statistics.mean(r["alloc_B_per_tx"] for r in subset),
                "mean_allocs_per_tx": statistics.mean(r["allocs_per_tx"] for r in subset),
                "mean_stage_ns": {stage: statistics.mean(r["mean_stage_ns"][stage] for r in subset)
                                  for stage in sorted(STAGES[profile])},
                "p99_precision": p99_cluster(groups[profile])}
        deltas = [h["mean_service_ns"] - cp["mean_service_ns"]
                  for h, cp in zip(by_profile["H"], by_profile["CP"])]
        agg["H_minus_CP_mean_service_ns"] = statistics.mean(deltas)
        agg["H_minus_CP_ci95_ns"] = r01.paired_ci(deltas)
    return rows, qc, agg


def write_outputs(session, plan):
    rows, qc, agg = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.r02.per_run.v1", "rows": rows})
    with (session / "result_per_run.csv").open("x", newline="", encoding="utf-8") as out:
        fields = ["run_index", "profile", "qc_status", "n", "mean_service_ns", "p50_service_ns",
                  "p95_service_ns", "p99_service_ns_descriptive", "mean_residual_ns",
                  "process_cpu_ns_per_tx", "alloc_B_per_tx", "allocs_per_tx", "wire_bytes", "genesis_sha256"]
        writer = csv.DictWriter(out, fieldnames=fields, extrasaction="ignore")
        writer.writeheader()
        writer.writerows(rows)
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", agg)
    (session / "figures").mkdir()
    r01.write_new(session / "figures/status.json", {"status": "NOT_GENERATED", "reason": "R02 tabular stage analysis"})
    decision = ("# R02 decision\n\n" + f"QC: {qc['status']}; independent process runs: {qc['n_valid']}/{RUNS}.\n\n" +
                (f"Direct CheckTx H minus CP mean: {agg['H_minus_CP_mean_service_ns']:.3f} ns; "
                 f"paired-bootstrap 95% CI: {agg['H_minus_CP_ci95_ns']}.\n\n"
                 if agg["status"] == "ANALYZED" else "No inferential contrast: insufficient runs.\n\n") +
                "Direct ABCI, recorder ON, one host; not the RQ1 live-network endpoint. "
                "Stage spans are wall time; process CPU and allocation deltas include instrumentation.\n")
    (session / "decision.md").write_text(decision)
    r01.write_new(session / "analysis_provenance.json", {"schema": "pluto.file02.analysis_provenance.v1",
                  "measurement_source_commit": plan["source_commit"], "analyzer_commit": r01.git("rev-parse", "HEAD"),
                  "previous_analysis": [p.name for p in sorted(session.glob("analysis-v*"))]})
    verify(session)
    print(json.dumps({"session": str(session), "qc": qc["status"], "aggregate": agg["status"],
                      "raw_manifest_sha256": qc["raw_manifest_sha256"]}, sort_keys=True))


def execute(session):
    plan = common.read_json(session / "plan.json")
    validate_plan(plan)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    for index, order in enumerate(plan["orders"], 1):
        env = os.environ.copy()
        env.update(GOMAXPROCS="1", FILE02_R02_ORDER=",".join(order))
        started = time.monotonic_ns()
        try:
            result = subprocess.run(COMMAND, cwd=ROOT, env=env, text=True, stdout=subprocess.PIPE,
                                    stderr=subprocess.PIPE, timeout=55)
            code, stdout, stderr = result.returncode, result.stdout, result.stderr
        except subprocess.TimeoutExpired as exc:
            code = 124
            stdout = exc.stdout.decode(errors="replace") if isinstance(exc.stdout, bytes) else (exc.stdout or "")
            stderr = "TIMEOUT"
        r01.write_new(raw / f"process-{index:02d}.json", {"index": index, "order": order,
                      "command": COMMAND, "returncode": code, "stdout": stdout, "stderr": stderr,
                      "elapsed_ns": time.monotonic_ns() - started})
        if code != 0:
            break
    common.freeze_raw(raw)
    write_outputs(session, plan)


def reanalyze(session):
    plan = common.read_json(session / "plan.json")
    validate_plan(plan, strict_source=False)
    common.verify_raw(session / "raw")
    previous = sorted(session.glob("analysis-v*"))
    archive = session / f"analysis-v{len(previous) + 1}"
    archive.mkdir(mode=0o700)
    for name in ("result_per_run.json", "result_per_run.csv", "qc_report.json",
                 "result_aggregate.json", "decision.md", "analysis_provenance.json", "figures"):
        path = session / name
        if path.exists():
            shutil.move(str(path), str(archive / name))
    common.freeze_raw(archive)
    write_outputs(session, plan)


def verify(session):
    plan = common.read_json(session / "plan.json")
    validate_plan(plan, strict_source=False)
    rows, qc, agg = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.r02.per_run.v1", "rows": rows}, "per-run result changed")
    require(common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == agg, "QC/aggregate changed")
    require((session / "result_per_run.csv").is_file() and (session / "decision.md").is_file() and
            (session / "figures/status.json").is_file(), "missing result artifact")
    provenance = common.read_json(session / "analysis_provenance.json")
    require(provenance.get("measurement_source_commit") == plan["source_commit"] and
            r01.git("cat-file", "-t", provenance.get("analyzer_commit", "")) == "commit",
            "analysis revision")
    for archive in session.glob("analysis-v*"):
        common.verify_raw(archive)
    print("R02_VERIFY=OK")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=("prepare", "run", "reanalyze", "verify"))
    parser.add_argument("session", type=Path)
    args = parser.parse_args()
    session = args.session.resolve()
    if args.action == "prepare":
        prepare(session)
    elif args.action == "run":
        execute(session)
    elif args.action == "reanalyze":
        reanalyze(session)
    else:
        verify(session)


if __name__ == "__main__":
    try:
        main()
    except (ValueError, OSError, subprocess.CalledProcessError) as exc:
        print(f"R02_ERROR: {exc}", file=sys.stderr)
        sys.exit(2)
