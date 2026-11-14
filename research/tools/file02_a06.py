#!/usr/bin/env python3
"""Locked A06 valid/invalid replay versus distinct stream experiment."""

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
CONDITIONS = ("valid_distinct", "valid_repeated", "invalid_distinct", "invalid_repeated")
SEED = 20261113


def require(ok, message):
    if not ok:
        raise ValueError(message)


def schedule():
    jobs = []
    for block in range(4):
        for order, condition in enumerate(CONDITIONS[block:] + CONDITIONS[:block]):
            jobs.append({"block": block, "order": order, "condition": condition,
                         "run_id": f"A06-b{block:02d}-{condition}"})
    return {"schema": "pluto.file02.a06.schedule.v1", "seed": SEED,
            "method": "four_rotated_blocks", "jobs": jobs}


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve() and not r01.git("status", "--porcelain"),
            "private session and clean source required")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT).returncode == 0,
            "session not ignored")
    session.mkdir(mode=0o700)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    common.write_once(raw / "schedule.json", json.dumps(schedule(), sort_keys=True, indent=2) + "\n")
    common.write_once(raw / "config.json", (ROOT / "research/config/s06_pilot.json").read_text())
    plan = {"schema": "pluto.file02.a06.plan.v1", "status": "LOCKED", "case_id": "A06",
            "source_commit": r01.git("rev-parse", "HEAD"),
            "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
            "created_utc": datetime.now(timezone.utc).isoformat(), "host": r01.cpu_inventory(),
            "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
            "module_sha256": common.digest(ROOT / "go.mod"),
            "config_sha256": common.digest(raw / "config.json"),
            "schedule_sha256": common.digest(raw / "schedule.json"),
            "protocol": "research/protocol/FILE02_A06_LOCKED_PLAN.md",
            "profile": "H", "chain_id": "pluto-file02-a06", "n_blocks": 4,
            "n_networks": 16, "fixed_legitimate_tx_s": 6, "background_tx_s": 14,
            "total_tx_s": 20, "total_bytes_s": 51360,
            "warmup_seconds": 1, "steady_seconds": 4, "drain_seconds": 3,
            "cooldown_seconds": 2,
            "primary": "application_new_count_invalid_repeated_minus_invalid_distinct",
            "ci": "paired_block_percentile_bootstrap_20000_seed_20261113"}
    r01.write_new(session / "plan.json", plan)
    return plan


def validate_plan(session, plan):
    require(plan.get("schema") == "pluto.file02.a06.plan.v1" and plan.get("status") == "LOCKED" and
            plan.get("case_id") == "A06" and plan.get("source_tree") ==
            r01.git("rev-parse", plan["source_commit"] + "^{tree}") and
            plan.get("host") == r01.cpu_inventory() and
            plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("config_sha256") == common.digest(session / "raw/config.json") ==
            common.digest(ROOT / "research/config/s06_pilot.json") and
            plan.get("schedule_sha256") == common.digest(session / "raw/schedule.json") and
            common.read_json(session / "raw/schedule.json") == schedule() and
            plan.get("n_networks") == 16 and plan.get("total_tx_s") == 20 and
            plan.get("total_bytes_s") == 51360, "A06 locked plan drift")


