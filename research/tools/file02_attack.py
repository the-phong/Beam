#!/usr/bin/env python3
"""Locked, raw-bound A01–A05 single-host matched attack experiment."""

import argparse
import csv
from datetime import datetime, timezone
import hashlib
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
import file02_r06 as r06

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
CASES = ("A01", "A02", "A03", "A04", "A05")
CASE_WORKLOAD = {"A01": "STRUCT_INVALID", "A02": "MIXED", "A03": "ECDSA_INVALID",
                 "A04": "MLDSA_INVALID", "A05": "STATE_INVALID"}
CASE_CAPABILITY = {"A01": "A", "A02": "A", "A03": "B", "A04": "B", "A05": "C"}
CASE_CODE_STAGE = {"A01": (1, "structural"), "A02": (11, "binding"),
                   "A03": (4, "secp"), "A04": (13, "ml"), "A05": (6, "state")}
WIRE_BYTES = 2568
SEED = 20261113


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def schedule(case):
    require(case in CASES, "unknown attack case")
    rng = random.Random(SEED ^ int(case[1:]))
    order = ["benign", "attack"] * 6
    rng.shuffle(order)
    jobs = []
    for pair in range(12):
        for position, condition in enumerate((order[pair], "attack" if order[pair] == "benign" else "benign")):
            jobs.append({"pair": pair, "order": position, "condition": condition,
                         "run_id": f"{case}-p{pair:02d}-{condition}"})
    return {"schema": "pluto.file02.attack_schedule.v1", "case_id": case,
            "master_seed": SEED, "jobs": jobs}


def session_path(path):
    path = Path(path).resolve()
    require(path.parent == ARTIFACTS.resolve() and path.is_dir(), "private session path")
    return path


def prepare(session, case):
    require(case in CASES and session.parent.resolve() == ARTIFACTS.resolve() and
            not r01.git("status", "--porcelain"), "source must be clean before lock")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT).returncode == 0,
            "session not ignored")
    session.mkdir(mode=0o700)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    common.write_once(raw / "schedule.json", json.dumps(schedule(case), sort_keys=True, indent=2) + "\n")
    common.write_once(raw / "config.json", (ROOT / "research/config/s06_pilot.json").read_text())
    plan = {"schema": "pluto.file02.attack_plan.v1", "status": "LOCKED", "case_id": case,
            "source_commit": r01.git("rev-parse", "HEAD"), "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
            "created_utc": datetime.now(timezone.utc).isoformat(), "host": r01.cpu_inventory(),
            "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
            "module_sha256": common.digest(ROOT / "go.mod"),
            "config_sha256": common.digest(raw / "config.json"),
            "schedule_sha256": common.digest(raw / "schedule.json"),
            "protocol": f"research/protocol/FILE02_{case}_LOCKED_PLAN.md",
            "profile": "H", "chain_id": "pluto-file02-" + case, "wire_bytes": WIRE_BYTES,
            "n_pairs": 12, "n_networks": 24, "fixed_legitimate_tx_s": 6,
            "background_tx_s": 14, "total_tx_s": 20, "total_bytes_s": WIRE_BYTES * 20,
            "warmup_seconds": 1, "steady_seconds": 12, "drain_seconds": 5,
            "cooldown_seconds": 2, "deadline_ms": 3000, "block_scan_interval_ms": 100,
            "ci": "paired_percentile_bootstrap_20000_seed_20261113",
            "decision_rule": "positive_only_if_paired_ci_lower_gt_zero_and_all_qc_pass"}
    r01.write_new(session / "plan.json", plan)
    return plan


def validate_plan(session, plan):
    case = plan.get("case_id")
    require(case in CASES and plan.get("schema") == "pluto.file02.attack_plan.v1" and
            plan.get("status") == "LOCKED" and plan.get("source_tree") ==
            r01.git("rev-parse", plan["source_commit"] + "^{tree}"), "attack source/status")
    require(plan.get("host") == r01.cpu_inventory() and plan.get("go_version") ==
            subprocess.check_output(["go", "version"], text=True).strip() and
            plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("config_sha256") == common.digest(session / "raw/config.json") ==
            common.digest(ROOT / "research/config/s06_pilot.json") and
            plan.get("schedule_sha256") == common.digest(session / "raw/schedule.json") and
            common.read_json(session / "raw/schedule.json") == schedule(case), "attack plan drift")
    require(plan.get("n_pairs") == 12 and plan.get("n_networks") == 24 and
            plan.get("fixed_legitimate_tx_s") == 6 and plan.get("background_tx_s") == 14 and
            plan.get("total_bytes_s") == 51360 and plan.get("deadline_ms") == 3000 and
            plan.get("ci") == "paired_percentile_bootstrap_20000_seed_20261113", "attack design drift")


