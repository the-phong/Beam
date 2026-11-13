#!/usr/bin/env python3
"""Locked R06 equal-tx-rate network execution and raw-bound analysis."""

import argparse
import csv
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import platform
import random
import shutil
import statistics
import subprocess
import sys
from uuid import uuid4

import file02_common as common
import file02_r01 as r01
import file02_t06_precision as precision

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
PROFILES = ("C", "CP", "P", "H")
WIRE = {"C": 148, "CP": 2568, "P": 2504, "H": 2568}
SEED = 20261113


def require(ok, message):
    if not ok:
        raise ValueError(message)


def schedule():
    rng = random.Random(SEED)
    order = list(PROFILES)
    rng.shuffle(order)
    jobs = []
    for block in range(12):
        seed = rng.randrange(1, 2**63)
        for position, profile in enumerate(order[block % 4:] + order[:block % 4]):
            jobs.append({"block": block, "order": position, "profile": profile,
                         "seed": seed, "run_id": f"r06-b{block:02d}-{profile.lower()}"})
    return {"schema": "pluto.file02.r06.schedule.v1", "master_seed": SEED,
            "method": "seeded_permutation_rotated_twelve_blocks", "jobs": jobs}


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve(), "private session parent")
    require(not r01.git("status", "--porcelain"), "source dirty before lock")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT).returncode == 0,
            "session not Git-ignored")
    session.mkdir(mode=0o700)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    common.write_once(raw / "schedule.json", json.dumps(schedule(), sort_keys=True, indent=2) + "\n")
    plan = {"schema": "pluto.file02.r06.plan.v1", "status": "LOCKED", "case_id": "R06",
            "source_commit": r01.git("rev-parse", "HEAD"), "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
            "created_utc": datetime.now(timezone.utc).isoformat(), "host": r01.cpu_inventory(),
            "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
            "module_sha256": common.digest(ROOT / "go.mod"),
            "safety_config_sha256": common.digest(ROOT / "research/config/s06_pilot.json"),
            "schedule_sha256": common.digest(raw / "schedule.json"),
            "protocol": "research/protocol/FILE02_R06_LOCKED_PLAN.md",
            "profiles": list(PROFILES), "wire_bytes": WIRE, "chain_id": "pluto-file02-rq1",
            "n_blocks": 12, "n_networks": 48, "offered_valid_tx_s": 20,
            "warmup_seconds": 1, "steady_seconds": 12, "drain_seconds": 5, "cooldown_seconds": 2,
            "primary": "validator0_valid_steady_CheckTx_mean_service_ns",
            "contrast": "H_minus_CP", "ci": "paired_percentile_bootstrap_20000_seed_20261113",
            "knee_status": "NOT_OBSERVED_WITHIN_SAFE_RANGE"}
    r01.write_new(session / "plan.json", plan)
    return plan


def validate_plan(session, plan):
    require(plan.get("schema") == "pluto.file02.r06.plan.v1" and plan.get("status") == "LOCKED" and
            plan.get("case_id") == "R06" and plan.get("source_tree") == r01.git("rev-parse", plan["source_commit"] + "^{tree}"),
            "plan source/status")
    require(plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("safety_config_sha256") == common.digest(ROOT / "research/config/s06_pilot.json") and
            plan.get("host") == r01.cpu_inventory() and plan.get("go_version") == subprocess.check_output(["go", "version"], text=True).strip(),
            "runtime drift")
    require(plan.get("schedule_sha256") == common.digest(session / "raw/schedule.json") and
            common.read_json(session / "raw/schedule.json") == schedule(), "schedule drift")
    require(plan.get("n_blocks") == 12 and plan.get("n_networks") == 48 and
            plan.get("offered_valid_tx_s") == 20 and plan.get("steady_seconds") == 12 and
            plan.get("wire_bytes") == WIRE and plan.get("knee_status") == "NOT_OBSERVED_WITHIN_SAFE_RANGE",
            "design drift")


def samples(path):
    return [json.loads(line) for line in path.read_text().splitlines() if line.strip()]