def bind_run_plan(run, network_root):
    run, network_root = Path(run).resolve(), Path(network_root).resolve()
    require(run.is_dir() and network_root.is_dir() and not (run / "research_plan/plan.json").exists(),
            "A06 run/network path")
    meta, workload, nodes = (common.read_json(run / name) for name in
                             ("metadata.json", "plan.json", "nodes.json"))
    condition = meta.get("condition")
    require(meta.get("case_id") == "A06" and meta.get("scope") == "A_CONFIRMATORY" and
            condition in CONDITIONS and len(nodes) == 4 and workload.get("run_id") == meta.get("run_id") and
            workload.get("profile") == 4 and workload.get("regime") == "equal_tx_s" and
            workload.get("target_tx_per_second") == 20 and len(workload.get("events", [])) == 81 and
            workload.get("warmup_ns") == 1_000_000_000 and
            workload.get("steady_ns") == 4_000_000_000 and
            workload.get("drain_ns") == 3_000_000_000 and
            workload.get("replay_mode", False) == condition.endswith("repeated") and
            all(n["rpc"].startswith("http://127.0.0.1:") for n in nodes), "A06 binding")
    config = common.read_json(run / "config.json")
    require(common.digest(run / "config.json") == meta["config_sha256"] ==
            common.digest(ROOT / "research/config/s06_pilot.json"), "A06 safety config")
    network, safety = config["network"], config["safety"]
    inventory = r01.cpu_inventory()
    host = {"cpu_model": inventory["model"], "cpu_features": inventory["flags"],
            "architecture": inventory["architecture"], "logical_cpus": inventory["logical_cpus"],
            "os": inventory["os"]}
    research_plan = {"schema": "pluto.file02.plan.v1", "status": "LOCKED",
        "run_id": workload["run_id"], "case_id": "A06", "protocol_version": "file02-common-v1",
        "research_question": "RQ1b", "hypothesis": "replay changes cache and verification work",
        "null_hypothesis": "repeated and distinct streams reach application equally often",
        "alternative_hypothesis": "repeated stream reaches application less often",
        "analysis_plan": {"primary_endpoint": "application_new_count_invalid_repeated_minus_invalid_distinct",
                          "independent_unit": "fresh_four_validator_network",
                          "ci_method": "paired_block_percentile_bootstrap_20000_seed_20261113",
                          "exclusion_rule": "failed arm excludes block; no replacement"},
        "raw_capture_plan": ["telemetry", "transaction_results", "validator_blocks_and_apphash",
                             "loadgen_timeline", "host_metrics"],
        "expected_oracle": "fixed valid exact once; repeated valid at most once; bad secp absent; four-validator agreement",
        "source_commit": meta["source_commit"], "source_tree": meta["source_tree"],
        "config_sha256": meta["config_sha256"], "corpus_sha256": common.digest(run / "plan.json"),
        "genesis_sha256": workload["genesis_sha256"], "profile": "H",
        "workload": "BENIGN_HIGH" if condition == "valid_distinct" else
                    "MIXED" if condition == "valid_repeated" else "ECDSA_INVALID",
        "actor_capability": "NONE" if condition == "valid_distinct" else "B",
        "regime": "equal_tx_s", "traffic_source": "RPC",
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
            "network_shape": "four_validator_single_host_full_mesh",
            "endpoint_hosts": ["127.0.0.1"]*4,
            "rpc_endpoints": [n["rpc"] for n in nodes],
            "config_toml_sha256": [common.digest(network_root / f"validator-{i}/config/config.toml")
                                   for i in range(4)]},
        "safety_config": safety, "seed": workload["seed"], "order_index": meta["order"],
        "phase_seconds": {"warmup": 1, "steady": 4, "drain": 3, "cooldown": 2},
        "planned": {"valid_tx_s": 20 if condition.startswith("valid") else 6,
                    "invalid_tx_s": 0 if condition.startswith("valid") else 14,
                    "total_bytes_s": 51360},
        "matching": {"matched_variable": "bytes_s",
                     "unmatched_variables": ["hash_uniqueness", "cache_state"],
                     "legitimate_stream_id": f"A06-b{meta['block']:02d}-fixed"},
        "pilot_reference": "research/evidence/file02/T06_FORMAL_CALIBRATION_AUDIT.md",
        "independent_run_target": 4,
        "p99_precision_rule": "INSUFFICIENT_PRECISION; p99 not primary"}
    common.validate_plan(research_plan)
    directory = run / "research_plan"
    directory.mkdir(mode=0o700)
    r01.write_new(directory / "plan.json", research_plan)


