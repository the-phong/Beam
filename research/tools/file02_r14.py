#!/usr/bin/env python3
"""Execute and audit R14's bounded CheckTx recheck case."""

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
SEED = 20261113


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
        for index, profile in enumerate(order[block:] + order[:block]):
            jobs.append({"block": block, "order": index, "profile": profile, "seed": seed,
                         "run_id": f"r14-b{block:02d}-{profile.lower()}"})
    return {"schema": "pluto.file02.r14.schedule.v1", "master_seed": SEED,
            "method": "four_rotated_blocks", "jobs": jobs}


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve() and not r01.git("status", "--porcelain"),
            "private session or dirty source")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT).returncode == 0,
            "session not ignored")
    session.mkdir(mode=0o700)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    common.write_once(raw / "schedule.json", json.dumps(schedule(), sort_keys=True, indent=2) + "\n")
    original = ROOT / "research/config/s06_pilot.json"
    common.write_once(raw / "config.json", original.read_text())
    plan = {"schema": "pluto.file02.r14.plan.v1", "status": "LOCKED", "case_id": "R14",
            "source_commit": r01.git("rev-parse", "HEAD"), "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
            "created_utc": datetime.now(timezone.utc).isoformat(), "host": r01.cpu_inventory(),
            "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
            "module_sha256": common.digest(ROOT / "go.mod"),
            "config_sha256": common.digest(raw / "config.json"),
            "schedule_sha256": common.digest(raw / "schedule.json"),
            "protocol": "research/protocol/FILE02_R14_LOCKED_PLAN.md",
            "chain_id": "pluto-file02-r14", "profiles": list(PROFILES),
            "n_blocks": 4, "n_networks": 16, "offered_valid_tx_s": 20,
            "warmup_seconds": 1, "steady_seconds": 4, "drain_seconds": 3, "cooldown_seconds": 2,
            "primary": "validator0_CheckTx_recheck_count",
            "ci": "percentile_bootstrap_20000_seed_20261113"}
    r01.write_new(session / "plan.json", plan)
    return plan


def validate_plan(session, plan):
    require(plan.get("schema") == "pluto.file02.r14.plan.v1" and plan.get("status") == "LOCKED" and
            plan.get("case_id") == "R14" and
            plan.get("source_tree") == r01.git("rev-parse", plan["source_commit"] + "^{tree}") and
            plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("host") == r01.cpu_inventory() and
            plan.get("schedule_sha256") == common.digest(session / "raw/schedule.json") and
            common.read_json(session / "raw/schedule.json") == schedule() and
            plan.get("config_sha256") == common.digest(session / "raw/config.json") ==
            common.digest(ROOT / "research/config/s06_pilot.json") and
            plan.get("n_blocks") == 4 and plan.get("n_networks") == 16 and
            plan.get("offered_valid_tx_s") == 20, "R14 locked plan drift")