def bind_run_plan(run, network_root):
    run, network_root = Path(run).resolve(), Path(network_root).resolve()
    require(run.is_dir() and network_root.is_dir() and not (run / "research_plan/plan.json").exists(),
            "run/network path or prior plan")
    meta, load_plan, nodes = (common.read_json(run / name) for name in
                              ("metadata.json", "plan.json", "nodes.json"))
    case, condition = meta.get("case_id"), meta.get("condition")
    require(case in CASES and condition in ("benign", "attack") and len(nodes) == 4 and
            meta.get("scope") == "R_CONFIRMATORY" and meta.get("run_id") == load_plan.get("run_id") and
            all(n["rpc"].startswith("http://127.0.0.1:") for n in nodes), "attack run binding")
    config_path = run / "config.json"
    config = common.read_json(config_path)
    require(common.digest(config_path) == meta["config_sha256"] ==
            common.digest(ROOT / "research/config/s06_pilot.json"), "safety/config mismatch")
    network, safety = config["network"], config["safety"]
    require(load_plan["profile"] == 4 and len(load_plan["events"]) == 241 and
            load_plan["warmup_ns"] == 1_000_000_000 and load_plan["steady_ns"] == 12_000_000_000 and
            load_plan["drain_ns"] == 5_000_000_000 and load_plan["seed"] == meta["seed"] and
            load_plan["regime"] == ("equal_tx_s" if condition == "benign" else
                                    "fixed_legitimate_plus_adversarial"), "attack schedule mismatch")
    inventory = r01.cpu_inventory()
    host = {"cpu_model": inventory["model"], "cpu_features": inventory["flags"],
            "architecture": inventory["architecture"], "logical_cpus": inventory["logical_cpus"],
            "os": inventory["os"]}
    node_hashes = [common.digest(network_root / f"validator-{i}/config/config.toml") for i in range(4)]
    research_plan = {"schema": "pluto.file02.plan.v1", "status": "LOCKED", "run_id": load_plan["run_id"],
        "case_id": case, "protocol_version": "file02-common-v1", "research_question": "RQ1b",
        "hypothesis": "invalid background lowers timely fixed legitimate inclusion",
        "null_hypothesis": "benign-minus-attack timely fraction <= 0",
        "alternative_hypothesis": "benign-minus-attack timely fraction > 0",
        "analysis_plan": {"primary_endpoint": "fixed_legitimate_all_four_observed_within_3000ms_fraction",
                          "independent_unit": "fresh_four_validator_network",
                          "ci_method": "paired_percentile_bootstrap_20000_seed_20261113",
                          "exclusion_rule": "failed arm excludes pair; no replacement"},
        "raw_capture_plan": ["telemetry", "transaction_results", "validator_blocks_and_apphash",
                             "loadgen_timeline", "host_metrics"],
        "expected_oracle": "fixed valid exact once, invalid structural/rejection code, four-validator agreement",
        "source_commit": meta["source_commit"], "source_tree": meta["source_tree"],
        "config_sha256": meta["config_sha256"], "corpus_sha256": common.digest(run / "plan.json"),
        "genesis_sha256": load_plan["genesis_sha256"], "profile": "H",
        "workload": "BENIGN_HIGH" if condition == "benign" else CASE_WORKLOAD[case],
        "actor_capability": "NONE" if condition == "benign" else CASE_CAPABILITY[case],
        "regime": load_plan["regime"], "traffic_source": "RPC",
        "topology": "FOUR_VALIDATOR_SINGLE_HOST_DIAGNOSTIC", "policy": "BASELINE_VERIFY_ORDER",
        "host_inventory": host,
        "software_versions": {"go": subprocess.check_output(["go", "version"], text=True).strip(),
                              "cometbft": network["comet_version"],
                              "crypto": "go.mod:" + common.digest(ROOT / "go.mod"), "os": platform.platform()},
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
            "rpc_endpoints": [n["rpc"] for n in nodes], "config_toml_sha256": node_hashes},
        "safety_config": safety, "seed": load_plan["seed"], "order_index": meta["order"],
        "phase_seconds": {"warmup": 1, "steady": 12, "drain": 5, "cooldown": 2},
        "planned": {"valid_tx_s": 20 if condition == "benign" else 6,
                    "invalid_tx_s": 0 if condition == "benign" else 14,
                    "total_bytes_s": WIRE_BYTES*20},
        "matching": {"matched_variable": "bytes_s", "unmatched_variables":
                     ["valid_background_inclusion", "verification_stage"],
                     "legitimate_stream_id": f"{case}-p{meta['pair']:02d}-fixed"},
        "pilot_reference": "research/evidence/file02/T06_FORMAL_CALIBRATION_AUDIT.md",
        "independent_run_target": 12,
        "p99_precision_rule": "rank>=0.95 and two_stage_cluster_CI_radius<=0.20; 2000 draws"}
    common.validate_plan(research_plan)
    directory = run / "research_plan"
    directory.mkdir(mode=0o700)
    r01.write_new(directory / "plan.json", research_plan)
    subprocess.run(["python3", str(ROOT / "research/tools/file02_common.py"), "validate-plan",
                    str(directory / "plan.json")], cwd=ROOT, check=True, capture_output=True)


