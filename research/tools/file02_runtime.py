#!/usr/bin/env python3
"""Capture actual File 02 run hardware, software and four-node configuration."""

import hashlib
import argparse
import json
import os
from pathlib import Path
import platform
import re
import subprocess
import sys
import time
from datetime import datetime, timezone

ROOT = Path(__file__).resolve().parents[2]
MODULES = {
    "github.com/cloudflare/circl": "circl",
    "github.com/cometbft/cometbft": "cometbft",
    "github.com/decred/dcrd/dcrec/secp256k1/v4": "secp256k1",
}


def require(ok, message):
    if not ok:
        raise ValueError(message)


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def git(*args):
    return subprocess.check_output(["git", *args], cwd=ROOT, text=True).strip()


def module_versions():
    source = (ROOT / "go.mod").read_text()
    versions = {}
    for path, label in MODULES.items():
        matches = re.findall(r"^\s*" + re.escape(path) + r"\s+(v\S+)\s*$", source, re.MULTILINE)
        require(len(matches) == 1, f"missing or duplicate module: {path}")
        versions[label] = matches[0]
    return versions


def cpu_inventory():
    source = Path("/proc/cpuinfo").read_text()
    models = re.findall(r"^model name\s*:\s*(.+)$", source, re.MULTILINE)
    flags = re.findall(r"^flags\s*:\s*(.+)$", source, re.MULTILINE)
    require(models and flags, "Linux CPU model/features unavailable")
    feature_sets = [set(row.split()) for row in flags]
    common_features = sorted(set.intersection(*feature_sets))
    require(common_features, "no common CPU features")
    return {"cpu_model": models[0].strip(), "cpu_features": common_features,
            "architecture": platform.machine(), "logical_cpus": os.cpu_count(),
            "kernel_release": Path("/proc/sys/kernel/osrelease").read_text().strip()}


def simple_toml_values(path):
    """Read the scalar keys written by Comet's config writer; reject ambiguity."""
    section = ""
    values = {}
    for line in Path(path).read_text().splitlines():
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        if stripped.startswith("[") and stripped.endswith("]"):
            section = stripped[1:-1]
            continue
        match = re.match(r"^([A-Za-z0-9_]+)\s*=\s*(.+?)(?:\s+#.*)?$", stripped)
        if not match:
            continue
        key = (section, match[1])
        require(key not in values, f"duplicate Comet config key: {key}")
        raw = match[2].strip()
        try:
            values[key] = json.loads(raw)
        except json.JSONDecodeError:
            # Comet writes TOML arrays and single-quoted values that JSON
            # cannot decode. Only the required scalar keys below are used.
            continue
    return values


def require_node_config(path, network):
    values = simple_toml_values(path)
    expected = {
        ("mempool", "type"): network["mempool_type"],
        ("mempool", "size"): network["mempool_size"],
        ("mempool", "max_txs_bytes"): network["mempool_max_bytes"],
        ("mempool", "cache_size"): network["mempool_cache_size"],
        ("mempool", "max_tx_bytes"): network["mempool_max_tx_bytes"],
        ("mempool", "recheck"): network["mempool_recheck"],
        ("mempool", "broadcast"): network["mempool_broadcast"],
        ("consensus", "create_empty_blocks"): network["create_empty_blocks"],
        ("consensus", "create_empty_blocks_interval"): f"{network['empty_block_interval_ms']}ms",
        ("consensus", "timeout_commit"): f"{network['timeout_commit_ms']}ms",
        ("instrumentation", "prometheus"): True,
    }
    for key, target in expected.items():
        # Comet can render whole-second durations as 1s rather than 1000ms.
        observed = values.get(key)
        if isinstance(target, str) and target.endswith("ms") and isinstance(observed, str) and observed.endswith("s"):
            try:
                observed = f"{int(float(observed[:-1]) * 1000)}ms"
            except ValueError:
                pass
        require(observed == target, f"actual Comet config mismatch: {key}")
    for key in (("", "proxy_app"), ("rpc", "laddr"), ("p2p", "laddr"),
                ("instrumentation", "prometheus_listen_addr")):
        value = values.get(key)
        require(isinstance(value, str) and (value.startswith("tcp://127.0.0.1:") or
                (key[0] == "instrumentation" and value.startswith("127.0.0.1:"))),
                f"non-loopback Comet listener: {key}")
    peers = values.get(("p2p", "persistent_peers"), "")
    require(len(peers.split(",")) == 3 and all("@127.0.0.1:" in peer for peer in peers.split(",")),
            "four-node peer topology mismatch")
    return {"config_toml_sha256": digest(path),
            "rpc_listener": values[("rpc", "laddr")],
            "p2p_listener": values[("p2p", "laddr")]}


