#!/usr/bin/env python3
"""R07 tolerance-matched encoded-byte-rate network run and integrity audit."""

import argparse
import csv
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import random
import statistics
import subprocess
import sys
from uuid import uuid4

import file02_common as common
import file02_r01 as r01
import file02_r06 as r06
import file02_t06_precision as precision

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
PROFILES = r06.PROFILES
WIRE = r06.WIRE
SEED = 20261113
RATE = {"C": 17, "CP": 1, "P": 1, "H": 1}


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
                         "seed": seed, "run_id": f"r07-b{block:02d}-{profile.lower()}"})
    return {"schema": "pluto.file02.r07.schedule.v1", "master_seed": SEED,
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
    plan = {"schema": "pluto.file02.r07.plan.v1", "status": "LOCKED", "case_id": "R07",
            "source_commit": r01.git("rev-parse", "HEAD"), "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
            "created_utc": datetime.now(timezone.utc).isoformat(), "host": r01.cpu_inventory(),
            "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
            "module_sha256": common.digest(ROOT / "go.mod"),
            "safety_config_sha256": common.digest(ROOT / "research/config/s06_pilot.json"),
            "schedule_sha256": common.digest(raw / "schedule.json"),
            "protocol": "research/protocol/FILE02_R07_LOCKED_PLAN.md",
            "profiles": list(PROFILES), "wire_bytes": WIRE, "chain_id": "pluto-file02-rq1",
            "n_blocks": 12, "n_networks": 48, "target_bytes_s": 2568,
            "planned_tx_s": RATE, "byte_match_tolerance": .05,
            "warmup_seconds": 1, "steady_seconds": 4, "drain_seconds": 3, "cooldown_seconds": 2,
            "primary": "validator0_valid_steady_CheckTx_mean_service_ns",
            "contrast": "H_minus_CP", "ci": "paired_percentile_bootstrap_20000_seed_20261113"}
    r01.write_new(session / "plan.json", plan)
    return plan


def validate_plan(session, plan):
    require(plan.get("schema") == "pluto.file02.r07.plan.v1" and plan.get("status") == "LOCKED" and
            plan.get("case_id") == "R07" and plan.get("source_tree") == r01.git("rev-parse", plan["source_commit"] + "^{tree}"),
            "plan source/status")
    require(plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("safety_config_sha256") == common.digest(ROOT / "research/config/s06_pilot.json") and
            plan.get("host") == r01.cpu_inventory() and
            plan.get("schedule_sha256") == common.digest(session / "raw/schedule.json") and
            common.read_json(session / "raw/schedule.json") == schedule(), "runtime or schedule drift")
    require(plan.get("n_blocks") == 12 and plan.get("n_networks") == 48 and
            plan.get("target_bytes_s") == 2568 and plan.get("planned_tx_s") == RATE and
            plan.get("byte_match_tolerance") == .05 and plan.get("steady_seconds") == 4,
            "design drift")


def audit_run(raw, job, plan):
    run = raw / "runs" / job["run_id"]
    meta, workload, result, oracle, commits, common_plan, nodes = (common.read_json(run / name) for name in
        ("metadata.json", "plan.json", "result.json", "oracle.json", "commit_observations.json",
         "research_plan/plan.json", "nodes.json"))
    common.validate_plan(common_plan)
    profile = job["profile"]
    expected_count = 1 + 4*RATE[profile]
    require(meta.get("schema") == "pluto.file02.r07_run.v1" and meta.get("scope") == "R_CONFIRMATORY" and
            meta.get("case_id") == common_plan.get("case_id") == "R07" and
            meta.get("recorder_mode") == "ON" and meta.get("commit_observer_mode") == "block_scan" and
            meta.get("source_commit") == common_plan.get("source_commit") == plan["source_commit"] and
            meta.get("source_tree") == common_plan.get("source_tree") == plan["source_tree"] and
            all(meta.get(key) == job[key] for key in ("block", "order", "profile", "seed")), "metadata/source")
    require(meta["run_id"] == workload["run_id"] == result["run_id"] == oracle["run_id"] == commits["run_id"] ==
            common_plan["run_id"] == job["run_id"] and
            result["plan_sha256"] == common_plan["corpus_sha256"] == common.digest(run / "plan.json") and
            result["genesis_sha256"] == common_plan["genesis_sha256"] == workload["genesis_sha256"] == common.digest(run / "genesis.bin") and
            result["config_sha256"] == common_plan["config_sha256"] == plan["safety_config_sha256"] and
            common_plan["network_config"]["rpc_endpoints"] == [node["rpc"] for node in nodes] and
            common_plan["seed"] == job["seed"] and common_plan["order_index"] == job["order"], "provenance")
    require(workload["regime"] == "equal_bytes_s" and workload["target_bytes_per_second"] == 2568 and
            workload["warmup_ns"] == 1_000_000_000 and workload["steady_ns"] == 4_000_000_000 and
            workload["drain_ns"] == 3_000_000_000 and
            common_plan["planned"]["valid_tx_s"] == RATE[profile] and
            common_plan["planned"]["total_bytes_s"] == RATE[profile]*WIRE[profile] and
            result["status"] == oracle["status"] == "OK" and
            result["planned_count"] == result["actual_send_count"] == oracle["valid_committed"] == expected_count and
            oracle["invalid_rejected"] == 0 and oracle["block_id"] and oracle["app_hash"] and
            len(commits["events"]) == expected_count, "load/oracle")
    by_commit = {entry["id"]: entry for entry in commits["events"]}
    require(len(by_commit) == expected_count, "duplicate commit observation")
    observed = r06.samples(run / "recorder-0.jsonl")
    checktx = {}
    for event in observed:
        if event.get("check_type") == "new" and event.get("tx_sha256"):
            require(event["tx_sha256"] not in checktx, "duplicate primary CheckTx")
            checktx[event["tx_sha256"]] = event
    hashes, slots = set(), set()
    windows = [0]*4
    service, rpc, inclusion = [], [], []
    for expected, actual in zip(workload["events"], result["events"]):
        h = hashlib.sha256(bytes.fromhex(expected["bytes_hex"])).hexdigest()
        require(h == expected["sha256"] and h not in hashes and expected["account_slot"] not in slots and
                expected["id"] == actual["id"] and expected["expected_code"] == actual["actual_code"] == 0 and
                actual["wire_bytes"] == WIRE[profile] and not actual.get("submission_error") and
                by_commit[expected["id"]]["sha256"] == h, "transaction identity/code")
        hashes.add(h)
        slots.add(expected["account_slot"])
        sent = actual["actual_offset_ns"]
        require(isinstance(sent, int) and 0 <= sent < 5_000_000_000 and
                (sent < 1_000_000_000) == (expected["offset_ns"] < 1_000_000_000), "phase spill")
        if expected["offset_ns"] >= 1_000_000_000:
            window = (sent-1_000_000_000)//1_000_000_000
            require(0 <= window < 4, "window overflow")
            windows[window] += 1
            event = checktx.get(h)
            require(event is not None and event.get("code") == 0 and event.get("profile") == workload["profile"] and
                    event.get("wire_bytes") == WIRE[profile] and isinstance(event.get("service_ns"), int) and
                    event["service_ns"] > 0, "missing primary span")
            service.append(event["service_ns"])
            rpc.append((precision.parse_go_rfc3339(actual["completed_utc"])-precision.parse_go_rfc3339(actual["actual_send_utc"])).total_seconds()*1000)
            inclusion.append((precision.parse_go_rfc3339(by_commit[expected["id"]]["all_four_seen_utc"])-precision.parse_go_rfc3339(actual["actual_send_utc"])).total_seconds()*1000)
    require(windows == [RATE[profile]]*4 and len(service) == 4*RATE[profile] and
            all(abs(n*WIRE[profile]-2568)/2568 <= .05 for n in windows) and
            all(x >= 0 for x in rpc+inclusion), "achieved byte rate or latency")
    observers = [r06.observer_metrics(run / f"observer-{i}.jsonl") for i in range(4)]
    return {"run_id": job["run_id"], "block": job["block"], "profile": profile,
            "order": job["order"], "seed": job["seed"], "genesis_sha256": workload["genesis_sha256"],
            "wire_bytes": WIRE[profile], "achieved_tx_s_windows": windows,
            "achieved_bytes_s_windows": [n*WIRE[profile] for n in windows],
            "valid_committed": expected_count, "mean_service_ns": statistics.mean(service),
            "service_ns_samples": service, "rpc_ms_samples": rpc,
            "inclusion_upper_bound_ms_samples": inclusion, "observer_validator0": observers[0],
            "observer_all": observers, "qc_status": "PASS"}


def derive(session, plan):
    raw = session / "raw"
    manifest = common.verify_raw(raw)
    rows, failures = [], []
    for job in schedule()["jobs"]:
        try:
            rows.append(audit_run(raw, job, plan))
        except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError) as exc:
            failures.append({"run_id": job["run_id"], "reason": str(exc)})
    for block in range(12):
        group = [r for r in rows if r["block"] == block]
        if len(group) == 4 and len({r["genesis_sha256"] for r in group}) != 1:
            failures.append({"block": block, "reason": "unmatched logical fixture"})
    if len({r["genesis_sha256"] for r in rows}) != 12:
        failures.append({"reason": "independent genesis count mismatch"})
    qc = {"schema": "pluto.file02.r07.qc.v1", "case_id": "R07",
          "status": "PASS" if len(rows) == 48 and not failures else "INVALID_FOR_INFERENCE",
          "n_planned": 48, "n_valid": len(rows), "failed": failures,
          "byte_target": 2568, "byte_tolerance": .05,
          "plan_sha256": common.digest(session / "plan.json"), "raw_manifest_sha256": manifest}
    aggregate = {"schema": "pluto.file02.r07.aggregate.v1", "case_id": "R07",
                 "status": "PENDING_VALID_QC", "n_networks": len(rows), "primary_H_minus_CP_ns": None,
                 "ci95_ns": None, "profile_summary": {}, "plan_sha256": qc["plan_sha256"],
                 "raw_manifest_sha256": manifest}
    if qc["status"] == "PASS":
        deltas = []
        for block in range(12):
            by = {r["profile"]: r for r in rows if r["block"] == block}
            deltas.append(by["H"]["mean_service_ns"]-by["CP"]["mean_service_ns"])
        aggregate["status"] = "ANALYZED_MATCHED_BYTES"
        aggregate["primary_H_minus_CP_ns"] = statistics.mean(deltas)
        aggregate["ci95_ns"] = r01.paired_ci(deltas)
        aggregate["pair_deltas_ns"] = deltas
        for profile in PROFILES:
            group = [r for r in rows if r["profile"] == profile]
            aggregate["profile_summary"][profile] = {
                "n_networks": 12, "planned_tx_s": RATE[profile], "achieved_bytes_s": RATE[profile]*WIRE[profile],
                "relative_byte_error": abs(RATE[profile]*WIRE[profile]-2568)/2568,
                "mean_service_ns": statistics.mean(r["mean_service_ns"] for r in group),
                "mean_rpc_ms": statistics.mean(statistics.mean(r["rpc_ms_samples"]) for r in group),
                "mean_inclusion_upper_bound_ms": statistics.mean(statistics.mean(r["inclusion_upper_bound_ms_samples"]) for r in group),
                "inclusion_p99_ms": None, "inclusion_p99_status": "INSUFFICIENT_PRECISION",
                "eventual_valid_included_per_network": 1+4*RATE[profile],
                "steady_sustainable_goodput_tx_s": None,
                "steady_goodput_reason": "eventual inclusion is not sustainable steady-state throughput",
                "mean_shared_app_cpu_delta_ns": statistics.mean(r["observer_validator0"]["application_process_cpu_delta_ns"] for r in group),
                "mean_shared_app_rss_peak_bytes": statistics.mean(r["observer_validator0"]["application_process_rss_peak_bytes"] for r in group),
                "mean_max_mempool_bytes": statistics.mean(r["observer_validator0"]["max_accepted_mempool_bytes"] for r in group),
                "mean_validator0_relay_send_envelope_bytes_delta": statistics.mean(r["observer_validator0"]["relay_send_envelope_bytes_delta"] for r in group)}
    return rows, qc, aggregate


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.r07.per_run.v1", "rows": rows})
    with (session / "result_per_run.csv").open("x", newline="", encoding="utf-8") as stream:
        keys = ("run_id", "block", "profile", "order", "seed", "genesis_sha256", "wire_bytes",
                "valid_committed", "mean_service_ns", "qc_status")
        writer = csv.DictWriter(stream, fieldnames=keys)
        writer.writeheader()
        writer.writerows({key: row[key] for key in keys} for row in rows)
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir()
    r01.write_new(session / "figures/status.json", {"status": "PENDING_CASE_REVIEW",
                                                     "reason": "No RQ figure drawn from this bounded case alone"})
    (session / "decision.md").write_text(
        f"# R07 decision\n\nQC: {qc['status']}; {len(rows)}/48 arms. "
        f"H-minus-CP effect: {aggregate['primary_H_minus_CP_ns']} ns, CI95 {aggregate['ci95_ns']}. "
        "Byte matching is within the preregistered ±5% tolerance, exact only for CP/H. "
        "The C/P cross-profile contrasts also change tx/s and canonical size. "
        "No p99, steady throughput, P2P-origin, multi-host, knee or DoS inference.\n", encoding="utf-8")
    return qc, aggregate


