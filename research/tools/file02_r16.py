#!/usr/bin/env python3
"""Run and independently audit the locked R16 benign-waveform comparison."""

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
PROFILES = tuple(r06.PROFILES)
ARMS = ("constant", "ramp", "burst")
WINDOWS = {"constant": [10, 10, 10, 10], "ramp": [4, 8, 12, 16],
           "burst": [0, 20, 0, 20]}
SEED = 20261113


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def schedule():
    rng = random.Random(SEED)
    base = list(PROFILES)
    rng.shuffle(base)
    jobs = []
    for block in range(4):
        seed = rng.randrange(1, 2**63)
        profiles = base[block:] + base[:block]
        for profile in profiles:
            arms = ARMS[block % 3:] + ARMS[:block % 3]
            for arm in arms:
                jobs.append({"block": block, "order": len(jobs) % 12,
                             "profile": profile, "arm": arm, "seed": seed,
                             "run_id": f"r16-b{block:02d}-{profile.lower()}-{arm}"})
    return {"schema": "pluto.file02.r16.schedule.v1", "master_seed": SEED,
            "method": "four_rotated_profile_blocks_latin_arm_rotation", "jobs": jobs}


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve() and
            not r01.git("status", "--porcelain"), "private session or dirty source")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT).returncode == 0,
            "session not Git-ignored")
    session.mkdir(mode=0o700)
    raw = session / "raw"
    raw.mkdir(mode=0o700)
    common.write_once(raw / "schedule.json", json.dumps(schedule(), sort_keys=True, indent=2) + "\n")
    config = ROOT / "research/config/s06_pilot.json"
    common.write_once(raw / "config.json", config.read_text())
    plan = {"schema": "pluto.file02.r16.plan.v1", "status": "LOCKED", "case_id": "R16",
            "source_commit": r01.git("rev-parse", "HEAD"),
            "source_tree": r01.git("rev-parse", "HEAD^{tree}"),
            "created_utc": datetime.now(timezone.utc).isoformat(),
            "host": r01.cpu_inventory(),
            "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
            "module_sha256": common.digest(ROOT / "go.mod"),
            "config_sha256": common.digest(raw / "config.json"),
            "schedule_sha256": common.digest(raw / "schedule.json"),
            "protocol": "research/protocol/FILE02_R16_LOCKED_PLAN.md",
            "chain_id": "pluto-file02-r16", "profiles": list(PROFILES),
            "arms": list(ARMS), "windows": WINDOWS,
            "n_blocks": 4, "n_networks": 48, "average_valid_tx_s": 10,
            "warmup_seconds": 1, "steady_seconds": 4,
            "drain_seconds": 3, "cooldown_seconds": 2,
            "primary": "burst_minus_constant_max_sampled_mempool_bytes",
            "ci": "paired_percentile_bootstrap_20000_seed_20261113"}
    r01.write_new(session / "plan.json", plan)
    return plan


def validate_plan(session, plan):
    require(plan.get("schema") == "pluto.file02.r16.plan.v1" and
            plan.get("status") == "LOCKED" and plan.get("case_id") == "R16" and
            plan.get("source_tree") == r01.git("rev-parse", plan["source_commit"] + "^{tree}") and
            plan.get("module_sha256") == common.digest(ROOT / "go.mod") and
            plan.get("host") == r01.cpu_inventory() and
            plan.get("config_sha256") == common.digest(session / "raw/config.json") ==
            common.digest(ROOT / "research/config/s06_pilot.json") and
            plan.get("schedule_sha256") == common.digest(session / "raw/schedule.json") and
            common.read_json(session / "raw/schedule.json") == schedule() and
            plan.get("profiles") == list(PROFILES) and plan.get("arms") == list(ARMS) and
            plan.get("windows") == WINDOWS and plan.get("n_networks") == 48,
            "R16 locked plan drift")


