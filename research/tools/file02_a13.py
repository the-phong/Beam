#!/usr/bin/env python3
"""Locked A13 attacker start/stop time-dynamics experiment."""

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
from file02_t06_precision import parse_go_rfc3339

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
SEED = 20261113
WINDOWS = [6]*3 + [20]*6 + [6]*3
PHASES = ("baseline", "on", "recovery")


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def schedule():
    jobs = []
    for block in range(4):
        for order, condition in enumerate(("benign", "attack") if block % 2 == 0 else ("attack", "benign")):
            jobs.append({"block": block, "order": order, "condition": condition,
                         "run_id": f"A13-b{block:02d}-{condition}"})
    return {"schema": "pluto.file02.a13.schedule.v1", "seed": SEED,
            "method": "balanced_order", "jobs": jobs}


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
    plan = {"schema": "pluto.file02.a13.plan.v1", "status": "LOCKED", "case_id": "A13",
            "source_commit": r01.git("rev-parse", "HEAD"),
            "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
            "created_utc": datetime.now(timezone.utc).isoformat(), "host": r01.cpu_inventory(),
            "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
            "module_sha256": common.digest(ROOT / "go.mod"),
            "config_sha256": common.digest(raw / "config.json"),
            "schedule_sha256": common.digest(raw / "schedule.json"),
            "protocol": "research/protocol/FILE02_A13_LOCKED_PLAN.md",
            "profile": "H", "chain_id": "pluto-file02-a13", "n_blocks": 4,
            "n_networks": 8, "fixed_legitimate_tx_s": 6,
            "window_tx_counts": WINDOWS, "max_window_bytes_s": 51360,
            "attack_onset_s": 4, "attack_stop_s": 10,
            "warmup_seconds": 1, "steady_seconds": 12, "drain_seconds": 5,
            "cooldown_seconds": 2, "deadline_ms": 3000,
            "primary": "on_phase_benign_minus_attack_fixed_timely_fraction",
            "ci": "paired_block_percentile_bootstrap_20000_seed_20261113",
            "alert_status": "UNAVAILABLE_NO_ACCEPTED_DETECTOR"}
    r01.write_new(session / "plan.json", plan)
    return plan


def validate_plan(session, plan):
    require(plan.get("schema") == "pluto.file02.a13.plan.v1" and plan.get("status") == "LOCKED" and
            plan.get("case_id") == "A13" and plan.get("source_tree") ==
            r01.git("rev-parse", plan["source_commit"] + "^{tree}") and
            plan.get("host") == r01.cpu_inventory() and
            plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("config_sha256") == common.digest(session / "raw/config.json") ==
            common.digest(ROOT / "research/config/s06_pilot.json") and
            plan.get("schedule_sha256") == common.digest(session / "raw/schedule.json") and
            common.read_json(session / "raw/schedule.json") == schedule() and
            plan.get("n_networks") == 8 and plan.get("window_tx_counts") == WINDOWS and
            plan.get("attack_onset_s") == 4 and plan.get("attack_stop_s") == 10 and
            plan.get("deadline_ms") == 3000, "A13 locked plan drift")


