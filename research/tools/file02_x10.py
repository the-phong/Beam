#!/usr/bin/env python3
"""Prospectively locked X10 three-arm single-host service experiment."""

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
import statistics
import subprocess
import sys
from uuid import uuid4

import file02_common as common
import file02_r01 as r01

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
PERMUTATIONS = ("LBI", "LIB", "BLI", "BIL", "ILB", "IBL")
SEED = 20261115
WIRE_BYTES = 2568


def require(condition, reason):
    if not condition:
        raise ValueError(reason)


def schedule():
    orders = list(PERMUTATIONS) * 4
    random.Random(SEED).shuffle(orders)
    jobs = [{"block": block, "order": order, "condition": arm,
             "run_id": f"X10-b{block:02d}-{arm}"}
            for block, permutation in enumerate(orders)
            for order, arm in enumerate(permutation)]
    return {"schema": "pluto.file02.x10_schedule.v1", "seed": SEED,
            "permutations": orders, "jobs": jobs}


def session_path(value):
    path = Path(value).resolve()
    require(path.parent == ARTIFACTS.resolve() and path.is_dir(), "private X10 session path")
    return path


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve() and not r01.git("status", "--porcelain"),
            "X10 source must be clean before plan lock")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT).returncode == 0,
            "X10 session not ignored")
    session.mkdir(mode=0o700)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    common.write_once(raw / "schedule.json", json.dumps(schedule(), indent=2, sort_keys=True) + "\n")
    common.write_once(raw / "config.json", (ROOT / "research/config/s06_pilot.json").read_text())
    plan = {"schema": "pluto.file02.x10_plan.v1", "status": "LOCKED", "case_id": "X10",
        "source_commit": r01.git("rev-parse", "HEAD"), "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
        "created_utc": datetime.now(timezone.utc).isoformat(), "host": r01.cpu_inventory(),
        "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
        "module_sha256": common.digest(ROOT / "go.mod"),
        "config_sha256": common.digest(raw / "config.json"),
        "schedule_sha256": common.digest(raw / "schedule.json"),
        "protocol": "research/protocol/LOCKED_EXTENSION_PROTOCOL.md",
        "service_contract": "research/protocol/SERVICE_METRICS_CONTRACT.md",
        "n_blocks": 24, "n_networks": 72, "arms": {"L": [6, 0], "B": [20, 0], "I": [6, 14]},
        "wire_bytes": WIRE_BYTES, "warmup_seconds": 1, "steady_seconds": 12,
        "drain_seconds": 5, "cooldown_seconds": 2, "deadline_ms": 3000,
        "block_scan_interval_ms": 100, "ci_seed": SEED, "ci_resamples": 20000,
        "primary_contrast": "I_minus_B_mean_first_all_four_observation_upper_ms",
        "margin_ms": 100, "target_halfwidth_ms": 40,
        "topology": "four_validator_single_physical_host_loopback_RPC_validator0",
        "safety_caps": [25, 65536, 16, 60]}
    r01.write_new(session / "plan.json", plan)
    return plan


def validate_plan(session, plan):
    require(plan.get("schema") == "pluto.file02.x10_plan.v1" and plan.get("status") == "LOCKED" and
            plan.get("case_id") == "X10" and plan.get("source_tree") ==
            r01.git("rev-parse", plan["source_commit"] + "^{tree}"), "X10 source/status")
    require(plan.get("host") == r01.cpu_inventory() and plan.get("go_version") ==
            subprocess.check_output(["go", "version"], text=True).strip() and
            plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("config_sha256") == common.digest(session / "raw/config.json") ==
            common.digest(ROOT / "research/config/s06_pilot.json") and
            plan.get("schedule_sha256") == common.digest(session / "raw/schedule.json") and
            common.read_json(session / "raw/schedule.json") == schedule(), "X10 plan/config/schedule drift")
    require(plan.get("n_blocks") == 24 and plan.get("n_networks") == 72 and
            plan.get("arms") == {"L": [6, 0], "B": [20, 0], "I": [6, 14]} and
            plan.get("safety_caps") == [25, 65536, 16, 60] and
            plan.get("ci_resamples") == 20000 and plan.get("margin_ms") == 100,
            "X10 design drift")


