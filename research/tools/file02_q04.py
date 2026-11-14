#!/usr/bin/env python3
"""Q04 offered socket-rate and encoded-byte fidelity audit."""

import argparse
import json
from pathlib import Path
import sys

import file02_a13 as a13
import file02_attack as attack
import file02_common as common
import file02_q01 as q01
import file02_q02 as q02

ROOT = Path(__file__).resolve().parents[2]
OUTPUT = ROOT / "research/evidence/file02/quality/Q04_RATE_FIDELITY_V2.json"
Q02_SESSION = "q02-20261114T044327Z-e7f236e2"
Q02_MANIFEST = "58265e00a8c0fc08a37cb4e2935633d85304a1e70d32142df5d9eaa4d1180f3f"
COMPONENTS = ("total", "fixed", "background")


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def component(event):
    return "fixed" if event["scenario"] == "fixed_legitimate" else "background"


def counts(events):
    windows = [{part: {"tx_s": 0, "bytes_s": 0} for part in COMPONENTS}
               for _ in range(12)]
    for offset, size, kind in events:
        if offset < 1_000_000_000:
            continue
        second = (offset - 1_000_000_000) // 1_000_000_000
        require(0 <= second < 12, "steady socket write outside 12 locked windows")
        for part in ("total", kind):
            windows[second][part]["tx_s"] += 1
            windows[second][part]["bytes_s"] += size
    return windows


def audit_run(run, case, manifest):
    meta, workload, result, oracle = (common.read_json(run / name) for name in
                                      ("metadata.json", "plan.json", "result.json", "oracle.json"))
    planned, actual = workload["events"], result["events"]
    require(meta["case_id"] == case and meta["run_id"] == workload["run_id"] ==
            result["run_id"] == oracle["run_id"] and result["status"] == oracle["status"] == "OK" and
            result["planned_count"] == result["actual_send_count"] == len(planned) == len(actual) and
            result["plan_sha256"] == common.digest(run / "plan.json") and
            result["config_sha256"] == meta["config_sha256"], "Q04 provenance/sends")
    target_events, offered_events, jitters = [], [], []
    for p, a in zip(planned, actual):
        require(p["id"] == a["id"] and not a.get("submission_error") and
                isinstance(a.get("actual_send_monotonic_ns"), int) and
                isinstance(a.get("actual_offset_ns"), int) and
                a["wire_bytes"] == len(p["bytes_hex"])//2, "Q04 actual socket write/size")
        kind = component(p)
        target_events.append((p["offset_ns"], a["wire_bytes"], kind))
        offered_events.append((a["actual_offset_ns"], a["wire_bytes"], kind))
        jitters.append(abs(a["actual_offset_ns"]-p["offset_ns"])/1_000_000)
    target, achieved = counts(target_events), counts(offered_events)
    violations = []
    for second, (t, a) in enumerate(zip(target, achieved)):
        for part in COMPONENTS:
            for unit in ("tx_s", "bytes_s"):
                expected, observed = t[part][unit], a[part][unit]
                if (expected == 0 and observed != 0) or (expected > 0 and
                    (abs(observed-expected)/expected > .05 or observed/expected < .90)):
                    violations.append({"second": second, "component": part, "unit": unit,
                                       "target": expected, "achieved": observed})
    return {"case_id": case, "run_id": workload["run_id"], "profile": meta["profile"],
            "condition": meta["condition"], "source_raw_manifest_sha256": manifest,
            "plan_sha256": result["plan_sha256"], "target_windows": target,
            "achieved_windows": achieved, "max_abs_socket_offset_jitter_ms": max(jitters),
            "offered_socket_count": result["actual_send_count"],
            "eventual_valid_committed": oracle["valid_committed"],
            "violations": violations, "qc_status": "PASS" if not violations else "EXCLUDE_RATE_FIDELITY"}


def source_runs():
    output = []
    for case, (name, manifest) in q01.INPUTS.items():
        session = q01.ARTIFACTS / name
        plan = common.read_json(session / "plan.json")
        if case == "A13":
            a13.verify(session)
            rows, qc, _ = a13.derive(session, plan)
            expected = 8
        else:
            attack.verify(session)
            rows, qc, _ = attack.derive(session, plan)
            expected = 24
        require(qc["status"] == "PASS" and qc["raw_manifest_sha256"] == manifest and
                len(rows) == expected, "A source QC")
        for row in rows:
            require(row["qc_status"] == "PASS", "A row QC")
            output.append(audit_run(session / "raw/runs" / row["run_id"], case, manifest))
    session = q02.ARTIFACTS / Q02_SESSION
    q02.verify(session)
    plan = common.read_json(session / "plan.json")
    rows, _, qc, _ = q02.derive(session, plan)
    require(qc["status"] == "PASS" and qc["raw_manifest_sha256"] == Q02_MANIFEST and
            len(rows) == 96, "Q02 source QC")
    for row in rows:
        output.append(audit_run(session / "raw/runs" / row["run_id"], "Q02", Q02_MANIFEST))
    require(len(output) == 248 and len({r["run_id"] for r in output}) == 248,
            "Q04 expected network coverage")
    return output


def derive():
    rows = source_runs()
    failed = [row["run_id"] for row in rows if row["qc_status"] != "PASS"]
    return {"schema": "pluto.file02.q04.rate_fidelity.v1",
            "status": "PASS" if not failed else "RATE_QC_EXCLUSIONS",
            "measurement": "actual_monotonic_socket_write_offered_tx_s_and_encoded_bytes_s",
            "n_networks": len(rows), "excluded_run_ids": failed,
            "threshold_fraction": .05, "minimum_window_fraction": .90,
            "runs": rows, "confirmed_goodput_used_as_offered": False}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("run", "verify"))
    args = parser.parse_args()
    try:
        report = derive()
        rendered = json.dumps(report, sort_keys=True, indent=2) + "\n"
        if args.command == "run":
            require(not OUTPUT.exists(), "Q04 output exists")
            OUTPUT.write_text(rendered)
        else:
            require(OUTPUT.read_text() == rendered, "Q04 output drift")
        print(f"Q04_{args.command.upper()}={report['status']} networks={report['n_networks']} "
              f"excluded={len(report['excluded_run_ids'])}")
        return 0 if report["status"] == "PASS" else 2
    except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError) as exc:
        print(f"Q04_ERROR={exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
