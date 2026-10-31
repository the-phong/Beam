#!/usr/bin/env python3
"""Render a text-only, uploadable Phase 03D source review artifact."""

from hashlib import sha256
from pathlib import Path
import subprocess

BASE = "7295b641254c91c53b3b80a0c2ec5f6c286e8dd1"
TESTED = "fa28e6f7b02b70020899c8294b74af02c7ab51d0"
FINAL_BEFORE_CLOSURE = "494341fca6a952587c16decdf4ebf05d98bf0b36"
REVIEWED = "87b504853cb0ff98790976879ea8832db92b83e0"
OUT = Path("research/evidence/phase_03d/closure/source_review_bundle.txt")

PATHS = [
    "Makefile",
    "internal/b0app/app.go",
    "internal/b0app/app_test.go",
    "internal/b0app/transaction_test.go",
    "internal/b0app/vector_test.go",
    "internal/b0app/benchmark_test.go",
    "internal/b0state/state.go",
    "internal/b0state/transition.go",
    "internal/b0state/state_test.go",
    "internal/b0state/benchmark_test.go",
    "internal/b0store/store.go",
    "internal/b0store/store_test.go",
    "internal/b0store/recovery_closure_test.go",
    "internal/b0store/benchmark_test.go",
    "internal/node/b0.go",
    "internal/node/node.go",
    "test/integration/b0_genesis_test.go",
    "test/integration/b0_genesis_recovery_test.go",
    "test/integration/b0_transaction_test.go",
    "test/integration/b0_recovery_test.go",
    "test/integration/b0_asymmetric_test.go",
    "test/integration/b0_m4_test.go",
    "test/integration/b0_m6_test.go",
    "test/integration/b0_mismatch_test.go",
    "research/evidence/phase_03d/remaining/run_verification.sh",
    "research/evidence/phase_03d/remaining/summarize_sync.py",
]


def git(*args: str) -> bytes:
    return subprocess.check_output(["git", *args])


def main() -> None:
    if git("cat-file", "-t", REVIEWED).strip() != b"tree":
        raise SystemExit("reviewed source tree is unavailable")
    changed = git("diff", "--name-status", BASE, REVIEWED)
    source_patch = git(
        "diff", "--binary", BASE, REVIEWED, "--",
        "Makefile", "internal/b0app", "internal/b0state", "internal/b0store", "internal/node",
        "test/integration", "research/evidence/phase_03d/remaining/run_verification.sh",
        "research/evidence/phase_03d/remaining/summarize_sync.py",
    )
    contents = [(p, git("show", f"{REVIEWED}:{p}")) for p in PATHS]
    header = (
        "PLUTO PHASE 03D — INDEPENDENT SOURCE REVIEW BUNDLE\n"
        "All following source blocks are verbatim Git blobs, not summaries.\n"
        f"accepted D-025 tree: {BASE}\n"
        f"prior tested tree: {TESTED}\n"
        f"prior final staged tree: {FINAL_BEFORE_CLOSURE}\n"
        f"reviewed source snapshot tree: {REVIEWED}\n"
        f"HEAD commit: {git('rev-parse', 'HEAD').decode().strip()}\n"
        f"branch: {git('branch', '--show-current').decode().strip()}\n"
        f"source patch SHA-256: {sha256(source_patch).hexdigest()}\n"
        f"source patch bytes: {len(source_patch)}\n"
        "The reviewed tree precedes this generated bundle to avoid a self-referential tree ID.\n"
        "The final staged tree is recorded in the separate closure handoff.\n\n"
        "=== CHANGED-FILE MANIFEST: ACCEPTED D-025 TO REVIEWED TREE ===\n"
    ).encode()
    parts = [header, changed, b"\n=== REVIEWED SOURCE FILE SHA-256 ===\n"]
    parts.extend(f"{sha256(data).hexdigest()}  {path}\n".encode() for path, data in contents)
    parts.extend([b"\n=== ACTUAL SOURCE PATCH (GIT DIFF --BINARY) ===\n", source_patch])
    for path, data in contents:
        parts.append(f"\n=== BEGIN VERBATIM {path} ({len(data)} bytes; sha256 {sha256(data).hexdigest()}) ===\n".encode())
        parts.append(data)
        if not data.endswith(b"\n"):
            parts.append(b"\n")
        parts.append(f"=== END VERBATIM {path} ===\n".encode())
    OUT.write_bytes(b"".join(parts))
    print(f"{OUT} bytes={OUT.stat().st_size} sha256={sha256(OUT.read_bytes()).hexdigest()}")


if __name__ == "__main__":
    main()