def bind_run_plan(run, network_root):
    run, network_root = Path(run).resolve(), Path(network_root).resolve()
    session = run.parents[2]
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    require(run.is_dir() and network_root.is_dir() and not (run / "research_plan/plan.json").exists(),
            "X10 run/network path or prior plan")
    meta, workload, nodes = (common.read_json(run / name) for name in
                             ("metadata.json", "plan.json", "nodes.json"))
    arm = meta.get("condition")
    require(arm in ("L", "B", "I") and meta.get("scope") in ("QA_ONLY", "X_CONFIRMATORY") and
            meta.get("source_commit") == plan["source_commit"] and
            meta.get("source_tree") == plan["source_tree"] and
            meta.get("run_id") == workload.get("run_id") and len(nodes) == 4 and
            all(n["rpc"].startswith("http://127.0.0.1:") and n["app_pid"] > 0
                for n in nodes), "X10 run identity/topology")
    config = common.read_json(run / "config.json")
    require(common.digest(run / "config.json") == plan["config_sha256"] and
            config["safety"]["max_tx_per_second"] == 25 and
            config["safety"]["max_bytes_per_second"] == 65536 and
            config["safety"]["max_outstanding"] == 16 and
            config["safety"]["max_duration_seconds"] == 60, "X10 safety/config")
    valid_rate, invalid_rate = plan["arms"][arm]
    require(workload["profile"] == 4 and len(workload["events"]) == (73 if arm == "L" else 241) and
            workload["warmup_ns"] == 1_000_000_000 and workload["steady_ns"] == 12_000_000_000 and
            workload["drain_ns"] == 5_000_000_000, "X10 workload shape")
    inventory = plan["host"]
    host = {"cpu_model": inventory["model"], "cpu_features": inventory["flags"],
            "architecture": inventory["architecture"], "logical_cpus": inventory["logical_cpus"],
            "os": inventory["os"]}
    network = config["network"]
    bound = {"schema": "pluto.file02.plan.v1", "status": "LOCKED", "run_id": workload["run_id"],
        "case_id": "X10", "protocol_version": "file02-mechanism-service-v1", "research_question": "RQ1b extension",
        "hypothesis": "late-invalid background may change fixed valid H observation upper bound",
        "null_hypothesis": "I minus B fixed valid mean observation upper bound <= 0",
        "alternative_hypothesis": "I minus B fixed valid mean observation upper bound > 0",
        "analysis_plan": {"primary_endpoint": "fixed_valid_mean_first_all_four_observation_upper_ms",
                          "independent_unit": "fresh_four_validator_network",
                          "ci_method": "paired_percentile_bootstrap_20000_seed_20261115",
                          "exclusion_rule": "failed arm excludes matched block; no replacement"},
        "raw_capture_plan": ["telemetry", "transaction_results", "validator_blocks_and_apphash",
                             "loadgen_timeline", "host_metrics"],
        "expected_oracle": "all valid exact once, late ML invalid stage 13, four-validator agreement",
        "source_commit": plan["source_commit"], "source_tree": plan["source_tree"],
        "config_sha256": plan["config_sha256"], "corpus_sha256": common.digest(run / "plan.json"),
        "genesis_sha256": workload["genesis_sha256"], "profile": "H",
        "workload": {"L": "BENIGN_LOW", "B": "BENIGN_HIGH", "I": "MLDSA_INVALID"}[arm],
        "actor_capability": "B" if arm == "I" else "NONE",
        "regime": workload["regime"], "traffic_source": "RPC",
        "topology": "FOUR_VALIDATOR_SINGLE_HOST_DIAGNOSTIC", "policy": "BASELINE_VERIFY_ORDER",
        "host_inventory": host,
        "software_versions": {"go": plan["go_version"], "cometbft": network["comet_version"],
                              "crypto": "go.mod:" + plan["module_sha256"], "os": platform.platform()},
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
        "planned": {"valid_tx_s": valid_rate, "invalid_tx_s": invalid_rate,
                    "total_bytes_s": (valid_rate + invalid_rate) * WIRE_BYTES},
        "matching": {"matched_variable": "none" if arm == "L" else "bytes_s",
                     "unmatched_variables": ["admission", "relay", "commit"],
                     "legitimate_stream_id": f"X10-b{meta['block']:02d}-fixed"},
        "pilot_reference": "research/evidence/file02/mechanism/INSTRUMENTATION_EQUIVALENCE_QC.md",
        "independent_run_target": 24,
        "p99_precision_rule": "INSUFFICIENT_PRECISION unless separately established"}
    common.validate_plan(bound)
    directory = run / "research_plan"
    directory.mkdir(mode=0o700)
    r01.write_new(directory / "plan.json", bound)


