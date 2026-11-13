#!/usr/bin/env python3
"""R09 valid-only mempool-byte limit experiment and raw-bound audit."""

import argparse
import csv
from datetime import datetime, timezone
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

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
PROFILES = r06.PROFILES
WIRE = r06.WIRE
SEED = 20261113
ARMS = {"baseline": 8_388_608, "bytes2700": 2700}


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def schedule():
    rng = random.Random(SEED)
    order = list(PROFILES)
    rng.shuffle(order)
    jobs = []
    for block in range(4):
        seed = rng.randrange(1, 2**63)
        rotated = order[block:] + order[:block]
        for profile in rotated:
            arms = ("baseline", "bytes2700") if block % 2 == 0 else ("bytes2700", "baseline")
            for arm in arms:
                jobs.append({"block": block, "order": len(jobs) % 8, "profile": profile,
                             "arm": arm, "seed": seed,
                             "run_id": f"r09-b{block:02d}-{profile.lower()}-{arm}"})
    return {"schema": "pluto.file02.r09.schedule.v1", "master_seed": SEED,
            "method": "four_rotated_blocks_alternating_byte_arm_order", "jobs": jobs}


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve(), "private session parent")
    require(not r01.git("status", "--porcelain"), "dirty source before lock")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT).returncode == 0,
            "session not Git-ignored")
    session.mkdir(mode=0o700)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    common.write_once(raw / "schedule.json", json.dumps(schedule(), indent=2, sort_keys=True) + "\n")
    original = ROOT / "research/config/s06_pilot.json"
    base = common.read_json(original)
    require(base["network"]["mempool_size"] == 128 and base["network"]["mempool_max_bytes"] == 8_388_608,
            "accepted baseline mempool config changed")
    common.write_once(raw / "config-baseline.json", original.read_text())
    changed = json.loads(json.dumps(base))
    changed["network"]["mempool_max_bytes"] = 2700
    common.write_once(raw / "config-bytes2700.json", json.dumps(changed, indent=2, sort_keys=True) + "\n")
    for arm, byte_cap in ARMS.items():
        c = common.read_json(raw / f"config-{arm}.json")
        require(c["safety"] == base["safety"] and c["network"]["mempool_max_bytes"] == byte_cap and
                all(c["network"][k] == base["network"][k] for k in base["network"] if k != "mempool_max_bytes"),
                "R09 changed a safety cap or other network variable")
    plan = {"schema": "pluto.file02.r09.plan.v1", "status": "LOCKED", "case_id": "R09",
            "source_commit": r01.git("rev-parse", "HEAD"), "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
            "created_utc": datetime.now(timezone.utc).isoformat(), "host": r01.cpu_inventory(),
            "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
            "module_sha256": common.digest(ROOT / "go.mod"),
            "original_safety_config_sha256": common.digest(original),
            "config_sha256": {arm: common.digest(raw / f"config-{arm}.json") for arm in ARMS},
            "schedule_sha256": common.digest(raw / "schedule.json"),
            "protocol": "research/protocol/FILE02_R09_LOCKED_PLAN.md",
            "chain_id": "pluto-file02-r09", "profiles": list(PROFILES), "wire_bytes": WIRE,
            "arms": ARMS, "n_blocks": 4, "n_networks": 32, "offered_valid_tx_s": 20,
            "warmup_seconds": 1, "steady_seconds": 4, "drain_seconds": 3, "cooldown_seconds": 2,
            "primary": "first_mempool_bytes_full_offset_ns",
            "ci": "paired_percentile_bootstrap_20000_seed_20261113"}
    r01.write_new(session / "plan.json", plan)
    return plan


