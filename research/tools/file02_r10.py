#!/usr/bin/env python3
"""Execute and audit the bounded R10 transaction-size boundary diagnostic."""

import argparse
import csv
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import subprocess
import sys
from uuid import uuid4

import file02_common as common
import file02_r01 as r01
import file02_r06 as r06

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def prepare(session):
    require(session.parent.resolve() == ARTIFACTS.resolve() and not r01.git("status", "--porcelain"),
            "private session or dirty source")
    require(subprocess.run(["git", "check-ignore", "-q", str(session)], cwd=ROOT).returncode == 0,
            "session not ignored")
    session.mkdir(mode=0o700)
    (session / "raw").mkdir(mode=0o700)
    r01.write_new(session / "plan.json", {
        "schema": "pluto.file02.r10.plan.v1", "status": "LOCKED", "case_id": "R10",
        "created_utc": datetime.now(timezone.utc).isoformat(), "source_commit": r01.git("rev-parse", "HEAD"),
        "source_tree": r01.git("rev-parse", "HEAD^{tree}"), "host": r01.cpu_inventory(),
        "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
        "safety_config_sha256": common.digest(ROOT / "research/config/s06_pilot.json"),
        "chain_id_length": 50, "max_canonical_H_bytes": 2602, "mempool_max_tx_bytes": 8192,
        "submitted_sizes_bytes": [2602, 8192, 8193], "spacing_seconds": 1,
        "origin": "RPC_LOOPBACK", "topology": "FOUR_VALIDATOR_SINGLE_HOST_DIAGNOSTIC",
        "primary": "RPC_accept_reject_and_CheckTx_stage", "ci": "UNAVAILABLE_SINGLE_NETWORK"})


def derive(session, plan):
    raw = session / "raw"
    manifest = common.verify_raw(raw)
    run = raw / "run"
    locked, rpc, oracle = (common.read_json(run / n) for n in
                           ("research_plan.json", "rpc_results.json", "oracle.json"))
    require(locked.get("schema") == "pluto.file02.r10.run_plan.v1" and locked.get("status") == "LOCKED" and
            locked.get("source_commit") == plan["source_commit"] and
            locked.get("source_tree") == plan["source_tree"] and
            locked.get("config_sha256") == plan["safety_config_sha256"] and
            locked.get("genesis_sha256") == common.digest(run / "genesis.bin") and
            locked.get("signed_corpus_plan_sha256") == common.digest(run / "signed_corpus_plan.json") and
            locked.get("sizes_bytes") == [2602, 8192, 8193] and
            locked.get("rpc_endpoint", "").startswith("http://127.0.0.1:") and
            len(locked.get("validator_config_sha256", [])) == 4,
            "runtime plan provenance")
    rows = rpc.get("rows")
    require(rpc.get("schema") == "pluto.file02.r10.rpc.v1" and isinstance(rows, list) and len(rows) == 3 and
            [r.get("label") for r in rows] == ["valid_max_canonical", "invalid_at_limit", "invalid_over_limit"] and
            [r.get("wire_bytes") for r in rows] == [2602, 8192, 8193] and
            all(isinstance(r.get("rpc_roundtrip_ns"), int) and r["rpc_roundtrip_ns"] > 0 for r in rows),
            "RPC rows")
    require(rows[0]["rpc_code"] == 0 and rows[0]["rpc_error"] is False and
            isinstance(rows[1]["rpc_code"], int) and rows[1]["rpc_code"] > 0 and
            rows[2]["rpc_reason"] == "mempool_tx_too_large" and rows[2]["rpc_error"] is True,
            "RPC boundary outcomes")
    require(oracle.get("schema") == "pluto.file02.r10.oracle.v1" and oracle.get("status") == "OK" and
            oracle.get("common_height", 0) >= 8 and oracle.get("block_id") and oracle.get("app_hash") and
            oracle.get("valid_max_committed") == 1 and oracle.get("invalid_at_limit_committed") == 0 and
            oracle.get("invalid_over_limit_committed") == 0 and
            oracle.get("invalid_at_limit_stage") == "structural" and
            oracle.get("over_limit_checktx_reached") is False and
            oracle.get("comet_precheck_cpu_ns") is None and
            oracle.get("comet_precheck_cpu_status") == "UNAVAILABLE", "four-validator oracle")
    for i in range(4):
        samples = r06.samples(run / f"observer-{i}.jsonl")
        require(samples and all(s.get("kind") == "node_sample" and s.get("dropped_samples") == 0 for s in samples),
                "observer missing/drop")
        recorder = common.read_json(run / f"recorder-{i}.json")
        require(recorder.get("dropped") == 0 and isinstance(recorder.get("events"), list), "recorder drop")
    qc = {"schema": "pluto.file02.r10.qc.v1", "case_id": "R10", "status": "PASS_BOUNDED_DIAGNOSTIC",
          "n_networks": 1, "n_submissions": 3, "plan_sha256": common.digest(session / "plan.json"),
          "raw_manifest_sha256": manifest}
    aggregate = {"schema": "pluto.file02.r10.aggregate.v1", "case_id": "R10",
                 "status": "PARTIAL_VALID_NEAR_8192_NOT_FEASIBLE",
                 "max_valid_canonical_bytes": 2602, "mempool_max_tx_bytes": 8192,
                 "at_limit_rejected_stage": "structural", "over_limit_rejected_before_checktx": True,
                 "valid_checktx_service_ns": oracle["valid_checktx_service_ns"],
                 "invalid_at_limit_checktx_service_ns": oracle["invalid_at_limit_checktx_service_ns"],
                 "comet_precheck_cpu_ns": None, "comet_precheck_cpu_status": "UNAVAILABLE",
                 "ci95": None, "ci_status": "INSUFFICIENT_INDEPENDENT_RUNS",
                 "plan_sha256": qc["plan_sha256"], "raw_manifest_sha256": manifest}
    return rows, qc, aggregate