def percentile(values, fraction):
    ordered = sorted(values)
    position = (len(ordered) - 1) * fraction
    lower = int(position)
    weight = position - lower
    return ordered[lower] * (1 - weight) + ordered[min(lower + 1, len(ordered) - 1)] * weight


def paired_ci(values):
    require(values, "empty paired bootstrap")
    rng = random.Random(SEED)
    means = sorted(statistics.mean(rng.choice(values) for _ in values) for _ in range(20000))
    return [percentile(means, .025), percentile(means, .975)]


def read_jsonl(path):
    return [json.loads(line) for line in Path(path).read_text().splitlines() if line.strip()]


def audit_run(raw, job, top_plan):
    arm = job["condition"]
    run = raw / "runs" / job["run_id"]
    meta, workload, result, oracle, commits, bound = (common.read_json(run / name) for name in
        ("metadata.json", "plan.json", "result.json", "oracle.json",
         "commit_observations.json", "research_plan/plan.json"))
    common.validate_plan(bound)
    require(meta.get("schema") == "pluto.file02.x10_run.v1" and
            meta.get("scope") == "X_CONFIRMATORY" and meta.get("case_id") == "X10" and
            meta.get("condition") == arm and meta.get("block") == job["block"] and
            meta.get("order") == job["order"] and meta.get("run_id") == job["run_id"] and
            meta.get("source_commit") == bound["source_commit"] == top_plan["source_commit"] and
            meta.get("source_tree") == bound["source_tree"] == top_plan["source_tree"] and
            meta.get("recorder_mode") == "ON" and meta.get("commit_observer_mode") == "block_scan",
            "X10 run metadata/source")
    require(bound["run_id"] == workload["run_id"] == result["run_id"] == oracle["run_id"] ==
            commits["run_id"] == job["run_id"] and
            bound["config_sha256"] == meta["config_sha256"] == workload["config_sha256"] ==
            result["config_sha256"] == top_plan["config_sha256"] == common.digest(run / "config.json") and
            bound["corpus_sha256"] == result["plan_sha256"] == common.digest(run / "plan.json") and
            bound["genesis_sha256"] == workload["genesis_sha256"] == result["genesis_sha256"] ==
            common.digest(run / "genesis.bin"), "X10 plan/actual provenance")
    valid_expected, invalid_expected = {"L": (73, 0), "B": (241, 0), "I": (73, 168)}[arm]
    total = valid_expected + invalid_expected
    require(result["status"] == oracle["status"] == "OK" and
            result["planned_count"] == result["actual_send_count"] == total and
            len(workload["events"]) == len(result["events"]) == total and
            oracle["valid_committed"] == valid_expected and
            oracle["invalid_rejected"] == invalid_expected and
            oracle["block_id"] and oracle["app_hash"] and
            commits["poll_interval_ms"] == 100 and len(commits["events"]) == valid_expected,
            "X10 execution/correctness")
    nodes = common.read_json(run / "nodes.json")
    require(len(nodes) == 4 and len({n["app_pid"] for n in nodes}) == 4 and
            all(n["app_pid"] != os.getpid() for n in nodes), "X10 dedicated app identity")
    by_commit = {item["id"]: item for item in commits["events"]}
    require(len(by_commit) == valid_expected, "X10 duplicate/missing commit observation")
    app_events = read_jsonl(run / "app-0-events.jsonl")
    checktx = {item["tx_sha256"]: item for item in app_events
               if item.get("check_type") == "new" and item.get("tx_sha256")}
    require(len(checktx) >= total, "X10 missing app CheckTx events")
    process_cpu = []
    for node in range(4):
        resources = read_jsonl(run / f"app-{node}-resources.jsonl")
        require(len(resources) >= 2 and all(item["pid"] == nodes[node]["app_pid"] and
                item["recorder_drops"] == 0 for item in resources) and
                resources[-1]["user_cpu_ns"] + resources[-1]["system_cpu_ns"] >=
                resources[0]["user_cpu_ns"] + resources[0]["system_cpu_ns"],
                "X10 app process samples/drops")
        process_cpu.append(resources[-1]["user_cpu_ns"] + resources[-1]["system_cpu_ns"] -
                           resources[0]["user_cpu_ns"] - resources[0]["system_cpu_ns"])
        observed = read_jsonl(run / f"observer-{node}.jsonl")
        require(len(observed) >= 2 and any((item.get("application_process") or {}).get("pid") ==
                                            nodes[node]["app_pid"] for item in observed),
                "X10 observer app PID mismatch")
    fixed, latencies, windows = [], [], [{"fixed": 0, "background": 0, "bytes": 0} for _ in range(12)]
    invalid_service_ns, ml_stage_ns = [], []
    all_hashes = set()
    for planned, actual in zip(workload["events"], result["events"]):
        wire = bytes.fromhex(planned["bytes_hex"])
        digest = hashlib.sha256(wire).hexdigest()
        require(digest == planned["sha256"] and digest not in all_hashes and
                planned["id"] == actual["id"] and len(wire) == actual["wire_bytes"] == WIRE_BYTES and
                actual["actual_code"] == planned["expected_code"] and
                not actual.get("submission_error") and
                isinstance(actual.get("actual_send_monotonic_ns"), int) and
                actual["actual_send_monotonic_ns"] > 0, "X10 event send/hash/code")
        all_hashes.add(digest)
        record = checktx.get(digest)
        require(record is not None and record.get("code") == planned["expected_code"] and
                record.get("last_stage") == planned["expected_stage"] and
                record.get("wire_bytes") == WIRE_BYTES and
                isinstance(record.get("service_ns"), int) and record["service_ns"] > 0 and
                record.get("app_ingress_utc") and record.get("app_lock_acquired_utc") and
                record.get("app_result_utc"), "X10 app event/stage/timestamp")
        if arm == "I" and planned["traffic"] == "adversarial":
            require(planned["expected_code"] == 13 and planned["expected_stage"] == "ml" and
                    planned["id"] not in by_commit and
                    {"structural", "binding", "secp", "ml"} <=
                    {span["stage"] for span in record.get("spans", [])}, "X10 bad-ML oracle")
            if planned["offset_ns"] >= 1_000_000_000:
                invalid_service_ns.append(record["service_ns"])
                ml_stage_ns.extend(span["duration_ns"] for span in record["spans"] if span["stage"] == "ml")
        else:
            observed = by_commit.get(planned["id"])
            require(observed is not None and observed["sha256"] == digest and observed["height"] >= 1,
                    "X10 valid commit observation")
            stamps = observed.get("per_validator_seen_monotonic_ns")
            require(isinstance(stamps, list) and len(stamps) == 4 and
                    all(isinstance(stamp, int) and stamp > 0 for stamp in stamps) and
                    observed["all_four_seen_monotonic_ns"] == max(stamps) and
                    len(observed.get("per_validator_seen_utc", [])) == 4,
                    "X10 per-validator observation")
        if planned["scenario"] == "fixed_legitimate":
            fixed.append((digest, planned["offset_ns"], planned["bytes_hex"]))
            if planned["offset_ns"] >= 1_000_000_000:
                seen = by_commit[planned["id"]]["all_four_seen_monotonic_ns"]
                require(seen >= actual["actual_send_monotonic_ns"], "X10 monotonic observation before send")
                latencies.append((seen - actual["actual_send_monotonic_ns"]) / 1_000_000)
        if planned["offset_ns"] >= 1_000_000_000:
            sent = actual["actual_offset_ns"]
            window = (sent - 1_000_000_000) // 1_000_000_000
            require(0 <= window < 12, "X10 phase/rate window")
            windows[window]["fixed" if planned["scenario"] == "fixed_legitimate" else "background"] += 1
            windows[window]["bytes"] += WIRE_BYTES
    expected_window = {"fixed": 6, "background": 0 if arm == "L" else 14,
                       "bytes": (6 if arm == "L" else 20) * WIRE_BYTES}
    require(len(all_hashes) == total and len(fixed) == 73 and len(latencies) == 72 and
            all(window == expected_window for window in windows) and
            (arm != "I" or len(invalid_service_ns) == len(ml_stage_ns) == 168),
            "X10 fixed stream, rate or invalid stage coverage")
    timely = sum(value <= 3000 for value in latencies)
    return {"run_id": job["run_id"], "block": job["block"], "order": job["order"],
        "condition": arm, "genesis_sha256": workload["genesis_sha256"], "fixed_stream": fixed,
        "fixed_mean_upper_ms": statistics.mean(latencies), "fixed_p50_upper_ms": percentile(latencies, .5),
        "fixed_p95_upper_ms": percentile(latencies, .95), "fixed_timely_count": timely,
        "fixed_timely_fraction": timely / 72, "fixed_n": 72,
        "invalid_mean_checktx_wall_ns": statistics.mean(invalid_service_ns) if invalid_service_ns else None,
        "invalid_mean_ml_stage_wall_ns": statistics.mean(ml_stage_ns) if ml_stage_ns else None,
        "app_process_cpu_delta_ns": process_cpu, "windows": windows, "qc_status": "PASS"}


