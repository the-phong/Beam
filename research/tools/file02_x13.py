#!/usr/bin/env python3
"""Locked X13 20/23/25 tx/s invalid-background sensitivity and verifier."""

import argparse
import csv
from datetime import datetime, timezone
import hashlib
import io
import json
import os
from pathlib import Path
import platform
import random
import signal
import statistics
import subprocess
import sys
import time
from uuid import uuid4

import file02_common as common
import file02_r01 as r01
import file02_x10 as x10

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
SEED = 20261115
RATES = (20, 23, 25)
WIRE = 2568


def require(ok, why):
    if not ok:
        raise ValueError(why)


def schedule():
    orders = [list(order) for order in [
        (20, 23, 25), (20, 25, 23), (23, 20, 25),
        (23, 25, 20), (25, 20, 23), (25, 23, 20)] * 2]
    random.Random(SEED).shuffle(orders)
    jobs = [{"block": block, "order": order, "rate": rate,
             "run_id": f"X13-b{block:02d}-r{rate}"}
            for block, rates in enumerate(orders) for order, rate in enumerate(rates)]
    return {"schema": "pluto.file02.x13_schedule.v1", "seed": SEED,
            "orders": orders, "jobs": jobs}


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve() and not r01.git("status", "--porcelain"),
            "X13 source must be clean")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT).returncode == 0,
            "X13 session must be ignored")
    session.mkdir(mode=0o700)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    common.write_once(raw / "schedule.json", json.dumps(schedule(), indent=2, sort_keys=True) + "\n")
    common.write_once(raw / "config.json", (ROOT / "research/config/s06_pilot.json").read_text())
    plan = {"schema": "pluto.file02.x13_plan.v1", "status": "LOCKED", "case_id": "X13",
        "source_commit": r01.git("rev-parse", "HEAD"), "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
        "created_utc": datetime.now(timezone.utc).isoformat(), "host": r01.cpu_inventory(),
        "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
        "module_sha256": common.digest(ROOT / "go.mod"),
        "config_sha256": common.digest(raw / "config.json"),
        "schedule_sha256": common.digest(raw / "schedule.json"),
        "protocol": "research/protocol/FILE02_X13_SAFE_RANGE_LOCKED_PLAN.md",
        "n_blocks": 12, "n_networks": 36, "rates": list(RATES),
        "fixed_valid_tx_s": 6, "background_by_rate": {"20": 14, "23": 17, "25": 19},
        "wire_bytes": WIRE, "warmup_seconds": 1, "steady_seconds": 12,
        "drain_seconds": 5, "cooldown_seconds": 2, "deadline_ms": 3000,
        "ci_seed": SEED, "ci_resamples": 20000,
        "topology": "four_validator_single_physical_host_loopback_RPC_validator0",
        "safety_caps": [25, 65536, 16, 60], "exclusion_policy": "none_all_36_required"}
    r01.write_new(session / "plan.json", plan)
    return plan


def validate_plan(session, plan):
    require(plan.get("schema") == "pluto.file02.x13_plan.v1" and plan.get("status") == "LOCKED" and
            plan.get("case_id") == "X13" and plan.get("source_tree") ==
            r01.git("rev-parse", plan["source_commit"] + "^{tree}"), "X13 source/status")
    require(plan.get("host") == r01.cpu_inventory() and plan.get("go_version") ==
            subprocess.check_output(["go", "version"], text=True).strip() and
            plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("config_sha256") == common.digest(session / "raw/config.json") ==
            common.digest(ROOT / "research/config/s06_pilot.json") and
            plan.get("schedule_sha256") == common.digest(session / "raw/schedule.json") and
            common.read_json(session / "raw/schedule.json") == schedule(), "X13 plan/config/schedule drift")
    require(plan.get("n_blocks") == 12 and plan.get("n_networks") == 36 and
            plan.get("rates") == list(RATES) and plan.get("fixed_valid_tx_s") == 6 and
            plan.get("background_by_rate") == {"20": 14, "23": 17, "25": 19} and
            plan.get("safety_caps") == [25, 65536, 16, 60] and
            plan.get("ci_resamples") == 20000 and
            plan.get("exclusion_policy") == "none_all_36_required", "X13 design drift")