def bind_run_plan(run, network_root):
    """Write and validate the COMMON plan after ports exist, before any send."""
    run = Path(run).resolve()
    network_root = Path(network_root).resolve()
    require(run.is_dir() and network_root.is_dir() and not (run / "research_plan/plan.json").exists(),
            "run/network path or previous common plan")
    metadata, load_plan, nodes = (common.read_json(run / name) for name in
                                  ("metadata.json", "plan.json", "nodes.json"))
    require(metadata.get("case_id") == "R06" and metadata.get("scope") == "R_CONFIRMATORY" and
            metadata.get("run_id") == load_plan.get("run_id") and len(nodes) == 4, "R06 identity")
    root = common.read_json(ROOT / "research/config/s06_pilot.json")
    network, safety = root["network"], root["safety"]
    profile = next((p for p in PROFILES if {"C": 1, "CP": 2, "P": 3, "H": 4}[p] == load_plan["profile"]), None)
    require(profile == metadata["profile"] and all(node["rpc"].startswith("http://127.0.0.1:") for node in nodes),
            "profile/endpoint mismatch")
    node_hashes = []
    for index in range(4):
        path = network_root / f"validator-{index}/config/config.toml"
        node_hashes.append(common.digest(path))
    inventory = r01.cpu_inventory()
    host = {"cpu_model": inventory["model"], "cpu_features": inventory["flags"],
            "architecture": inventory["architecture"], "logical_cpus": inventory["logical_cpus"],
            "os": inventory["os"]}
    plan = {"schema": "pluto.file02.plan.v1", "status": "LOCKED", "run_id": load_plan["run_id"],
            "case_id": "R06", "protocol_version": "file02-common-v1", "research_question": "RQ1",
            "hypothesis": "H valid CheckTx service exceeds exact-size CP at equal offered rate",
            "null_hypothesis": "H-minus-CP mean service <= 0",
            "alternative_hypothesis": "H-minus-CP mean service > 0",
            "analysis_plan": {"primary_endpoint": "validator0_valid_steady_CheckTx_mean_service_ns",
                              "independent_unit": "fresh_four_validator_network",
                              "ci_method": "paired_percentile_bootstrap_20000_seed_20261113",
                              "exclusion_rule": "any failed locked-arm QC excludes paired block; no replacement"},
            "raw_capture_plan": ["telemetry", "transaction_results", "validator_blocks_and_apphash",
                                 "loadgen_timeline", "host_metrics"],
            "expected_oracle": "241 valid exact-once commits and four-validator AppHash agreement",
            "source_commit": metadata["source_commit"], "source_tree": metadata["source_tree"],
            "config_sha256": metadata["config_sha256"], "corpus_sha256": common.digest(run / "plan.json"),
            "genesis_sha256": load_plan["genesis_sha256"], "profile": profile,
            "workload": "BENIGN_HIGH", "actor_capability": "NONE", "regime": "equal_tx_s",
            "traffic_source": "RPC", "topology": "FOUR_VALIDATOR_SINGLE_HOST_DIAGNOSTIC",
            "policy": "BASELINE_VERIFY_ORDER", "host_inventory": host,
            "software_versions": {"go": subprocess.check_output(["go", "version"], text=True).strip(),
                                  "cometbft": network["comet_version"],
                                  "crypto": "go.mod:" + common.digest(ROOT / "go.mod"),
                                  "os": platform.platform()},
            "network_config": {"validator_count": 4, "mempool_type": network["mempool_type"],
                               "mempool_size": network["mempool_size"], "mempool_max_bytes": network["mempool_max_bytes"],
                               "mempool_cache_size": network["mempool_cache_size"],
                               "mempool_max_tx_bytes": network["mempool_max_tx_bytes"],
                               "mempool_recheck": network["mempool_recheck"],
                               "mempool_broadcast": network["mempool_broadcast"],
                               "max_block_bytes": network["max_block_bytes"],
                               "empty_block_interval_ms": network["empty_block_interval_ms"],
                               "timeout_commit_ms": network["timeout_commit_ms"],
                               "network_shape": "four_validator_single_host_full_mesh",
                               "endpoint_hosts": ["127.0.0.1"]*4,
                               "rpc_endpoints": [node["rpc"] for node in nodes],
                               "config_toml_sha256": node_hashes},
            "safety_config": safety, "seed": load_plan["seed"], "order_index": metadata["order"],
            "phase_seconds": {"warmup": 1, "steady": 12, "drain": 5, "cooldown": 2},
            "planned": {"valid_tx_s": 20, "invalid_tx_s": 0,
                        "total_bytes_s": WIRE[profile]*20},
            "matching": {"matched_variable": "tx_s", "unmatched_variables": ["wire_bytes", "verification_path"]},
            "pilot_reference": "research/evidence/file02/T06_KNEE_PILOT.md",
            "independent_run_target": 12,
            "p99_precision_rule": "rank>=0.95 and two_stage_cluster_CI_radius<=0.20; 2000 draws"}
    common.validate_plan(plan)
    target = run / "research_plan"
    target.mkdir(mode=0o700)
    r01.write_new(target / "plan.json", plan)
    subprocess.run(["python3", str(ROOT / "research/tools/file02_common.py"), "validate-plan",
                    str(target / "plan.json")], cwd=ROOT, check=True, capture_output=True)
    print(f"R06_COMMON_PLAN_LOCKED={load_plan['run_id']}")


