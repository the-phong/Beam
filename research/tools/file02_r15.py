#!/usr/bin/env python3
"""Execute and audit the formal bounded R15 benign capacity ramp."""

import argparse
import csv
from datetime import datetime, timezone, timedelta
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
import file02_t06_knee as knee

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
PROFILES = r06.PROFILES
RATES = (20, 23, 25)
SEED = 20261113


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def jobs():
    result = []
    for rate in RATES:
        for repetition in range(3):
            for position in range(4):
                profile = PROFILES[(position+repetition) % 4]
                result.append({"rate": rate, "repetition": repetition, "position": position,
                               "profile": profile, "seed": 20000+rate*100+repetition*4+position,
                               "run_id": f"r15-r{rate:02d}-n{repetition}-{profile}"})
    return result


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve() and not r01.git("status", "--porcelain"),
            "private session or dirty source")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT).returncode == 0,
            "session not ignored")
    session.mkdir(mode=0o700)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    common.write_once(raw / "schedule.json", json.dumps({"schema": "pluto.file02.r15.schedule.v1", "jobs": jobs()},
                                                       sort_keys=True, indent=2) + "\n")
    original = ROOT / "research/config/s06_pilot.json"
    common.write_once(raw / "config.json", original.read_text())
    plan = {"schema": "pluto.file02.r15.plan.v1", "status": "LOCKED", "case_id": "R15",
            "source_commit": r01.git("rev-parse", "HEAD"), "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
            "created_utc": datetime.now(timezone.utc).isoformat(), "host": r01.cpu_inventory(),
            "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
            "module_sha256": common.digest(ROOT / "go.mod"),
            "config_sha256": common.digest(raw / "config.json"),
            "schedule_sha256": common.digest(raw / "schedule.json"),
            "protocol": "research/protocol/FILE02_R15_LOCKED_PLAN.md",
            "chain_id": "pluto-file02-r15", "profiles": list(PROFILES), "rates_tx_s": list(RATES),
            "n_repetitions": 3, "n_networks": 36, "warmup_seconds": 1,
            "steady_seconds": 12, "drain_seconds": 3, "cooldown_seconds": 2,
            "eligible_steady_seconds": 9, "alignment_grace_seconds": 3,
            "ci": "percentile_bootstrap_20000_seed_20261113"}
    r01.write_new(session / "plan.json", plan)
    return plan


def validate_plan(session, plan):
    require(plan.get("schema") == "pluto.file02.r15.plan.v1" and plan.get("status") == "LOCKED" and
            plan.get("case_id") == "R15" and plan.get("source_tree") ==
            r01.git("rev-parse", plan["source_commit"] + "^{tree}") and
            plan.get("host") == r01.cpu_inventory() and
            plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("config_sha256") == common.digest(session / "raw/config.json") ==
            common.digest(ROOT / "research/config/s06_pilot.json") and
            plan.get("schedule_sha256") == common.digest(session / "raw/schedule.json") and
            common.read_json(session / "raw/schedule.json") ==
            {"schema": "pluto.file02.r15.schedule.v1", "jobs": jobs()} and
            plan.get("rates_tx_s") == list(RATES) and plan.get("n_networks") == 36 and
            plan.get("eligible_steady_seconds") == 9, "R15 plan drift")


def row(raw, job, plan):
    run = raw / "runs" / job["run_id"]
    meta, common_plan, load_plan, result, commits = (common.read_json(run / name) for name in
        ("metadata.json", "research_plan/plan.json", "plan.json", "result.json", "commit_observations.json"))
    common.validate_plan(common_plan)
    require(meta.get("schema") == "pluto.file02.r15_run.v1" and meta.get("scope") == "R_CONFIRMATORY" and
            meta.get("case_id") == common_plan.get("case_id") == "R15" and
            all(meta.get(k) == job[k] for k in ("run_id", "rate", "repetition", "position", "profile", "seed")) and
            meta.get("source_commit") == common_plan.get("source_commit") == plan["source_commit"] and
            meta.get("source_tree") == common_plan.get("source_tree") == plan["source_tree"] and
            common_plan.get("config_sha256") == plan["config_sha256"] == common.digest(run / "config.json") and
            common_plan.get("corpus_sha256") == common.digest(run / "plan.json") and
            common_plan.get("genesis_sha256") == common.digest(run / "genesis.bin"),
            "R15 runtime plan/provenance")
    base = knee.row(run, plan["source_commit"], plan["source_tree"],
                    job["rate"], job["profile"], job["repetition"])
    require(base["steady_tx_windows"] == [job["rate"]]*12 and
            base["achieved_offered_bytes_s"] <= 65536, "R15 offered rate/safety")
    for i in range(4):
        events = r06.samples(run / f"recorder-{i}.jsonl")
        require(events and all(e.get("kind") == "check_tx" for e in events), "R15 recorder")
    by_id = {item["id"]: item for item in commits["events"]}
    require(len(by_id) == len(load_plan["events"]), "R15 commit identity")
    start = knee.parse_go_rfc3339(result["started_utc"])
    deadline = start + timedelta(seconds=13)
    eligible = 0
    timely = 0
    for event in load_plan["events"]:
        if 1_000_000_000 <= event["offset_ns"] < 10_000_000_000:
            eligible += 1
            if knee.parse_go_rfc3339(by_id[event["id"]]["all_four_seen_utc"]) < deadline:
                timely += 1
    require(eligible == 9*job["rate"], "R15 aligned denominator")
    base.update({"position": job["position"], "seed": job["seed"],
                 "eligible_send_count": eligible, "aligned_timely_count": timely,
                 "aligned_timely_fraction": timely/eligible, "qc_status": "PASS"})
    return base