def bind_run_plan(run, network_root):
    run, network_root = Path(run).resolve(), Path(network_root).resolve()
    session = run.parents[2]
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    meta, workload, nodes = (common.read_json(run / name) for name in
                             ("metadata.json", "plan.json", "nodes.json"))
    require(run.is_dir() and network_root.is_dir() and
            not (run / "research_plan/plan.json").exists() and
            meta.get("case_id") == "X13" and meta.get("rate") in RATES and
            meta.get("scope") in ("QA_ONLY", "X_CONFIRMATORY") and
            meta.get("source_commit") == plan["source_commit"] and
            meta.get("source_tree") == plan["source_tree"] and
            meta.get("run_id") == workload.get("run_id") and len(nodes) == 4 and
            all(n["rpc"].startswith("http://127.0.0.1:") and n["app_pid"] > 0 for n in nodes),
            "X13 run identity/topology")
    config = common.read_json(run / "config.json")
    require(common.digest(run / "config.json") == plan["config_sha256"] and
            [config["safety"][key] for key in ("max_tx_per_second", "max_bytes_per_second",
                                                   "max_outstanding", "max_duration_seconds")] ==
            plan["safety_caps"] and len(workload["events"]) == 1+12*meta["rate"],
            "X13 workload/safety")
    inventory = plan["host"]
    host = {"cpu_model": inventory["model"], "cpu_features": inventory["flags"],
            "architecture": inventory["architecture"], "logical_cpus": inventory["logical_cpus"],
            "os": inventory["os"]}
    network = config["network"]
    bound = {"schema": "pluto.file02.plan.v1", "status": "LOCKED", "run_id": workload["run_id"],
        "case_id": "X13", "protocol_version": "file02-mechanism-service-v1",
        "research_question": "RQ1b safe-range sensitivity",
        "hypothesis": "fixed-valid observation may vary with invalid background rate",
        "null_hypothesis": "no descriptive rate response in observation upper bound",
        "alternative_hypothesis": "descriptive rate response in observation upper bound",
        "analysis_plan": {"primary_endpoint": "fixed_valid_mean_first_all_four_observation_upper_ms",
            "independent_unit": "fresh_four_validator_network",
            "ci_method": "paired_block_bootstrap_20000_seed_20261115",
            "exclusion_rule": "any failed arm invalidates session; no replacement"},
        "raw_capture_plan": ["telemetry", "transaction_results", "validator_blocks_and_apphash",
                             "loadgen_timeline", "host_metrics"],
        "expected_oracle": "valid exact once, bad ML rejected, all-four AppHash agreement",
        "source_commit": plan["source_commit"], "source_tree": plan["source_tree"],
        "config_sha256": plan["config_sha256"], "corpus_sha256": common.digest(run / "plan.json"),
        "genesis_sha256": workload["genesis_sha256"], "profile": "H", "workload": "MLDSA_INVALID",
        "actor_capability": "B", "regime": workload["regime"], "traffic_source": "RPC",
        "topology": "FOUR_VALIDATOR_SINGLE_HOST_DIAGNOSTIC", "policy": "BASELINE_VERIFY_ORDER",
        "host_inventory": host,
        "software_versions": {"go": plan["go_version"], "cometbft": network["comet_version"],
                              "crypto": "go.mod:"+plan["module_sha256"], "os": platform.platform()},
        "network_config": {"validator_count": 4, "mempool_type": network["mempool_type"],
            "mempool_size": network["mempool_size"], "mempool_max_bytes": network["mempool_max_bytes"],
            "mempool_cache_size": network["mempool_cache_size"],
            "mempool_max_tx_bytes": network["mempool_max_tx_bytes"],
            "mempool_recheck": network["mempool_recheck"],
            "mempool_broadcast": network["mempool_broadcast"],
            "max_block_bytes": network["max_block_bytes"],
            "empty_block_interval_ms": network["empty_block_interval_ms"],
            "timeout_commit_ms": network["timeout_commit_ms"],
            "network_shape": "four_validator_single_host_full_mesh", "endpoint_hosts": ["127.0.0.1"]*4,
            "rpc_endpoints": [n["rpc"] for n in nodes],
            "config_toml_sha256": [common.digest(network_root / f"validator-{i}/config/config.toml") for i in range(4)]},
        "safety_config": config["safety"], "seed": workload["seed"], "order_index": meta["order"],
        "phase_seconds": {"warmup": 1, "steady": 12, "drain": 5, "cooldown": 2},
        "planned": {"valid_tx_s": 6, "invalid_tx_s": meta["rate"]-6,
                    "total_bytes_s": meta["rate"]*WIRE},
        "matching": {"matched_variable": "none", "unmatched_variables": ["total_rate", "offered_bytes"],
                     "legitimate_stream_id": f"X13-b{meta['block']:02d}-fixed"},
        "pilot_reference": "research/evidence/file02/mechanism/X10_THREE_ARM_AUDIT.md",
        "independent_run_target": 12,
        "p99_precision_rule": "INSUFFICIENT_PRECISION unless separately established"}
    common.validate_plan(bound)
    directory = run / "research_plan"
    directory.mkdir(mode=0o700)
    r01.write_new(directory / "plan.json", bound)