def observer_metrics(path):
    rows = samples(path)
    require(rows and all(r.get("kind") == "node_sample" and r.get("dropped_samples") == 0 for r in rows),
            "observer missing/drop")
    result = {"sample_count": len(rows)}
    for key in ("accepted_mempool_bytes", "accepted_mempool_count"):
        values = [r[key] for r in rows if isinstance(r.get(key), int)]
        result["max_" + key] = max(values) if values else None
    for process in ("application_process", "comet_process"):
        cpu = [r[process]["cpu_ns"] for r in rows if isinstance(r.get(process), dict) and isinstance(r[process].get("cpu_ns"), int)]
        rss = [r[process]["rss_bytes"] for r in rows if isinstance(r.get(process), dict) and isinstance(r[process].get("rss_bytes"), int)]
        result[process + "_cpu_delta_ns"] = max(cpu) - min(cpu) if len(cpu) >= 2 else None
        result[process + "_rss_peak_bytes"] = max(rss) if rss else None
    for key in ("relay_receive_envelope_bytes", "relay_send_envelope_bytes"):
        values = [r[key] for r in rows if isinstance(r.get(key), int)]
        result[key + "_delta"] = max(values)-min(values) if len(values) >= 2 else None
    return result


def audit_run(raw, job, plan):
    run = raw / "runs" / job["run_id"]
    meta, workload, result, oracle, commits = (common.read_json(run / f) for f in
        ("metadata.json", "plan.json", "result.json", "oracle.json", "commit_observations.json"))
    common_plan = common.read_json(run / "research_plan/plan.json")
    common.validate_plan(common_plan)
    nodes = common.read_json(run / "nodes.json")
    require(common_plan["run_id"] == job["run_id"] and common_plan["case_id"] == "R06" and
            common_plan["source_commit"] == plan["source_commit"] and
            common_plan["source_tree"] == plan["source_tree"] and
            common_plan["config_sha256"] == plan["safety_config_sha256"] and
            common_plan["corpus_sha256"] == common.digest(run / "plan.json") and
            common_plan["genesis_sha256"] == common.digest(run / "genesis.bin") and
            common_plan["network_config"]["rpc_endpoints"] == [node["rpc"] for node in nodes] and
            common_plan["seed"] == job["seed"] and common_plan["order_index"] == job["order"] and
            common_plan["planned"]["total_bytes_s"] == WIRE[job["profile"]]*20,
            "common plan provenance/endpoint mismatch")
    require(meta.get("schema") == "pluto.file02.r06_run.v1" and meta.get("scope") == "R_CONFIRMATORY" and
            meta.get("case_id") == "R06" and meta.get("recorder_mode") == "ON" and
            meta.get("commit_observer_mode") == "block_scan" and
            meta.get("source_commit") == plan["source_commit"] and meta.get("source_tree") == plan["source_tree"] and
            all(meta.get(k) == job[k] for k in ("block", "order", "profile", "seed")), "metadata/source mismatch")
    require(meta["run_id"] == workload["run_id"] == result["run_id"] == oracle["run_id"] == commits["run_id"] == job["run_id"] and
            result["plan_sha256"] == common.digest(run / "plan.json") and
            workload["genesis_sha256"] == result["genesis_sha256"] == common.digest(run / "genesis.bin") and
            workload["config_sha256"] == result["config_sha256"] == plan["safety_config_sha256"],
            "run binding")
    require(workload["seed"] == job["seed"] and workload["regime"] == "equal_tx_s" and
            workload["target_tx_per_second"] == 20 and workload["warmup_ns"] == 1_000_000_000 and
            workload["steady_ns"] == 12_000_000_000 and workload["drain_ns"] == 5_000_000_000 and
            result["status"] == oracle["status"] == "OK" and
            result["planned_count"] == result["actual_send_count"] == oracle["valid_committed"] == 241 and
            oracle["invalid_rejected"] == 0 and oracle["block_id"] and oracle["app_hash"] and
            len(commits["events"]) == 241, "load/oracle failure")
    observed = samples(run / "recorder-0.jsonl")
    checktx = {}
    for event in observed:
        if event.get("check_type") == "new" and event.get("tx_sha256"):
            require(event["tx_sha256"] not in checktx, "duplicate primary CheckTx hash")
            checktx[event["tx_sha256"]] = event
    by_commit = {event["id"]: event for event in commits["events"]}
    require(len(by_commit) == 241, "duplicate inclusion observation")
    windows = [0]*12
    service, rpc, inclusion = [], [], []
    hashes = set()
    for expected, actual in zip(workload["events"], result["events"]):
        h = hashlib.sha256(bytes.fromhex(expected["bytes_hex"])).hexdigest()
        require(h == expected["sha256"] and h not in hashes and expected["id"] == actual["id"] and
                actual["actual_code"] == expected["expected_code"] == 0 and
                actual["wire_bytes"] == WIRE[job["profile"]] and not actual.get("submission_error") and
                by_commit[expected["id"]]["sha256"] == h, "transaction identity/code/size")
        hashes.add(h)
        sent = actual["actual_offset_ns"]
        require(isinstance(sent, int) and 0 <= sent < 13_000_000_000 and
                (sent < 1_000_000_000) == (expected["offset_ns"] < 1_000_000_000), "phase spill")
        if expected["offset_ns"] >= 1_000_000_000:
            window = (sent-1_000_000_000)//1_000_000_000
            require(0 <= window < 12, "window overflow")
            windows[window] += 1
            event = checktx.get(h)
            require(event is not None and event.get("code") == 0 and
                    event.get("profile") == workload["profile"] and event.get("wire_bytes") == WIRE[job["profile"]] and
                    isinstance(event.get("service_ns"), int) and event["service_ns"] > 0, "missing primary span")
            service.append(event["service_ns"])
            rpc.append((precision.parse_go_rfc3339(actual["completed_utc"])-precision.parse_go_rfc3339(actual["actual_send_utc"])).total_seconds()*1000)
            inclusion.append((precision.parse_go_rfc3339(by_commit[expected["id"]]["all_four_seen_utc"])-precision.parse_go_rfc3339(actual["actual_send_utc"])).total_seconds()*1000)
    require(len(hashes) == 241 and len(service) == 240 and windows == [20]*12, "steady rate/sample count")
    require(all(x >= 0 for x in rpc + inclusion), "negative latency")
    obs = [observer_metrics(run / f"observer-{i}.jsonl") for i in range(4)]
    return {"run_id": job["run_id"], "block": job["block"], "profile": job["profile"],
            "order": job["order"], "seed": job["seed"], "genesis_sha256": workload["genesis_sha256"],
            "wire_bytes": WIRE[job["profile"]], "achieved_tx_s_windows": windows,
            "achieved_bytes_s_windows": [n*WIRE[job["profile"]] for n in windows],
            "valid_committed": 241, "mean_service_ns": statistics.mean(service),
            "service_ns_samples": service, "rpc_ms_samples": rpc,
            "inclusion_upper_bound_ms_samples": inclusion, "observer_validator0": obs[0],
            "observer_all": obs, "qc_status": "PASS"}