def validate_plan(session, plan):
    require(plan.get("schema") == "pluto.file02.r09.plan.v1" and plan.get("status") == "LOCKED" and
            plan.get("case_id") == "R09" and plan.get("source_tree") == r01.git("rev-parse", plan["source_commit"] + "^{tree}"),
            "source/status")
    require(plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("original_safety_config_sha256") == common.digest(ROOT / "research/config/s06_pilot.json") and
            plan.get("host") == r01.cpu_inventory() and
            plan.get("schedule_sha256") == common.digest(session / "raw/schedule.json") and
            common.read_json(session / "raw/schedule.json") == schedule(), "source/config/schedule drift")
    require(plan.get("arms") == ARMS and plan.get("n_blocks") == 4 and plan.get("n_networks") == 32 and
            plan.get("offered_valid_tx_s") == 20 and plan.get("steady_seconds") == 4 and
            plan.get("wire_bytes") == WIRE, "design drift")
    base = common.read_json(ROOT / "research/config/s06_pilot.json")
    for arm, byte_cap in ARMS.items():
        config = session / "raw" / f"config-{arm}.json"
        value = common.read_json(config)
        require(plan["config_sha256"][arm] == common.digest(config) and
                value["safety"] == base["safety"] and value["network"]["mempool_max_bytes"] == byte_cap and
                all(value["network"][k] == base["network"][k] for k in base["network"] if k != "mempool_max_bytes"),
                "arm config changed")


def bootstrap(values):
    require(len(values) == 4, "four matched blocks required")
    rng = random.Random(SEED)
    draws = [statistics.mean(values[rng.randrange(4)] for _ in range(4)) for _ in range(20_000)]
    return [r01.percentile(draws, .025), r01.percentile(draws, .975)]