def audit_run(raw, job, top):
    run = raw / "runs" / job["run_id"]
    meta, workload, result, oracle, commits, bound, nodes = (common.read_json(run / name) for name in
        ("metadata.json", "plan.json", "result.json", "oracle.json",
         "commit_observations.json", "research_plan/plan.json", "nodes.json"))
    common.validate_plan(bound)
    rate = job["rate"]
    total, invalid = 1+12*rate, 12*(rate-6)
    require(meta.get("schema") == "pluto.file02.x13_run.v1" and
            meta.get("scope") in ("QA_ONLY", "X_CONFIRMATORY") and
            meta.get("case_id") == "X13" and meta.get("block") == job["block"] and
            meta.get("order") == job["order"] and meta.get("rate") == rate and
            meta.get("run_id") == job["run_id"] and
            meta.get("source_commit") == bound["source_commit"] == top["source_commit"] and
            meta.get("source_tree") == bound["source_tree"] == top["source_tree"] and
            meta.get("recorder_mode") == "ON" and meta.get("commit_observer_mode") == "block_scan",
            "X13 source/metadata")
    require(bound["run_id"] == workload["run_id"] == result["run_id"] ==
            oracle["run_id"] == commits["run_id"] == job["run_id"] and
            bound["config_sha256"] == meta["config_sha256"] == workload["config_sha256"] ==
            result["config_sha256"] == top["config_sha256"] == common.digest(run / "config.json") and
            bound["corpus_sha256"] == result["plan_sha256"] == common.digest(run / "plan.json") and
            bound["genesis_sha256"] == workload["genesis_sha256"] == result["genesis_sha256"] ==
            common.digest(run / "genesis.bin"), "X13 provenance")
    require(result["status"] == oracle["status"] == "OK" and
            result["planned_count"] == result["actual_send_count"] == total and
            len(workload["events"]) == len(result["events"]) == total and
            oracle["valid_committed"] == 73 and oracle["invalid_rejected"] == invalid and
            oracle["block_id"] and oracle["app_hash"] and
            commits["poll_interval_ms"] == 100 and len(commits["events"]) == 73,
            "X13 execution/correctness")
    require(len(nodes) == 4 and len({n["app_pid"] for n in nodes}) == 4 and
            all(n["rpc"].startswith("http://127.0.0.1:") and n["app_pid"] > 0 for n in nodes),
            "X13 process/topology")
    by_commit = {item["id"]: item for item in commits["events"]}
    require(len(by_commit) == 73, "X13 duplicate/missing valid observation")
    app_events = x10.read_jsonl(run / "app-0-events.jsonl")
    checktx = {item["tx_sha256"]: item for item in app_events
               if item.get("check_type") == "new" and item.get("tx_sha256")}
    require(len(checktx) >= total, "X13 app CheckTx coverage")
    cpu = []
    for node in range(4):
        samples = x10.read_jsonl(run / f"app-{node}-resources.jsonl")
        require(len(samples) >= 2 and all(s["pid"] == nodes[node]["app_pid"] and
                s["recorder_drops"] == 0 for s in samples), "X13 app process samples/drops")
        first = samples[0]["user_cpu_ns"] + samples[0]["system_cpu_ns"]
        last = samples[-1]["user_cpu_ns"] + samples[-1]["system_cpu_ns"]
        require(last >= first, "X13 negative process CPU delta")
        cpu.append(last-first)
        observer = x10.read_jsonl(run / f"observer-{node}.jsonl")
        require(len(observer) >= 2 and any((s.get("application_process") or {}).get("pid") ==
                                            nodes[node]["app_pid"] for s in observer) and
                all(s.get("dropped_samples") == 0 for s in observer),
                "X13 observer/PID mismatch")
    fixed, latencies = [], []
    windows = [{"fixed": 0, "background": 0, "bytes": 0} for _ in range(12)]
    invalid_wall = []
    hashes = set()
    for planned, actual in zip(workload["events"], result["events"]):
        wire = bytes.fromhex(planned["bytes_hex"])
        digest = hashlib.sha256(wire).hexdigest()
        require(len(wire) == actual["wire_bytes"] == WIRE and digest == planned["sha256"] and
                digest not in hashes and planned["id"] == actual["id"] and
                actual["actual_code"] == planned["expected_code"] and
                not actual.get("submission_error") and
                isinstance(actual.get("actual_send_monotonic_ns"), int) and
                actual["actual_send_monotonic_ns"] > 0, "X13 event send/hash/code")
        hashes.add(digest)
        record = checktx.get(digest)
        require(record is not None and record.get("code") == planned["expected_code"] and
                record.get("last_stage") == planned["expected_stage"] and
                record.get("wire_bytes") == WIRE and record.get("service_ns", 0) > 0 and
                record.get("app_ingress_utc") and record.get("app_lock_acquired_utc") and
                record.get("app_result_utc"), "X13 app stage/timestamps")
        if planned["traffic"] == "adversarial":
            require(planned["expected_code"] == 13 and planned["expected_stage"] == "ml" and
                    planned["id"] not in by_commit and
                    {"structural", "binding", "secp", "ml"} <=
                    {span["stage"] for span in record.get("spans", [])}, "X13 late bad ML")
            if planned["offset_ns"] >= 1_000_000_000:
                invalid_wall.append(record["service_ns"])
        else:
            observed = by_commit.get(planned["id"])
            require(observed and observed["sha256"] == digest and observed["height"] >= 1 and
                    len(observed.get("per_validator_seen_monotonic_ns", [])) == 4 and
                    all(isinstance(stamp, int) and stamp > 0 for stamp in
                        observed["per_validator_seen_monotonic_ns"]) and
                    len(observed.get("per_validator_seen_utc", [])) == 4 and
                    observed["all_four_seen_monotonic_ns"] ==
                    max(observed["per_validator_seen_monotonic_ns"]),
                    "X13 valid all-four observation")
        if planned["scenario"] == "fixed_legitimate":
            fixed.append([digest, planned["offset_ns"], planned["bytes_hex"]])
            if planned["offset_ns"] >= 1_000_000_000:
                stamp = by_commit[planned["id"]]["all_four_seen_monotonic_ns"]
                require(stamp >= actual["actual_send_monotonic_ns"], "X13 observation before send")
                latencies.append((stamp-actual["actual_send_monotonic_ns"])/1_000_000)
        if planned["offset_ns"] >= 1_000_000_000:
            index = (actual["actual_offset_ns"]-1_000_000_000)//1_000_000_000
            require(0 <= index < 12, "X13 send outside steady windows")
            windows[index]["fixed" if planned["scenario"] == "fixed_legitimate" else "background"] += 1
            windows[index]["bytes"] += WIRE
    require(len(hashes) == total and len(fixed) == 73 and len(latencies) == 72 and
            len(invalid_wall) == invalid and
            all(w == {"fixed": 6, "background": rate-6, "bytes": rate*WIRE} for w in windows),
            "X13 fixed/rate/invalid coverage")
    return {"run_id": job["run_id"], "block": job["block"], "order": job["order"],
        "rate": rate, "scope": meta["scope"], "genesis_sha256": workload["genesis_sha256"],
        "fixed_stream": fixed, "fixed_mean_upper_ms": statistics.mean(latencies),
        "fixed_p50_upper_ms": x10.percentile(latencies, .5),
        "fixed_p95_upper_ms": x10.percentile(latencies, .95),
        "fixed_timely_count": sum(ms <= 3000 for ms in latencies), "fixed_n": 72,
        "invalid_mean_checktx_wall_ns": statistics.mean(invalid_wall),
        "app_process_cpu_delta_ns": cpu, "windows": windows, "qc_status": "PASS"}


