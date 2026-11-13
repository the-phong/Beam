#!/usr/bin/env python3
"""Audit File 02 T02 transaction byte breakdown from frozen T01 raw."""

import argparse
import json
from pathlib import Path
import sys

import file02_common as common
import file02_t01 as t01
from file02_common03 import private_session, require

PROFILE_CODES = {"C": 1, "CP": 2, "P": 3, "H": 4}
ML_BYTES = 2420
SECP_BYTES = 64


def inspect_wire(raw, profile, chain):
    p = PROFILE_CODES[profile]
    require(raw[:8] == b"PLUTOTX4" and int.from_bytes(raw[8:10], "big") == 4 and
            raw[10] == p and raw[11] == len(chain) and raw[12:12+len(chain)] == chain,
            "T02 envelope mismatch")
    fields = {"magic_version_profile_chain_length": 12, "chain_id": len(chain),
              "sender": 20, "receiver": 20, "amount": 8, "nonce": 8,
              "secp_signature": SECP_BYTES if profile in ("C", "CP", "H") else 0,
              "mldsa_signature": ML_BYTES if profile in ("P", "H") else 0,
              "canonical_zero_padding": ML_BYTES if profile == "CP" else 0,
              "on_wire_public_key": 0}
    require(len(raw) == sum(fields.values()), "T02 wire length mismatch")
    if profile == "CP":
        require(raw[-ML_BYTES:] == bytes(ML_BYTES), "T02 noncanonical CP padding")
    return fields


def audit(t01_path):
    session = private_session(t01_path)
    manifest = common.verify_raw(session)
    prior = common.read_json(session / "summary.json")
    require(t01.audit(session, prior["source_commit"], prior["source_tree"]) == prior,
            "T01 source audit mismatch")
    rows = []
    for profile in PROFILE_CODES:
        match = next(row for row in prior["runs"] if row["profile"] == profile)
        plan = common.read_json(session / "runs" / match["run_id"] / "plan.json")
        chain = b"pluto-file02-t01"
        fields = [inspect_wire(bytes.fromhex(event["bytes_hex"]), profile, chain)
                  for event in plan["events"]]
        require(len(fields) == 9 and all(item == fields[0] for item in fields),
                "T02 distribution or field mismatch")
        size = sum(fields[0].values())
        rows.append({"profile": profile, "n": 9, "min_bytes": size, "max_bytes": size,
                     "mean_bytes": size, "breakdown_bytes": fields[0],
                     "canonical_parse_and_accepted": True,
                     "on_wire_public_key_status": "ABSENT_BY_GENESIS_ONLY_PROTOCOL"})
    sizes = {row["profile"]: row["mean_bytes"] for row in rows}
    require(sizes["CP"] == sizes["H"] and sizes["C"] < sizes["CP"] and
            sizes["P"] < sizes["H"], "T02 size control failed")
    return {"schema": "pluto.file02.t02_size_audit.v1", "source_t01_manifest_sha256": manifest,
            "source_commit": prior["source_commit"], "profiles": rows,
            "cp_h_exact_byte_match": True,
            "on_wire_key_variant": "UNSUPPORTED_REQUIRES_SEPARATE_PROTOCOL_DECISION"}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("t01_session")
    args = parser.parse_args()
    try:
        print(json.dumps(audit(Path(args.t01_session)), indent=2, sort_keys=True))
        return 0
    except (OSError, ValueError, KeyError, TypeError, StopIteration) as exc:
        print(f"T02_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__": sys.exit(main())
