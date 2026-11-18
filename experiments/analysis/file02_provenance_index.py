#!/usr/bin/env python3
"""Build a publication-safe index of frozen File 02 session provenance."""

from __future__ import annotations

import argparse
import csv
import hashlib
import io
import json
from pathlib import Path


FIELDS = (
    "case_id",
    "session_id",
    "record_kind",
    "run_id",
    "row_ref",
    "session_qc_status",
    "row_qc_status",
    "source_commit",
    "plan_sha256",
    "raw_manifest_sha256",
    "provenance_status",
    "inference_role",
)

SUPERSEDED_SESSIONS = {
    "q02-20261114T025508Z-a4b258cb": "SUPERSEDED_ORDER_CONFOUNDING",
}
CURRENT_SESSIONS = {
    "q02-20261114T044327Z-e7f236e2": "CURRENT_Q02_V2",
}


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def build_rows(root: Path) -> list[dict[str, str]]:
    records: list[dict[str, str]] = []
    qc_paths = sorted(root.glob("*/qc_report.json"))
    if not qc_paths:
        raise ValueError(f"no QC reports under {root}")
    for qc_path in qc_paths:
        session = qc_path.parent
        plan_path = session / "plan.json"
        manifest_path = session / "raw" / "manifest.json"
        if not plan_path.is_file() or not manifest_path.is_file():
            raise ValueError(f"missing plan or frozen raw manifest: {session.name}")
        qc = json.loads(qc_path.read_text(encoding="utf-8"))
        plan = json.loads(plan_path.read_text(encoding="utf-8"))
        plan_hash, manifest_hash = sha256(plan_path), sha256(manifest_path)
        if qc.get("plan_sha256") != plan_hash:
            raise ValueError(f"QC plan hash mismatch: {session.name}")
        if qc.get("raw_manifest_sha256") != manifest_hash:
            raise ValueError(f"QC raw manifest hash mismatch: {session.name}")

        result_path = session / "result_per_run.json"
        result = json.loads(result_path.read_text(encoding="utf-8")) if result_path.is_file() else {}
        rows = result.get("rows", []) if isinstance(result, dict) else result
        if not isinstance(rows, list):
            raise ValueError(f"invalid result rows: {session.name}")
        qc_status = str(qc.get("status", "UNAVAILABLE"))
        role = SUPERSEDED_SESSIONS.get(session.name) or CURRENT_SESSIONS.get(session.name)
        if role is None:
            role = "HISTORICAL_QC_PASS" if qc_status.startswith("PASS") else "INVALID_FOR_INFERENCE"
        for index, row in enumerate(rows or [{}], start=1):
            if not isinstance(row, dict):
                raise ValueError(f"invalid result row {index}: {session.name}")
            run_id = str(row.get("run_id") or "")
            kind = "RUN" if run_id else ("RESULT_ROW_NO_RUN_ID" if rows else "SESSION_NO_RESULT_ROWS")
            records.append({
                "case_id": str(qc.get("case_id") or plan.get("case_id") or ""),
                "session_id": session.name,
                "record_kind": kind,
                "run_id": run_id,
                "row_ref": f"result_per_run.json#rows[{index-1}]" if rows else "",
                "session_qc_status": qc_status,
                "row_qc_status": str(row.get("qc_status") or "UNAVAILABLE"),
                "source_commit": str(plan.get("source_commit") or "UNAVAILABLE_IN_PLAN"),
                "plan_sha256": plan_hash,
                "raw_manifest_sha256": manifest_hash,
                "provenance_status": "QC_PLAN_AND_MANIFEST_HASH_MATCH",
                "inference_role": role,
            })
    return records


def render(rows: list[dict[str, str]]) -> str:
    out = io.StringIO(newline="")
    writer = csv.DictWriter(out, fieldnames=FIELDS, lineterminator="\n")
    writer.writeheader()
    writer.writerows(rows)
    return out.getvalue()


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("write", "check"))
    parser.add_argument("--root", type=Path, default=Path("research/artifacts/file02"))
    parser.add_argument("--output", type=Path, default=Path("research/evidence/file02/quality/DATA_PROVENANCE_INDEX.csv"))
    args = parser.parse_args()
    expected = render(build_rows(args.root))
    if args.mode == "write":
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(expected, encoding="utf-8")
    elif args.output.read_text(encoding="utf-8") != expected:
        raise SystemExit("provenance index does not match frozen sessions")


if __name__ == "__main__":
    main()