def derive(session, plan):
    raw = session / "raw"
    manifest = common.verify_raw(raw)
    rows, failures = [], []
    for job in schedule()["jobs"]:
        try:
            rows.append(audit_run(raw, job, plan))
        except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError, AssertionError) as exc:
            failures.append({"run_id": job["run_id"], "reason": str(exc)})
    blocks = []
    by_block = {index: {} for index in range(24)}
    for row in rows:
        by_block[row["block"]][row["condition"]] = row
    for block, arms in by_block.items():
        if set(arms) != {"L", "B", "I"}:
            failures.append({"block": block, "reason": "missing arm; block excluded without replacement"})
            continue
        if (len({row["genesis_sha256"] for row in arms.values()}) != 1 or
                len({json.dumps(row["fixed_stream"]) for row in arms.values()}) != 1):
            failures.append({"block": block, "reason": "genesis/fixed stream mismatch"})
            continue
        blocks.append({"block": block,
            "i_minus_b_mean_upper_ms": arms["I"]["fixed_mean_upper_ms"] - arms["B"]["fixed_mean_upper_ms"],
            "i_minus_l_mean_upper_ms": arms["I"]["fixed_mean_upper_ms"] - arms["L"]["fixed_mean_upper_ms"],
            "i_minus_b_timely_fraction": arms["I"]["fixed_timely_fraction"] - arms["B"]["fixed_timely_fraction"]})
    status = "PASS" if len(rows) == 72 and len(blocks) == 24 and not failures else "INVALID_FOR_INFERENCE"
    qc = {"schema": "pluto.file02.x10_qc.v1", "status": status, "n_planned": 72,
          "n_valid": len(rows), "n_blocks": len(blocks), "failures": failures,
          "raw_manifest_sha256": manifest, "plan_sha256": common.digest(session / "plan.json")}
    primary = [block["i_minus_b_mean_upper_ms"] for block in blocks]
    secondary = [block["i_minus_l_mean_upper_ms"] for block in blocks]
    if status == "PASS":
        primary_ci = paired_ci(primary)
        secondary_ci = paired_ci(secondary)
        effect = statistics.mean(primary)
        if primary_ci[0] > 100:
            decision = "RESEARCH_RELEVANT_LATENCY_EFFECT"
        elif primary_ci[0] > 0:
            decision = "POSITIVE_DIRECTION_PRACTICAL_SIZE_INCONCLUSIVE"
        elif primary_ci[0] > -100 and primary_ci[1] < 100 and all(
                row["fixed_timely_count"] == 72 for row in rows):
            decision = "NO_RESEARCH_RELEVANT_MEAN_EFFECT_WITHIN_REGISTERED_MARGIN"
        else:
            decision = "INCONCLUSIVE_CONTINUOUS_EFFECT"
    else:
        primary_ci = secondary_ci = effect = None
        decision = "WITHHELD_QC_FAILURE"
    aggregate = {"schema": "pluto.file02.x10_aggregate.v1", "status": "ANALYZED" if status == "PASS" else "WITHHELD_QC_FAILURE",
        "n_blocks": len(blocks), "primary_i_minus_b_mean_upper_ms": effect, "primary_ci95_ms": primary_ci,
        "i_minus_l_mean_upper_ms": statistics.mean(secondary) if status == "PASS" else None,
        "i_minus_l_ci95_ms": secondary_ci, "target_halfwidth_ms": 40,
        "achieved_halfwidth_ms": (primary_ci[1]-primary_ci[0])/2 if status == "PASS" else None,
        "original_3000ms_i_minus_b_fraction": statistics.mean(
            block["i_minus_b_timely_fraction"] for block in blocks) if status == "PASS" else None,
        "original_3000ms_ci95_fraction": paired_ci([b["i_minus_b_timely_fraction"] for b in blocks]) if status == "PASS" else None,
        "decision": decision, "blocks": blocks if status == "PASS" else None,
        "cpu_crypto_ns": None, "cpu_crypto_reason": "UNAVAILABLE_STAGE_CPU_ATTRIBUTION",
        "exact_commit_latency_ms": None, "exact_commit_reason": "UNAVAILABLE_EXACT_COMMIT_TIMESTAMP",
        "raw_manifest_sha256": manifest, "plan_sha256": qc["plan_sha256"]}
    return rows, qc, aggregate


