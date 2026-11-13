#!/usr/bin/env python3
"""Audit fresh-network and disjoint monotonic phases for File 02 COMMON-02 QA."""

import argparse
import json
import os
from pathlib import Path
import subprocess
import sys

import file02_common as common
import s07_orchestrate as s07

ROOT = Path(__file__).resolve().parents[2]
PHASES = ("setup", "warm-up", "execute", "drain", "oracle", "collect", "cooldown", "analyze", "freeze")


def require(ok, message):
    if not ok:
        raise ValueError(message)


def validate_profile(run):
    metadata = common.read_json(run / "metadata.json")
    plan = common.read_json(run / "plan.json")
    result = common.read_json(run / "result.json")
    cooldown = common.read_json(run / "cooldown.json")
    phases = [json.loads(line) for line in (run / "phases.jsonl").read_text().splitlines()]
    require(tuple(row.get("phase") for row in phases) == PHASES, "missing or overlapping phase sequence")
    stamps = [row.get("elapsed_monotonic_ns") for row in phases[:7]]
    require(all(isinstance(ns, int) and ns >= 0 for ns in stamps) and
            all(a < b for a, b in zip(stamps, stamps[1:])), "nonmonotonic phase markers")
    run_id = metadata["run_id"]
    require(run_id == plan.get("run_id") == result.get("run_id") == cooldown.get("run_id"), "run identity mismatch")
    require(cooldown.get("schema") == "pluto.file02.cooldown_qa.v1" and
            cooldown.get("planned_ms") == 1000 and cooldown.get("network_live") is True,
            "cooldown configuration or live-network check")
    start_ns, end_ns = cooldown.get("start_monotonic_ns"), cooldown.get("end_monotonic_ns")
    require(isinstance(start_ns, int) and isinstance(end_ns, int) and
            start_ns > stamps[5] and end_ns >= start_ns + 1_000_000_000 and
            phases[6]["elapsed_monotonic_ns"] >= end_ns and
            cooldown.get("elapsed_ns", 0) >= 1_000_000_000, "cooldown duration/order")
    warmup, steady, drain = plan.get("warmup_ns"), plan.get("steady_ns"), plan.get("drain_ns")
    require(all(isinstance(ns, int) and ns > 0 for ns in (warmup, steady, drain)), "plan phase durations")
    planned = plan.get("events", [])
    actual = result.get("events", [])
    require(len(planned) == len(actual) == result.get("actual_send_count") and len(actual) > 0,
            "planned/actual event count")
    counts = {"warmup": 0, "steady": 0}
    for expected, observed in zip(planned, actual):
        target, sent = expected.get("offset_ns"), observed.get("actual_offset_ns")
        require(expected.get("id") == observed.get("id") and target == observed.get("planned_offset_ns")
                and isinstance(target, int) and isinstance(sent, int) and 0 <= target < warmup + steady
                and 0 <= sent < warmup + steady, "send entered drain/cooldown or event mismatch")
        planned_phase = "warmup" if target < warmup else "steady"
        actual_phase = "warmup" if sent < warmup else "steady"
        require(planned_phase == actual_phase, "send crossed warmup/steady boundary")
        counts[actual_phase] += 1
    require(counts["warmup"] > 0 and counts["steady"] > 0, "empty warmup or steady phase")
    homes = [run / "network" / f"validator-{i}" for i in range(4)]
    require(all(home.is_dir() and (home / "config" / "config.toml").is_file() for home in homes)
            and len({home.stat().st_ino for home in homes}) == 4, "four fresh validator homes missing")
    return {"run_id": run_id, "profile": metadata["profile"],
            "warmup_actual_sends": counts["warmup"], "steady_actual_sends": counts["steady"],
            "drain_actual_sends": 0, "cooldown_actual_sends": 0,
            "cooldown_elapsed_ns": cooldown["elapsed_ns"],
            "network_home_count": 4, "monotonic_phase_markers": len(stamps)}


def audit_session(session_path):
    session = s07.private_session(Path(session_path))
    verified = s07.verify(session)
    rows = [validate_profile(session / f"profile-{profile}") for profile in ("C", "CP", "P", "H")]
    require(len({row["run_id"] for row in rows}) == 4, "reused run ID")
    output = ROOT / "research" / "artifacts" / "file02" / ("common02-" + session.name)
    output.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    output.mkdir(mode=0o700)
    summary = {"schema": "pluto.file02.common02_audit.v1", "scope": "QA_ONLY",
               "source_commit": verified["source_commit"],
               "s07_manifest_sha256": common.digest(session / "manifest.json"), "runs": rows}
    common.write_once(output / "summary.json", json.dumps(summary, indent=2, sort_keys=True) + "\n")
    return output, common.freeze_raw(output)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    p = sub.add_parser("audit")
    p.add_argument("session")
    sub.add_parser("smoke")
    args = parser.parse_args()
    try:
        if args.command == "smoke":
            env = os.environ.copy()
            env["S07_FILE02_COOLDOWN_MS"] = "1000"
            completed = subprocess.run([sys.executable, str(ROOT / "research/tools/s07_orchestrate.py"), "smoke"],
                                       cwd=ROOT, env=env, capture_output=True, text=True, timeout=1200)
            print(completed.stdout, end="")
            require(completed.returncode == 0, f"bounded four-profile smoke failed: {completed.stderr[-1000:]}")
            sessions = [line.removeprefix("S07_SESSION=") for line in completed.stdout.splitlines()
                        if line.startswith("S07_SESSION=")]
            require(len(sessions) == 1, "S07 session ID missing")
            session = sessions[0]
        else:
            session = args.session
        path, manifest = audit_session(session)
        print(f"COMMON02_STATUS=OK runs=4 audit={path} manifest_sha256={manifest}")
        return 0
    except (ValueError, OSError, KeyError, TypeError, subprocess.TimeoutExpired) as exc:
        print(f"COMMON02_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
