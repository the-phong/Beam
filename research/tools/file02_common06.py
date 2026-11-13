#!/usr/bin/env python3
"""Counterbalanced independent-run QA and distribution-free p99 rank gate."""

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
import s07_orchestrate as s07

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
PROFILES = ("C", "CP", "P", "H")


def binomial_upper_tail(n, probability, rank):
    return sum(math.exp(math.lgamma(n + 1) - math.lgamma(j + 1) - math.lgamma(n - j + 1)
                        + j * math.log(probability) + (n-j) * math.log1p(-probability))
               for j in range(rank, n + 1))


def rank_coverage(n):
    """P(true percentile rank of sample p99 is within [98.5%, 99.5%])."""
    require(isinstance(n, int) and n >= 1, "sample count")
    rank = math.ceil(.99 * n)
    return binomial_upper_tail(n, .995, rank) - binomial_upper_tail(n, .985, rank)


def schedule(master_seed):
    rng = random.Random(master_seed)
    order = list(PROFILES)
    rng.shuffle(order)
    jobs = []
    for block in range(4):
        for position, profile in enumerate(order[block:] + order[:block]):
            jobs.append({"block": block, "order": position, "profile": profile,
                         "seed": rng.randrange(1, 2**63), "run_id": f"common06-b{block}-{profile.lower()}"})
    return {"schema": "pluto.file02.common06_schedule.v1", "master_seed": master_seed,
            "method": "seeded_random_initial_permutation_then_four_cyclic_rotations", "jobs": jobs}


def validate_schedule(value):
    require(value.get("schema") == "pluto.file02.common06_schedule.v1" and
            value.get("method") == "seeded_random_initial_permutation_then_four_cyclic_rotations" and
            isinstance(value.get("master_seed"), int) and 0 <= value["master_seed"] < 2**64 and
            schedule(value["master_seed"]) == value, "schedule replay mismatch")
    jobs = value["jobs"]
    require(len(jobs) == 16 and len({j["seed"] for j in jobs}) == 16 and
            len({j["run_id"] for j in jobs}) == 16, "nonunique seeds or runs")
    for block in range(4):
        require({j["profile"] for j in jobs[4*block:4*block+4]} == set(PROFILES), "unbalanced block")
    for position in range(4):
        require({j["profile"] for j in jobs if j["order"] == position} == set(PROFILES), "unbalanced order")


def validate_run(run, job, source, tree):
    meta, plan, result, oracle, commits = (common.read_json(run / name) for name in
                                         ("metadata.json", "plan.json", "result.json", "oracle.json", "commit_observations.json"))
    require(meta.get("schema") == "pluto.file02.common06_run.v1" and meta.get("scope") == "QA_ONLY" and
            meta.get("source_commit") == source and meta.get("source_tree") == tree and
            meta.get("network") == "fresh_four_validator_single_host_loopback" and
            all(meta.get(key) == job[key] for key in ("profile", "block", "order", "seed")), "metadata mismatch")
    require(meta["run_id"] == plan["run_id"] == result["run_id"] == oracle["run_id"] == commits["run_id"] == job["run_id"] and
            result["plan_sha256"] == common.digest(run / "plan.json") and
            result["genesis_sha256"] == plan["genesis_sha256"] == common.digest(run / "genesis.bin") and
            result["config_sha256"] == plan["config_sha256"] == meta["config_sha256"], "run binding mismatch")
    require(plan["seed"] == job["seed"] and plan["regime"] == "equal_tx_s" and
            plan["target_tx_per_second"] == 2 and result["status"] == oracle["status"] == "OK" and
            oracle["valid_committed"] == len(plan["events"]) == result["actual_send_count"] == result["planned_count"] == 9 and
            oracle["invalid_rejected"] == 0 and oracle["common_height"] >= 8 and
            commits["schema"] == "pluto.s07.commits.v1" and len(commits["events"]) == 9,
            "execution or oracle mismatch")
    by_id = {item["id"]: item for item in commits["events"]}
    require(len(by_id) == 9, "duplicate commit observation")
    latencies = []
    steady = [0]*4
    for expected, actual in zip(plan["events"], result["events"]):
        raw = bytes.fromhex(expected["bytes_hex"])
        require(hashlib.sha256(raw).hexdigest() == expected["sha256"] and
                expected["id"] == actual["id"] and expected["expected_code"] == actual["actual_code"] == 0 and
                expected["traffic"] == actual["traffic"] == "legitimate" and not actual.get("submission_error") and
                actual["actual_offset_ns"] is not None and
                (expected["offset_ns"] < plan["warmup_ns"]) == (actual["actual_offset_ns"] < plan["warmup_ns"]),
                "event mismatch")
        if expected["offset_ns"] >= plan["warmup_ns"]:
            index = (actual["actual_offset_ns"]-plan["warmup_ns"]) // plan["window_ns"]
            require(0 <= index < 4, "steady send outside window")
            steady[index] += 1
        commit = by_id[expected["id"]]
        require(commit["sha256"] == expected["sha256"] and 1 <= commit["height"] <= oracle["common_height"],
                "commit observation mismatch")
        sent, seen = s07.parse_time(actual["actual_send_utc"]), s07.parse_time(commit["all_four_seen_utc"])
        require(seen >= sent, "negative inclusion upper bound")
        latencies.append((seen-sent).total_seconds()*1000)
    require(steady == [2]*4, "achieved tx/s window mismatch")
    return {"run_id": job["run_id"], "profile": job["profile"], "block": job["block"],
            "order": job["order"], "seed": job["seed"], "genesis_sha256": plan["genesis_sha256"],
            "n_inclusion_upper_bound": len(latencies), "inclusion_upper_bound_ms": latencies,
            "steady_tx_s_windows": steady, "oracle_valid_committed": oracle["valid_committed"]}