def csv_text(rows):
    output = io.StringIO()
    keys = ("run_id", "block", "order", "condition", "genesis_sha256", "fixed_mean_upper_ms",
            "fixed_p50_upper_ms", "fixed_p95_upper_ms", "fixed_timely_count", "fixed_n",
            "fixed_timely_fraction", "invalid_mean_checktx_wall_ns", "qc_status")
    writer = csv.DictWriter(output, fieldnames=keys, lineterminator="\n")
    writer.writeheader()
    writer.writerows({key: row[key] for key in keys} for row in rows)
    return output.getvalue()


def decision_text(qc, aggregate):
    return ("# X10 three-arm service decision\n\n"
            f"QC: {qc['status']}; valid networks {qc['n_valid']}/72; matched blocks {qc['n_blocks']}/24.\n\n"
            f"Primary I−B mean upper-bound effect: {aggregate['primary_i_minus_b_mean_upper_ms']} ms, "
            f"paired CI95 {aggregate['primary_ci95_ms']} ms.\n\n"
            f"Secondary I−L mean upper-bound effect: {aggregate['i_minus_l_mean_upper_ms']} ms, "
            f"paired CI95 {aggregate['i_minus_l_ci95_ms']} ms.\n\n"
            f"Historical 3-s endpoint retained; new I−B fraction effect "
            f"{aggregate['original_3000ms_i_minus_b_fraction']} CI95 "
            f"{aggregate['original_3000ms_ci95_fraction']}.\n\n"
            f"Decision: {aggregate['decision']}. Observation is an upper bound, not exact commit. "
            "Single physical host and RPC origin only; no DoS, knee or distributed-network claim.\n")


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.x10_per_run.v1", "rows": rows})
    common.write_once(session / "result_per_run.csv", csv_text(rows))
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir(mode=0o700)
    r01.write_new(session / "figures/status.json", {"status": "PENDING_X17_FIGURES",
        "reason": "no plot is used as a substitute for per-run QC and aggregate"})
    common.write_once(session / "decision.md", decision_text(qc, aggregate))
    return qc, aggregate