def bind_run_plan(run, network_root):
    run, network_root = Path(run).resolve(), Path(network_root).resolve()
    require(run.is_dir() and network_root.is_dir() and not (run / "research_plan/plan.json").exists(),
            "A13 run/network path")
    meta, workload, nodes = (common.read_json(run / name) for name in
                             ("metadata.json", "plan.json", "nodes.json"))
    condition = meta.get("condition")
    require(meta.get("case_id") == "A13" and meta.get("scope") == "A_CONFIRMATORY" and
            condition in ("benign", "attack") and len(nodes) == 4 and
            workload.get("run_id") == meta.get("run_id") and
            workload.get("profile") == 4 and workload.get("regime") == "equal_tx_s" and
            workload.get("waveform") == "step" and workload.get("target_tx_per_second") == 13 and
            len(workload.get("events", [])) == 157 and
            workload.get("warmup_ns") == 1_000_000_000 and
            workload.get("steady_ns") == 12_000_000_000 and
            workload.get("drain_ns") == 5_000_000_000 and
            all(n["rpc"].startswith("http://127.0.0.1:") for n in nodes), "A13 binding")
    config = common.read_json(run / "config.json")
    require(common.digest(run / "config.json") == meta["config_sha256"] ==
            common.digest(ROOT / "research/config/s06_pilot.json"), "A13 safety config")
    network, safety = config["network"], config["safety"]
    inventory = r01.cpu_inventory()
    host = {"cpu_model": inventory["model"], "cpu_features": inventory["flags"],
            "architecture": inventory["architecture"], "logical_cpus": inventory["logical_cpus"],
            "os": inventory["os"]}
    research_plan = {"schema": "pluto.file02.plan.v1", "status": "LOCKED",
        "run_id": workload["run_id"], "case_id": "A13", "protocol_version": "file02-common-v1",
        "research_question": "RQ1b", "hypothesis": "attacker on phase lowers fixed timely inclusion",
        "null_hypothesis": "on-phase benign-minus-attack timely fraction <= 0",
        "alternative_hypothesis": "on-phase benign-minus-attack timely fraction > 0",
        "analysis_plan": {"primary_endpoint": "on_phase_fixed_legitimate_all_four_within_3000ms_fraction",
                          "independent_unit": "fresh_four_validator_network",
                          "ci_method": "paired_block_percentile_bootstrap_20000_seed_20261113",
                          "exclusion_rule": "failed arm excludes pair; no replacement"},
        "raw_capture_plan": ["telemetry", "transaction_results", "validator_blocks_and_apphash",
                             "loadgen_timeline", "host_metrics"],
        "expected_oracle": "fixed valid exact once; invalid 4/5/5 on phase rejected; all-four agreement",
        "source_commit": meta["source_commit"], "source_tree": meta["source_tree"],
        "config_sha256": meta["config_sha256"], "corpus_sha256": common.digest(run / "plan.json"),
        "genesis_sha256": workload["genesis_sha256"], "profile": "H",
        "workload": "BENIGN_BURST" if condition == "benign" else "MIXED",
        "actor_capability": "NONE" if condition == "benign" else "C",
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
            "network_shape": "four_validator_single_host_full_mesh", "endpoint_hosts": ["127.0.0.1"]*4,
            "rpc_endpoints": [n["rpc"] for n in nodes],
            "config_toml_sha256": [common.digest(network_root / f"validator-{i}/config/config.toml")
                                   for i in range(4)]},
        "safety_config": safety, "seed": workload["seed"], "order_index": meta["order"],
        "phase_seconds": {"warmup": 1, "steady": 12, "drain": 5, "cooldown": 2},
        "planned": {"valid_tx_s": 13 if condition == "benign" else 6,
                    "invalid_tx_s": 0 if condition == "benign" else 7,
                    "total_bytes_s": 33384},
        "matching": {"matched_variable": "bytes_s", "unmatched_variables": ["verification_stage"],
                     "legitimate_stream_id": f"A13-b{meta['block']:02d}-fixed"},
        "pilot_reference": "research/evidence/file02/T06_FORMAL_CALIBRATION_AUDIT.md",
        "independent_run_target": 4,
        "p99_precision_rule": "INSUFFICIENT_PRECISION; p99 not primary"}
    common.validate_plan(research_plan)
    directory = run / "research_plan"
    directory.mkdir(mode=0o700)
    r01.write_new(directory / "plan.json", research_plan)


