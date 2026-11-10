#!/usr/bin/env python3
"""Hash a fixed Git revision without copying or publishing historical raw data."""

import argparse
import hashlib
import json
import subprocess
from pathlib import Path


def git(*args: str) -> bytes:
    return subprocess.check_output(["git", *args])


def category(path: str) -> str | None:
    if path in {"go.mod", "go.sum", "Makefile", ".github/workflows/verify.yml"}:
        return "source"
    if path.startswith(("cmd/", "internal/", "test/")) and path.endswith(".go"):
        return "source"
    if path.startswith("research/specs/") and path.endswith(".md"):
        return "spec"
    if path in {
        "research/ARCHITECTURE.md",
        "research/DECISIONS.md",
        "research/MASTER_PLAN.md",
        "research/PHASE_STATUS.md",
        "research/config/manifest.yaml",
    }:
        return "spec"
    if path.startswith("research/evidence/"):
        # Raw logs and binary review bundles remain at the immutable source
        # revision; this public manifest lists only selected summary records.
        parts = Path(path).parts
        if "raw" in parts or path.endswith((".log", ".patch", ".patch.gz", ".patch.b64")):
            return None
        if path.endswith((".md", ".txt", ".json", ".csv", ".svg", ".sha256")):
            return "published_evidence"
    return None


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--ref", required=True, help="immutable source commit")
    parser.add_argument("--out", required=True, type=Path)
    args = parser.parse_args()
    commit = git("rev-parse", "--verify", f"{args.ref}^{{commit}}").decode().strip()
    tree = git("rev-parse", f"{commit}^{{tree}}").decode().strip()
    paths = git("ls-tree", "-r", "-z", "--name-only", commit).decode().split("\0")
    entries = []
    for path in paths:
        kind = category(path)
        if kind is None:
            continue
        raw = git("show", f"{commit}:{path}")
        entries.append({
            "category": kind,
            "path": path,
            "sha256": hashlib.sha256(raw).hexdigest(),
            "bytes": len(raw),
        })
    result = {
        "schema": "pluto.research-v2.baseline-manifest.v1",
        "source_commit": commit,
        "source_tree": tree,
        "selection": "Go source, build/CI, normative specs, selected already-published summary evidence; no raw logs or review bundles",
        "entries": entries,
    }
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(result, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(f"wrote {len(entries)} digests to {args.out}")


if __name__ == "__main__":
    main()
