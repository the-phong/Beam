#!/usr/bin/env python3
"""Run, analyze, freeze and verify a bounded four-profile S07 smoke.

Private raw files stay below research/artifacts/s07. The public sample contains
only hashes and derived, explicitly non-scientific measurements.
"""

from __future__ import annotations

import argparse
import csv
import hashlib
import json
import os
from pathlib import Path
import re
import statistics
import subprocess
import sys
from datetime import datetime, timezone
from uuid import uuid4

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research" / "artifacts" / "s07"
CONFIG = ROOT / "research" / "config" / "s06_pilot.json"
CORPUS = ROOT / "research" / "corpus" / "s06_scenarios.json"
PROFILES = ("C", "CP", "P", "H")
RAW_NAMES = ("metadata.json", "plan.json", "genesis.bin", "nodes.json", "result.json", "oracle.json",
             "commit_observations.json", "phases.jsonl", "load.log")
FIELDS = (
    "run_id", "profile", "status", "reason", "source_commit", "seed", "config_sha256", "plan_sha256",
    "planned_count", "actual_send_count", "planned_bytes", "actual_send_bytes", "legitimate_committed",
    "invalid_rejected", "goodput_observed_lower_bound_tx_per_steady_second", "rpc_latency_n", "rpc_latency_p50_ms",
    "inclusion_latency_n", "inclusion_latency_exact_ms", "inclusion_latency_upper_bound_p50_ms",
    "wire_bytes_min", "wire_bytes_max",
    "telemetry_samples", "comet_cpu_cores", "app_cpu_cores", "comet_peak_rss_bytes",
    "accepted_mempool_max_count", "accepted_mempool_max_bytes", "checktx_queue_samples",
    "checktx_queue_mean_ms", "relay_send_delta_bytes_observed", "relay_send_streams_n",
    "relay_receive_delta_bytes_observed", "relay_receive_streams_n",
    "p2p_ingress_delta_txs_observed", "p2p_ingress_streams_n",
    "independent_run_n", "ci95_low", "ci95_high", "ci_note",
)


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1 << 20), b""):
            digest.update(chunk)
    return digest.hexdigest()


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="microseconds").replace("+00:00", "Z")


def parse_time(raw: str) -> datetime:
    normalized = raw.replace("Z", "+00:00")
    # Go's RFC3339Nano emits up to nine fraction digits. Python datetime
    # stores microseconds; truncate sub-microsecond precision for summaries.
    normalized = re.sub(r"(\.\d{6})\d+(?=[+-]\d{2}:\d{2}$)", r"\1", normalized)
    return datetime.fromisoformat(normalized)


def private_session(path: Path) -> Path:
    root = ARTIFACTS.resolve()
    target = path.resolve(strict=True)
    if target.parent != root or not target.is_dir():
        raise ValueError("session must be one directory below ignored research/artifacts/s07")
    return target


def create_exclusive(path: Path, data: bytes) -> None:
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, "wb") as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())


def write_json(path: Path, value: object) -> None:
    create_exclusive(path, (json.dumps(value, indent=2, sort_keys=True, allow_nan=False) + "\n").encode())


def read_json(path: Path) -> dict:
    value = json.loads(path.read_text())
    if not isinstance(value, dict):
        raise ValueError(f"{path.name} must be an object")
    return value