def ci(values):
    require(len(values) == 3, "three independent networks")
    rng = random.Random(SEED)
    draws = [statistics.mean(values[rng.randrange(3)] for _ in range(3)) for _ in range(20_000)]
    return [r01.percentile(draws, .025), r01.percentile(draws, .975)]


def measured_mean_ci(cells, key):
    values = [r[key] for r in cells]
    if any(v is None for v in values):
        return None, None
    return statistics.mean(values), ci(values)


def decide(rows):
    output = {}
    for profile in PROFILES:
        cells = {rate: [r for r in rows if r["profile"] == profile and r["rate_tx_s"] == rate]
                 for rate in RATES}
        require(all(len(cells[rate]) == 3 for rate in RATES), "R15 cell count")
        p95_ref = statistics.median(r["inclusion_upper_bound_p95_ms"] for r in cells[20])
        waits = [r["checktx_socket_queue_wait_mean_ms"] for r in cells[20]]
        q_ref = statistics.median(waits) if all(v is not None and v > 0 for v in waits) else None
        reference_ok = sum(r["aligned_timely_fraction"] >= .95 for r in cells[20]) >= 2
        flags = {}
        for rate in (23, 25):
            aligned = sum(r["aligned_timely_fraction"] < .90 for r in cells[rate])
            latency_queue = sum(r["inclusion_upper_bound_p95_ms"] >= 2*p95_ref and
                                q_ref is not None and r["checktx_socket_queue_wait_mean_ms"] is not None and
                                r["checktx_socket_queue_wait_mean_ms"] >= 2*q_ref for r in cells[rate])
            flags[str(rate)] = {"aligned_goodput_runs": aligned, "latency_and_socket_queue_runs": latency_queue,
                                "qualifies": aligned >= 2 or latency_queue >= 2}
        status = ("NOT_IDENTIFIABLE_REFERENCE" if not reference_ok else
                  "KNEE_BRACKET_20_TO_23_OBSERVED" if all(x["qualifies"] for x in flags.values()) else
                  "NOT_OBSERVED_WITHIN_SAFE_RANGE")
        levels = {}
        for rate in RATES:
            selected = cells[rate]
            level = {"n_networks": 3,
                     "aligned_fraction": [r["aligned_timely_fraction"] for r in selected]}
            for source, target in (("aligned_timely_fraction", "aligned_fraction"),
                                   ("eventual_goodput_tx_s", "eventual_goodput_tx_s"),
                                   ("inclusion_upper_bound_p95_ms", "p95_upper_ms"),
                                   ("checktx_socket_queue_wait_mean_ms", "socket_queue_wait_ms"),
                                   ("max_accepted_mempool_bytes", "max_sampled_mempool_bytes")):
                mean, interval = measured_mean_ci(selected, source)
                level["mean_"+target] = mean
                level["ci95_mean_"+target] = interval
            levels[str(rate)] = level
        output[profile] = {"status": status, "reference_aligned_ok": reference_ok,
                           "reference_p95_upper_ms": p95_ref, "reference_socket_queue_ms": q_ref,
                           "criteria": flags, "levels": levels}
    return output