def audit_run(raw, job, plan):
    run = raw / "runs" / job["run_id"]
    meta, workload, result, oracle, common_plan, nodes = (common.read_json(run / name) for name in
        ("metadata.json", "plan.json", "result.json", "oracle.json", "research_plan/plan.json", "nodes.json"))
    common.validate_plan(common_plan)
    arm, profile = job["arm"], job["profile"]
    require(meta.get("schema") == "pluto.file02.r09_run.v1" and meta.get("scope") == "R_CONFIRMATORY" and
            meta.get("case_id") == common_plan.get("case_id") == "R09" and meta.get("recorder_mode") == "ON" and
            all(meta.get(k) == job[k] for k in ("block", "order", "profile", "arm", "seed")) and
            meta.get("source_commit") == common_plan.get("source_commit") == plan["source_commit"] and
            meta.get("source_tree") == common_plan.get("source_tree") == plan["source_tree"], "metadata/source")
    require(meta["run_id"] == workload["run_id"] == result["run_id"] == oracle["run_id"] ==
            common_plan["run_id"] == job["run_id"] and
            result["plan_sha256"] == common_plan["corpus_sha256"] == common.digest(run / "plan.json") and
            result["genesis_sha256"] == common_plan["genesis_sha256"] == workload["genesis_sha256"] == common.digest(run / "genesis.bin") and
            result["config_sha256"] == common_plan["config_sha256"] == plan["config_sha256"][arm] == common.digest(run / "config.json") and
            common_plan["network_config"]["mempool_size"] == 128 and
            common_plan["network_config"]["mempool_max_bytes"] == ARMS[arm] and
            common_plan["network_config"]["rpc_endpoints"] == [node["rpc"] for node in nodes] and
            common_plan["seed"] == job["seed"] and common_plan["order_index"] == job["order"], "provenance")
    require(workload["regime"] == "equal_tx_s" and workload["target_tx_per_second"] == 20 and
            workload["warmup_ns"] == 1_000_000_000 and workload["steady_ns"] == 4_000_000_000 and
            workload["drain_ns"] == 3_000_000_000 and len(workload["events"]) ==
            result["planned_count"] == result["actual_send_count"] == 81 and
            oracle["schema"] == "pluto.file02.r09_oracle.v1" and oracle["status"] == "OK" and
            oracle["block_id"] and oracle["app_hash"] and oracle["common_height"] >= 8, "execution/oracle")
    accepted, rejected = oracle["valid_accepted"], oracle["valid_rejected_bytes_full"]
    require(accepted + rejected == 81 and accepted >= 1 and
            result["status"] == ("OK" if rejected == 0 else "ORACLE_MISMATCH"), "admission totals/status")
    windows = [0]*4
    hashes, slots = set(), set()
    bytes_full = []
    for expected, actual in zip(workload["events"], result["events"]):
        require(expected["sha256"] not in hashes and expected["account_slot"] not in slots and
                expected["id"] == actual["id"] and actual["wire_bytes"] == WIRE[profile] and
                actual["actual_offset_ns"] is not None and 0 <= actual["actual_offset_ns"] < 5_000_000_000 and
                (actual["actual_offset_ns"] < 1_000_000_000) == (expected["offset_ns"] < 1_000_000_000),
                "send identity/phase")
        hashes.add(expected["sha256"])
        slots.add(expected["account_slot"])
        if expected["offset_ns"] >= 1_000_000_000:
            windows[(actual["actual_offset_ns"]-1_000_000_000)//1_000_000_000] += 1
        if actual["reason"] == "mempool_bytes_full":
            bytes_full.append(actual["actual_offset_ns"])
        else:
            require(actual["actual_code"] == 0 and actual.get("submission_error", "") == "", "unexpected rejection reason")
    require(windows == [20]*4 and len(bytes_full) == rejected and
            oracle["first_bytes_full_offset_ns"] == (bytes_full[0] if bytes_full else None),
            "offered rate/onset mismatch")
    samples = r06.samples(run / "observer-0.jsonl")
    require(samples and all(sample.get("kind") == "node_sample" and sample.get("dropped_samples") == 0 for sample in samples),
            "observer drop/missing")
    curve = [{"observed_utc": sample["observed_utc"], "accepted_count": sample.get("accepted_mempool_count"),
              "accepted_bytes": sample.get("accepted_mempool_bytes")} for sample in samples]
    observed = r06.observer_metrics(run / "observer-0.jsonl")
    recorder = r06.samples(run / "recorder-0.jsonl")
    require(recorder and all(item.get("kind") == "check_tx" for item in recorder), "recorder missing")
    return {"run_id": job["run_id"], "block": job["block"], "order": job["order"],
            "profile": profile, "arm": arm, "seed": job["seed"],
            "genesis_sha256": workload["genesis_sha256"], "wire_bytes": WIRE[profile],
            "offered_tx_s_windows": windows, "offered_bytes_s_windows": [n*WIRE[profile] for n in windows],
            "valid_accepted": accepted, "accepted_bytes": accepted*WIRE[profile],
            "valid_rejected_bytes_full": rejected, "first_bytes_full_offset_ms":
            oracle["first_bytes_full_offset_ns"]/1_000_000 if rejected else None,
            "mempool_bytes_curve": curve, "max_sampled_mempool_count": observed["max_accepted_mempool_count"],
            "max_sampled_mempool_bytes": observed["max_accepted_mempool_bytes"],
            "qc_status": "PASS"}


def derive(session, plan):
    raw = session / "raw"
    manifest = common.verify_raw(raw)
    rows, failures = [], []
    for job in schedule()["jobs"]:
        try:
            rows.append(audit_run(raw, job, plan))
        except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError, IndexError) as exc:
            failures.append({"run_id": job["run_id"], "reason": str(exc)})
    for block in range(4):
        group = [row for row in rows if row["block"] == block]
        if len(group) == 8 and len({row["genesis_sha256"] for row in group}) != 1:
            failures.append({"block": block, "reason": "unmatched logical fixture"})
    if len({row["genesis_sha256"] for row in rows}) != 4:
        failures.append({"reason": "independent genesis count mismatch"})
    qc = {"schema": "pluto.file02.r09.qc.v1", "case_id": "R09",
          "status": "PASS" if len(rows) == 32 and not failures else "INVALID_FOR_INFERENCE",
          "n_planned": 32, "n_valid": len(rows), "failed": failures,
          "plan_sha256": common.digest(session / "plan.json"), "raw_manifest_sha256": manifest}
    aggregate = {"schema": "pluto.file02.r09.aggregate.v1", "case_id": "R09",
                 "status": "PENDING_VALID_QC", "n_networks": len(rows), "profile_summary": {},
                 "plan_sha256": qc["plan_sha256"], "raw_manifest_sha256": manifest}
    if qc["status"] == "PASS":
        all_binding = True
        for profile in PROFILES:
            baseline = {r["block"]: r for r in rows if r["profile"] == profile and r["arm"] == "baseline"}
            bytes2700 = {r["block"]: r for r in rows if r["profile"] == profile and r["arm"] == "bytes2700"}
            effects = [baseline[b]["valid_accepted"]-bytes2700[b]["valid_accepted"] for b in range(4)]
            onsets = [bytes2700[b]["first_bytes_full_offset_ms"] for b in range(4)]
            binding = all(baseline[b]["valid_rejected_bytes_full"] == 0 and
                          bytes2700[b]["valid_rejected_bytes_full"] > 0 and onsets[b] is not None and
                          bytes2700[b]["max_sampled_mempool_count"] < 128 and
                          bytes2700[b]["max_sampled_mempool_bytes"] <= 2700 for b in range(4))
            all_binding = all_binding and binding
            aggregate["profile_summary"][profile] = {
                "n_matched_blocks": 4, "byte_bound_observed_all_blocks": binding,
                "baseline_accepted_counts": [baseline[b]["valid_accepted"] for b in range(4)],
                "bytes2700_accepted_counts": [bytes2700[b]["valid_accepted"] for b in range(4)],
                "bytes2700_rejected_valid_counts": [bytes2700[b]["valid_rejected_bytes_full"] for b in range(4)],
                "bytes2700_first_full_offset_ms": onsets,
                "mean_baseline_minus_bytes2700_accepted": statistics.mean(effects),
                "ci95_baseline_minus_bytes2700_accepted": bootstrap(effects),
                "mean_bytes2700_accepted_bytes": statistics.mean(bytes2700[b]["accepted_bytes"] for b in range(4)),
                "mean_bytes2700_max_sampled_mempool_bytes": statistics.mean(bytes2700[b]["max_sampled_mempool_bytes"] for b in range(4))}
        aggregate["status"] = "BYTE_LIMIT_OBSERVED_BEFORE_COUNT" if all_binding else "BYTE_LIMIT_NOT_OBSERVED_ALL_BLOCKS"
    return rows, qc, aggregate


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.r09.per_run.v1", "rows": rows})
    with (session / "result_per_run.csv").open("x", newline="", encoding="utf-8") as stream:
        keys = ("run_id", "block", "profile", "arm", "seed", "valid_accepted", "accepted_bytes",
                "valid_rejected_bytes_full", "first_bytes_full_offset_ms", "max_sampled_mempool_count",
                "max_sampled_mempool_bytes", "qc_status")
        writer = csv.DictWriter(stream, fieldnames=keys)
        writer.writeheader()
        writer.writerows({key: row[key] for key in keys} for row in rows)
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir()
    r01.write_new(session / "figures/status.json", {"status": "NOT_REQUIRED",
                                                     "reason": "Per-run mempool curves and registered aggregate are numeric artifacts"})
    (session / "decision.md").write_text(
        f"# R09 decision\n\nQC {qc['status']} on {len(rows)}/32 arms. Result: {aggregate['status']}. "
        "Only the mempool byte constraint changed; S06 safety and count caps did not. "
        "These valid-only observations do not show a knee or DoS.\n", encoding="utf-8")
    return qc, aggregate