def audit_run(raw, job, plan):
    case, condition = plan["case_id"], job["condition"]
    run = raw / "runs" / job["run_id"]
    meta, workload, result, oracle, commits, bound = (common.read_json(run / name) for name in
        ("metadata.json", "plan.json", "result.json", "oracle.json",
         "commit_observations.json", "research_plan/plan.json"))
    common.validate_plan(bound)
    require(meta.get("schema") == "pluto.file02.attack_run.v1" and meta.get("scope") == "R_CONFIRMATORY" and
            all(meta.get(key) == job[key] for key in ("run_id", "pair", "order", "condition")) and
            meta.get("case_id") == case and meta.get("profile") == "H" and
            meta.get("recorder_mode") == "ON" and meta.get("commit_observer_mode") == "block_scan" and
            meta.get("source_commit") == bound["source_commit"] == plan["source_commit"] and
            meta.get("source_tree") == bound["source_tree"] == plan["source_tree"], "attack metadata")
    require(bound["run_id"] == workload["run_id"] == result["run_id"] == oracle["run_id"] ==
            commits["run_id"] == job["run_id"] and bound["case_id"] == case and
            bound["config_sha256"] == meta["config_sha256"] == workload["config_sha256"] ==
            result["config_sha256"] == common.digest(run / "config.json") == plan["config_sha256"] and
            bound["corpus_sha256"] == result["plan_sha256"] == common.digest(run / "plan.json") and
            bound["genesis_sha256"] == workload["genesis_sha256"] == result["genesis_sha256"] ==
            common.digest(run / "genesis.bin") and bound["network_config"]["rpc_endpoints"] ==
            [node["rpc"] for node in common.read_json(run / "nodes.json")], "attack provenance")
    require(workload["profile"] == 4 and workload["seed"] == 2026111300 + job["pair"] and
            workload["warmup_ns"] == 1_000_000_000 and workload["steady_ns"] == 12_000_000_000 and
            workload["drain_ns"] == 5_000_000_000 and workload["window_ns"] == 1_000_000_000 and
            workload["regime"] == ("equal_tx_s" if condition == "benign" else
                                    "fixed_legitimate_plus_adversarial") and
            result["status"] == oracle["status"] == "OK" and
            result["planned_count"] == result["actual_send_count"] == len(workload["events"]) ==
            len(result["events"]) == 241 and
            oracle["valid_committed"] == (241 if condition == "benign" else 73) and
            oracle["invalid_rejected"] == (0 if condition == "benign" else 168) and
            oracle["block_id"] and oracle["app_hash"], "attack execution/oracle")
    expected_commit_count = oracle["valid_committed"]
    by_commit = {item["id"]: item for item in commits["events"]}
    require(len(by_commit) == len(commits["events"]) == expected_commit_count, "commit observation identity")
    checktx = {}
    for event in r06.samples(run / "recorder-0.jsonl"):
        if event.get("check_type") == "new" and event.get("tx_sha256"):
            require(event["tx_sha256"] not in checktx, "duplicate CheckTx new hash")
            checktx[event["tx_sha256"]] = event
    fixed_hashes, all_hashes = [], set()
    windows = [{"fixed": 0, "background": 0, "bytes": 0} for _ in range(12)]
    fixed_latency_ms, invalid_service_ns, invalid_structural_ns = [], [], []
    invalid_stage_spans = {stage: [] for stage in ("structural", "binding", "secp", "ml", "state")}
    invalid_verify_reached = invalid_ml_reached = 0
    invalid_variants = {}
    for planned, actual in zip(workload["events"], result["events"]):
        wire = bytes.fromhex(planned["bytes_hex"])
        h = hashlib.sha256(wire).hexdigest()
        require(h == planned["sha256"] and h not in all_hashes and
                planned["id"] == actual["id"] and actual["actual_code"] == planned["expected_code"] and
                actual["wire_bytes"] == len(wire) == WIRE_BYTES and not actual.get("submission_error") and
                isinstance(actual.get("actual_send_monotonic_ns"), int) and
                actual["actual_send_monotonic_ns"] > 0, "attack transaction send/code/size")
        all_hashes.add(h)
        sent = actual["actual_offset_ns"]
        require(isinstance(sent, int) and 0 <= sent < 13_000_000_000 and
                (sent < 1_000_000_000) == (planned["offset_ns"] < 1_000_000_000), "phase crossing")
        record = checktx.get(h)
        require(record is not None and record.get("code") == planned["expected_code"] and
                record.get("last_stage") == planned["expected_stage"] and
                record.get("wire_bytes") == WIRE_BYTES and record.get("profile") == 4 and
                isinstance(record.get("service_ns"), int) and record["service_ns"] > 0,
                "CheckTx stage/hash missing")
        if planned["traffic"] == "adversarial":
            require(condition == "attack" and
                    (planned["expected_code"], planned["expected_stage"]) == CASE_CODE_STAGE[case] and
                    planned["id"] not in by_commit, "invalid oracle/commit")
            invalid_variants[planned["scenario"]] = invalid_variants.get(planned["scenario"], 0) + 1
            if planned["offset_ns"] >= 1_000_000_000:
                invalid_service_ns.append(record["service_ns"])
                spans = record.get("spans", [])
                span_stages = {span.get("stage") for span in spans}
                required = {"A01": {"structural"}, "A02": {"structural", "binding"},
                            "A03": {"structural", "binding", "secp"},
                            "A04": {"structural", "binding", "secp", "ml"},
                            "A05": {"structural", "binding", "secp", "ml", "state"}}[case]
                require(required <= span_stages and
                        ("secp" in span_stages) == (case in ("A03", "A04", "A05")) and
                        ("ml" in span_stages) == (case in ("A04", "A05")) and
                        ("state" in span_stages) == (case == "A05"),
                        "invalid verification-stage oracle")
                invalid_verify_reached += int("secp" in span_stages)
                invalid_ml_reached += int("ml" in span_stages)
                for span in spans:
                    if span.get("stage") in invalid_stage_spans:
                        invalid_stage_spans[span["stage"]].append(span["duration_ns"])
                if case == "A01":
                    require(not any(span.get("stage") in ("secp", "ml") for span in spans),
                            "structural reject reached crypto")
                    invalid_structural_ns.extend(span["duration_ns"] for span in spans
                                                 if span.get("stage") == "structural")
        else:
            require(planned["id"] in by_commit and by_commit[planned["id"]]["sha256"] == h,
                    "valid commit observation")
        if planned["scenario"] == "fixed_legitimate":
            fixed_hashes.append(h)
            if planned["offset_ns"] >= 1_000_000_000:
                seen = by_commit[planned["id"]].get("all_four_seen_monotonic_ns")
                require(isinstance(seen, int) and seen >= actual["actual_send_monotonic_ns"],
                        "missing/negative same-host monotonic inclusion bound")
                fixed_latency_ms.append((seen - actual["actual_send_monotonic_ns"]) / 1_000_000)
        if planned["offset_ns"] >= 1_000_000_000:
            window = (sent - 1_000_000_000) // 1_000_000_000
            require(0 <= window < 12, "steady window overflow")
            windows[window]["fixed" if planned["scenario"] == "fixed_legitimate" else "background"] += 1
            windows[window]["bytes"] += len(wire)
    require(len(all_hashes) == 241 and len(fixed_hashes) == 73 and len(fixed_latency_ms) == 72 and
            all(w == {"fixed": 6, "background": 14, "bytes": 51360} for w in windows),
            "fixed stream/achieved rate")
    if condition == "attack":
        require(len(invalid_service_ns) == 168 and
                (case != "A01" or all(invalid_variants.get(v, 0) > 0 for v in
                    ("A01_wrong_magic", "A01_wrong_declared_length", "A01_bad_chain_encoding"))),
                "attack distinct class coverage")
    else:
        require(not invalid_service_ns and not invalid_variants, "benign arm has invalid traffic")
    observer = [r06.observer_metrics(run / f"observer-{i}.jsonl") for i in range(4)]
    row = {"run_id": job["run_id"], "pair": job["pair"], "order": job["order"],
            "condition": condition, "profile": "H", "seed": workload["seed"],
            "genesis_sha256": workload["genesis_sha256"], "fixed_hashes": fixed_hashes,
            "windows": windows, "valid_committed": oracle["valid_committed"],
            "invalid_rejected": oracle["invalid_rejected"], "invalid_variants": invalid_variants,
            "fixed_timely_count": sum(x <= plan["deadline_ms"] for x in fixed_latency_ms),
            "fixed_eligible_count": 72,
            "fixed_timely_fraction": sum(x <= plan["deadline_ms"] for x in fixed_latency_ms) / 72,
            "fixed_inclusion_upper_ms": fixed_latency_ms,
            "invalid_mean_checktx_service_ns": statistics.mean(invalid_service_ns) if invalid_service_ns else None,
            "invalid_mean_structural_span_ns": statistics.mean(invalid_structural_ns) if invalid_structural_ns else None,
            "observer_validator0": observer[0], "qc_status": "PASS"}
    if case != "A01":
        row["invalid_verify_reach_fraction"] = invalid_verify_reached / 168 if condition == "attack" else None
        row["invalid_ml_reach_fraction"] = invalid_ml_reached / 168 if condition == "attack" else None
        row["invalid_mean_stage_wall_ns"] = {
            stage: statistics.mean(samples) if samples else None
            for stage, samples in invalid_stage_spans.items()}
    return row