def derive(session, plan):
    raw = session / "raw"
    manifest = common.verify_raw(raw)
    jobs = schedule()["jobs"]
    rows, failed = [], []
    for job in jobs:
        try:
            rows.append(audit_run(raw, job, plan))
        except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError) as exc:
            failed.append({"run_id": job["run_id"], "reason": str(exc)})
    by_block = {block: [r for r in rows if r["block"] == block] for block in range(12)}
    for block, group in by_block.items():
        if len(group) == 4:
            geneses = {r["genesis_sha256"] for r in group}
            if len(geneses) != 1:
                failed.append({"block": block, "reason": "logical fixture/genesis not matched"})
    if len({r["genesis_sha256"] for r in rows}) != 12:
        failed.append({"reason": "independent block genesis count mismatch"})
    qc = {"schema": "pluto.file02.r06.qc.v1", "case_id": "R06",
          "status": "PASS_BOUNDED_NETWORK" if len(rows) == 48 and not failed else "INVALID_FOR_INFERENCE",
          "n_planned": 48, "n_valid": len(rows), "failed": failed,
          "under_knee": "NOT_OBSERVED_WITHIN_SAFE_RANGE", "plan_sha256": common.digest(session / "plan.json"),
          "raw_manifest_sha256": manifest}
    aggregate = {"schema": "pluto.file02.r06.aggregate.v1", "case_id": "R06",
                 "status": "PENDING_VALID_QC", "n_networks": len(rows),
                 "under_knee": "NOT_OBSERVED_WITHIN_SAFE_RANGE", "primary_H_minus_CP_ns": None,
                 "ci95_ns": None, "profile_summary": {}, "plan_sha256": qc["plan_sha256"],
                 "raw_manifest_sha256": manifest}
    if qc["status"] == "PASS_BOUNDED_NETWORK":
        deltas = []
        for block in range(12):
            by = {r["profile"]: r for r in by_block[block]}
            deltas.append(by["H"]["mean_service_ns"]-by["CP"]["mean_service_ns"])
        aggregate["status"] = "ANALYZED_BOUNDED_NETWORK"
        aggregate["primary_H_minus_CP_ns"] = statistics.mean(deltas)
        aggregate["ci95_ns"] = r01.paired_ci(deltas)
        aggregate["pair_deltas_ns"] = deltas
        for profile in PROFILES:
            group = [r for r in rows if r["profile"] == profile]
            inclusion_groups = [r["inclusion_upper_bound_ms_samples"] for r in group]
            precision_result = precision.cluster_bootstrap(inclusion_groups, SEED ^ sum(map(ord, profile)))
            aggregate["profile_summary"][profile] = {
                "n_networks": len(group), "mean_service_ns": statistics.mean(r["mean_service_ns"] for r in group),
                "mean_rpc_ms": statistics.mean(statistics.mean(r["rpc_ms_samples"]) for r in group),
                "mean_inclusion_upper_bound_ms": statistics.mean(statistics.mean(r["inclusion_upper_bound_ms_samples"]) for r in group),
                "inclusion_upper_bound_p99_precision": precision_result,
                "offered_valid_tx_s_per_window": 20,
                "offered_bytes_s_per_window": 20 * WIRE[profile],
                "eventual_valid_included_per_network": 241,
                "eventual_goodput_tx_s_steady_window": None,
                "eventual_goodput_reason": "inclusion may occur after steady window; no sustainable throughput inference",
                "mean_max_mempool_bytes": statistics.mean(r["observer_validator0"]["max_accepted_mempool_bytes"] for r in group),
                "mean_shared_app_cpu_delta_ns": statistics.mean(r["observer_validator0"]["application_process_cpu_delta_ns"] for r in group),
                "mean_shared_app_rss_peak_bytes": statistics.mean(r["observer_validator0"]["application_process_rss_peak_bytes"] for r in group),
                "mean_validator0_comet_cpu_delta_ns": statistics.mean(r["observer_validator0"]["comet_process_cpu_delta_ns"] for r in group),
                "mean_validator0_comet_rss_peak_bytes": statistics.mean(r["observer_validator0"]["comet_process_rss_peak_bytes"] for r in group),
                "mean_validator0_relay_send_envelope_bytes_delta": statistics.mean(r["observer_validator0"]["relay_send_envelope_bytes_delta"] for r in group),
                "mean_validator0_relay_receive_envelope_bytes_delta": statistics.mean(r["observer_validator0"]["relay_receive_envelope_bytes_delta"] for r in group)}
    return rows, qc, aggregate


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.r06.per_run.v1", "rows": rows})
    with (session / "result_per_run.csv").open("x", newline="", encoding="utf-8") as stream:
        keys = ("run_id", "block", "profile", "order", "seed", "genesis_sha256", "wire_bytes",
                "valid_committed", "mean_service_ns", "qc_status")
        writer = csv.DictWriter(stream, fieldnames=keys)
        writer.writeheader()
        writer.writerows({key: row[key] for key in keys} for row in rows)
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir()
    r01.write_new(session / "figures/status.json", {"status": "PENDING_CASE_REVIEW" if qc["status"] == "PASS_BOUNDED_NETWORK" else "NOT_RUN",
                                                     "reason": "No scientific figure is required to establish the registered primary contrast"})
    verdict = "POSITIVE_BOUNDED_H_MINUS_CP" if aggregate["ci95_ns"] and aggregate["ci95_ns"][0] > 0 else "INCONCLUSIVE_OR_NEGATIVE"
    (session / "decision.md").write_text(
        f"# R06 decision\n\nQC: {qc['status']}; {len(rows)}/48 network arms. "
        f"Primary bounded-network decision: {verdict if qc['status'] == 'PASS_BOUNDED_NETWORK' else 'WITHHELD'}. "
        "The R06 under-knee requirement is NOT_OBSERVED_WITHIN_SAFE_RANGE, so R06 remains partial. "
        "No DoS, sustainable throughput, peer propagation or multi-host claim.\n", encoding="utf-8")
    return qc, aggregate