def execute():
    session = ARTIFACTS / ("r07-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8])
    plan = prepare(session)
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT, stdout=stream, stderr=subprocess.STDOUT)
    require(build.returncode == 0, f"build failed: {session}")
    env = os.environ.copy()
    env.update(FILE02_R07_ROOT=str(raw), FILE02_SOURCE_COMMIT=plan["source_commit"], FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02R07$", "-timeout", "2400s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=2500)
    manifest = common.freeze_raw(raw)
    qc, aggregate = write_outputs(session, plan)
    print(f"R07 session={session} integration_exit={run.returncode} qc={qc['status']} "
          f"effect_ns={aggregate['primary_H_minus_CP_ns']} ci95_ns={aggregate['ci95_ns']} raw_manifest_sha256={manifest}")
    return 0 if run.returncode == 0 and qc["status"] == "PASS" else 2


def verify(session):
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve(), "private session")
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    rows, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") == {"schema": "pluto.file02.r07.per_run.v1", "rows": rows} and
            common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate, "derived artifact mismatch")
    with (session / "result_per_run.csv").open(newline="", encoding="utf-8") as stream:
        csv_rows = list(csv.DictReader(stream))
    require(len(csv_rows) == len(rows) and all(str(row["mean_service_ns"]) == entry["mean_service_ns"] and
            row["run_id"] == entry["run_id"] for row, entry in zip(rows, csv_rows)), "CSV mismatch")
    print(f"R07_VERIFY=OK qc={qc['status']} n={len(rows)} raw_manifest_sha256={qc['raw_manifest_sha256']}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("run")
    command = sub.add_parser("verify")
    command.add_argument("session")
    args = parser.parse_args()
    try:
        sys.exit(execute() if args.command == "run" else (verify(args.session) or 0))
    except (ValueError, OSError, KeyError, TypeError, subprocess.TimeoutExpired) as exc:
        print(f"R07_ERROR: {exc}", file=sys.stderr)
        sys.exit(2)