def derive(session, plan):
    raw = session / "raw"
    manifest = common.verify_raw(raw)
    rows, failed = [], []
    for job in schedule(plan["case_id"])["jobs"]:
        try:
            rows.append(audit_run(raw, job, plan))
        except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError) as exc:
            failed.append({"run_id": job["run_id"], "reason": str(exc)})
    pair_deltas = []
    for pair in range(12):
        group = [row for row in rows if row["pair"] == pair]
        if len(group) != 2:
            continue
        by = {row["condition"]: row for row in group}
        if (set(by) != {"benign", "attack"} or
                by["benign"]["genesis_sha256"] != by["attack"]["genesis_sha256"] or
                by["benign"]["fixed_hashes"] != by["attack"]["fixed_hashes"]):
            failed.append({"pair": pair, "reason": "fixed legitimate stream/genesis mismatch"})
            continue
        pair_deltas.append(by["benign"]["fixed_timely_fraction"] - by["attack"]["fixed_timely_fraction"])
    qc_status = ("PASS" if len(rows) == 24 and len(pair_deltas) == 12 and not failed and
                 len({row["genesis_sha256"] for row in rows}) == 12 else "INVALID_FOR_INFERENCE")
    qc = {"schema": "pluto.file02.attack_qc.v1", "case_id": plan["case_id"],
          "status": qc_status, "n_planned": 24, "n_valid": len(rows), "failed": failed,
          "raw_manifest_sha256": manifest, "plan_sha256": common.digest(session / "plan.json")}
    ci = r01.paired_ci(pair_deltas) if qc_status == "PASS" else None
    decision = ("POSITIVE_BOUNDED_SERVICE_DEGRADATION" if ci and ci[0] > 0 else
                "NO_POSITIVE_EFFECT_IN_TESTED_STRATUM" if ci and ci[1] <= 0 else
                "INCONCLUSIVE" if ci else "WITHHELD_QC_FAILURE")
    aggregate = {"schema": "pluto.file02.attack_aggregate.v1", "case_id": plan["case_id"],
                 "status": "ANALYZED" if qc_status == "PASS" else "WITHHELD_QC_FAILURE",
                 "n_pairs": len(pair_deltas), "primary_benign_minus_attack_fraction":
                 statistics.mean(pair_deltas) if ci else None, "ci95_fraction": ci,
                 "pair_deltas": pair_deltas if ci else None, "decision": decision,
                 "raw_manifest_sha256": manifest, "plan_sha256": qc["plan_sha256"],
                 "invalid_mean_service_ns": statistics.mean(row["invalid_mean_checktx_service_ns"]
                    for row in rows if row["condition"] == "attack") if qc_status == "PASS" else None,
                 "shared_app_cpu_status": "UNAVAILABLE_SEPARATE_FROM_INTEGRATION_HARNESS"}
    return rows, qc, aggregate


