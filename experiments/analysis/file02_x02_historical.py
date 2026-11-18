#!/usr/bin/env python3
"""Recheck File 02 raw with the common integrity tool at each locked source commit."""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path

import file02_common as common


ROOT = Path(__file__).resolve().parents[2]
KNOWN_EXCLUDED_RAW_MISMATCH = "r15-20261113T074320Z-62398090"


def run() -> dict:
    sessions = sorted((ROOT / "research/artifacts/file02").glob("*/plan.json"))
    if not sessions:
        raise ValueError("no frozen sessions")
    rows = []
    with tempfile.TemporaryDirectory(prefix="file02-x02-history-") as parent:
        checkout = Path(parent) / "source"
        first = common.read_json(sessions[0])["source_commit"]
        subprocess.run(["git", "worktree", "add", "--detach", "--quiet", str(checkout), first],
                       cwd=ROOT, check=True, capture_output=True)
        try:
            active = first
            for plan_path in sessions:
                plan = common.read_json(plan_path)
                source = plan["source_commit"]
                if source != active:
                    subprocess.run(["git", "checkout", "--detach", "--quiet", source],
                                   cwd=checkout, check=True, capture_output=True)
                    active = source
                raw = plan_path.parent / "raw"
                environment = os.environ.copy()
                environment["PYTHONPATH"] = str(checkout / "research/tools")
                environment["PYTHONDONTWRITEBYTECODE"] = "1"
                command = [sys.executable, "-c", "import file02_common as c, sys; print(c.verify_raw(sys.argv[1]))", str(raw)]
                result = subprocess.run(command, cwd=checkout, env=environment,
                                        capture_output=True, text=True)
                status = "PASS" if result.returncode == 0 else "FAIL"
                rows.append({"session_id": plan_path.parent.name,
                             "case_id": plan.get("case_id"),
                             "tool_source_commit": source,
                             "historical_raw_verify": status,
                             "raw_manifest_sha256": common.digest(raw / "manifest.json"),
                             "diagnostic": None if status == "PASS" else result.stderr.strip().splitlines()[-1]})
        finally:
            subprocess.run(["git", "worktree", "remove", "--force", str(checkout)],
                           cwd=ROOT, check=True, capture_output=True)
    failures = [row["session_id"] for row in rows if row["historical_raw_verify"] == "FAIL"]
    if failures != [KNOWN_EXCLUDED_RAW_MISMATCH]:
        raise ValueError("unexpected historical-tool raw failures: " + repr(failures))
    return {"schema": "pluto.file02.x02_historical_tool_raw.v1",
            "sessions": len(rows), "pass": len(rows) - len(failures),
            "fail_excluded": len(failures), "rows": rows}


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("write", "check"))
    parser.add_argument("--output", type=Path,
                        default=ROOT / "research/evidence/file02/mechanism/HISTORICAL_TOOL_RAW_AUDIT.json")
    args = parser.parse_args()
    expected = json.dumps(run(), sort_keys=True, indent=2) + "\n"
    if args.mode == "write":
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(expected, encoding="utf-8")
    elif args.output.read_text(encoding="utf-8") != expected:
        raise SystemExit("historical-tool raw audit drift")


if __name__ == "__main__":
    main()