def derive(session, plan):
    raw = session / "raw"
    manifest = common.verify_raw(raw)
    jobs = schedule()["jobs"]
    qa_only = (raw / "runs").is_dir() and len(list((raw / "runs").iterdir())) == 3
    if qa_only:
        jobs = jobs[:3]
    rows, failures = [], []
    for job in jobs:
        try:
            rows.append(audit_run(raw, job, plan))
        except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
            failures.append({"run_id": job["run_id"], "reason": str(exc)})
    by_block = {}
    for row in rows:
        by_block.setdefault(row["block"], {})[row["rate"]] = row
    for block, arms in by_block.items():
        if set(arms) != set(RATES) or len({r["genesis_sha256"] for r in arms.values()}) != 1 or \
                len({json.dumps(r["fixed_stream"]) for r in arms.values()}) != 1:
            failures.append({"block": block, "reason": "matched rate block mismatch"})
    status = ("QA_ONLY" if qa_only and len(rows) == 3 and not failures and
              all(r["scope"] == "QA_ONLY" for r in rows) else
              "PASS" if not qa_only and len(rows) == 36 and len(by_block) == 12 and not failures and
              all(r["scope"] == "X_CONFIRMATORY" for r in rows) else "INVALID_FOR_INFERENCE")
    qc = {"schema": "pluto.file02.x13_qc.v1", "status": status,
          "n_planned": 3 if qa_only else 36, "n_valid": len(rows),
          "failures": failures, "raw_manifest_sha256": manifest,
          "plan_sha256": common.digest(session / "plan.json")}
    estimates = {}
    if status == "PASS":
        for rate in RATES:
            means = [by_block[b][rate]["fixed_mean_upper_ms"] for b in range(12)]
            timely = [by_block[b][rate]["fixed_timely_count"]/72 for b in range(12)]
            estimates[str(rate)] = {"n_networks": 12, "mean_upper_ms": statistics.mean(means),
                "ci95_mean_upper_ms": x10.paired_ci(means),
                "timely_fraction": statistics.mean(timely),
                "ci95_timely_fraction": x10.paired_ci(timely),
                "invalid_checktx_wall_ns": statistics.mean(
                    by_block[b][rate]["invalid_mean_checktx_wall_ns"] for b in range(12))}
        differences = {str(rate): {"mean_upper_ms": statistics.mean([
            by_block[b][rate]["fixed_mean_upper_ms"] - by_block[b][20]["fixed_mean_upper_ms"]
            for b in range(12)]), "ci95_ms": x10.paired_ci([
                by_block[b][rate]["fixed_mean_upper_ms"] - by_block[b][20]["fixed_mean_upper_ms"]
                for b in range(12)])} for rate in (23, 25)}
    else:
        differences = None
    aggregate = {"schema": "pluto.file02.x13_aggregate.v1",
        "status": "DESCRIPTIVE_SENSITIVITY" if status == "PASS" else "WITHHELD_QC_OR_QA",
        "estimates": estimates if status == "PASS" else None,
        "paired_rate_minus_20": differences,
        "knee_status": "NOT_OBSERVED_WITHIN_SAFE_RANGE" if status == "PASS" else None,
        "exact_commit_latency_ms": None, "exact_commit_reason": "UNAVAILABLE_EXACT_COMMIT_TIMESTAMP",
        "cpu_crypto_ns": None, "cpu_crypto_reason": "UNAVAILABLE_STAGE_CPU_ATTRIBUTION",
        "raw_manifest_sha256": manifest, "plan_sha256": qc["plan_sha256"]}
    return rows, qc, aggregate