def execute():
    session = ARTIFACTS / ("r09-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8])
    plan = prepare(session)
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT, stdout=stream, stderr=subprocess.STDOUT)
    require(build.returncode == 0, f"build failed: {session}")
    env = os.environ.copy()
    env.update(FILE02_R09_ROOT=str(raw), FILE02_SOURCE_COMMIT=plan["source_commit"], FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02R09$", "-timeout", "1800s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=1900)
    manifest = common.freeze_raw(raw)
    qc, aggregate = write_outputs(session, plan)
    print(f"R09 session={session} integration_exit={run.returncode} qc={qc['status']} "
          f"status={aggregate['status']} raw_manifest_sha256={manifest}")
    return 0 if run.returncode == 0 and qc["status"] == "PASS" else 2


def verify(session):
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve(), "private session")
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    rows, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") == {"schema": "pluto.file02.r09.per_run.v1", "rows": rows} and
            common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate, "derived artifact mismatch")
    with (session / "result_per_run.csv").open(newline="", encoding="utf-8") as stream:
        csv_rows = list(csv.DictReader(stream))
    require(len(csv_rows) == len(rows) and all(row["run_id"] == item["run_id"] and
            row["valid_accepted"] == int(item["valid_accepted"]) for row, item in zip(rows,csv_rows)), "CSV mismatch")
    print(f"R09_VERIFY=OK qc={qc['status']} n={len(rows)} raw_manifest_sha256={qc['raw_manifest_sha256']}")


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
        print(f"R09_ERROR: {exc}", file=sys.stderr)
        sys.exit(2)