def execute():
    session = ARTIFACTS / ("r06-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8])
    plan = prepare(session)
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT, stdout=stream, stderr=subprocess.STDOUT)
    require(build.returncode == 0, f"build failed: {session}")
    env = os.environ.copy()
    env.update(FILE02_R06_ROOT=str(raw), FILE02_SOURCE_COMMIT=plan["source_commit"], FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02R06$", "-timeout", "3600s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=3700)
    manifest = common.freeze_raw(raw)
    qc, aggregate = write_outputs(session, plan)
    print(f"R06 session={session} integration_exit={run.returncode} qc={qc['status']} "
          f"effect_ns={aggregate['primary_H_minus_CP_ns']} ci95_ns={aggregate['ci95_ns']} raw_manifest_sha256={manifest}")
    return 0 if run.returncode == 0 and qc["status"] == "PASS_BOUNDED_NETWORK" else 2


def rederive(session):
    """Keep the first derived version byte-for-byte, then extend secondary metrics."""
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve(), "private session")
    require(common.verify_raw(session / "raw"), "raw invalid")
    archive = session / "analysis-v1"
    require(not archive.exists(), "analysis-v1 already archived")
    archive.mkdir(mode=0o700)
    names = ("result_per_run.json", "result_per_run.csv", "qc_report.json", "result_aggregate.json", "decision.md")
    for name in names:
        shutil.copy2(session / name, archive / name)
    shutil.copytree(session / "figures", archive / "figures")
    common.freeze_raw(archive)
    for name in names:
        (session / name).unlink()
    shutil.rmtree(session / "figures")
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    qc, aggregate = write_outputs(session, plan)
    print(f"R06_REDERIVE=OK qc={qc['status']} effect_ns={aggregate['primary_H_minus_CP_ns']} "
          f"raw_manifest_sha256={qc['raw_manifest_sha256']}")


