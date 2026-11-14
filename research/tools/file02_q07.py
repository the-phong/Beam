#!/usr/bin/env python3
"""Q07 frozen-run oracle, exact inclusion and invalid rejection audit."""

import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import sys

import file02_common as common
import file02_q01 as q01
import file02_q02 as q02
import file02_q04 as q04
import file02_r06 as r06

ROOT = Path(__file__).resolve().parents[2]
OUTPUT = ROOT / "research/evidence/file02/quality/Q07_NO_INVALID_ACCEPTED.json"
ORACLE_PATH = "test/integration/s07_reproduce_test.go"
ORACLE_SHA256 = "1cde090e38452986d1718bec81650e21ed5bcf0df0138b9b5b0cc05169c5a859"
REPLAY_BINARY = ROOT / "build/file02-q07-replay"


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def oracle_source_digest(source_commit):
    raw = subprocess.check_output(["git", "show", source_commit + ":" + ORACLE_PATH], cwd=ROOT)
    require(b"func s07Oracle(" in raw and b"func s07EqualCounts(" in raw,
            "live oracle function missing")
    function = raw.split(b"func s07Oracle(", 1)[1].split(b"func s07EqualCounts(", 1)[0]
    return hashlib.sha256(function).hexdigest()


def assert_commit_set(valid, invalid, committed):
    require(len(valid) == len(set(valid)) and len(invalid) == len(set(invalid)) and
            not set(valid) & set(invalid), "Q07 planned hash duplication")
    require(len(committed) == len(set(committed)) and set(committed) == set(valid) and
            not set(committed) & set(invalid), "Q07 valid exact-once/invalid absence")


def session_for(case):
    return q01.ARTIFACTS / q01.INPUTS[case][0] if case in q01.INPUTS else q02.ARTIFACTS / q04.Q02_SESSION


def replay_state(run, case):
    session = session_for(case)
    chain_id = common.read_json(session / "plan.json")["chain_id"]
    proc = subprocess.run([str(REPLAY_BINARY), "-run-dir", str(run), "-chain-id", chain_id],
                          cwd=ROOT, text=True, capture_output=True)
    require(proc.returncode == 0, "Q07 deterministic state replay: " + proc.stderr.strip())
    result = json.loads(proc.stdout)
    require(result["schema"] == "pluto.file02.q07_replay.v1" and
            result["status"] == "PASS" and result["run_id"] == run.name,
            "Q07 replay output identity")
    return result