def audit_run(raw, job, plan):
    run = raw / "runs" / job["run_id"]
    meta, workload, result, oracle, commits, bound = (common.read_json(run / name) for name in
        ("metadata.json", "plan.json", "result.json", "oracle.json",
         "commit_observations.json", "research_plan/plan.json"))
    common.validate_plan(bound)
    condition = job["condition"]
    require(meta.get("schema") == "pluto.file02.a13_run.v1" and meta.get("scope") == "A_CONFIRMATORY" and
            all(meta.get(key) == job[key] for key in ("run_id", "block", "order", "condition")) and
            meta.get("source_commit") == bound["source_commit"] == plan["source_commit"] and
            meta.get("source_tree") == bound["source_tree"] == plan["source_tree"] and
            bound["run_id"] == workload["run_id"] == result["run_id"] == oracle["run_id"] == commits["run_id"] and
            bound["corpus_sha256"] == result["plan_sha256"] == common.digest(run / "plan.json") and
            bound["genesis_sha256"] == result["genesis_sha256"] == workload["genesis_sha256"] ==
            common.digest(run / "genesis.bin") and
            bound["config_sha256"] == result["config_sha256"] == plan["config_sha256"] ==
            common.digest(run / "config.json"), "A13 provenance")
    require(result["status"] == oracle["status"] == "OK" and
            result["planned_count"] == result["actual_send_count"] == len(result["events"]) ==
            len(workload["events"]) == 157 and
            oracle["valid_committed"] == (157 if condition == "benign" else 73) and
            oracle["invalid_rejected"] == (0 if condition == "benign" else 84) and
            len(commits["events"]) == oracle["valid_committed"] and
            oracle["block_id"] and oracle["app_hash"], "A13 execution/oracle")
    by_commit = {item["id"]: item for item in commits["events"]}
    require(len(by_commit) == len(commits["events"]), "A13 commit identity")
    fixed, windows, phase_latencies, classes = [], [0]*12, {p: [] for p in PHASES}, {}
    hashes = set()
    for expected, actual in zip(workload["events"], result["events"]):
        wire = bytes.fromhex(expected["bytes_hex"])
        digest = hashlib.sha256(wire).hexdigest()
        require(expected["id"] == actual["id"] and expected["sha256"] == digest and
                digest not in hashes and len(wire) == actual["wire_bytes"] == 2568 and
                actual.get("actual_code") == expected["expected_code"] and
                not actual.get("submission_error") and
                isinstance(actual.get("actual_send_monotonic_ns"), int), "A13 send/code/size")
        hashes.add(digest)
        if expected["traffic"] == "adversarial":
            require(condition == "attack" and expected["id"] not in by_commit and
                    (expected["expected_code"], expected["expected_stage"]) == {
                        "A13_structural": (1, "structural"), "A13_bad_ml": (13, "ml"),
                        "A13_state_invalid": (6, "state")}.get(expected["scenario"]),
                    "A13 invalid stage/commit")
            classes[expected["scenario"]] = classes.get(expected["scenario"], 0)+1
        else:
            require(by_commit[expected["id"]]["sha256"] == digest, "A13 valid commit")
        if expected["offset_ns"] >= 1_000_000_000:
            second = (actual["actual_offset_ns"]-1_000_000_000)//1_000_000_000
            require(0 <= second < 12, "A13 phase crossing")
            windows[second] += 1
            if expected["scenario"] == "fixed_legitimate":
                fixed.append(digest)
                seen = by_commit[expected["id"]].get("all_four_seen_monotonic_ns")
                require(isinstance(seen, int) and seen >= actual["actual_send_monotonic_ns"],
                        "A13 fixed monotonic inclusion")
                phase = "baseline" if second < 3 else "on" if second < 9 else "recovery"
                phase_latencies[phase].append((seen-actual["actual_send_monotonic_ns"])/1_000_000)
    require(len(fixed) == 72 and windows == WINDOWS and
            {key: len(value) for key, value in phase_latencies.items()} ==
            {"baseline": 18, "on": 36, "recovery": 18} and
            (condition != "attack" or classes == {"A13_structural": 24,
                "A13_bad_ml": 30, "A13_state_invalid": 30}), "A13 fixed stream/phase/rate")
    records = {record["tx_sha256"]: record for record in r06.samples(run / "recorder-0.jsonl")
               if record.get("check_type") == "new" and record.get("tx_sha256")}
    require(len(records) == 157 and all(records[event["sha256"]]["code"] == event["expected_code"] and
            records[event["sha256"]]["last_stage"] == event["expected_stage"]
            for event in workload["events"]), "A13 recorder stage/hash")
    observer = r06.observer_metrics(run / "observer-0.jsonl")
    started = parse_go_rfc3339(result["started_utc"])
    curve = []
    for sample in r06.samples(run / "observer-0.jsonl"):
        observed = parse_go_rfc3339(sample["observed_utc"])
        curve.append({"elapsed_s": (observed-started).total_seconds(),
                      "accepted_mempool_bytes": sample.get("accepted_mempool_bytes"),
                      "app_cpu_ns": (sample.get("application_process") or {}).get("cpu_ns"),
                      "checktx_socket_queue_seconds": sample.get("checktx_socket_queue_seconds")})
    return {"run_id": job["run_id"], "block": job["block"], "order": job["order"],
            "condition": condition, "genesis_sha256": workload["genesis_sha256"],
            "fixed_hashes": fixed, "windows": windows, "invalid_classes": classes,
            "valid_committed": oracle["valid_committed"], "invalid_rejected": oracle["invalid_rejected"],
            "phase_fixed_latency_upper_ms": phase_latencies,
            "phase_fixed_timely_fraction": {p: sum(x <= plan["deadline_ms"] for x in values)/len(values)
                                            for p, values in phase_latencies.items()},
            "observer_validator0": observer, "resource_curve_validator0": curve,
            "alert_time_s": None, "alert_status": "UNAVAILABLE_NO_ACCEPTED_DETECTOR",
            "qc_status": "PASS"}