def csv_text(rows):
    from io import StringIO
    buffer = StringIO()
    keys = ("run_id", "pair", "order", "condition", "seed", "genesis_sha256",
            "valid_committed", "invalid_rejected", "fixed_timely_count", "fixed_eligible_count",
            "fixed_timely_fraction", "invalid_mean_checktx_service_ns", "qc_status")
    writer = csv.DictWriter(buffer, fieldnames=keys, lineterminator="\n")
    writer.writeheader()
    writer.writerows({key: row[key] for key in keys} for row in rows)
    return buffer.getvalue()


def decision_text(qc, aggregate):
    return (f"# {qc['case_id']} bounded decision\n\nQC: {qc['status']}; "
            f"{qc['n_valid']}/24 network arms. Primary decision: {aggregate['decision']}. "
            "H-minus-CP RQ1 is unchanged. This is a same-host RPC-origin RQ1b workload contrast; "
            "no DoS, knee, sustainable throughput, P2P-origin or multi-host conclusion.\n")


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.attack_per_run.v1", "rows": rows})
    common.write_once(session / "result_per_run.csv", csv_text(rows))
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir(mode=0o700)
    r01.write_new(session / "figures/status.json", {"status": "PENDING_CASE_REVIEW",
        "reason": "registered numeric primary contrast is recorded without a scientific figure"})
    common.write_once(session / "decision.md", decision_text(qc, aggregate))
    return qc, aggregate