def audit_run(raw, job, plan):
    run = raw / "runs" / job["run_id"]
    meta, workload, result, oracle, common_plan, nodes = (common.read_json(run / name) for name in
        ("metadata.json", "plan.json", "result.json", "oracle.json", "research_plan/plan.json", "nodes.json"))
    common.validate_plan(common_plan)
    require(meta.get("schema") == "pluto.file02.r14_run.v1" and meta.get("scope") == "R_CONFIRMATORY" and
            meta.get("case_id") == common_plan.get("case_id") == "R14" and
            all(meta.get(k) == job[k] for k in ("block", "order", "profile", "seed")) and
            meta.get("source_commit") == common_plan.get("source_commit") == plan["source_commit"] and
            meta.get("source_tree") == common_plan.get("source_tree") == plan["source_tree"] and
            meta.get("run_id") == workload.get("run_id") == result.get("run_id") == oracle.get("run_id") ==
            common_plan.get("run_id") == job["run_id"], "R14 identity/source")
    require(result.get("plan_sha256") == common_plan.get("corpus_sha256") == common.digest(run / "plan.json") and
            result.get("genesis_sha256") == common_plan.get("genesis_sha256") ==
            workload.get("genesis_sha256") == common.digest(run / "genesis.bin") and
            result.get("config_sha256") == common_plan.get("config_sha256") ==
            plan["config_sha256"] == common.digest(run / "config.json") and
            common_plan["network_config"]["mempool_recheck"] is True and
            common_plan["network_config"]["rpc_endpoints"] == [node["rpc"] for node in nodes],
            "R14 runtime provenance")
    require(workload.get("regime") == "equal_tx_s" and workload.get("waveform") == "constant" and
            workload.get("target_tx_per_second") == 20 and len(workload.get("events", [])) == 81 and
            result.get("planned_count") == result.get("actual_send_count") == 81 and
            result.get("status") == "OK" and oracle.get("status") == "OK" and
            oracle.get("valid_committed") == 81 and oracle.get("invalid_rejected") == 0 and
            oracle.get("common_height", 0) >= 8 and oracle.get("block_id") and oracle.get("app_hash"),
            "R14 execution/oracle")
    windows = [0]*4
    planned = {event["sha256"] for event in workload["events"]}
    require(len(planned) == 81, "duplicate planned transaction")
    for expected, actual in zip(workload["events"], result["events"]):
        require(expected["id"] == actual["id"] and actual.get("actual_code") == 0 and
                actual.get("submission_error", "") == "" and actual.get("actual_offset_ns") is not None,
                "R14 offered transaction mismatch")
        if expected["offset_ns"] >= 1_000_000_000:
            window = (actual["actual_offset_ns"]-1_000_000_000)//1_000_000_000
            require(0 <= window < 4, "R14 send outside steady window")
            windows[window] += 1
    require(windows == [20]*4, "R14 rate windows")
    for index in range(4):
        samples = r06.samples(run / f"observer-{index}.jsonl")
        require(samples and all(s.get("kind") == "node_sample" and s.get("dropped_samples") == 0 for s in samples),
                "R14 observer missing/drop")
        recorder = r06.samples(run / f"recorder-{index}.jsonl")
        require(recorder and all(e.get("kind") == "check_tx" for e in recorder), "R14 recorder missing")
    samples = r06.samples(run / "observer-0.jsonl")
    recorder = r06.samples(run / "recorder-0.jsonl")
    new = [e for e in recorder if e.get("check_type") == "new"]
    recheck = [e for e in recorder if e.get("check_type") == "recheck"]
    require(len(new) == 81 and {e.get("tx_sha256") for e in new} == planned and
            all(e.get("code") == 0 for e in new) and
            all(e.get("tx_sha256") in planned and isinstance(e.get("code"), int) and
                isinstance(e.get("service_ns"), int) for e in recheck),
            "R14 NEW/RECHECK attribution")
    metric = max((s.get("recheck_total") or 0) for s in samples)
    require(metric == len(recheck), "Comet recheck counter/recorder mismatch")
    curve = [{"observed_utc": s["observed_utc"], "accepted_count": s.get("accepted_mempool_count"),
              "accepted_bytes": s.get("accepted_mempool_bytes"), "recheck_total": s.get("recheck_total")}
             for s in samples]
    return {"run_id": job["run_id"], "block": job["block"], "order": job["order"],
            "profile": job["profile"], "seed": job["seed"],
            "genesis_sha256": workload["genesis_sha256"], "new_count": len(new),
            "recheck_count": len(recheck), "recheck_code_ok": sum(e["code"] == 0 for e in recheck),
            "recheck_code_reject": sum(e["code"] != 0 for e in recheck),
            "recheck_mean_service_ns": statistics.mean(e["service_ns"] for e in recheck) if recheck else None,
            "comet_recheck_counter": metric, "mempool_curve": curve, "qc_status": "PASS"}