def csv_text(rows):
    output = io.StringIO()
    keys = ("run_id", "block", "order", "rate", "fixed_mean_upper_ms",
            "fixed_p50_upper_ms", "fixed_p95_upper_ms", "fixed_timely_count",
            "fixed_n", "invalid_mean_checktx_wall_ns", "qc_status")
    writer = csv.DictWriter(output, fieldnames=keys, lineterminator="\n")
    writer.writeheader()
    writer.writerows({key: row[key] for key in keys} for row in rows)
    return output.getvalue()


def decision_text(qc, aggregate):
    return ("# X13 safe-range sensitivity decision\n\n"
            f"QC: {qc['status']}; valid networks {qc['n_valid']}/{qc['n_planned']}.\n\n"
            f"Rate-specific first-all-four observation upper bounds and paired rate contrasts: "
            f"{json.dumps(aggregate['estimates'], sort_keys=True)}; "
            f"{json.dumps(aggregate['paired_rate_minus_20'], sort_keys=True)}.\n\n"
            f"Knee: {aggregate['knee_status']}. No capacity or sustainable-throughput "
            "claim. Exact commit time and crypto-stage CPU unavailable. Single-host RPC only.\n")


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.x13_per_run.v1", "rows": rows})
    common.write_once(session / "result_per_run.csv", csv_text(rows))
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir(mode=0o700)
    r01.write_new(session / "figures/status.json", {"status": "PENDING_X17_FIGURES"})
    common.write_once(session / "decision.md", decision_text(qc, aggregate))
    return qc, aggregate