def execute(case):
    require(case in CASES and (ROOT / f"research/protocol/FILE02_{case}_LOCKED_PLAN.md").is_file(),
            "case plan missing")
    session = ARTIFACTS / (case.lower() + "-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") +
                           "-" + uuid4().hex[:8])
    plan = prepare(session, case)
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT,
                               stdout=stream, stderr=subprocess.STDOUT)
    require(build.returncode == 0, f"build failed; private session {session}")
    env = os.environ.copy()
    env.update(FILE02_A_ROOT=str(raw), FILE02_A_CASE=case,
               FILE02_SOURCE_COMMIT=plan["source_commit"], FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02AttackCase$", "-timeout", "3600s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=3700)
    manifest = common.freeze_raw(raw)
    qc, aggregate = write_outputs(session, plan)
    print(f"ATTACK_CASE={case} session={session} integration_exit={run.returncode} "
          f"qc={qc['status']} effect={aggregate['primary_benign_minus_attack_fraction']} "
          f"ci95={aggregate['ci95_fraction']} raw_manifest_sha256={manifest}")
    return 0 if run.returncode == 0 and qc["status"] == "PASS" else 2


def verify(session):
    session = session_path(session)
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    rows, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.attack_per_run.v1", "rows": rows} and
            (session / "result_per_run.csv").read_text() == csv_text(rows) and
            common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate and
            (session / "decision.md").read_text() == decision_text(qc, aggregate) and
            common.read_json(session / "figures/status.json") ==
            {"status": "PENDING_CASE_REVIEW", "reason":
             "registered numeric primary contrast is recorded without a scientific figure"},
            "derived artifact mismatch")
    print(f"ATTACK_VERIFY=OK case={plan['case_id']} qc={qc['status']} raw_manifest_sha256={qc['raw_manifest_sha256']}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    bind = sub.add_parser("bind-run-plan")
    bind.add_argument("run")
    bind.add_argument("network_root")
    run = sub.add_parser("run")
    run.add_argument("case", choices=CASES)
    check = sub.add_parser("verify")
    check.add_argument("session")
    args = parser.parse_args()
    try:
        if args.command == "bind-run-plan":
            bind_run_plan(args.run, args.network_root)
            return 0
        if args.command == "run":
            return execute(args.case)
        verify(args.session)
        return 0
    except (OSError, ValueError, KeyError, TypeError, subprocess.CalledProcessError) as exc:
        print(f"ATTACK_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