def audit_run(source):
    case, run_id = source["case_id"], source["run_id"]
    run = session_for(case) / "raw/runs" / run_id
    meta, plan, result, oracle, commits = (common.read_json(run / name) for name in
        ("metadata.json", "plan.json", "result.json", "oracle.json", "commit_observations.json"))
    require(meta["case_id"] == case and meta["run_id"] == plan["run_id"] ==
            result["run_id"] == oracle["run_id"] == commits["run_id"] == run_id and
            result["status"] == oracle["status"] == "OK" and
            oracle["schema"] == "pluto.s07.oracle.v1" and
            oracle["common_height"] >= 8 and oracle["block_id"] and oracle["app_hash"] and
            meta["source_commit"] and meta["source_tree"] and
            result["plan_sha256"] == common.digest(run / "plan.json"),
            "Q07 provenance/live oracle")
    digest = oracle_source_digest(meta["source_commit"])
    require(digest == ORACLE_SHA256 and meta["source_tree"] ==
            subprocess.check_output(["git", "rev-parse", meta["source_commit"]+"^{tree}"],
                                    cwd=ROOT, text=True).strip(),
            "Q07 live oracle implementation/source tree")
    planned = plan["events"]
    actual = result["events"]
    require(result["planned_count"] == result["actual_send_count"] == len(planned) ==
            len(actual) and len(planned) in (157, 241), "Q07 send coverage")
    records = {record["tx_sha256"]: record for record in r06.samples(run / "recorder-0.jsonl")
               if record.get("check_type") == "new" and record.get("tx_sha256")}
    require(len(records) == len(planned), "Q07 recorder coverage")
    valid, invalid, slots = [], [], []
    for p, a in zip(planned, actual):
        wire_hash = hashlib.sha256(bytes.fromhex(p["bytes_hex"])).hexdigest()
        require(p["id"] == a["id"] and p["sha256"] == wire_hash and
                a["actual_code"] == p["expected_code"] and
                not a.get("submission_error") and
                records[wire_hash]["code"] == p["expected_code"] and
                records[wire_hash]["last_stage"] == p["expected_stage"],
                "Q07 hash/CheckTx/stage")
        slots.append(p["account_slot"])
        (valid if p["traffic"] == "legitimate" else invalid).append(wire_hash)
    require(len(set(slots)) == len(slots) and all(0 <= slot < 241 for slot in slots),
            "Q07 one-use funded sender slots")
    committed = [event["sha256"] for event in commits["events"]]
    assert_commit_set(valid, invalid, committed)
    require(oracle["valid_committed"] == len(valid) and
            oracle["invalid_rejected"] == len(invalid), "Q07 oracle counts")
    replay = replay_state(run, case)
    require(replay["genesis_accounts"] == 242 and
            replay["initial_supply"] == replay["final_supply"] == 482 and
            replay["valid_replayed"] == len(valid) and
            replay["invalid_not_applied"] == len(invalid) and
            replay["app_hash"] == oracle["app_hash"], "Q07 full state replay/supply")
    return {"case_id": case, "run_id": run_id, "source_raw_manifest_sha256":
            source["source_raw_manifest_sha256"], "oracle_source_sha256": digest,
            "common_height": oracle["common_height"], "block_id": oracle["block_id"],
            "app_hash": oracle["app_hash"], "valid_exact_once": len(valid),
            "invalid_rejected_and_absent": len(invalid),
            "genesis_accounts": replay["genesis_accounts"],
            "initial_supply": replay["initial_supply"],
            "final_supply": replay["final_supply"],
            "state_validation": "REPLAYED_APPHASH_SUPPLY_AND_LIVE_ORACLE_ALL_FOUR_USED_ACCOUNT_NONCE",
            "qc_status": "PASS"}


def derive():
    build = subprocess.run(["go", "build", "-o", str(REPLAY_BINARY),
                            "./cmd/research/q07-replay"], cwd=ROOT, text=True,
                           capture_output=True)
    require(build.returncode == 0, "Q07 replay build: " + build.stderr.strip())
    rate = q04.derive()
    require(rate["status"] == "PASS" and rate["n_networks"] == 248 and
            not rate["excluded_run_ids"], "Q07 source rate/QC")
    rows, failed = [], []
    for source in rate["runs"]:
        try:
            rows.append(audit_run(source))
        except (ValueError, KeyError, TypeError, OSError, subprocess.CalledProcessError) as exc:
            failed.append({"run_id": source["run_id"], "reason": str(exc)})
    return {"schema": "pluto.file02.q07.no_invalid_accepted.v1",
            "status": "PASS" if len(rows) == 248 and not failed else "INVALID_FOR_INFERENCE",
            "n_planned": 248, "n_valid": len(rows), "failed_or_excluded": failed,
            "oracle_function_sha256": ORACLE_SHA256,
            "state_evidence_limit": "per-validator account snapshots not serialized; committed AppHash matches full deterministic genesis replay at header height minus one",
            "runs": rows}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("run", "verify"))
    args = parser.parse_args()
    try:
        report = derive()
        rendered = json.dumps(report, sort_keys=True, indent=2) + "\n"
        if args.command == "run":
            require(not OUTPUT.exists(), "Q07 output exists")
            OUTPUT.write_text(rendered)
        else:
            require(OUTPUT.read_text() == rendered, "Q07 output drift")
        print(f"Q07_{args.command.upper()}={report['status']} "
              f"valid={report['n_valid']} excluded={len(report['failed_or_excluded'])}")
        return 0 if report["status"] == "PASS" else 2
    except (ValueError, KeyError, TypeError, OSError, subprocess.CalledProcessError,
            json.JSONDecodeError) as exc:
        print(f"Q07_ERROR={exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
