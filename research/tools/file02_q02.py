#!/usr/bin/env python3
"""Q02 locked structural-invalid cross-profile decomposition."""
import argparse
import csv
from datetime import datetime, timezone
import hashlib
from io import StringIO
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
PROFILES = ("C", "CP", "P", "H")
WIRE = {"C": 148, "CP": 2568, "P": 2504, "H": 2568}
PROFILE_CODE = {"C": 1, "CP": 2, "P": 3, "H": 4}
SEED = 20261114


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def schedule(legacy=False):
    jobs = []
    for block in range(12):
        rotated = PROFILES[block % 4:] + PROFILES[:block % 4]
        for index, profile in enumerate(rotated):
            # v1 accidentally tied arm order to profile parity after rotation.
            # Keep it replayable for frozen historical sessions; v2 balances
            # each profile across six benign-first and six attack-first blocks.
            first_benign = (block + index) % 2 == 0 if legacy else block % 2 == 0
            arms = ("benign", "attack") if first_benign else ("attack", "benign")
            for condition in arms:
                jobs.append({"block": block, "order": len(jobs) % 8,
                             "profile": profile, "condition": condition,
                             "run_id": f"Q02-b{block:02d}-{profile}-{condition}"})
    return {"schema": "pluto.file02.q02.schedule.v1" if legacy else
            "pluto.file02.q02.schedule.v2", "seed": SEED,
            "method": "rotated_profile_balanced_arm_order" if legacy else
            "rotated_profile_with_per_profile_balanced_arm_order", "jobs": jobs}


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve() and not r01.git("status", "--porcelain"),
            "clean source/private Q02 session required")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT).returncode == 0,
            "Q02 session not ignored")
    session.mkdir(mode=0o700)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    common.write_once(raw / "schedule.json", json.dumps(schedule(), sort_keys=True, indent=2) + "\n")
    common.write_once(raw / "config.json", (ROOT / "research/config/s06_pilot.json").read_text())
    plan = {"schema": "pluto.file02.q02.plan.v1", "status": "LOCKED", "case_id": "Q02",
            "source_commit": r01.git("rev-parse", "HEAD"), "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
            "created_utc": datetime.now(timezone.utc).isoformat(), "host": r01.cpu_inventory(),
            "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
            "module_sha256": common.digest(ROOT / "go.mod"),
            "config_sha256": common.digest(raw / "config.json"),
            "schedule_sha256": common.digest(raw / "schedule.json"),
            "schedule_schema": "pluto.file02.q02.schedule.v2",
            "protocol": "research/protocol/FILE02_Q02_LOCKED_PLAN.md",
            "chain_id": "pluto-file02-q02", "profiles": list(PROFILES), "wire_bytes": WIRE,
            "n_blocks": 12, "n_networks": 96, "fixed_legitimate_tx_s": 6,
            "background_tx_s": 14, "total_tx_s": 20, "deadline_ms": 3000,
            "did": "(H_attack-H_benign)-(C_attack-C_benign)",
            "ci": "block_percentile_bootstrap_20000_seed_20261114"}
    r01.write_new(session / "plan.json", plan)
    return plan


def validate_plan(session, plan):
    locked_schedule = common.read_json(session / "raw/schedule.json")
    legacy = locked_schedule.get("schema") == "pluto.file02.q02.schedule.v1"
    require(plan.get("schema") == "pluto.file02.q02.plan.v1" and plan.get("status") == "LOCKED" and
            plan.get("case_id") == "Q02" and plan.get("source_tree") ==
            r01.git("rev-parse", plan["source_commit"] + "^{tree}") and
            plan.get("host") == r01.cpu_inventory() and
            plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("config_sha256") == common.digest(session / "raw/config.json") ==
            common.digest(ROOT / "research/config/s06_pilot.json") and
            plan.get("schedule_sha256") == common.digest(session / "raw/schedule.json") and
            locked_schedule == schedule(legacy=legacy) and
            plan.get("schedule_schema") == (None if legacy else "pluto.file02.q02.schedule.v2") and
            plan.get("profiles") == list(PROFILES) and plan.get("wire_bytes") == WIRE and
            plan.get("n_blocks") == 12 and plan.get("n_networks") == 96 and
            plan.get("total_tx_s") == 20 and plan.get("deadline_ms") == 3000 and
            plan.get("ci") == "block_percentile_bootstrap_20000_seed_20261114", "Q02 locked plan drift")