def audit(session, source, tree):
    planned = common.read_json(session / "schedule.json")
    validate_schedule(planned)
    rows = [validate_run(session / "runs" / job["run_id"], job, source, tree) for job in planned["jobs"]]
    require(len({r["genesis_sha256"] for r in rows}) == 16, "genesis reused across independent runs")
    by_profile = {}
    for profile in PROFILES:
        group = [r for r in rows if r["profile"] == profile]
        n = sum(r["n_inclusion_upper_bound"] for r in group)
        require(len(group) == 4 and n == 36, "independent run or latency sample count")
        by_profile[profile] = {"n_independent_runs": 4, "n_transaction_samples": n,
                               "n_per_run": [r["n_inclusion_upper_bound"] for r in group],
                               "p99_ms": None, "p99_status": "INSUFFICIENT_PRECISION"}
    return {"schema": "pluto.file02.common06_audit.v1", "scope": "QA_ONLY", "source_commit": source,
            "source_tree": tree, "schedule_sha256": common.digest(session / "schedule.json"),
            "rank_precision": {"target_quantile": .99, "rank_band": [.985, .995],
                               "conservative_iid_sample_floor_n": 2000,
                               "coverage_at_n_2000": rank_coverage(2000),
                               "assumption": "independent identically distributed transaction samples; optimistic floor, not run-level CI"},
            "by_profile": by_profile, "runs": rows}


def smoke():
    require(not git_value("status", "--porcelain"), "commit source before QA")
    source, tree = git_value("rev-parse", "HEAD"), git_value("rev-parse", "HEAD^{tree}")
    ARTIFACTS.mkdir(parents=True, exist_ok=True, mode=0o700)
    session = ARTIFACTS / ("common06-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8])
    session.mkdir(mode=0o700)
    common.write_once(session / "schedule.json", json.dumps(schedule(secrets.randbits(64)), indent=2, sort_keys=True) + "\n")
    with (session / "build.log").open("wb") as log:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT, stdout=log, stderr=subprocess.STDOUT)
    require(build.returncode == 0, "build failed; inspect private build.log")
    env = os.environ.copy()
    env.update(FILE02_COMMON06_ROOT=str(session), FILE02_SOURCE_COMMIT=source, FILE02_SOURCE_TREE=tree)
    with (session / "integration.log").open("wb") as log:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02Common06CounterbalancedRepeats$", "-timeout", "1200s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=log, stderr=subprocess.STDOUT, timeout=1300)
    require(run.returncode == 0, "integration failed; inspect private integration.log")
    summary = audit(session, source, tree)
    common.write_once(session / "summary.json", json.dumps(summary, indent=2, sort_keys=True) + "\n")
    manifest = common.freeze_raw(session)
    print(f"COMMON06_STATUS=QA_ONLY runs=16 p99=INSUFFICIENT_PRECISION session={session} manifest_sha256={manifest}")


def verify(path):
    session = private_session(path)
    manifest = common.verify_raw(session)
    prior = common.read_json(session / "summary.json")
    require(prior.get("schema") == "pluto.file02.common06_audit.v1", "summary schema")
    require(audit(session, prior["source_commit"], prior["source_tree"]) == prior, "summary differs from raw")
    print(f"COMMON06_VERIFY=OK runs=16 manifest_sha256={manifest}")


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
        print(f"COMMON06_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__": sys.exit(main())
