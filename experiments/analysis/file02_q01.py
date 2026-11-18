#!/usr/bin/env python3
"""Q01 raw-bound pair audit; publish only keyless matched-service summaries."""

import argparse
import csv
from io import StringIO
import json
from pathlib import Path
import sys

import file02_a13 as a13
import file02_attack as attack
import file02_common as common

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "research/artifacts/file02"
OUTPUT = ROOT / "research/evidence/file02/quality"
INPUTS = {
    "A01": ("a01-20261113T170536Z-9bdf0a45", "e8ea05c17bdb405e0339e895cd0f562a1c55ad7eae29745965fa54124f10c98a"),
    "A02": ("a02-20261113T171854Z-5021cdef", "9acf1b51ac4e0e9ba7132730df35de0698e06cc4c84f904dc6890727ea71646d"),
    "A03": ("a03-20261113T172941Z-5a42d176", "46855a3034478e02dc49e20d1815b4e47da18e21dd24fc95e92b0ad9f268c9a1"),
    "A04": ("a04-20261113T174032Z-edd720e5", "41d2bf5bef4aef6cc3c01720710a80b84704952da2c6458e3c0f68194c29c37d"),
    "A05": ("a05-20261113T175129Z-edc80e07", "b90bc2b95ac43388d9d7db8e85cfa3a2dc052258c421623a56cbe2d1ccc1026b"),
    "A08": ("a08-20261113T182741Z-0dff8ab6", "18eed7952c91b5cff08a418055a0a742ffaeb3f70e06f94f115f4e99e5ad8b4d"),
    "A13": ("a13-20261113T190017Z-d2e1e0a6", "6892315cdc842ad0539b78d4dfe5a1c73b1863c49155319bdebf62cddb2dbb61"),
}
FIELDS = ("case_id", "pair", "benign_run_id", "attack_run_id", "seed", "profile",
          "topology", "fixed_offered_tx_s", "background_on_tx_s", "total_on_tx_s",
          "total_on_bytes_s", "eligible_per_arm", "benign_timely", "attack_timely",
          "benign_fraction", "attack_fraction", "benign_minus_attack_fraction",
          "source_raw_manifest_sha256", "qc_status")


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def paired_row(case, pair, by, manifest):
    require(set(by) == {"benign", "attack"}, "missing or duplicate arm")
    b, a = by["benign"], by["attack"]
    require(b["qc_status"] == a["qc_status"] == "PASS" and
            b["genesis_sha256"] == a["genesis_sha256"] and
            b["fixed_hashes"] == a["fixed_hashes"], "pair QC/genesis/legitimate stream")
    if case == "A13":
        require(b["block"] == a["block"] == pair and
                b["windows"] == a["windows"] == [6]*3 + [20]*6 + [6]*3 and
                len(b["phase_fixed_latency_upper_ms"]["on"]) ==
                len(a["phase_fixed_latency_upper_ms"]["on"]) == 36,
                "A13 phase/rate/eligible")
        bt = round(b["phase_fixed_timely_fraction"]["on"] * 36)
        at = round(a["phase_fixed_timely_fraction"]["on"] * 36)
        eligible = 36
        seed = 2026111300 + pair
    else:
        require(b["pair"] == a["pair"] == pair and
                b["seed"] == a["seed"] and
                all(window == {"fixed": 6, "background": 14, "bytes": 51360}
                    for row in (b, a) for window in row["windows"]) and
                b["fixed_eligible_count"] == a["fixed_eligible_count"] == 72,
                "attack rate/seed/eligible")
        bt, at, eligible = b["fixed_timely_count"], a["fixed_timely_count"], 72
        seed = b["seed"]
    require(0 <= bt <= eligible and 0 <= at <= eligible, "invalid timely count")
    return {"case_id": case, "pair": pair, "benign_run_id": b["run_id"],
            "attack_run_id": a["run_id"], "seed": seed, "profile": "H",
            "topology": "FOUR_VALIDATOR_SINGLE_HOST_DIAGNOSTIC",
            "fixed_offered_tx_s": 6, "background_on_tx_s": 14,
            "total_on_tx_s": 20, "total_on_bytes_s": 51360,
            "eligible_per_arm": eligible, "benign_timely": bt,
            "attack_timely": at, "benign_fraction": bt/eligible,
            "attack_fraction": at/eligible,
            "benign_minus_attack_fraction": (bt-at)/eligible,
            "source_raw_manifest_sha256": manifest, "qc_status": "PASS"}


def pair_source_identity(session, by):
    fixed_streams, controls = [], []
    for condition in ("benign", "attack"):
        run = session / "raw/runs" / by[condition]["run_id"]
        workload = common.read_json(run / "plan.json")
        metadata = common.read_json(run / "metadata.json")
        bound = common.read_json(run / "research_plan/plan.json")
        fixed_streams.append([(event["sha256"], event["offset_ns"],
                               event["account_slot"], event["nonce"])
                              for event in workload["events"]
                              if event["scenario"] in ("fixed_legitimate", "warmup")])
        controls.append((workload["seed"], workload["genesis_sha256"],
                         workload["config_sha256"], metadata["source_tree"],
                         metadata["network"], bound["profile"],
                         bound["topology"], bound["traffic_source"],
                         bound["host_inventory"], bound["software_versions"]))
    require(fixed_streams[0] == fixed_streams[1] and len(fixed_streams[0]) == 73 and
            controls[0] == controls[1] and controls[0][4] ==
            "fresh_four_validator_single_host_loopback" and
            controls[0][5:8] == ("H", "FOUR_VALIDATOR_SINGLE_HOST_DIAGNOSTIC", "RPC"),
            "pair fixed send offsets/seed/config/topology/software drift")