def capture(config_path, node_homes, run_id, profile):
    require(len(node_homes) == 4 and len({str(Path(p).resolve()) for p in node_homes}) == 4,
            "four distinct validator homes required")
    config_path = Path(config_path)
    config = json.loads(config_path.read_text())
    network = config["network"]
    require(network["validator_count"] == 4 and network["placement"] == "single_host_pilot",
            "unsupported topology")
    versions = module_versions()
    require(versions["cometbft"] == network["comet_version"], "Comet module/config mismatch")
    nodes = []
    genesis_hashes = set()
    for home in node_homes:
        home = Path(home)
        node = require_node_config(home / "config" / "config.toml", network)
        genesis_path = home / "config" / "genesis.json"
        genesis_hashes.add(digest(genesis_path))
        genesis = json.loads(genesis_path.read_text())
        require(int(genesis["consensus_params"]["block"]["max_bytes"]) == network["max_block_bytes"],
                "actual genesis block bytes mismatch")
        nodes.append(node)
    require(len(genesis_hashes) == 1, "validator genesis divergence")
    require(len({n["rpc_listener"] for n in nodes}) == 4 and
            len({n["p2p_listener"] for n in nodes}) == 4, "duplicate validator listener")
    binary_names = ("cometbft-s05", "profile-load", "profile-observer")
    binaries = {name: digest(ROOT / "build" / name) for name in binary_names}
    go_version = subprocess.check_output(["go", "version"], text=True).strip()
    clean = not bool(git("status", "--porcelain"))
    return {"schema": "pluto.file02.runtime_inventory.v1", "run_id": run_id,
            "profile": profile, "captured_utc": datetime.now(timezone.utc).isoformat(),
            "captured_monotonic_ns": time.monotonic_ns(), "source_commit": git("rev-parse", "HEAD"),
            "source_tree": git("rev-parse", "HEAD^{tree}"), "source_clean": clean,
            "config_sha256": digest(config_path), "go_mod_sha256": digest(ROOT / "go.mod"),
            "go_sum_sha256": digest(ROOT / "go.sum"),
            "software_versions": {"go": go_version, "cometbft": versions["cometbft"],
                                  "crypto": f"circl={versions['circl']};secp256k1={versions['secp256k1']}",
                                  "os": platform.system() + " " + platform.release()},
            "host_inventory": cpu_inventory(),
            "network_config": {"validator_count": 4, "mempool_type": network["mempool_type"],
                               "mempool_size": network["mempool_size"],
                               "mempool_max_bytes": network["mempool_max_bytes"],
                               "mempool_cache_size": network["mempool_cache_size"],
                               "mempool_max_tx_bytes": network["mempool_max_tx_bytes"],
                               "mempool_recheck": network["mempool_recheck"],
                               "mempool_broadcast": network["mempool_broadcast"],
                               "max_block_bytes": network["max_block_bytes"],
                               "empty_block_interval_ms": network["empty_block_interval_ms"],
                               "timeout_commit_ms": network["timeout_commit_ms"],
                               "network_shape": "four_validator_single_host_loopback",
                               "endpoint_hosts": ["127.0.0.1"]},
            "node_configs": nodes, "genesis_json_sha256": next(iter(genesis_hashes)),
            "binary_sha256": binaries}


def capture_s07_session(session_path):
    import file02_common as common
    import s07_orchestrate as s07

    session = s07.private_session(Path(session_path))
    manifest = s07.verify(session)
    session_meta = json.loads((session / "session.json").read_text())
    config_path = session / "config.json"
    require(session_meta["source_commit"] == git("rev-parse", "HEAD") and
            session_meta["source_tree"] == git("rev-parse", "HEAD^{tree}"),
            "S07 session source differs from current checkout")
    output = ROOT / "research" / "artifacts" / "file02" / ("common01-" + session.name)
    output.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    output.mkdir(mode=0o700)
    records = []
    for profile in ("C", "CP", "P", "H"):
        run_root = session / ("profile-" + profile)
        metadata = json.loads((run_root / "metadata.json").read_text())
        homes = [run_root / "network" / f"validator-{i}" for i in range(4)]
        inventory = capture(config_path, homes, metadata["run_id"], profile)
        require(inventory["source_clean"] and inventory["source_commit"] == session_meta["source_commit"]
                and inventory["source_tree"] == session_meta["source_tree"]
                and inventory["config_sha256"] == metadata["config_sha256"],
                "runtime inventory/source/config mismatch")
        common.write_once(output / f"profile-{profile}.json", json.dumps(inventory, sort_keys=True, indent=2) + "\n")
        records.append({"profile": profile, "run_id": metadata["run_id"],
                        "inventory_sha256": digest(output / f"profile-{profile}.json")})
    summary = {"schema": "pluto.file02.common01_audit.v1", "scope": "QA_ONLY",
               "source_commit": session_meta["source_commit"], "source_tree": session_meta["source_tree"],
               "s07_manifest_sha256": digest(session / "manifest.json"),
               "s07_frozen_file_count": len(manifest["files"]), "records": records}
    common.write_once(output / "summary.json", json.dumps(summary, sort_keys=True, indent=2) + "\n")
    frozen_sha = common.freeze_raw(output)
    return output, frozen_sha


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    p = sub.add_parser("capture-s07")
    p.add_argument("session")
    args = parser.parse_args()
    try:
        output, frozen_sha = capture_s07_session(args.session)
        print(f"COMMON01_STATUS=OK inventory={output} manifest_sha256={frozen_sha}")
        return 0
    except (ValueError, OSError, KeyError, subprocess.CalledProcessError) as exc:
        print(f"COMMON01_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
