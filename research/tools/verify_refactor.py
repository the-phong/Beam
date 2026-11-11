#!/usr/bin/env python3
"""Check that moved Go files differ only in imports, fixture paths and format."""

from pathlib import Path
import subprocess
import sys

BASE = "ed6bc8f474c13ed709d5ab5f57b3856edcbf602a"
MOVES = {
    "internal/profiles/load/": "internal/experiment/load/",
    "internal/profiles/corpus/": "internal/experiment/corpus/profiles/",
    "internal/profiles/": "internal/experiment/profiles/",
    "internal/benchmarks/corpus/": "internal/experiment/corpus/b0b1/",
    "internal/benchmarks/bench/": "internal/experiment/benchmark/",
    "internal/telemetry/": "internal/experiment/telemetry/",
    "internal/b0app/": "internal/classical/app/",
    "internal/b0genesis/": "internal/classical/genesis/",
    "internal/b0state/": "internal/classical/state/",
    "internal/b0store/": "internal/classical/store/",
    "internal/tx/": "internal/classical/tx/",
    "internal/b1app/": "internal/hybrid/app/",
    "internal/b1genesis/": "internal/hybrid/genesis/",
    "internal/b1state/": "internal/hybrid/state/",
    "internal/b1store/": "internal/hybrid/store/",
    "internal/b1test/": "internal/hybrid/testutil/",
    "internal/b1tx/": "internal/hybrid/tx/",
    "internal/app/": "internal/legacy/app/",
    "internal/store/": "internal/legacy/store/",
    "cmd/b0b1-corpus/": "cmd/research/corpus/b0b1/",
    "cmd/profile-corpus/": "cmd/research/corpus/profiles/",
    "cmd/profile-workload-corpus/": "cmd/research/corpus/workload/",
    "cmd/profile-vectors/": "cmd/research/vectors/profiles/",
    "cmd/profile-load/": "cmd/research/workload/",
    "cmd/profile-ingress/": "cmd/research/ingress/",
    "cmd/profile-observer/": "cmd/research/observer/",
}


def git(*args: str) -> str:
    return subprocess.check_output(["git", *args], text=True)


def main() -> int:
    paths = git("ls-tree", "-r", "--name-only", BASE, "cmd", "internal", "test").splitlines()
    imports = {old.rstrip("/"): new.rstrip("/") for old, new in MOVES.items()
               if old.startswith("internal/")}
    failures = []
    checked = 0
    for old_path in paths:
        if not old_path.endswith(".go"):
            continue
        new_path = old_path
        for old, new in MOVES.items():
            if old_path.startswith(old):
                new_path = new + old_path[len(old):]
                break
        expected = git("show", BASE + ":" + old_path)
        for old, new in sorted(imports.items(), key=lambda pair: -len(pair[0])):
            expected = expected.replace("example.com/pluto/" + old + '"',
                                        "example.com/pluto/" + new + '"')
        for old, new in sorted(imports.items(), key=lambda pair: -len(pair[0])):
            expected = expected.replace("example.com/pluto/" + old + "/",
                                        "example.com/pluto/" + new + "/")
        if new_path.startswith(("internal/classical/", "internal/hybrid/", "internal/legacy/")):
            expected = expected.replace('"../../research/', '"../../../research/')
        if new_path.startswith("internal/experiment/corpus/b0b1/"):
            expected = expected.replace('"../../../research/', '"../../../../research/')
        if new_path.startswith("internal/experiment/profiles/app/"):
            expected = expected.replace('"..", "..", "..", "research"',
                                        '"..", "..", "..", "..", "research"')
        expected = expected.replace('"../b1tx/testdata/', '"../tx/testdata/')
        expected = expected.replace('"../../internal/b1tx/testdata/',
                                    '"../../internal/hybrid/tx/testdata/')
        expected = expected.replace("cmd/profile-load.", "cmd/research/workload.")
        expected = subprocess.check_output(["gofmt"], input=expected, text=True)
        checked += 1
        if not Path(new_path).exists() or Path(new_path).read_text() != expected:
            failures.append((old_path, new_path))
    print(f"go_files_checked={checked} unexpected_diffs={len(failures)}")
    for old, new in failures:
        print(f"unexpected: {old} -> {new}", file=sys.stderr)
    data_paths = git("ls-tree", "-r", "--name-only", BASE, "research/corpus",
                     "research/evidence", "internal/b1tx/testdata",
                     "internal/profiles/testdata").splitlines()
    data_failures = []
    for old_path in data_paths:
        new_path = old_path
        for old, new in MOVES.items():
            if old_path.startswith(old):
                new_path = new + old_path[len(old):]
                break
        old_bytes = subprocess.check_output(["git", "show", BASE + ":" + old_path])
        if not Path(new_path).exists() or Path(new_path).read_bytes() != old_bytes:
            data_failures.append((old_path, new_path))
    print(f"tracked_data_checked={len(data_paths)} byte_mismatches={len(data_failures)}")
    for old, new in data_failures:
        print(f"data mismatch: {old} -> {new}", file=sys.stderr)
    return int(bool(failures or data_failures))


if __name__ == "__main__":
    sys.exit(main())