def derive():
    pairs, summary = [], []
    for case, (name, expected_manifest) in INPUTS.items():
        session = ARTIFACTS / name
        require(session.is_dir() and session.parent == ARTIFACTS, "private source session")
        plan = common.read_json(session / "plan.json")
        require(plan["case_id"] == case and plan["profile"] == "H" and
                plan["status"] == "LOCKED", "source plan identity")
        if case == "A13":
            a13.verify(session)
            source_rows, qc, aggregate = a13.derive(session, plan)
            count, endpoint_ci = 4, aggregate["phase_ci95_fraction"]["on"]
            endpoint_effect = aggregate["phase_benign_minus_attack_fraction"]["on"]
        else:
            attack.verify(session)
            source_rows, qc, aggregate = attack.derive(session, plan)
            count, endpoint_ci = 12, aggregate["ci95_fraction"]
            endpoint_effect = aggregate["primary_benign_minus_attack_fraction"]
        require(qc["status"] == "PASS" and qc["raw_manifest_sha256"] == expected_manifest and
                aggregate["status"] == "ANALYZED" and len(source_rows) == 2*count,
                "source QC/manifest/coverage")
        case_pairs = []
        for pair in range(count):
            group = [row for row in source_rows if row.get("pair", row.get("block")) == pair]
            require(len(group) == 2 and {row["condition"] for row in group} ==
                    {"benign", "attack"}, "pair membership")
            by = {row["condition"]: row for row in group}
            pair_source_identity(session, by)
            case_pairs.append(paired_row(case, pair, by, expected_manifest))
        effects = [row["benign_minus_attack_fraction"] for row in case_pairs]
        require(all(effect == source for effect, source in zip(effects,
            aggregate["pair_deltas"] if case != "A13" else [
                next(row for row in source_rows if row["block"] == p and row["condition"] == "benign")
                    ["phase_fixed_timely_fraction"]["on"] -
                next(row for row in source_rows if row["block"] == p and row["condition"] == "attack")
                    ["phase_fixed_timely_fraction"]["on"] for p in range(4)])) and
                sum(effects)/count == endpoint_effect,
                "source estimator mismatch")
        pairs.extend(case_pairs)
        summary.append({"case_id": case, "n_independent_pairs": count,
                        "n_networks": 2*count, "eligible_per_arm": case_pairs[0]["eligible_per_arm"],
                        "effect_fraction": endpoint_effect, "ci95_fraction": endpoint_ci,
                        "source_raw_manifest_sha256": expected_manifest,
                        "qc_status": "PASS"})
    require(len(pairs) == 76 and len(summary) == 7, "Q01 coverage")
    return pairs, {"schema": "pluto.file02.q01.summary.v1",
                   "status": "PASS", "endpoint": "fixed_legitimate_all_four_observed_within_3000ms_fraction",
                   "n_cases": 7, "n_pairs": 76, "n_networks": 152,
                   "case_summaries": summary, "pooled_ci95": None,
                   "pooled_ci_status": "NOT_ESTIMATED_ATTACK_FAMILY_HETEROGENEITY",
                   "excluded_cases": {"A06": "replay mechanism; no fixed legitimate benign service contrast",
                                      "A07": "valid-only load; no invalid background",
                                      "A14": "process fault; no invalid background"},
                   "failed_or_excluded_pairs": []}


def csv_text(rows):
    buffer = StringIO()
    writer = csv.DictWriter(buffer, fieldnames=FIELDS, lineterminator="\n")
    writer.writeheader()
    writer.writerows({key: row[key] for key in FIELDS} for row in rows)
    return buffer.getvalue()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("run", "verify"))
    args = parser.parse_args()
    try:
        rows, summary = derive()
        csv_path, json_path = OUTPUT / "Q01_PAIRED_COMPARISON.csv", OUTPUT / "Q01_SUMMARY.json"
        if args.command == "run":
            OUTPUT.mkdir(parents=True, exist_ok=True)
            require(not csv_path.exists() and not json_path.exists(), "Q01 output exists")
            csv_path.write_text(csv_text(rows), encoding="utf-8")
            json_path.write_text(json.dumps(summary, sort_keys=True, indent=2) + "\n", encoding="utf-8")
        else:
            require(csv_path.read_text() == csv_text(rows) and
                    json.loads(json_path.read_text()) == summary, "Q01 derived output mismatch")
        print(f"Q01_{args.command.upper()}=PASS cases=7 pairs=76 networks=152")
        return 0
    except (ValueError, KeyError, TypeError, OSError, json.JSONDecodeError) as exc:
        print(f"Q01_ERROR={exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