def verify(session):
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve(), "private session")
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    rows, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") == {"schema": "pluto.file02.r06.per_run.v1", "rows": rows} and
            common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate,
            "derived artifact mismatch")
    with (session / "result_per_run.csv").open(newline="", encoding="utf-8") as stream:
        csv_rows = list(csv.DictReader(stream))
    require(len(csv_rows) == len(rows) and all(str(row["mean_service_ns"]) == entry["mean_service_ns"] and
            row["run_id"] == entry["run_id"] for row, entry in zip(rows, csv_rows)), "CSV mismatch")
    print(f"R06_VERIFY=OK qc={qc['status']} n={len(rows)} raw_manifest_sha256={qc['raw_manifest_sha256']}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("run")
    command = sub.add_parser("bind-run-plan")
    command.add_argument("run_dir")
    command.add_argument("network_root")
    command = sub.add_parser("verify")
    command.add_argument("session")
    command = sub.add_parser("rederive")
    command.add_argument("session")
    args = parser.parse_args()
    try:
        sys.exit((bind_run_plan(args.run_dir, args.network_root) or 0) if args.command == "bind-run-plan" else
                 execute() if args.command == "run" else
                 (rederive(args.session) or 0) if args.command == "rederive" else (verify(args.session) or 0))
    except (ValueError, OSError, KeyError, TypeError, subprocess.TimeoutExpired) as exc:
        print(f"R06_ERROR: {exc}", file=sys.stderr)
        sys.exit(2)