def execute(qa=False):
    session = ARTIFACTS / ("x13-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") +
                           "-" + uuid4().hex[:8])
    plan = prepare(session)
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT,
                               stdout=stream, stderr=subprocess.STDOUT)
    require(build.returncode == 0, f"X13 build failed: {session}")
    env = os.environ.copy()
    if qa:
        env["FILE02_X13_QA_LIMIT"] = "1"
    else:
        env.pop("FILE02_X13_QA_LIMIT", None)
    env.update(FILE02_X13_ROOT=str(raw), FILE02_SOURCE_COMMIT=plan["source_commit"],
               FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        completed = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                                    "-run", "^TestFile02X13Sensitivity$", "-timeout", "7200s", "-count=1", "-v"],
                                   cwd=ROOT, env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=7300)
    quiesce_session_loads(session)
    manifest = common.freeze_raw(raw)
    qc, aggregate = write_outputs(session, plan)
    print(f"X13_SESSION={session} INTEGRATION_EXIT={completed.returncode} QC={qc['status']} "
          f"KNEE={aggregate['knee_status']} RAW_MANIFEST={manifest}")
    return 0 if completed.returncode == 0 and qc["status"] == ("QA_ONLY" if qa else "PASS") else 2


def quiesce_session_loads(session):
    """Prevent a detached load generator writing into raw after freeze."""
    matched = []
    for path in Path("/proc").glob("[0-9]*/cmdline"):
        try:
            command = path.read_bytes().replace(b"\0", b" ").decode(errors="replace")
        except OSError:
            continue
        if str(session) in command and str(ROOT / "build/profile-load") in command:
            matched.append(int(path.parent.name))
    for pid in matched:
        try:
            os.kill(pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
    for _ in range(100):
        if not any(Path(f"/proc/{pid}").exists() for pid in matched):
            return
        time.sleep(.1)
    require(False, f"X13 load process still active after integration exit: {matched}")


def verify(value):
    session = Path(value).resolve()
    require(session.parent == ARTIFACTS.resolve() and session.is_dir(), "private X13 session path")
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    rows, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.x13_per_run.v1", "rows": rows} and
            (session / "result_per_run.csv").read_text() == csv_text(rows) and
            common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate and
            common.read_json(session / "figures/status.json") == {"status": "PENDING_X17_FIGURES"} and
            (session / "decision.md").read_text() == decision_text(qc, aggregate),
            "X13 derived artifact drift")
    print(f"X13_VERIFY=OK qc={qc['status']} raw_manifest_sha256={qc['raw_manifest_sha256']}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    bind = sub.add_parser("bind-run-plan")
    bind.add_argument("run")
    bind.add_argument("network_root")
    run = sub.add_parser("run")
    run.add_argument("--qa", action="store_true")
    check = sub.add_parser("verify")
    check.add_argument("session")
    args = parser.parse_args()
    try:
        if args.command == "bind-run-plan":
            bind_run_plan(args.run, args.network_root)
            return 0
        if args.command == "run":
            return execute(args.qa)
        verify(args.session)
        return 0
    except (OSError, ValueError, KeyError, TypeError, subprocess.CalledProcessError) as exc:
        print(f"X13_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())