def ci(values):
    require(len(values) == 4, "four independent networks")
    rng = random.Random(SEED)
    draws = [statistics.mean(values[rng.randrange(4)] for _ in range(4)) for _ in range(20_000)]
    return [r01.percentile(draws, .025), r01.percentile(draws, .975)]


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
        group = [r for r in rows if r["block"] == block]
        if len(group) == 4 and len({r["genesis_sha256"] for r in group}) != 1:
            failures.append({"block": block, "reason": "unmatched fixture"})
    if len({r["genesis_sha256"] for r in rows}) != 4:
        failures.append({"reason": "independent genesis count"})
    qc = {"schema": "pluto.file02.r14.qc.v1", "case_id": "R14",
          "status": "PASS" if len(rows) == 16 and not failures else "INVALID_FOR_INFERENCE",
          "n_planned": 16, "n_valid": len(rows), "failed": failures,
          "plan_sha256": common.digest(session / "plan.json"), "raw_manifest_sha256": manifest}
    aggregate = {"schema": "pluto.file02.r14.aggregate.v1", "case_id": "R14",
                 "status": "PENDING_VALID_QC", "n_networks": len(rows),
                 "profile_summary": {}, "plan_sha256": qc["plan_sha256"], "raw_manifest_sha256": manifest}
    if qc["status"] == "PASS":
        for profile in PROFILES:
            selected = sorted((r for r in rows if r["profile"] == profile), key=lambda r: r["block"])
            counts = [r["recheck_count"] for r in selected]
            aggregate["profile_summary"][profile] = {
                "n_networks": 4, "new_counts": [r["new_count"] for r in selected],
                "recheck_counts": counts, "mean_recheck_count": statistics.mean(counts),
                "ci95_mean_recheck_count": ci(counts),
                "recheck_code_ok": sum(r["recheck_code_ok"] for r in selected),
                "recheck_code_reject": sum(r["recheck_code_reject"] for r in selected)}
        aggregate["status"] = ("RECHECK_OBSERVED" if all(r["recheck_count"] > 0 for r in rows)
                               else "RECHECK_NOT_OBSERVED_IN_ALL_RUNS")
    return rows, qc, aggregate


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.r14.per_run.v1", "rows": rows})
    with (session / "result_per_run.csv").open("x", newline="", encoding="utf-8") as stream:
        keys = ("run_id", "block", "profile", "seed", "new_count", "recheck_count",
                "recheck_code_ok", "recheck_code_reject", "recheck_mean_service_ns", "comet_recheck_counter", "qc_status")
        writer = csv.DictWriter(stream, fieldnames=keys)
        writer.writeheader()
        writer.writerows({key: row[key] for key in keys} for row in rows)
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir()
    r01.write_new(session / "figures/status.json", {"status": "NOT_REQUIRED",
                                                     "reason": "numeric per-run queue curves and bootstrap aggregate"})
    (session / "decision.md").write_text(
        f"# R14 decision\n\nQC {qc['status']} on {len(rows)}/16 networks. Result {aggregate['status']}. "
        "Recheck events are internal mempool work, not new offered transactions. "
        "Exact trigger commit, Comet per-event CPU, and full queue residence are UNAVAILABLE. "
        "No knee or DoS inference.\n", encoding="utf-8")
    return qc, aggregate


def execute():
    session = ARTIFACTS / ("r14-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8])
    plan = prepare(session)
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT, stdout=stream, stderr=subprocess.STDOUT)
    require(build.returncode == 0, f"build failed: {session}")
    env = os.environ.copy()
    env.update(FILE02_R14_ROOT=str(raw), FILE02_SOURCE_COMMIT=plan["source_commit"], FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02R14$", "-timeout", "1200s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=1300)
    manifest = common.freeze_raw(raw)
    qc, aggregate = write_outputs(session, plan)
    print(f"R14 session={session} integration_exit={run.returncode} qc={qc['status']} "
          f"status={aggregate['status']} raw_manifest_sha256={manifest}")
    return 0 if run.returncode == 0 and qc["status"] == "PASS" else 2


def verify(session):
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve(), "private session")
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    rows, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.r14.per_run.v1", "rows": rows} and
            common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate,
            "R14 derived artifact mismatch")
    with (session / "result_per_run.csv").open(newline="", encoding="utf-8") as stream:
        csv_rows = list(csv.DictReader(stream))
    require(len(csv_rows) == len(rows) and all(item["run_id"] == row["run_id"] for item, row in zip(csv_rows, rows)),
            "R14 CSV mismatch")
    print(f"R14_VERIFY=OK qc={qc['status']} n={len(rows)} raw_manifest_sha256={qc['raw_manifest_sha256']}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("run")
    p = sub.add_parser("verify")
    p.add_argument("session")
    args = parser.parse_args()
    try:
        sys.exit(execute() if args.command == "run" else (verify(args.session) or 0))
    except (ValueError, OSError, KeyError, TypeError, subprocess.TimeoutExpired) as exc:
        print(f"R14_ERROR: {exc}", file=sys.stderr)
        sys.exit(2)
