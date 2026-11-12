#!/usr/bin/env python3
"""Audit preregistered twelve-block precision pilot with cluster bootstrap."""

import argparse
from datetime import datetime, timezone
import hashlib
import json
import math
import os
from pathlib import Path
import random
import secrets
import subprocess
import sys
from uuid import uuid4

import file02_common as common
from file02_common03 import private_session, git_value, require
from file02_common06 import rank_coverage
import s07_orchestrate as s07

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
PROFILES = ("C", "CP", "P", "H")
BLOCKS = 12
RATE = 20
STEADY_SECONDS = 12
BOOTSTRAP_DRAWS = 2000
MAX_RELATIVE_CI_RADIUS = .20


def schedule(master_seed):
    rng = random.Random(master_seed)
    order = list(PROFILES)
    rng.shuffle(order)
    jobs = []
    for block in range(BLOCKS):
        for position, profile in enumerate(order[block % 4:] + order[:block % 4]):
            jobs.append({"block": block, "order": position, "profile": profile,
                         "seed": rng.randrange(1, 2**63), "run_id": f"t06p-b{block:02d}-{profile.lower()}"})
    return {"schema": "pluto.file02.t06_precision_schedule.v1", "master_seed": master_seed,
            "method": "seeded_random_initial_order_then_twelve_cyclic_blocks", "jobs": jobs}


def validate_schedule(value):
    require(value.get("schema") == "pluto.file02.t06_precision_schedule.v1" and
            value.get("method") == "seeded_random_initial_order_then_twelve_cyclic_blocks" and
            isinstance(value.get("master_seed"), int) and 0 <= value["master_seed"] < 2**64 and
            schedule(value["master_seed"]) == value, "schedule replay mismatch")
    jobs = value["jobs"]
    require(len(jobs) == 48 and len({j["seed"] for j in jobs}) == 48 and
            len({j["run_id"] for j in jobs}) == 48, "duplicate seed/run")
    for block in range(BLOCKS):
        require({j["profile"] for j in jobs[4*block:4*block+4]} == set(PROFILES), "unbalanced block")
    for position in range(4):
        require([j["profile"] for j in jobs if j["order"] == position].count("H") == 3 and
                all([j["profile"] for j in jobs if j["order"] == position].count(p) == 3 for p in PROFILES),
                "unbalanced order position")


def quantile99(values):
    require(values, "empty quantile input")
    ordered = sorted(values)
    return ordered[math.ceil(.99*len(ordered))-1]


def cluster_bootstrap(groups, seed):
    """Two-stage bootstrap: whole networks, then transactions within each."""
    require(len(groups) == BLOCKS and all(len(group) == 1+RATE*STEADY_SECONDS for group in groups),
            "bootstrap requires twelve complete independent runs")
    point = quantile99([v for group in groups for v in group])
    rng = random.Random(seed)
    estimates = []
    for _ in range(BOOTSTRAP_DRAWS):
        sampled = []
        for _ in range(BLOCKS):
            group = groups[rng.randrange(BLOCKS)]
            sampled.extend(group[rng.randrange(len(group))] for _ in group)
        estimates.append(quantile99(sampled))
    estimates.sort()
    low, high = estimates[math.floor(.025*BOOTSTRAP_DRAWS)], estimates[math.ceil(.975*BOOTSTRAP_DRAWS)-1]
    radius = max(point-low, high-point)/point if point > 0 else math.inf
    coverage = rank_coverage(sum(map(len, groups)))
    passed = coverage >= .95 and radius <= MAX_RELATIVE_CI_RADIUS
    return {"n_independent_runs": BLOCKS, "n_transaction_samples": sum(map(len, groups)),
            "n_per_run": [len(group) for group in groups],
            "p99_upper_bound_ms": point if passed else None,
            "cluster_bootstrap_ci95_ms": [low, high] if passed else None,
            "relative_ci_radius": radius, "rank_coverage_iid_optimistic": coverage,
            "status": "PRECISION_PASS" if passed else "INSUFFICIENT_PRECISION"}