def ci(deltas):
    require(len(deltas) == 4, "four independent blocks")
    rng = random.Random(SEED)
    draws = [statistics.mean(deltas[rng.randrange(4)] for _ in range(4)) for _ in range(20_000)]
    return [r01.percentile(draws, .025), r01.percentile(draws, .975)]


def derive(session, plan):
    manifest = common.verify_raw(session / "raw")
    rows, failed, phase_deltas = [], [], {p: [] for p in PHASES}
    for job in schedule()["jobs"]:
        try:
            rows.append(audit_run(session / "raw", job, plan))
        except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError) as exc:
            failed.append({"run_id": job["run_id"], "reason": str(exc)})
    for block in range(4):
        group = [row for row in rows if row["block"] == block]
        if (len(group) != 2 or {row["condition"] for row in group} != {"benign", "attack"} or
                len({row["genesis_sha256"] for row in group}) != 1 or
                len({tuple(row["fixed_hashes"]) for row in group}) != 1):
            failed.append({"block": block, "reason": "A13 matched phase/fixed stream"})
            continue
        by = {row["condition"]: row for row in group}
        for phase in PHASES:
            phase_deltas[phase].append(by["benign"]["phase_fixed_timely_fraction"][phase] -
                                       by["attack"]["phase_fixed_timely_fraction"][phase])
    status = "PASS" if len(rows) == 8 and all(len(v) == 4 for v in phase_deltas.values()) and not failed else "INVALID_FOR_INFERENCE"
    qc = {"schema": "pluto.file02.a13.qc.v1", "case_id": "A13", "status": status,
          "n_planned": 8, "n_valid": len(rows), "failed": failed,
          "raw_manifest_sha256": manifest, "plan_sha256": common.digest(session / "plan.json")}
    aggregate = {"schema": "pluto.file02.a13.aggregate.v1", "case_id": "A13",
                 "status": "ANALYZED" if status == "PASS" else "WITHHELD_QC_FAILURE",
                 "n_blocks": 4 if status == "PASS" else None,
                 "phase_benign_minus_attack_fraction": {p: statistics.mean(v) for p, v in phase_deltas.items()}
                    if status == "PASS" else None,
                 "phase_ci95_fraction": {p: ci(v) for p, v in phase_deltas.items()}
                    if status == "PASS" else None,
                 "scheduled_attack_onset_s": 4, "scheduled_attack_stop_s": 10,
                 "service_violation_status": "NOT_OBSERVED" if status == "PASS" and
                    all(v == 0 for v in phase_deltas["on"]) else "UNRESOLVED",
                 "recovery_time_s": None,
                 "recovery_status": "NOT_APPLICABLE_NO_OBSERVED_DEGRADATION" if status == "PASS" and
                    all(v == 0 for v in phase_deltas["on"]) else "UNAVAILABLE_NO_LOCKED_THRESHOLD",
                 "alert_lead_time_s": None, "false_alert_transients": None,
                 "alert_status": "UNAVAILABLE_NO_ACCEPTED_DETECTOR",
                 "raw_manifest_sha256": manifest, "plan_sha256": qc["plan_sha256"]}
    return rows, qc, aggregate