def write_outputs(session, plan):
    rows, qc, aggregate = derive(session, plan)
    r01.write_new(session / "result_per_run.json", {"schema": "pluto.file02.r10.per_run.v1", "rows": rows})
    with (session / "result_per_run.csv").open("x", newline="", encoding="utf-8") as stream:
        keys = ("label", "sha256", "wire_bytes", "rpc_code", "rpc_reason", "rpc_error", "rpc_roundtrip_ns")
        writer = csv.DictWriter(stream, fieldnames=keys)
        writer.writeheader()
        writer.writerows({key: row[key] for key in keys} for row in rows)
    r01.write_new(session / "qc_report.json", qc)
    r01.write_new(session / "result_aggregate.json", aggregate)
    (session / "figures").mkdir()
    r01.write_new(session / "figures/status.json", {"status": "NOT_REQUIRED", "reason": "one boundary network; no scientific curve"})
    (session / "decision.md").write_text(
        "# R10 decision\n\nA valid canonical H transaction cannot approach the 8,192-B mempool limit: "
        "maximum is 2,602 B. The 8,192-B malformed extension reaches structural CheckTx rejection; "
        "8,193 B is rejected by Comet before CheckTx. Per-request Comet CPU and CI are UNAVAILABLE. "
        "R10 remains partial; no DoS inference.\n", encoding="utf-8")
    return qc, aggregate


def execute():
    session = ARTIFACTS / ("r10-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8])
    prepare(session)
    plan = common.read_json(session / "plan.json")
    raw = session / "raw"
    with (raw / "build.log").open("wb") as stream:
        build = subprocess.run(["make", "build", "build-comet-s05"], cwd=ROOT, stdout=stream, stderr=subprocess.STDOUT)
    require(build.returncode == 0, f"build failed: {session}")
    env = os.environ.copy()
    env.update(FILE02_R10_ROOT=str(raw), FILE02_SOURCE_COMMIT=plan["source_commit"], FILE02_SOURCE_TREE=plan["source_tree"])
    with (raw / "integration.log").open("wb") as stream:
        run = subprocess.run(["go", "test", "-tags", "integration fourval", "./test/integration",
                              "-run", "^TestFile02R10$", "-timeout", "180s", "-count=1", "-v"],
                             cwd=ROOT, env=env, stdout=stream, stderr=subprocess.STDOUT, timeout=200)
    manifest = common.freeze_raw(raw)
    if run.returncode != 0:
        print(f"R10_INTEGRATION_FAILED session={session} raw_manifest_sha256={manifest}")
        return 2
    qc, aggregate = write_outputs(session, plan)
    print(f"R10 session={session} qc={qc['status']} status={aggregate['status']} raw_manifest_sha256={manifest}")
    return 0


def verify(session):
    session = Path(session).resolve()
    require(session.parent == ARTIFACTS.resolve(), "private session")
    plan = common.read_json(session / "plan.json")
    require(plan.get("schema") == "pluto.file02.r10.plan.v1" and plan.get("status") == "LOCKED" and
            plan.get("source_tree") == r01.git("rev-parse", plan["source_commit"] + "^{tree}") and
            plan.get("safety_config_sha256") == common.digest(ROOT / "research/config/s06_pilot.json"),
            "locked plan provenance")
    rows, qc, aggregate = derive(session, plan)
    require(common.read_json(session / "result_per_run.json") == {"schema": "pluto.file02.r10.per_run.v1", "rows": rows} and
            common.read_json(session / "qc_report.json") == qc and
            common.read_json(session / "result_aggregate.json") == aggregate, "derived artifact mismatch")
    with (session / "result_per_run.csv").open(newline="", encoding="utf-8") as stream:
        csv_rows = list(csv.DictReader(stream))
    require(len(csv_rows) == 3 and all(item["label"] == row["label"] for item, row in zip(csv_rows, rows)), "CSV mismatch")
    print(f"R10_VERIFY=OK qc={qc['status']} raw_manifest_sha256={qc['raw_manifest_sha256']}")


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
        print(f"R10_ERROR: {exc}", file=sys.stderr)
        sys.exit(2)