def bind_run_plan(run, network_root):
    run, network_root = Path(run).resolve(), Path(network_root).resolve()
    require(run.is_dir() and network_root.is_dir() and not (run / "research_plan/plan.json").exists(),
            "Q02 run/network path")
    meta, workload, nodes = (common.read_json(run / name) for name in
                             ("metadata.json", "plan.json", "nodes.json"))
    profile, condition = meta.get("profile"), meta.get("condition")
    require(meta.get("case_id") == "Q02" and meta.get("scope") == "Q_CONFIRMATORY" and
            profile in PROFILES and condition in ("benign", "attack") and len(nodes) == 4 and
            workload.get("run_id") == meta.get("run_id") and
            workload.get("profile") == PROFILE_CODE[profile] and
            workload.get("regime") == ("equal_tx_s" if condition == "benign" else
                                       "fixed_legitimate_plus_adversarial") and
            workload.get("waveform") == "constant" and len(workload.get("events", [])) == 241 and
            workload.get("warmup_ns") == 1_000_000_000 and
            workload.get("steady_ns") == 12_000_000_000 and
            workload.get("drain_ns") == 5_000_000_000 and
            all(n["rpc"].startswith("http://127.0.0.1:") for n in nodes), "Q02 binding")
    config = common.read_json(run / "config.json")
    require(common.digest(run / "config.json") == meta["config_sha256"] ==
            common.digest(ROOT / "research/config/s06_pilot.json"), "Q02 safety config")
    network, safety = config["network"], config["safety"]
    inventory = r01.cpu_inventory()
    host = {"cpu_model": inventory["model"], "cpu_features": inventory["flags"],
            "architecture": inventory["architecture"], "logical_cpus": inventory["logical_cpus"],
            "os": inventory["os"]}
    research_plan = {"schema": "pluto.file02.plan.v1", "status": "LOCKED",
        "run_id": workload["run_id"], "case_id": "Q02", "protocol_version": "file02-common-v1",
        "research_question": "RQ1b", "hypothesis": "invalid background changes fixed legitimate service by profile",
        "null_hypothesis": "profile-specific benign-minus-attack timely fraction <= 0",
        "alternative_hypothesis": "profile-specific benign-minus-attack timely fraction > 0",
        "analysis_plan": {"primary_endpoint": "fixed_legitimate_all_four_observed_within_3000ms_fraction",
                          "independent_unit": "fresh_four_validator_network",
                          "ci_method": "block_percentile_bootstrap_20000_seed_20261114",
                          "exclusion_rule": "failed arm excludes eight-arm fixture block; no replacement"},
        "raw_capture_plan": ["telemetry", "transaction_results", "validator_blocks_and_apphash",
                             "loadgen_timeline", "host_metrics"],
        "expected_oracle": "73/241 valid exact-once; 168 structural rejects in attack; all-four agreement",
        "source_commit": meta["source_commit"], "source_tree": meta["source_tree"],
        "config_sha256": meta["config_sha256"], "corpus_sha256": common.digest(run / "plan.json"),
        "genesis_sha256": workload["genesis_sha256"], "profile": profile,
        "workload": "BENIGN_HIGH" if condition == "benign" else "STRUCT_INVALID",
        "actor_capability": "NONE" if condition == "benign" else "A",
        "regime": workload["regime"], "traffic_source": "RPC",
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
        "planned": {"valid_tx_s": 20 if condition == "benign" else 6,
                    "invalid_tx_s": 0 if condition == "benign" else 14,
                    "total_bytes_s": 20*WIRE[profile]},
        "matching": {"matched_variable": "tx_s", "unmatched_variables":
                     ["profile_wire_bytes", "valid_background_inclusion", "verification_stage"],
                     "legitimate_stream_id": f"Q02-b{meta['block']:02d}-{profile}-fixed"},
        "pilot_reference": "research/evidence/file02/T06_FORMAL_CALIBRATION_AUDIT.md",
        "independent_run_target": 12,
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
    profile, condition = job["profile"], job["condition"]
    require(meta.get("schema") == "pluto.file02.q02_run.v1" and meta.get("scope") == "Q_CONFIRMATORY" and
            all(meta.get(k) == job[k] for k in ("run_id", "block", "order", "profile", "condition")) and
            meta.get("source_commit") == bound["source_commit"] == plan["source_commit"] and
            meta.get("source_tree") == bound["source_tree"] == plan["source_tree"] and
            bound["run_id"] == workload["run_id"] == result["run_id"] == oracle["run_id"] ==
            commits["run_id"] and bound["profile"] == profile and
            bound["topology"] == "FOUR_VALIDATOR_SINGLE_HOST_DIAGNOSTIC" and
            bound["traffic_source"] == "RPC" and
            bound["corpus_sha256"] == result["plan_sha256"] == common.digest(run / "plan.json") and
            bound["genesis_sha256"] == result["genesis_sha256"] == workload["genesis_sha256"] ==
            common.digest(run / "genesis.bin") and
            bound["config_sha256"] == result["config_sha256"] == plan["config_sha256"] ==
            common.digest(run / "config.json"), "Q02 provenance")
    require(result["status"] == oracle["status"] == "OK" and
            result["planned_count"] == result["actual_send_count"] == len(result["events"]) ==
            len(workload["events"]) == 241 and
            oracle["valid_committed"] == (241 if condition == "benign" else 73) and
            oracle["invalid_rejected"] == (0 if condition == "benign" else 168) and
            len(commits["events"]) == oracle["valid_committed"] and
            oracle["block_id"] and oracle["app_hash"], "Q02 execution/oracle")
    by_commit = {event["id"]: event for event in commits["events"]}
    require(len(by_commit) == len(commits["events"]), "Q02 commit identity")
    records = {event["tx_sha256"]: event for event in r06.samples(run / "recorder-0.jsonl")
               if event.get("check_type") == "new" and event.get("tx_sha256")}
    require(len(records) == 241, "Q02 CheckTx recorder coverage")
    fixed, fixed_offsets, fixed_slots, windows, latencies, distinct = [], [], [], [0]*12, [], set()
    for expected, actual in zip(workload["events"], result["events"]):
        wire = bytes.fromhex(expected["bytes_hex"])
        digest = hashlib.sha256(wire).hexdigest()
        require(expected["id"] == actual["id"] and expected["sha256"] == digest and
                digest not in distinct and actual["wire_bytes"] == len(wire) == WIRE[profile] and
                actual["actual_code"] == expected["expected_code"] and
                not actual.get("submission_error") and
                isinstance(actual.get("actual_send_monotonic_ns"), int) and
                records[digest]["code"] == expected["expected_code"] and
                records[digest]["last_stage"] == expected["expected_stage"],
                "Q02 send/hash/size/code/stage")
        distinct.add(digest)
        if expected["traffic"] == "adversarial":
            require(condition == "attack" and expected["scenario"] == "Q02_structural" and
                    expected["expected_code"] == 1 and expected["expected_stage"] == "structural" and
                    expected["id"] not in by_commit and
                    not any(s.get("stage") in ("secp", "ml") for s in records[digest].get("spans", [])),
                    "Q02 structural feasibility")
        else:
            require(by_commit[expected["id"]]["sha256"] == digest, "Q02 valid commit")
        if expected["scenario"] == "fixed_legitimate":
            fixed.append(digest)
            fixed_offsets.append(expected["offset_ns"])
            fixed_slots.append(expected["account_slot"])
            if expected["offset_ns"] >= 1_000_000_000:
                seen = by_commit[expected["id"]].get("all_four_seen_monotonic_ns")
                require(isinstance(seen, int) and seen >= actual["actual_send_monotonic_ns"],
                        "Q02 fixed observation/clock")
                latencies.append((seen-actual["actual_send_monotonic_ns"])/1_000_000)
        if expected["offset_ns"] >= 1_000_000_000:
            second = (actual["actual_offset_ns"]-1_000_000_000)//1_000_000_000
            require(0 <= second < 12, "Q02 actual window")
            windows[second] += 1
    require(len(distinct) == 241 and len(fixed) == 73 and len(latencies) == 72 and
            windows == [20]*12, "Q02 fixed stream/window rate")
    for node in range(4):
        r06.observer_metrics(run / f"observer-{node}.jsonl")
    timely = sum(x <= plan["deadline_ms"] for x in latencies)
    return {"run_id": job["run_id"], "block": job["block"], "order": job["order"],
            "profile": profile, "condition": condition, "seed": workload["seed"],
            "genesis_sha256": workload["genesis_sha256"], "fixed_hashes": fixed,
            "fixed_offsets_ns": fixed_offsets, "fixed_slots": fixed_slots,
            "wire_bytes": WIRE[profile],
            "offered_tx_s": 20, "offered_bytes_s": 20*WIRE[profile],
            "fixed_eligible": 72, "fixed_timely": timely,
            "fixed_timely_fraction": timely/72, "fixed_inclusion_upper_ms": latencies,
            "valid_committed": oracle["valid_committed"],
            "invalid_rejected": oracle["invalid_rejected"], "qc_status": "PASS"}


def ci(values):
    require(len(values) == 12, "Q02 twelve independent fixture blocks")
    rng = random.Random(SEED)
    draws = [statistics.mean(values[rng.randrange(12)] for _ in range(12))
             for _ in range(20_000)]
    return [r01.percentile(draws, .025), r01.percentile(draws, .975)]


def block_contrasts(group):
    require(len(group) == 8 and {row["profile"] for row in group} == set(PROFILES) and
            len({row["genesis_sha256"] for row in group}) == 1 and
            len({row["seed"] for row in group}) == 1, "Q02 eight-arm fixture block incomplete")
    by = {(row["profile"], row["condition"]): row for row in group}
    require(len(by) == 8 and all(by[(p, "benign")]["fixed_hashes"] ==
            by[(p, "attack")]["fixed_hashes"] and
            by[(p, "benign")]["fixed_offsets_ns"] ==
            by[(p, "attack")]["fixed_offsets_ns"] for p in PROFILES),
            "Q02 matched profile pair drift")
    require(len({tuple(row["fixed_offsets_ns"]) for row in group}) == 1 and
            len({tuple(row["fixed_slots"]) for row in group}) == 1,
            "Q02 cross-profile intent schedule drift")
    d = {p: by[(p, "benign")]["fixed_timely_fraction"] -
            by[(p, "attack")]["fixed_timely_fraction"] for p in PROFILES}
    return {"block": group[0]["block"], "effects_benign_minus_attack": d,
            "did_H_C": d["C"]-d["H"],
            "condition_contrasts": {condition: {
                "CP_minus_C": by[("CP", condition)]["fixed_timely_fraction"] -
                              by[("C", condition)]["fixed_timely_fraction"],
                "H_minus_CP": by[("H", condition)]["fixed_timely_fraction"] -
                              by[("CP", condition)]["fixed_timely_fraction"],
                "H_minus_P": by[("H", condition)]["fixed_timely_fraction"] -
                             by[("P", condition)]["fixed_timely_fraction"]}
                for condition in ("benign", "attack")}}


def derive(session, plan):
    manifest = common.verify_raw(session / "raw")
    rows, failed, blocks = [], [], []
    for job in common.read_json(session / "raw/schedule.json")["jobs"]:
        try:
            rows.append(audit_run(session / "raw", job, plan))
        except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError) as exc:
            failed.append({"run_id": job["run_id"], "reason": str(exc)})
    for block in range(12):
        try:
            blocks.append(block_contrasts([row for row in rows if row["block"] == block]))
        except (ValueError, KeyError, TypeError) as exc:
            failed.append({"block": block, "reason": str(exc)})
    status = "PASS" if len(rows) == 96 and len(blocks) == 12 and not failed else "INVALID_FOR_INFERENCE"
    qc = {"schema": "pluto.file02.q02.qc.v1", "case_id": "Q02", "status": status,
          "n_planned": 96, "n_valid": len(rows), "failed": failed,
          "raw_manifest_sha256": manifest, "plan_sha256": common.digest(session / "plan.json")}
    aggregate = {"schema": "pluto.file02.q02.aggregate.v1", "case_id": "Q02",
        "status": "ANALYZED" if status == "PASS" else "WITHHELD_QC_FAILURE",
        "n_blocks": 12 if status == "PASS" else None,
        "profile_effects_fraction": {p: {"mean": statistics.mean(
            b["effects_benign_minus_attack"][p] for b in blocks), "ci95": ci([
            b["effects_benign_minus_attack"][p] for b in blocks])} for p in PROFILES}
            if status == "PASS" else None,
        "did_H_C_fraction": {"mean": statistics.mean(b["did_H_C"] for b in blocks),
                             "ci95": ci([b["did_H_C"] for b in blocks])} if status == "PASS" else None,
        "condition_contrasts_fraction": {condition: {name: {"mean": statistics.mean(
            b["condition_contrasts"][condition][name] for b in blocks), "ci95": ci([
            b["condition_contrasts"][condition][name] for b in blocks])}
            for name in ("CP_minus_C", "H_minus_CP", "H_minus_P")}
            for condition in ("benign", "attack")} if status == "PASS" else None,
        "wire_bytes": WIRE, "cross_profile_byte_match": {"CP_H": True, "C_H": False, "P_H": False},
        "causal_limit": "STRUCTURAL_INVALID_ONLY; C/P_WIRE_SIZE_CONFOUND",
        "raw_manifest_sha256": manifest, "plan_sha256": qc["plan_sha256"]}
    return rows, blocks, qc, aggregate