def audit_run(raw, job, plan):
    run = raw / "runs" / job["run_id"]
    meta, workload, result, oracle, bound = (common.read_json(run / name) for name in
        ("metadata.json", "plan.json", "result.json", "oracle.json", "research_plan/plan.json"))
    common.validate_plan(bound)
    condition = job["condition"]
    require(meta.get("schema") == "pluto.file02.a06_run.v1" and meta.get("scope") == "A_CONFIRMATORY" and
            all(meta.get(key) == job[key] for key in ("run_id", "block", "order", "condition")) and
            meta.get("source_commit") == bound["source_commit"] == plan["source_commit"] and
            meta.get("source_tree") == bound["source_tree"] == plan["source_tree"] and
            bound["run_id"] == workload["run_id"] == result["run_id"] == oracle["run_id"] and
            bound["corpus_sha256"] == result["plan_sha256"] == common.digest(run / "plan.json") and
            bound["genesis_sha256"] == result["genesis_sha256"] == workload["genesis_sha256"] ==
            common.digest(run / "genesis.bin") and
            bound["config_sha256"] == result["config_sha256"] == plan["config_sha256"] ==
            common.digest(run / "config.json"), "A06 provenance")
    require(result["status"] in (("OK", "ORACLE_MISMATCH") if condition.endswith("repeated") else ("OK",)) and
            result["planned_count"] == result["actual_send_count"] == len(result["events"]) ==
            len(workload["events"]) == 81 and oracle["status"] == "OK" and
            oracle["valid_committed"] == (81 if condition == "valid_distinct" else
                                          26 if condition == "valid_repeated" else 25) and
            oracle["block_id"] and oracle["app_hash"], "A06 execution/oracle")
    fixed, hashes, windows = [], set(), [0]*4
    outcomes = {"app_code_0": 0, "app_code_4": 0, "app_code_6": 0, "duplicate_cache": 0}
    for expected, actual in zip(workload["events"], result["events"]):
        raw_bytes = bytes.fromhex(expected["bytes_hex"])
        digest = hashlib.sha256(raw_bytes).hexdigest()
        require(expected["id"] == actual["id"] and expected["sha256"] == digest and
                len(raw_bytes) == actual["wire_bytes"] == 2568 and
                isinstance(actual.get("actual_send_monotonic_ns"), int) and
                actual["actual_send_monotonic_ns"] > 0 and
                isinstance(actual.get("actual_offset_ns"), int), "A06 send/size")
        hashes.add(digest)
        if expected["scenario"] == "fixed_legitimate":
            fixed.append(digest)
        code = actual.get("actual_code")
        key = "duplicate_cache" if actual.get("reason") == "duplicate_cache" else f"app_code_{code}"
        require(key in outcomes, "A06 unclassified RPC outcome")
        outcomes[key] += 1
        if expected["offset_ns"] >= 1_000_000_000:
            window = (actual["actual_offset_ns"] - 1_000_000_000)//1_000_000_000
            require(0 <= window < 4, "A06 rate phase")
            windows[window] += 1
    require(len(fixed) == 25 and windows == [20]*4 and
            len(hashes) == (26 if condition.endswith("repeated") else 81),
            "A06 fixed stream/rate/hash uniqueness")
    expected_committed = {event["sha256"]: 1 for event in workload["events"]
                          if event["scenario"] in ("fixed_legitimate", "benign_background", "A06_valid_replay")}
    require(oracle.get("committed_hashes") == expected_committed and
            oracle.get("validator_committed_hashes") == [expected_committed]*4,
            "A06 immutable four-validator committed hash oracle")
    records = r06.samples(run / "recorder-0.jsonl")
    new = [record for record in records if record.get("check_type") == "new"]
    recheck = [record for record in records if record.get("check_type") == "recheck"]
    background_hashes = {event["sha256"] for event in workload["events"]
                         if event["scenario"] != "fixed_legitimate"}
    app_background_new = [record for record in new if record.get("tx_sha256") in background_hashes]
    require(all(record.get("kind") == "check_tx" and isinstance(record.get("service_ns"), int)
                for record in records) and
            all(record.get("code") == (4 if condition.startswith("invalid") else 0 if condition == "valid_distinct" else record.get("code"))
                for record in app_background_new), "A06 recorder attribution")
    if condition.startswith("invalid"):
        require(all(record.get("code") == 4 and record.get("last_stage") == "secp" for record in app_background_new),
                "A06 invalid repeated reached unexpected stage")
    observer = r06.observer_metrics(run / "observer-0.jsonl")
    return {"run_id": job["run_id"], "block": job["block"], "order": job["order"],
            "condition": condition, "genesis_sha256": workload["genesis_sha256"],
            "fixed_hashes": fixed, "unique_hash_count": len(hashes), "windows": windows,
            "rpc_outcomes": outcomes, "app_background_new_count": len(app_background_new),
            "app_background_new_secp_count": sum(record.get("last_stage") == "secp" for record in app_background_new),
            "app_background_mean_service_ns": statistics.mean(record["service_ns"] for record in app_background_new)
                if app_background_new else None,
            "app_recheck_count": len(recheck), "valid_committed": oracle["valid_committed"],
            "observer_validator0": observer, "qc_status": "PASS"}


def ci(deltas):
    require(len(deltas) == 4, "four independent blocks")
    rng = random.Random(SEED)
    draws = [statistics.mean(deltas[rng.randrange(4)] for _ in range(4)) for _ in range(20_000)]
    return [r01.percentile(draws, .025), r01.percentile(draws, .975)]