def csv_text(rows):
    from io import StringIO
    buffer = StringIO()
    keys = ("run_id", "block", "order", "condition", "valid_committed",
            "invalid_rejected", "qc_status")
    writer = csv.DictWriter(buffer, fieldnames=keys, lineterminator="\n")
    writer.writeheader()
    writer.writerows({key: row[key] for key in keys} for row in rows)
    return buffer.getvalue()


def decision_text(qc, aggregate):
    return (f"# A13 bounded attack dynamics\n\nQC: {qc['status']}; {qc['n_valid']}/8 arms. "
            f"Phase effects: {aggregate['phase_benign_minus_attack_fraction']}; "
            f"CI95: {aggregate['phase_ci95_fraction']}. Alert lead/false-alert values are "
            "UNAVAILABLE_NO_ACCEPTED_DETECTOR; recovery is not assigned a zero time. "
            "Single-host RPC-origin only; no DoS or knee inference.\n")


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.a13.per_run.v1", "rows": rows})
    common.write_once(session / "result_per_run.csv", csv_text(rows))
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir(mode=0o700)
    r01.write_new(session / "figures/status.json", {"status": "PENDING_CASE_REVIEW",
        "reason": "numeric phase curves retained without scientific figure"})
    common.write_once(session / "decision.md", decision_text(qc, aggregate))
    return qc, aggregate


def execute():
    session = ARTIFACTS / ("a13-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") +
                           "-" + uuid4().hex[:8])
    plan = prepare(session)
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT,
                               stdout=stream, stderr=subprocess.STDOUT)
    require(build.returncode == 0, f"build failed: {session}")
    env = os.environ.copy()
    env.update(FILE02_A13_ROOT=str(raw), FILE02_SOURCE_COMMIT=plan["source_commit"],
               FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02A13Dynamics$", "-timeout", "3600s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=3700)
    manifest = common.freeze_raw(raw)
    qc, aggregate = write_outputs(session, plan)
    print(f"A13 session={session} integration_exit={run.returncode} qc={qc['status']} "
          f"on_effect={None if aggregate['phase_benign_minus_attack_fraction'] is None else aggregate['phase_benign_minus_attack_fraction']['on']} "
          f"raw_manifest_sha256={manifest}")
    return 0 if run.returncode == 0 and qc["status"] == "PASS" else 2


def verify(session):
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve() and session.is_dir(), "private A13 session")
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    rows, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.a13.per_run.v1", "rows": rows} and
            (session / "result_per_run.csv").read_text() == csv_text(rows) and
            common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate and
            (session / "decision.md").read_text() == decision_text(qc, aggregate) and
            common.read_json(session / "figures/status.json")["status"] == "PENDING_CASE_REVIEW",
            "A13 derived artifact mismatch")
    print(f"A13_VERIFY=OK qc={qc['status']} raw_manifest_sha256={qc['raw_manifest_sha256']}")


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
        print(f"A13_ERROR={exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