def validate_run(run, job, source, tree):
    meta, plan, result, oracle, commits, resources = (common.read_json(run / name) for name in
        ("metadata.json", "plan.json", "result.json", "oracle.json", "commit_observations.json",
         "generator_resources.json"))
    require(meta.get("schema") == "pluto.file02.t06_precision_run.v1" and meta.get("scope") == "QA_ONLY" and
            meta.get("commit_observer_mode") == "block_scan" and meta.get("source_commit") == source and
            meta.get("source_tree") == tree and meta.get("network") == "fresh_four_validator_single_host_loopback" and
            all(meta.get(key) == job[key] for key in ("profile", "block", "order", "seed")), "metadata mismatch")
    require(meta["run_id"] == plan["run_id"] == result["run_id"] == oracle["run_id"] == commits["run_id"] ==
            resources["run_id"] == job["run_id"] and result["plan_sha256"] == common.digest(run / "plan.json") and
            plan["genesis_sha256"] == result["genesis_sha256"] == common.digest(run / "genesis.bin") and
            plan["config_sha256"] == result["config_sha256"] == meta["config_sha256"], "source/run binding mismatch")
    count = 1+RATE*STEADY_SECONDS
    require(plan["seed"] == job["seed"] and plan["profile"] == result["profile"] and
            plan["regime"] == "equal_tx_s" and plan["target_tx_per_second"] == RATE and
            plan["steady_ns"] == STEADY_SECONDS*1_000_000_000 and
            result["status"] == oracle["status"] == "OK" and
            oracle["valid_committed"] == result["planned_count"] == result["actual_send_count"] == len(plan["events"]) == count and
            oracle["invalid_rejected"] == 0 and oracle["common_height"] >= 8 and
            oracle["block_id"] and oracle["app_hash"] and
            commits["schema"] == "pluto.s07.commits.v1" and len(commits["events"]) == count and
            resources["schema"] == "pluto.file02.generator_resources.v1", "execution/oracle mismatch")
    by_id = {entry["id"]: entry for entry in commits["events"]}
    require(len(by_id) == count, "duplicate commit observations")
    windows = [0]*STEADY_SECONDS
    hashes, slots, latencies = set(), set(), []
    for expected, actual in zip(plan["events"], result["events"]):
        raw = bytes.fromhex(expected["bytes_hex"])
        target, sent = expected["offset_ns"], actual["actual_offset_ns"]
        require(hashlib.sha256(raw).hexdigest() == expected["sha256"] and expected["sha256"] not in hashes and
                expected["account_slot"] not in slots and expected["nonce"] == 0 and
                expected["id"] == actual["id"] and target == actual["planned_offset_ns"] and
                expected["traffic"] == actual["traffic"] == "legitimate" and
                expected["expected_code"] == actual["actual_code"] == 0 and not actual.get("submission_error") and
                isinstance(sent, int) and 0 <= sent < plan["warmup_ns"]+plan["steady_ns"] and
                (target < plan["warmup_ns"]) == (sent < plan["warmup_ns"]), "event/send mismatch")
        hashes.add(expected["sha256"])
        slots.add(expected["account_slot"])
        if target >= plan["warmup_ns"]:
            window = (sent-plan["warmup_ns"]) // plan["window_ns"]
            require(0 <= window < STEADY_SECONDS, "steady window overflow")
            windows[window] += 1
        commit = by_id[expected["id"]]
        require(commit["sha256"] == expected["sha256"] and 1 <= commit["height"] <= oracle["common_height"],
                "all-four commit observation mismatch")
        sent_utc, seen_utc = s07.parse_time(actual["actual_send_utc"]), s07.parse_time(commit["all_four_seen_utc"])
        require(seen_utc >= sent_utc, "negative inclusion upper bound")
        latencies.append((seen_utc-sent_utc).total_seconds()*1000)
    require(windows == [RATE]*STEADY_SECONDS and len(hashes) == len(slots) == count,
            "achieved rate or distinct transaction count")
    return {"run_id": job["run_id"], "profile": job["profile"], "block": job["block"],
            "order": job["order"], "seed": job["seed"], "genesis_sha256": plan["genesis_sha256"],
            "n_inclusion_upper_bound": len(latencies), "inclusion_upper_bound_ms": latencies,
            "steady_tx_s_windows": windows, "oracle_valid_committed": oracle["valid_committed"],
            "generator_cpu_core_fraction": (resources["user_cpu_ns"]+resources["system_cpu_ns"])/resources["wall_ns"]}