def audit_run(raw, job, plan):
    run = raw / "runs" / job["run_id"]
    meta, workload, result, oracle, commits, cp, nodes = (common.read_json(run / name) for name in
        ("metadata.json", "plan.json", "result.json", "oracle.json", "commit_observations.json",
         "research_plan/plan.json", "nodes.json"))
    common.validate_plan(cp)
    require(meta.get("schema") == "pluto.file02.r16_run.v1" and meta.get("scope") == "R_CONFIRMATORY" and
            meta.get("case_id") == cp.get("case_id") == "R16" and
            all(meta.get(k) == job[k] for k in ("block", "order", "profile", "arm", "seed")) and
            meta.get("recorder_mode") == "ON" and meta.get("commit_observer_mode") == "block_scan" and
            meta.get("source_commit") == cp.get("source_commit") == plan["source_commit"] and
            meta.get("source_tree") == cp.get("source_tree") == plan["source_tree"] and
            meta.get("run_id") == workload.get("run_id") == result.get("run_id") ==
            oracle.get("run_id") == commits.get("run_id") == cp.get("run_id") == job["run_id"],
            "R16 run/source identity")
    require(result.get("plan_sha256") == cp.get("corpus_sha256") == common.digest(run / "plan.json") and
            result.get("genesis_sha256") == cp.get("genesis_sha256") ==
            workload.get("genesis_sha256") == common.digest(run / "genesis.bin") and
            result.get("config_sha256") == cp.get("config_sha256") ==
            plan["config_sha256"] == common.digest(run / "config.json") and
            cp["network_config"]["rpc_endpoints"] == [n["rpc"] for n in nodes] and
            cp["seed"] == job["seed"] and cp["order_index"] == job["order"],
            "R16 runtime provenance")
    require(workload.get("regime") == "equal_tx_s" and workload.get("waveform") == job["arm"] and
            workload.get("target_tx_per_second") == 10 and workload.get("steady_ns") == 4_000_000_000 and
            workload.get("drain_ns") == 3_000_000_000 and len(workload.get("events", [])) == 41 and
            result.get("status") == oracle.get("status") == "OK" and
            result.get("planned_count") == result.get("actual_send_count") == oracle.get("valid_committed") == 41 and
            oracle.get("invalid_rejected") == 0 and oracle.get("common_height", 0) >= 8 and
            oracle.get("block_id") and oracle.get("app_hash") and len(commits.get("events", [])) == 41,
            "R16 load/consensus oracle")
    wire_size = r06.WIRE[job["profile"]]
    require(all(len(bytes.fromhex(e["bytes_hex"])) == wire_size for e in workload["events"]),
            "R16 profile wire size")
    by_commit = {e["id"]: e for e in commits["events"]}
    require(len(by_commit) == 41, "R16 duplicate commit ID")
    planned_hashes, slots = set(), set()
    windows = [0]*4
    byte_windows = [0]*4
    latency_ms = []
    for expected, actual in zip(workload["events"], result["events"]):
        h = hashlib.sha256(bytes.fromhex(expected["bytes_hex"])).hexdigest()
        require(h == expected["sha256"] and h not in planned_hashes and
                expected["account_slot"] not in slots and
                expected["id"] == actual["id"] and actual.get("actual_code") == 0 and
                not actual.get("submission_error") and actual.get("wire_bytes") == wire_size and
                by_commit[expected["id"]]["sha256"] == h and
                isinstance(actual.get("actual_offset_ns"), int), "R16 event identity/oracle")
        planned_hashes.add(h)
        slots.add(expected["account_slot"])
        actual_offset = actual["actual_offset_ns"]
        if expected["offset_ns"] >= 1_000_000_000:
            window = (actual_offset-1_000_000_000)//1_000_000_000
            require(0 <= window < 4 and (expected["offset_ns"]-1_000_000_000)//1_000_000_000 == window,
                    "R16 send slipped from planned window")
            windows[window] += 1
            byte_windows[window] += wire_size
            sent = precision.parse_go_rfc3339(actual["actual_send_utc"])
            seen = precision.parse_go_rfc3339(by_commit[expected["id"]]["all_four_seen_utc"])
            require(seen >= sent, "R16 negative inclusion bound")
            latency_ms.append((seen-sent).total_seconds()*1000)
        else:
            require(actual_offset < 1_000_000_000, "R16 warmup slip")
    require(windows == WINDOWS[job["arm"]] and max(windows) <= 25 and
            max(byte_windows) <= 65536 and len(latency_ms) == 40,
            "R16 offered schedule/caps")
    observed = []
    for i in range(4):
        samples = r06.samples(run / f"observer-{i}.jsonl")
        recorder = r06.samples(run / f"recorder-{i}.jsonl")
        require(samples and all(s.get("kind") == "node_sample" and s.get("dropped_samples") == 0 for s in samples) and
                recorder and all(e.get("kind") == "check_tx" for e in recorder),
                "R16 observer/recorder drop")
        observed.append(samples)
    ingress = r06.samples(run / "recorder-0.jsonl")
    new = {e.get("tx_sha256"): e for e in ingress if e.get("check_type") == "new"}
    require(len(new) == 41 and set(new) == planned_hashes and all(e.get("code") == 0 for e in new.values()),
            "R16 application NEW attribution")
    samples = observed[0]
    queue = [s["accepted_mempool_count"] for s in samples if isinstance(s.get("accepted_mempool_count"), int)]
    qbytes = [s["accepted_mempool_bytes"] for s in samples if isinstance(s.get("accepted_mempool_bytes"), int)]
    require(queue and qbytes, "R16 mempool samples unavailable")
    first, last = samples[0], samples[-1]
    count0, count1 = first.get("checktx_socket_queue_count"), last.get("checktx_socket_queue_count")
    sum0, sum1 = first.get("checktx_socket_queue_seconds"), last.get("checktx_socket_queue_seconds")
    queue_wait_ms = ((sum1-sum0)/(count1-count0)*1000 if
                     all(isinstance(v, (int, float)) for v in (count0, count1, sum0, sum1)) and
                     count1 > count0 and sum1 >= sum0 else None)
    last_send = max(precision.parse_go_rfc3339(e["actual_send_utc"]) for e in result["events"])
    post = [precision.parse_go_rfc3339(s["observed_utc"]) for s in samples if
            isinstance(s.get("accepted_mempool_count"), int) and s["accepted_mempool_count"] == 0 and
            precision.parse_go_rfc3339(s["observed_utc"]) >= last_send]
    recovery_ms = (min(post)-last_send).total_seconds()*1000 if post else None
    relay = [s.get("p2p_ingress_tx_bytes_total") for s in samples]
    relay_delta = (max(relay)-min(relay) if all(isinstance(v, (int, float)) for v in relay) else None)
    relay_rates = [s.get("p2p_ingress_bytes_per_second") for s in samples
                   if isinstance(s.get("p2p_ingress_bytes_per_second"), (int, float))]
    generator = common.read_json(run / "generator_resources.json")
    require(generator.get("wall_ns", 0) > 0 and
            (generator.get("user_cpu_ns", 0)+generator.get("system_cpu_ns", 0)) < .9*generator["wall_ns"],
            "R16 generator CPU safety QC")
    return {"run_id": job["run_id"], "profile": job["profile"], "arm": job["arm"],
            "block": job["block"], "order": job["order"], "seed": job["seed"],
            "genesis_sha256": workload["genesis_sha256"], "wire_bytes": wire_size,
            "planned_hash_digest": hashlib.sha256("\n".join(sorted(planned_hashes)).encode()).hexdigest(),
            "offered_tx_s_windows": windows, "offered_bytes_s_windows": byte_windows,
            "valid_committed": 41, "max_sampled_mempool_count": max(queue),
            "max_sampled_mempool_bytes": max(qbytes),
            "mempool_count_curve": [{"observed_utc": s["observed_utc"],
                                     "count": s.get("accepted_mempool_count"),
                                     "bytes": s.get("accepted_mempool_bytes")}
                                    for s in samples],
            "checktx_socket_queue_wait_ms": queue_wait_ms,
            "inclusion_upper_p95_ms": r01.percentile(latency_ms, .95),
            "inclusion_upper_p99_ms": None,
            "inclusion_upper_p99_status": "INSUFFICIENT_PRECISION",
            "recovery_upper_ms": recovery_ms,
            "recovery_status": "OBSERVED_SAMPLE_BOUND" if recovery_ms is not None else "UNAVAILABLE_NO_ZERO_SAMPLE",
            "p2p_ingress_bytes_delta": relay_delta,
            "p2p_ingress_peak_bytes_s": max(relay_rates) if relay_rates else None,
            "p2p_ingress_status": "OBSERVED" if relay_delta is not None else "UNAVAILABLE",
            "qc_status": "PASS"}