def execute():
    session = ARTIFACTS / ("x10-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") +
                           "-" + uuid4().hex[:8])
    plan = prepare(session)
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT,
                               stdout=stream, stderr=subprocess.STDOUT)
    require(build.returncode == 0, f"X10 build failed; private session {session}")
    env = os.environ.copy()
    env.pop("FILE02_X10_QA_LIMIT", None)
    env.update(FILE02_X10_ROOT=str(raw), FILE02_SOURCE_COMMIT=plan["source_commit"],
               FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        completed = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                                    "-run", "^TestFile02X10ThreeArm$", "-timeout", "7200s", "-count=1", "-v"],
                                   cwd=ROOT, env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=7300)
    manifest = common.freeze_raw(raw)
    qc, aggregate = write_outputs(session, plan)
    print(f"X10_SESSION={session} INTEGRATION_EXIT={completed.returncode} QC={qc['status']} "
          f"DECISION={aggregate['decision']} RAW_MANIFEST={manifest}")
    return 0 if completed.returncode == 0 and qc["status"] == "PASS" else 2


def verify(session):
    session = session_path(session)
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    rows, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.x10_per_run.v1", "rows": rows} and
            (session / "result_per_run.csv").read_text() == csv_text(rows) and
            common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate and
            (session / "decision.md").read_text() == decision_text(qc, aggregate) and
            common.read_json(session / "figures/status.json") ==
            {"status": "PENDING_X17_FIGURES", "reason":
             "no plot is used as a substitute for per-run QC and aggregate"},
            "X10 derived artifact mismatch")
    print(f"X10_VERIFY=OK qc={qc['status']} raw_manifest_sha256={qc['raw_manifest_sha256']}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    bind = sub.add_parser("bind-run-plan")
    bind.add_argument("run")
    bind.add_argument("network_root")
    sub.add_parser("run")
    check = sub.add_parser("verify")
    check.add_argument("session")
    args = parser.parse_args()
    try:
        if args.command == "bind-run-plan":
            bind_run_plan(args.run, args.network_root)
            return 0
        if args.command == "run":
            return execute()
        verify(args.session)
        return 0
    except (OSError, ValueError, KeyError, TypeError, subprocess.CalledProcessError) as exc:
        print(f"X10_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