def csv_text(rows):
    buffer = StringIO()
    fields = ("run_id", "block", "order", "profile", "condition", "seed",
              "wire_bytes", "offered_tx_s", "offered_bytes_s", "fixed_eligible",
              "fixed_timely", "fixed_timely_fraction", "valid_committed",
              "invalid_rejected", "qc_status")
    writer = csv.DictWriter(buffer, fieldnames=fields, lineterminator="\n")
    writer.writeheader()
    writer.writerows({key: row[key] for key in fields} for row in rows)
    return buffer.getvalue()


def decision_text(qc, aggregate):
    return (f"# Q02 cross-profile structural-invalid decomposition\n\n"
            f"QC: {qc['status']}; {qc['n_valid']}/96 arms. "
            f"H/C DiD: {aggregate['did_H_C_fraction']}. "
            "This is the 3-s all-four observation fraction, not exact commit time. "
            "C/P wire sizes differ from H; no ML-specific, knee or DoS inference.\n")


def write_outputs(session, plan):
    rows, blocks, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.q02.per_run.v1", "rows": rows})
    common.write_once(session / "result_per_run.csv", csv_text(rows))
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    r01.write_new(session / "paired_blocks.json", {"schema": "pluto.file02.q02.blocks.v1", "blocks": blocks})
    (session / "figures").mkdir(mode=0o700)
    r01.write_new(session / "figures/status.json", {"status": "PENDING_CASE_REVIEW",
        "reason": "numeric contrasts retained without scientific plot"})
    common.write_once(session / "decision.md", decision_text(qc, aggregate))
    return qc, aggregate