def bootstrap(values):
    require(len(values) == 4, "R16 requires four paired blocks")
    rng = random.Random(SEED)
    draws = [statistics.mean(values[rng.randrange(4)] for _ in range(4)) for _ in range(20_000)]
    return [r01.percentile(draws, .025), r01.percentile(draws, .975)]


def derive(session, plan):
    manifest = common.verify_raw(session / "raw")
    rows, failures = [], []
    for job in schedule()["jobs"]:
        try:
            rows.append(audit_run(session / "raw", job, plan))
        except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError, IndexError) as exc:
            failures.append({"run_id": job["run_id"], "reason": str(exc)})
    for block in range(4):
        block_rows = [r for r in rows if r["block"] == block]
        if len(block_rows) == 12 and len({r["genesis_sha256"] for r in block_rows}) != 1:
            failures.append({"block": block, "reason": "unmatched genesis"})
        for profile in PROFILES:
            paired = [r for r in block_rows if r["profile"] == profile]
            if len(paired) == 3 and len({r["planned_hash_digest"] for r in paired}) != 1:
                failures.append({"block": block, "profile": profile,
                                 "reason": "waveform transaction set not matched"})
    if len({r["genesis_sha256"] for r in rows}) != 4:
        failures.append({"reason": "independent genesis count"})
    qc = {"schema": "pluto.file02.r16.qc.v1", "case_id": "R16",
          "status": "PASS" if len(rows) == 48 and not failures else "INVALID_FOR_INFERENCE",
          "n_planned": 48, "n_valid": len(rows), "failed": failures,
          "plan_sha256": common.digest(session / "plan.json"), "raw_manifest_sha256": manifest}
    aggregate = {"schema": "pluto.file02.r16.aggregate.v1", "case_id": "R16",
                 "status": "PENDING_VALID_QC", "n_networks": len(rows), "profile_summary": {},
                 "plan_sha256": qc["plan_sha256"], "raw_manifest_sha256": manifest}
    if qc["status"] == "PASS":
        for profile in PROFILES:
            groups = {arm: {r["block"]: r for r in rows if r["profile"] == profile and r["arm"] == arm}
                      for arm in ARMS}
            require(all(len(groups[a]) == 4 for a in ARMS), "R16 profile block pairing")
            difference = [groups["burst"][b]["max_sampled_mempool_bytes"] -
                          groups["constant"][b]["max_sampled_mempool_bytes"] for b in range(4)]
            ramp_difference = [groups["ramp"][b]["max_sampled_mempool_bytes"] -
                               groups["constant"][b]["max_sampled_mempool_bytes"] for b in range(4)]
            aggregate["profile_summary"][profile] = {
                "n_matched_blocks": 4, "burst_minus_constant_peak_bytes": difference,
                "mean_burst_minus_constant_peak_bytes": statistics.mean(difference),
                "ci95_burst_minus_constant_peak_bytes": bootstrap(difference),
                "ramp_minus_constant_peak_bytes": ramp_difference,
                "mean_ramp_minus_constant_peak_bytes": statistics.mean(ramp_difference),
                "ci95_ramp_minus_constant_peak_bytes": bootstrap(ramp_difference),
                "benign_false_alert_candidate_blocks":
                    sum(difference[b] > 0 or ramp_difference[b] > 0 for b in range(4)),
                "recovery_upper_ms_by_arm": {a: [groups[a][b]["recovery_upper_ms"] for b in range(4)]
                                             for a in ARMS}}
        aggregate["status"] = "BENIGN_WAVEFORM_CONTROL_MEASURED"
    return rows, qc, aggregate


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.r16.per_run.v1", "rows": rows})
    with (session / "result_per_run.csv").open("x", newline="", encoding="utf-8") as stream:
        keys = ("run_id", "block", "profile", "arm", "seed", "wire_bytes", "valid_committed",
                "max_sampled_mempool_count", "max_sampled_mempool_bytes",
                "checktx_socket_queue_wait_ms", "inclusion_upper_p95_ms",
                "recovery_upper_ms", "p2p_ingress_bytes_delta", "p2p_ingress_peak_bytes_s",
                "qc_status")
        writer = csv.DictWriter(stream, fieldnames=keys)
        writer.writeheader()
        writer.writerows({key: r[key] for key in keys} for r in rows)
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir()
    r01.write_new(session / "figures/status.json", {"status": "NOT_REQUIRED",
        "reason": "numeric per-run curves and registered aggregate; no RQ2 detector/figure"})
    (session / "decision.md").write_text(
        f"# R16 decision\n\nQC {qc['status']} on {len(rows)}/48 networks. "
        f"Result: {aggregate['status']}. Benign transient is descriptive only; "
        "no detector, false-positive rate, knee or DoS claim.\n", encoding="utf-8")
    return qc, aggregate


