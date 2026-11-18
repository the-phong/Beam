#!/usr/bin/env python3
"""Verify publication-safe result files and unchanged moved public fixtures."""

import csv
import hashlib
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]
MECHANISM = ROOT / "research/evidence/file02/mechanism"
BASELINE_SHA256 = {
    "experiments/scenarios/corpus/s04_public_corpus.json": "d0f4bfaea4c541ee6f71547b6ea47e44c30aef588704051459e57e04d6478c83",
    "experiments/scenarios/corpus/s06_scenarios.json": "6f43b725156f60deb0f0d2e6c451605bae479a5e890ff58f7e3be46c5a8ca0e2",
    "experiments/scenarios/corpus/b0b1/public_corpus.json": "242e5c848aa3c97f731a3e70f9ca0876a9ec68bffbbbab2ead8c222c005cd535",
    "experiments/scenarios/corpus/b0b1/public_corpus_contention.json": "459148e8d350cecd1509130d88de909b9ce37c2420ef2999d224db9265ec1646",
    "tests/fixtures/b0_golden_vectors.json": "3d08e616583eed8eb3aa408cc55e25a016355de363a44d51021c49a10baeeb70",
    "tests/fixtures/b0_state_vectors.json": "3d8491b174f44ae51c99796d48c4001ba56e86e008c5dbd08f7d1b68007d69f7",
}


def require_sha(path: Path, expected: str) -> None:
    if not path.is_file() or hashlib.sha256(path.read_bytes()).hexdigest() != expected:
        raise ValueError(f"missing or changed public artifact: {path.relative_to(ROOT)}")


def verify_manifest(name: str) -> int:
    manifest = json.loads((MECHANISM / name).read_text())
    if not manifest["qc"].startswith("PASS_") or not manifest["raw_manifest_sha256"]:
        raise ValueError(f"invalid publication provenance: {name}")
    for relative, digest in manifest["files"].items():
        path = (MECHANISM / relative).resolve()
        if not path.is_relative_to(MECHANISM.resolve()):
            raise ValueError(f"unsafe artifact path: {relative}")
        require_sha(path, digest)
    return len(manifest["files"])


def verify() -> None:
    for relative, digest in BASELINE_SHA256.items():
        require_sha(ROOT / relative, digest)
    count = sum(verify_manifest(name) for name in (
        "X07_QUEUE_V2_PUBLIC_MANIFEST.json", "X16_CONFIRMATORY_PUBLIC_MANIFEST.json"))
    figures = json.loads((MECHANISM / "figures/manifest.json").read_text())
    if figures["schema"] != "pluto.file02.x17_figures.v1":
        raise ValueError("X17 figure manifest schema")
    for name, digest in figures["figures"].items():
        if Path(name).name != name:
            raise ValueError("unsafe figure name")
        require_sha(MECHANISM / "figures" / name, digest)
    table = ROOT / "results/tables/primary_effects.csv"
    with table.open(newline="") as handle:
        rows = list(csv.DictReader(handle))
    if len(rows) != 6 or len({(r["result_id"], r["contrast"]) for r in rows}) != 6:
        raise ValueError("primary effect table incomplete or duplicated")
    x16 = json.loads((MECHANISM / "X16_CONFIRMATORY_AGGREGATE_20261116.json").read_text())
    row = next(r for r in rows if (r["result_id"], r["contrast"]) == ("X16", "B-I"))
    effect = x16["paired_B_minus_I"]["app_cpu_s"]
    if (float(row["estimate"]), float(row["ci95_low"]), float(row["ci95_high"])) != (
            effect["mean"], *effect["ci95"]):
        raise ValueError("X16 table differs from public aggregate")
    print(f"PUBLIC_ARTIFACTS=PASS fixtures={len(BASELINE_SHA256)} manifest_files={count} x17_figures={len(figures['figures'])} table_rows={len(rows)}")


if __name__ == "__main__":
    try:
        verify()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
        print(f"PUBLIC_ARTIFACTS=FAIL {exc}", file=sys.stderr)
        sys.exit(2)