def derive(session, plan):
    manifest = common.verify_raw(session / "raw")
    rows, failed = [], []
    for job in schedule()["jobs"]:
        try:
            rows.append(audit_run(session / "raw", job, plan))
        except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError) as exc:
            failed.append({"run_id": job["run_id"], "reason": str(exc)})
    deltas = []
    for block in range(4):
        group = [row for row in rows if row["block"] == block]
        if (len(group) != 4 or {row["condition"] for row in group} != set(CONDITIONS) or
                len({row["genesis_sha256"] for row in group}) != 1 or
                len({tuple(row["fixed_hashes"]) for row in group}) != 1):
            failed.append({"block": block, "reason": "A06 four-arm matching"})
            continue
        by = {row["condition"]: row for row in group}
        deltas.append(by["invalid_repeated"]["app_background_new_count"] -
                      by["invalid_distinct"]["app_background_new_count"])
    status = "PASS" if len(rows) == 16 and len(deltas) == 4 and not failed else "INVALID_FOR_INFERENCE"
    qc = {"schema": "pluto.file02.a06.qc.v1", "case_id": "A06", "status": status,
          "n_planned": 16, "n_valid": len(rows), "failed": failed,
          "raw_manifest_sha256": manifest, "plan_sha256": common.digest(session / "plan.json")}
    mean = statistics.mean(deltas) if status == "PASS" else None
    aggregate = {"schema": "pluto.file02.a06.aggregate.v1", "case_id": "A06",
                 "status": "ANALYZED" if status == "PASS" else "WITHHELD_QC_FAILURE",
                 "n_blocks": len(deltas), "invalid_repeated_minus_distinct_app_new_count": mean,
                 "ci95_count": ci(deltas) if status == "PASS" else None,
                 "block_deltas": deltas if status == "PASS" else None,
                 "raw_manifest_sha256": manifest, "plan_sha256": qc["plan_sha256"],
                 "service_degradation_status": "NOT_TESTED_BY_A06"}
    return rows, qc, aggregate


def csv_text(rows):
    from io import StringIO
    buffer = StringIO()
    keys = ("run_id", "block", "order", "condition", "unique_hash_count",
            "app_background_new_count", "app_background_new_secp_count", "app_recheck_count",
            "valid_committed", "qc_status")
    writer = csv.DictWriter(buffer, fieldnames=keys, lineterminator="\n")
    writer.writeheader()
    writer.writerows({key: row[key] for key in keys} for row in rows)
    return buffer.getvalue()


def decision_text(qc, aggregate):
    return (f"# A06 bounded replay mechanism\n\nQC: {qc['status']}; {qc['n_valid']}/16 arms. "
            f"Invalid repeated-minus-distinct application NEW count: "
            f"{aggregate['invalid_repeated_minus_distinct_app_new_count']}; "
            f"CI95: {aggregate['ci95_count']}. This tests cache and crypto reachability, "
            "not service degradation or DoS. Single-host RPC-origin only.\n")


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.a06.per_run.v1", "rows": rows})
    common.write_once(session / "result_per_run.csv", csv_text(rows))
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir(mode=0o700)
    r01.write_new(session / "figures/status.json", {"status": "PENDING_CASE_REVIEW",
        "reason": "numeric replay mechanism recorded without scientific figure"})
    common.write_once(session / "decision.md", decision_text(qc, aggregate))
    return qc, aggregate


def execute():
    session = ARTIFACTS / ("a06-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") +
                           "-" + uuid4().hex[:8])
    plan = prepare(session)
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT,
                               stdout=stream, stderr=subprocess.STDOUT)
    require(build.returncode == 0, f"build failed: {session}")
    env = os.environ.copy()
    env.update(FILE02_A06_ROOT=str(raw), FILE02_SOURCE_COMMIT=plan["source_commit"],
               FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02A06Replay$", "-timeout", "3600s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=3700)
    manifest = common.freeze_raw(raw)
    qc, aggregate = write_outputs(session, plan)
    print(f"A06 session={session} integration_exit={run.returncode} qc={qc['status']} "
          f"effect={aggregate['invalid_repeated_minus_distinct_app_new_count']} "
          f"ci95={aggregate['ci95_count']} raw_manifest_sha256={manifest}")
    return 0 if run.returncode == 0 and qc["status"] == "PASS" else 2


def verify(session):
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve() and session.is_dir(), "private A06 session")
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    rows, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.a06.per_run.v1", "rows": rows} and
            (session / "result_per_run.csv").read_text() == csv_text(rows) and
            common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate and
            (session / "decision.md").read_text() == decision_text(qc, aggregate) and
            common.read_json(session / "figures/status.json")["status"] == "PENDING_CASE_REVIEW",
            "A06 derived artifact mismatch")
    print(f"A06_VERIFY=OK qc={qc['status']} raw_manifest_sha256={qc['raw_manifest_sha256']}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    binder = sub.add_parser("bind-run-plan")
    binder.add_argument("run")
    binder.add_argument("network_root")
    sub.add_parser("run")
    check = sub.add_parser("verify")
    check.add_argument("session")
    args = parser.parse_args()
    try:
        if args.command == "bind-run-plan":
            bind_run_plan(args.run, args.network_root)
        elif args.command == "run":
            return execute()
        else:
            verify(args.session)
        return 0
    except (ValueError, OSError, subprocess.SubprocessError) as exc:
        print(f"A06_ERROR={exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