def derive(session, plan):
    raw = session / "raw"
    manifest = common.verify_raw(raw)
    rows, failures = [], []
    for job in jobs():
        try:
            rows.append(row(raw, job, plan))
        except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError, IndexError, ZeroDivisionError) as exc:
            failures.append({"run_id": job["run_id"], "reason": str(exc)})
    if len({r["genesis_sha256"] for r in rows}) != 36:
        failures.append({"reason": "independent genesis count"})
    qc = {"schema": "pluto.file02.r15.qc.v1", "case_id": "R15",
          "status": "PASS" if len(rows) == 36 and not failures else "INVALID_FOR_INFERENCE",
          "n_planned": 36, "n_valid": len(rows), "failed": failures,
          "plan_sha256": common.digest(session / "plan.json"), "raw_manifest_sha256": manifest}
    aggregate = {"schema": "pluto.file02.r15.aggregate.v1", "case_id": "R15",
                 "status": "PENDING_VALID_QC", "n_networks": len(rows), "decisions": {},
                 "plan_sha256": qc["plan_sha256"], "raw_manifest_sha256": manifest}
    if qc["status"] == "PASS":
        aggregate["decisions"] = decide(rows)
        aggregate["status"] = "BOUNDED_RAMP_EVALUATED"
    return rows, qc, aggregate


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.r15.per_run.v1", "rows": rows})
    with (session / "result_per_run.csv").open("x", newline="", encoding="utf-8") as stream:
        keys = ("run_id", "profile", "rate_tx_s", "repetition", "achieved_offered_tx_s",
                "achieved_offered_bytes_s", "aligned_timely_fraction", "eventual_goodput_tx_s",
                "inclusion_upper_bound_p95_ms", "checktx_socket_queue_wait_mean_ms",
                "max_accepted_mempool_bytes", "qc_status")
        writer = csv.DictWriter(stream, fieldnames=keys)
        writer.writeheader()
        writer.writerows({key: r[key] for key in keys} for r in rows)
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir()
    r01.write_new(session / "figures/status.json", {"status": "NOT_REQUIRED",
                                                     "reason": "numeric per-run and per-level tables with uncertainty"})
    (session / "decision.md").write_text(
        "# R15 decision\n\nQC: " + qc["status"] + ". " +
        ", ".join(p+"="+d["status"] for p, d in aggregate["decisions"].items()) +
        ". Timely goodput uses a three-second observation grace; eventual goodput is not sustainable throughput. "
        "No inference above 25 tx/s, multi-host, P2P-origin or DoS.\n", encoding="utf-8")
    return qc, aggregate


def execute():
    session = ARTIFACTS / ("r15-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8])
    plan = prepare(session)
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT, stdout=stream, stderr=subprocess.STDOUT)
    require(build.returncode == 0, f"build failed: {session}")
    env = os.environ.copy()
    env.update(FILE02_R15_ROOT=str(raw), FILE02_SOURCE_COMMIT=plan["source_commit"], FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02R15$", "-timeout", "2400s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=2500)
    manifest = common.freeze_raw(raw)
    qc, aggregate = write_outputs(session, plan)
    print(f"R15 session={session} integration_exit={run.returncode} qc={qc['status']} "
          f"status={aggregate['status']} raw_manifest_sha256={manifest}")
    return 0 if run.returncode == 0 and qc["status"] == "PASS" else 2


def verify(session):
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve(), "private session")
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    rows, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.r15.per_run.v1", "rows": rows} and
            common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate,
            "R15 derived artifact mismatch")
    with (session / "result_per_run.csv").open(newline="", encoding="utf-8") as stream:
        csv_rows = list(csv.DictReader(stream))
    require(len(csv_rows) == len(rows) and all(item["run_id"] == r["run_id"] for item, r in zip(csv_rows, rows)),
            "R15 CSV mismatch")
    revision = session / "analysis_revision_001.json"
    if revision.exists():
        record = common.read_json(revision)
        require(record.get("raw_manifest_sha256") == qc["raw_manifest_sha256"] and
                record.get("original_aggregate_sha256") ==
                common.digest(session / "analysis_revision_000_aggregate.json") and
                record.get("corrected_aggregate_sha256") == common.digest(session / "result_aggregate.json"),
                "R15 analysis correction provenance")
    print(f"R15_VERIFY=OK qc={qc['status']} n={len(rows)} raw_manifest_sha256={qc['raw_manifest_sha256']}")


def rederive(session):
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve() and not (session / "analysis_revision_001.json").exists(),
            "private unrevised session required")
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    rows, qc, aggregate = derive(session, plan)
    require(qc["status"] == "PASS" and common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.r15.per_run.v1", "rows": rows},
            "R15 source raw/QC changed")
    prior = session / "result_aggregate.json"
    old = prior.read_text(encoding="utf-8")
    common.write_once(session / "analysis_revision_000_aggregate.json", old)
    corrected = json.dumps(aggregate, sort_keys=True, indent=2) + "\n"
    temporary = session / "result_aggregate.tmp"
    common.write_once(temporary, corrected)
    os.replace(temporary, prior)
    r01.write_new(session / "analysis_revision_001.json", {
        "schema": "pluto.file02.r15.analysis_revision.v1", "reason": "add registered CI for all measured rate-level means",
        "analysis_source_commit": r01.git("rev-parse", "HEAD"),
        "raw_manifest_sha256": qc["raw_manifest_sha256"],
        "original_aggregate_sha256": hashlib.sha256(old.encode()).hexdigest(),
        "corrected_aggregate_sha256": common.digest(prior)})
    verify(session)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("run")
    p = sub.add_parser("verify")
    p.add_argument("session")
    p = sub.add_parser("rederive")
    p.add_argument("session")
    args = parser.parse_args()
    try:
        sys.exit(execute() if args.command == "run" else
                 (rederive(args.session) or 0) if args.command == "rederive" else
                 (verify(args.session) or 0))
    except (ValueError, OSError, KeyError, TypeError, subprocess.TimeoutExpired) as exc:
        print(f"R15_ERROR: {exc}", file=sys.stderr)
        sys.exit(2)
