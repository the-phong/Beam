#!/usr/bin/env python3
"""Audit frozen R/A/Q sessions against their recorded historical source commits."""

from __future__ import annotations

import argparse
import hashlib
import json
import subprocess
from collections import Counter
from pathlib import Path

import file02_common as common


ROOT = Path(__file__).resolve().parents[2]
KNOWN_EXCLUDED_RAW_MISMATCH = "r15-20261113T074320Z-62398090"


def git_bytes(*args: str) -> bytes:
    return subprocess.check_output(["git", *args], cwd=ROOT)


def audit(root: Path) -> dict:
    rows = []
    for plan_path in sorted(root.glob("*/plan.json")):
        session = plan_path.parent
        plan = common.read_json(plan_path)
        qc_path = session / "qc_report.json"
        qc = common.read_json(qc_path) if qc_path.is_file() else None
        source = plan["source_commit"]
        tree = git_bytes("rev-parse", source + "^{tree}").decode().strip()
        source_tree_status = "MATCH" if tree == plan["source_tree"] else "MISMATCH"
        module_status = "UNAVAILABLE_IN_PLAN"
        if plan.get("module_sha256"):
            historical_module = hashlib.sha256(git_bytes("show", source + ":go.mod")).hexdigest()
            module_status = "MATCH" if historical_module == plan["module_sha256"] else "MISMATCH"
        config_status = "UNAVAILABLE_IN_PLAN"
        raw_config = session / "raw" / "config.json"
        if plan.get("config_sha256") and raw_config.is_file():
            config_status = "MATCH" if common.digest(raw_config) == plan["config_sha256"] else "MISMATCH"
        elif plan.get("config_sha256"):
            config_status = "UNAVAILABLE_RAW_CONFIG"

        manifest_path = session / "raw" / "manifest.json"
        manifest_sha = common.digest(manifest_path)
        raw_status = "PASS"
        raw_reason = None
        try:
            common.verify_raw(session / "raw")
        except (ValueError, OSError, KeyError, TypeError) as exc:
            raw_status, raw_reason = "FAIL", str(exc)
        qc_status = qc.get("status") if qc else "NO_QC_REPORT"
        qc_plan_status = "NO_QC_REPORT" if qc is None else (
            "MATCH" if qc.get("plan_sha256") == common.digest(plan_path) else "MISMATCH")
        qc_manifest_status = "NO_QC_REPORT" if qc is None else (
            "MATCH" if qc.get("raw_manifest_sha256") == manifest_sha else "MISMATCH")
        role = "EXCLUDED" if qc_status == "INVALID_FOR_INFERENCE" or session.name == KNOWN_EXCLUDED_RAW_MISMATCH else "QC_ELIGIBLE"
        if session.name == "q02-20261114T025508Z-a4b258cb":
            role = "SUPERSEDED_ORDER_CONFOUNDING"
        if session.name == "q02-20261114T044327Z-e7f236e2":
            role = "CURRENT_Q02_V2"
        rows.append({
            "case_id": plan.get("case_id"), "session_id": session.name,
            "source_commit": source, "source_tree_status": source_tree_status,
            "historical_module_status": module_status,
            "raw_config_status": config_status,
            "qc_status": qc_status, "qc_plan_hash_status": qc_plan_status,
            "qc_manifest_hash_status": qc_manifest_status,
            "raw_file_integrity": raw_status, "raw_failure_reason": raw_reason,
            "raw_manifest_sha256": manifest_sha, "inference_role": role,
        })
    if not rows:
        raise ValueError("no File 02 sessions")
    unexpected = [r for r in rows if r["source_tree_status"] != "MATCH" or
                  r["historical_module_status"] == "MISMATCH" or
                  r["raw_config_status"] == "MISMATCH" or
                  r["qc_plan_hash_status"] == "MISMATCH" or
                  r["qc_manifest_hash_status"] == "MISMATCH" or
                  (r["raw_file_integrity"] == "FAIL" and r["session_id"] != KNOWN_EXCLUDED_RAW_MISMATCH)]
    if unexpected:
        raise ValueError("unexpected provenance/integrity failures: " + ", ".join(r["session_id"] for r in unexpected))
    return {"schema": "pluto.file02.x02_historical_audit.v1",
            "counts": {"sessions": len(rows), "raw_pass": sum(r["raw_file_integrity"] == "PASS" for r in rows),
                       "raw_fail": sum(r["raw_file_integrity"] == "FAIL" for r in rows),
                       "qc_status": dict(Counter(r["qc_status"] for r in rows))},
            "rows": rows}


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("write", "check"))
    parser.add_argument("--root", type=Path, default=ROOT / "research/artifacts/file02")
    parser.add_argument("--output", type=Path, default=ROOT / "research/evidence/file02/mechanism/HISTORICAL_EVIDENCE_QC.json")
    args = parser.parse_args()
    expected = json.dumps(audit(args.root), sort_keys=True, indent=2) + "\n"
    if args.mode == "write":
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(expected, encoding="utf-8")
    elif args.output.read_text(encoding="utf-8") != expected:
        raise SystemExit("historical audit output drift")


if __name__ == "__main__":
    main()