def execute():
    session = ARTIFACTS / ("q02-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") +
                           "-" + uuid4().hex[:8])
    plan = prepare(session)
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT,
                               stdout=stream, stderr=subprocess.STDOUT)
    require(build.returncode == 0, f"build failed: {session}")
    env = os.environ.copy()
    env.update(FILE02_Q02_ROOT=str(raw), FILE02_SOURCE_COMMIT=plan["source_commit"],
               FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02Q02Decomposition$", "-timeout", "7200s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=7300)
    manifest = common.freeze_raw(raw)
    qc, aggregate = write_outputs(session, plan)
    print(f"Q02 session={session} integration_exit={run.returncode} qc={qc['status']} "
          f"did={aggregate['did_H_C_fraction']} raw_manifest_sha256={manifest}")
    return 0 if run.returncode == 0 and qc["status"] == "PASS" else 2


def verify(session):
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve() and session.is_dir(), "private Q02 session")
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    rows, blocks, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.q02.per_run.v1", "rows": rows} and
            (session / "result_per_run.csv").read_text() == csv_text(rows) and
            common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate and
            common.read_json(session / "paired_blocks.json") ==
            {"schema": "pluto.file02.q02.blocks.v1", "blocks": blocks} and
            (session / "decision.md").read_text() == decision_text(qc, aggregate) and
            common.read_json(session / "figures/status.json")["status"] == "PENDING_CASE_REVIEW",
            "Q02 derived artifact mismatch")
    print(f"Q02_VERIFY=OK qc={qc['status']} raw_manifest_sha256={qc['raw_manifest_sha256']}")


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
    except (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError) as exc:
        print(f"Q02_ERROR={exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