def execute():
    session = ARTIFACTS / ("r16-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8])
    plan = prepare(session)
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT, stdout=stream,
                               stderr=subprocess.STDOUT)
    require(build.returncode == 0, f"build failed: {session}")
    env = os.environ.copy()
    env.update(FILE02_R16_ROOT=str(raw), FILE02_SOURCE_COMMIT=plan["source_commit"],
               FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02R16$", "-timeout", "2400s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=2500)
    manifest = common.freeze_raw(raw)
    qc, aggregate = write_outputs(session, plan)
    print(f"R16 session={session} integration_exit={run.returncode} qc={qc['status']} "
          f"status={aggregate['status']} raw_manifest_sha256={manifest}")
    return 0 if run.returncode == 0 and qc["status"] == "PASS" else 2


def verify(session):
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve(), "private session")
    plan = common.read_json(session / "plan.json")
    validate_plan(session, plan)
    rows, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") ==
            {"schema": "pluto.file02.r16.per_run.v1", "rows": rows} and
            common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate,
            "R16 derived artifact mismatch")
    with (session / "result_per_run.csv").open(newline="", encoding="utf-8") as stream:
        csv_rows = list(csv.DictReader(stream))
    require(len(csv_rows) == len(rows) and all(item["run_id"] == row["run_id"]
                                          for item, row in zip(csv_rows, rows)), "R16 CSV mismatch")
    print(f"R16_VERIFY=OK qc={qc['status']} n={len(rows)} raw_manifest_sha256={qc['raw_manifest_sha256']}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("run")
    item = sub.add_parser("verify")
    item.add_argument("session")
    args = parser.parse_args()
    try:
        sys.exit(execute() if args.command == "run" else (verify(args.session) or 0))
    except (ValueError, OSError, KeyError, TypeError, subprocess.TimeoutExpired) as exc:
        print(f"R16_ERROR: {exc}", file=sys.stderr)
        sys.exit(2)