def audit(session, source, tree):
    planned = common.read_json(session / "schedule.json")
    validate_schedule(planned)
    rows = [validate_run(session / "runs" / job["run_id"], job, source, tree) for job in planned["jobs"]]
    require(len({r["genesis_sha256"] for r in rows}) == 48, "genesis reused")
    precision = {}
    for profile in PROFILES:
        groups = [r["inclusion_upper_bound_ms"] for r in rows if r["profile"] == profile]
        precision[profile] = cluster_bootstrap(groups, planned["master_seed"] ^ sum(map(ord, profile)))
    return {"schema": "pluto.file02.t06_precision_audit.v1", "scope": "PILOT_ONLY",
            "source_commit": source, "source_tree": tree, "schedule_sha256": common.digest(session / "schedule.json"),
            "fresh_networks": 48, "bootstrap_draws": BOOTSTRAP_DRAWS,
            "maximum_relative_ci_radius": MAX_RELATIVE_CI_RADIUS,
            "precision": precision, "runs": rows}


def smoke():
    require(not git_value("status", "--porcelain"), "commit source before pilot")
    source, tree = git_value("rev-parse", "HEAD"), git_value("rev-parse", "HEAD^{tree}")
    ARTIFACTS.mkdir(parents=True, exist_ok=True, mode=0o700)
    session = ARTIFACTS / ("t06-precision-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8])
    session.mkdir(mode=0o700)
    common.write_once(session / "schedule.json", json.dumps(schedule(secrets.randbits(64)), indent=2, sort_keys=True) + "\n")
    with (session / "build.log").open("wb") as log:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT, stdout=log, stderr=subprocess.STDOUT)
    require(build.returncode == 0, "build failed; inspect private build.log")
    env = os.environ.copy()
    env.update(FILE02_T06_PRECISION_ROOT=str(session), FILE02_SOURCE_COMMIT=source, FILE02_SOURCE_TREE=tree)
    with (session / "integration.log").open("wb") as log:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02T06PrecisionPilot$", "-timeout", "2400s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=2500)
    require(run.returncode == 0, "precision integration failed; inspect private integration.log")
    summary = audit(session, source, tree)
    common.write_once(session / "summary.json", json.dumps(summary, indent=2, sort_keys=True) + "\n")
    manifest = common.freeze_raw(session)
    statuses = ",".join(f"{p}:{summary['precision'][p]['status']}" for p in PROFILES)
    print(f"T06_PRECISION=QA_ONLY runs=48 {statuses} session={session} manifest_sha256={manifest}")


def verify(path):
    session = private_session(path)
    manifest = common.verify_raw(session)
    prior = common.read_json(session / "summary.json")
    require(prior.get("schema") == "pluto.file02.t06_precision_audit.v1", "summary schema")
    require(audit(session, prior["source_commit"], prior["source_tree"]) == prior, "summary differs from raw")
    print(f"T06_PRECISION_VERIFY=OK runs=48 manifest_sha256={manifest}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("smoke")
    p = sub.add_parser("verify")
    p.add_argument("session")
    args = parser.parse_args()
    try:
        if args.command == "smoke": smoke()
        else: verify(args.session)
        return 0
    except (OSError, ValueError, KeyError, TypeError, subprocess.TimeoutExpired) as exc:
        print(f"T06_PRECISION_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__": sys.exit(main())