def require(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError(message)


def run_command(command: list[str], log_path: Path, env: dict[str, str] | None = None) -> int:
    fd = os.open(log_path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, "wb") as log:
        completed = subprocess.run(command, cwd=ROOT, env=env, stdout=log, stderr=subprocess.STDOUT, check=False)
        log.flush()
        os.fsync(log.fileno())
    return completed.returncode


def sample_stream(path: Path) -> list[dict]:
    samples = []
    with path.open() as stream:
        for number, line in enumerate(stream, 1):
            require(line.endswith("\n"), f"truncated observer sample {path.name}:{number}")
            sample = json.loads(line)
            require(isinstance(sample, dict) and sample.get("schema") == "pluto.telemetry.v1"
                    and sample.get("kind") == "node_sample", f"invalid observer sample {path.name}:{number}")
            require(sample.get("dropped_samples") == 0 and not sample.get("unavailable"),
                    f"incomplete observer sample {path.name}:{number}")
            samples.append(sample)
    require(len(samples) >= 2, f"insufficient observer samples: {path.name}")
    times = [parse_time(sample["observed_utc"]) for sample in samples]
    require(all(a < b for a, b in zip(times, times[1:])), f"nonmonotonic observer times: {path.name}")
    return samples


def counter_delta(samples: list[dict], key: str) -> float | None:
    values = [sample.get(key) for sample in samples if sample.get(key) is not None]
    if len(values) < 2:
        return None
    delta = float(values[-1]) - float(values[0])
    require(delta >= 0, f"counter reset: {key}")
    return delta


def process_cpu(samples: list[dict], key: str) -> tuple[float | None, int | None]:
    values = [(parse_time(sample["observed_utc"]), sample[key]) for sample in samples if sample.get(key)]
    if len(values) < 2:
        return None, None
    elapsed = (values[-1][0] - values[0][0]).total_seconds()
    first, last = values[0][1], values[-1][1]
    require(elapsed > 0 and first["pid"] == last["pid"] and first["start_ticks"] == last["start_ticks"]
            and last["cpu_ns"] >= first["cpu_ns"], f"process identity/counter failure: {key}")
    return (last["cpu_ns"] - first["cpu_ns"]) / (elapsed * 1e9), max(v[1]["rss_bytes"] for v in values)


def analyze_run(run: Path, source_commit: str, config_hash: str, corpus_hash: str) -> dict:
    for name in RAW_NAMES:
        require((run / name).is_file(), f"missing raw {name}")
    metadata, plan, result, oracle, commits = (read_json(run / name) for name in
                                      ("metadata.json", "plan.json", "result.json", "oracle.json", "commit_observations.json"))
    require(metadata.get("schema") == "pluto.s07.run.v1" and oracle.get("schema") == "pluto.s07.oracle.v1",
            "invalid S07 schema")
    require(metadata.get("source_commit") == source_commit and metadata.get("config_sha256") == config_hash
            and metadata.get("corpus_sha256") == corpus_hash, "source/config/corpus identity mismatch")
    require(isinstance(metadata.get("source_tree"), str) and len(metadata["source_tree"]) == 40
            and isinstance(metadata.get("host"), str) and metadata["host"], "source tree or host missing")
    require(metadata.get("run_id") == plan.get("run_id") == result.get("run_id") == oracle.get("run_id"),
            "run ID mismatch")
    require(metadata.get("profile") == run.name.removeprefix("profile-") and metadata["profile"] in PROFILES,
            "profile mismatch")
    require(metadata.get("seed") == plan.get("seed") and metadata.get("genesis_sha256") == sha256(run / "genesis.bin")
            == plan.get("genesis_sha256") == result.get("genesis_sha256"), "seed/genesis mismatch")
    require(metadata.get("target_legitimate_tx_per_second") == plan.get("target_legitimate_tx_per_second")
            and metadata.get("target_invalid_tx_per_second") == plan.get("target_invalid_tx_per_second")
            and metadata.get("regime") == plan.get("regime"), "offered-rate metadata mismatch")
    require(result.get("config_sha256") == config_hash and result.get("plan_sha256") == sha256(run / "plan.json"),
            "result provenance mismatch")
    require(result.get("status") == oracle.get("status") == "OK", "run or oracle did not pass")
    require(commits.get("schema") == "pluto.s07.commits.v1" and commits.get("run_id") == metadata["run_id"]
            and commits.get("poll_interval_ms") == 100 and isinstance(commits.get("events"), list),
            "commit observation schema mismatch")
    planned, actual = plan.get("events"), result.get("events")
    require(isinstance(planned, list) and isinstance(actual, list) and len(planned) == len(actual) > 0,
            "event count mismatch")
    require(result.get("planned_count") == result.get("actual_send_count") == len(planned), "offered count mismatch")
    valid = invalid = 0
    rpc_ms = []
    inclusion_upper_ms = []
    commit_by_id = {item["id"]: item for item in commits["events"]}
    require(len(commit_by_id) == len(commits["events"]), "duplicate commit observations")
    started = parse_time(result["started_utc"])
    steady_start = started.timestamp() + plan["warmup_ns"] / 1e9
    steady_end = steady_start + plan["steady_ns"] / 1e9
    observed_in_steady = 0
    wire_sizes = []
    for expected, observed in zip(planned, actual):
        require(expected["id"] == observed["id"] and expected["expected_code"] == observed["expected_code"]
                == observed["actual_code"] and observed.get("actual_offset_ns") is not None
                and not observed.get("submission_error"), "event oracle/send mismatch")
        raw = bytes.fromhex(expected["bytes_hex"])
        require(hashlib.sha256(raw).hexdigest() == expected["sha256"] and len(raw) == observed["wire_bytes"],
                "wire hash/size mismatch")
        wire_sizes.append(len(raw))
        sent = parse_time(observed["actual_send_utc"])
        completed = parse_time(observed["completed_utc"])
        require(completed >= sent, "negative RPC latency")
        rpc_ms.append((completed - sent).total_seconds() * 1000)
        if expected["traffic"] == "legitimate":
            valid += 1
            commit = commit_by_id.get(expected["id"])
            require(commit is not None and commit["sha256"] == expected["sha256"]
                    and 1 <= commit["height"] <= oracle["common_height"], "commit observation mismatch")
            seen = parse_time(commit["all_four_seen_utc"])
            require(seen >= sent, "invalid commit observation time")
            inclusion_upper_ms.append((seen - sent).total_seconds() * 1000)
            if steady_start <= seen.timestamp() < steady_end:
                observed_in_steady += 1
        else:
            invalid += 1
    require(oracle.get("valid_committed") == valid and oracle.get("invalid_rejected") == invalid
            and oracle.get("common_height", 0) >= 8 and oracle.get("block_id") and oracle.get("app_hash"),
            "block/state oracle mismatch")
    require(len(commit_by_id) == valid, "commit observation count mismatch")
    streams = [sample_stream(run / f"observer-{i}.jsonl") for i in range(4)]
    cpu = [process_cpu(stream, "comet_process") for stream in streams]
    app_cpu, _ = process_cpu(streams[0], "application_process")
    queue_counts = [counter_delta(stream, "checktx_socket_queue_count") for stream in streams]
    queue_seconds = [counter_delta(stream, "checktx_socket_queue_seconds") for stream in streams]
    queue_n = sum(v for v in queue_counts if v is not None)
    queue_s = sum(v for v in queue_seconds if v is not None)
    relay_send = [counter_delta(stream, "relay_send_envelope_bytes") for stream in streams]
    relay_recv = [counter_delta(stream, "relay_receive_envelope_bytes") for stream in streams]
    ingress = [counter_delta(stream, "p2p_ingress_tx_total") for stream in streams]
    mempool_counts = [sample.get("accepted_mempool_count") for stream in streams for sample in stream]
    mempool_bytes = [sample.get("accepted_mempool_bytes") for stream in streams for sample in stream]
    require(all(v is not None for v in queue_counts + queue_seconds)
            and all(v[0] is not None and v[1] is not None for v in cpu) and app_cpu is not None
            and all(v is not None for v in mempool_counts + mempool_bytes)
            and any(v is not None for v in relay_send) and any(v is not None for v in relay_recv)
            and queue_n > 0 and any(v is not None and v > 0 for v in ingress), "required telemetry unavailable")
    steady_seconds = plan["steady_ns"] / 1e9
    require(steady_seconds > 0, "invalid steady interval")
    return {
        "run_id": metadata["run_id"], "profile": metadata["profile"], "status": "SMOKE_OK", "reason": "",
        "source_commit": source_commit, "seed": metadata["seed"], "config_sha256": config_hash,
        "plan_sha256": sha256(run / "plan.json"), "planned_count": len(planned),
        "actual_send_count": result["actual_send_count"], "planned_bytes": result["planned_bytes"],
        "actual_send_bytes": result["actual_send_bytes"], "legitimate_committed": valid,
        "invalid_rejected": invalid,
        "goodput_observed_lower_bound_tx_per_steady_second": observed_in_steady / steady_seconds,
        "rpc_latency_n": len(rpc_ms), "rpc_latency_p50_ms": round(statistics.median(rpc_ms), 3),
        "inclusion_latency_n": len(inclusion_upper_ms), "inclusion_latency_exact_ms": None,
        "inclusion_latency_upper_bound_p50_ms": round(statistics.median(inclusion_upper_ms), 3),
        "wire_bytes_min": min(wire_sizes), "wire_bytes_max": max(wire_sizes),
        "telemetry_samples": sum(map(len, streams)),
        "comet_cpu_cores": round(sum(v[0] for v in cpu if v[0] is not None), 4),
        "app_cpu_cores": round(app_cpu, 4) if app_cpu is not None else None,
        "comet_peak_rss_bytes": max(v[1] for v in cpu if v[1] is not None),
        "accepted_mempool_max_count": max(v for v in mempool_counts if v is not None),
        "accepted_mempool_max_bytes": max(v for v in mempool_bytes if v is not None),
        "checktx_queue_samples": int(queue_n), "checktx_queue_mean_ms": round(queue_s * 1000 / queue_n, 4),
        "relay_send_delta_bytes_observed": sum(v for v in relay_send if v is not None),
        "relay_send_streams_n": sum(v is not None for v in relay_send),
        "relay_receive_delta_bytes_observed": sum(v for v in relay_recv if v is not None),
        "relay_receive_streams_n": sum(v is not None for v in relay_recv),
        "p2p_ingress_delta_txs_observed": sum(v for v in ingress if v is not None),
        "p2p_ingress_streams_n": sum(v is not None for v in ingress),
        "independent_run_n": 1, "ci95_low": None, "ci95_high": None,
        "ci_note": "not estimable from one independent run per profile; transaction samples are clustered",
    }


def frozen_files(session: Path) -> dict[str, str]:
    files = {}
    for path in session.rglob("*"):
        if "network" in path.relative_to(session).parts:
            continue
        require(not path.is_symlink(), f"symlink in frozen session: {path}")
        if not path.is_file():
            continue
        if path.name in ("manifest.json", "MANIFEST_SHA256"):
            continue
        files[path.relative_to(session).as_posix()] = sha256(path)
    return dict(sorted(files.items()))


def freeze(session: Path, source_commit: str, rows: list[dict], exclusions: list[dict]) -> None:
    write_json(session / "exclusions.json", {"schema": "pluto.s07.exclusions.v1", "entries": exclusions})
    write_json(session / "derived.json", {"schema": "pluto.s07.derived.v1", "scope": "SMOKE_ONLY", "runs": rows})
    with (session / "derived.csv").open("x", newline="") as stream:
        writer = csv.DictWriter(stream, fieldnames=FIELDS)
        writer.writeheader()
        writer.writerows(rows)
    for profile in PROFILES:
        phase = session / f"profile-{profile}" / "phases.jsonl"
        if phase.exists():
            with phase.open("a") as stream:
                stream.write(json.dumps({"phase": "analyze", "detail": "strict schema and per-run metric extraction", "observed_utc": utc_now()}) + "\n")
                stream.write(json.dumps({"phase": "freeze", "detail": "SHA-256 manifest and read-only raw/derived files", "observed_utc": utc_now()}) + "\n")
    manifest = {"schema": "pluto.s07.manifest.v1", "source_commit": source_commit,
                "created_utc": utc_now(), "scope": "SMOKE_ONLY", "files": frozen_files(session)}
    write_json(session / "manifest.json", manifest)
    create_exclusive(session / "MANIFEST_SHA256", (sha256(session / "manifest.json") + "\n").encode())
    for relative in manifest["files"]:
        (session / relative).chmod(0o400)
    (session / "manifest.json").chmod(0o400)
    (session / "MANIFEST_SHA256").chmod(0o400)


def verify(session: Path) -> dict:
    session = private_session(session)
    manifest = read_json(session / "manifest.json")
    require(manifest.get("schema") == "pluto.s07.manifest.v1" and isinstance(manifest.get("files"), dict),
            "invalid manifest schema")
    require((session / "MANIFEST_SHA256").read_text().strip() == sha256(session / "manifest.json"),
            "manifest digest mismatch")
    require(manifest["files"] == frozen_files(session), "missing, extra or modified frozen file")
    derived = read_json(session / "derived.json")
    exclusions = read_json(session / "exclusions.json")
    require(derived.get("schema") == "pluto.s07.derived.v1" and derived.get("scope") == "SMOKE_ONLY"
            and isinstance(derived.get("runs"), list), "invalid derived schema")
    require(exclusions.get("schema") == "pluto.s07.exclusions.v1" and isinstance(exclusions.get("entries"), list),
            "invalid exclusion ledger")
    source = read_json(session / "session.json")
    require(source.get("schema") == "pluto.s07.session.v1" and source.get("source_commit") == manifest["source_commit"]
            and source.get("config_sha256") == sha256(session / "config.json")
            and source.get("corpus_sha256") == sha256(session / "corpus.json"),
            "session source/config/corpus schema or digest mismatch")
    rows = derived["runs"]
    require(len({row["run_id"] for row in rows}) == len(rows), "duplicate run IDs")
    for row in rows:
        run = session / f"profile-{row['profile']}"
        phase_names = [json.loads(line)["phase"] for line in (run / "phases.jsonl").read_text().splitlines()]
        standard_phases = ["setup", "warm-up", "execute", "drain", "oracle", "collect", "analyze", "freeze"]
        common02_phases = ["setup", "warm-up", "execute", "drain", "oracle", "collect", "cooldown", "analyze", "freeze"]
        require(phase_names in (standard_phases, common02_phases),
                f"incomplete stage sequence for {row['profile']}")
        require(row == analyze_run(run, source["source_commit"], source["config_sha256"], source["corpus_sha256"]),
                f"derived result differs from raw/oracle for {row['profile']}")
    require(len(rows) + sum(entry.get("profile") in PROFILES for entry in exclusions["entries"]) == len(PROFILES),
            "exclusion ledger does not account for all profiles")
    return manifest


def smoke(publish_sample: Path | None) -> int:
    ARTIFACTS.mkdir(parents=True, exist_ok=True, mode=0o700)
    session_id = "s07-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ") + "-" + uuid4().hex[:8]
    session = ARTIFACTS / session_id
    session.mkdir(mode=0o700)
    source = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    tree = subprocess.check_output(["git", "rev-parse", "HEAD^{tree}"], cwd=ROOT, text=True).strip()
    config_hash, corpus_hash = sha256(CONFIG), sha256(CORPUS)
    create_exclusive(session / "config.json", CONFIG.read_bytes())
    create_exclusive(session / "corpus.json", CORPUS.read_bytes())
    write_json(session / "session.json", {"schema": "pluto.s07.session.v1", "session_id": session_id,
               "source_commit": source, "source_tree": tree, "config_sha256": config_hash,
               "corpus_sha256": corpus_hash, "created_utc": utc_now(), "profiles": PROFILES,
               "placement": "four_validators_one_host_smoke"})
    print(f"S07_SESSION={session}", flush=True)
    build = run_command(["make", "build", "build-comet-s05", "corpus-s04-verify", "corpus-s06-verify"], session / "build.log")
    test = -1
    if build == 0:
        env = os.environ.copy()
        env.update(S07_RUN_ROOT=str(session), S07_SESSION_ID=session_id, S07_SOURCE_COMMIT=source,
                   S07_SOURCE_TREE=tree, S07_CORPUS_SHA256=corpus_hash)
        test = run_command(["go", "test", "-tags", "integration fourval", "./test/integration",
                            "-run", "^TestS07ReproSmoke$", "-timeout", "600s", "-count=1", "-v"],
                           session / "integration.log", env)
    rows, exclusions = [], []
    for profile in PROFILES:
        run = session / f"profile-{profile}"
        try:
            require(run.is_dir(), "run directory missing")
            rows.append(analyze_run(run, source, config_hash, corpus_hash))
        except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError) as error:
            exclusions.append({"profile": profile, "run_id": f"{session_id}-{profile}",
                               "status": "INVALID_FOR_ANALYSIS", "reason": str(error)[:240]})
    if build != 0:
        exclusions.append({"profile": "all", "status": "INVALID_FOR_ANALYSIS", "reason": "build failed"})
    if test != 0:
        exclusions.append({"profile": "all", "status": "INVALID_FOR_ANALYSIS", "reason": f"integration command exit {test}"})
    freeze(session, source, rows, exclusions)
    verify(session)
    if publish_sample is not None and not exclusions and len(rows) == 4:
        require(publish_sample.resolve().parent == (ROOT / "research" / "evidence" / "s07").resolve(),
                "public sample must be under research/evidence/s07")
        public = {"schema": "pluto.s07.public_smoke.v1", "scope": "SMOKE_ONLY_NOT_SCIENTIFIC",
                  "source_commit": source, "config_sha256": config_hash, "corpus_sha256": corpus_hash,
                  "manifest_sha256": sha256(session / "manifest.json"),
                  "runs": [{k: row[k] for k in FIELDS if k not in ("run_id",)} for row in rows]}
        write_json(publish_sample, public)
    print(f"S07_STATUS={'OK' if build == 0 and test == 0 and len(rows) == 4 and not exclusions else 'INVALID_FOR_ANALYSIS'} "+
          f"runs={len(rows)} excluded={len(exclusions)} manifest_sha256={sha256(session / 'manifest.json')}", flush=True)
    return 0 if build == 0 and test == 0 and len(rows) == 4 and not exclusions else 1


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    smoke_parser = sub.add_parser("smoke")
    smoke_parser.add_argument("--publish-sample", type=Path)
    verify_parser = sub.add_parser("verify")
    verify_parser.add_argument("session", type=Path)
    args = parser.parse_args()
    try:
        if args.command == "smoke":
            return smoke(args.publish_sample)
        manifest = verify(args.session)
        print(f"verified {len(manifest['files'])} frozen files, source={manifest['source_commit']}")
        return 0
    except (ValueError, OSError, KeyError, json.JSONDecodeError) as error:
        print(f"S07 verification error: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
