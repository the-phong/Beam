#!/usr/bin/env python3
"""File 02 common plan, integrity and QC gate. No experiment runner here."""

import argparse
import csv
import hashlib
import json
import math
import os
from pathlib import Path
import statistics
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
FACTORS = json.loads((ROOT / "config/file02_factors.json").read_text())
METRICS = {
    "workload": "offered_valid_tx_s offered_invalid_tx_s offered_tx_bytes_s achieved_valid_tx_s achieved_invalid_tx_s achieved_tx_bytes_s burstiness distinct_tx_count",
    "crypto": "rejection_stage verify_secp_us verify_mldsa_us total_checktx_us cpu_ns_per_tx alloc_B_per_tx",
    "ingress": "ingress_tx_s ingress_bytes_s queue_depth queue_wait_ms ingress_source",
    "mempool": "accepted_tx_count accepted_bytes full_count full_bytes rejected_by_reason eviction_count",
    "relay": "peer_rx_bytes_s peer_tx_bytes_s relay_queue propagation_p50_ms propagation_p95_ms propagation_p99_ms",
    "consensus": "valid_included_count legitimate_goodput_tx_s inclusion_latency_p50_ms inclusion_latency_p95_ms inclusion_latency_p99_ms finality_measure_defined blocks matched_AppHash",
    "node": "CPU_total CPU_app CPU_crypto RSS_peak GC_pause process_count thread_count",
    "detection": "TP FP TN FN PR_AUC recall_at_fixed_FPR FP_per_hour lead_time_ms inference_p99_ms detector_CPU detector_RSS",
    "quality": "n_samples n_runs CI95 exclude_reason observer_drop_count achieved_rate_error",
}
METRICS = {group: names.split() for group, names in METRICS.items()}
METRIC_REGISTRY = json.loads((ROOT / "config/file02_metrics.json").read_text())
FLAGS = (
    "fresh_network", "oracle_pass", "four_validators_live", "apphash_agreement",
    "clock_continuous", "observer_drops_within_limit", "generator_not_saturated",
    "safety_guard_clear", "distinct_transactions", "phases_separate",
)


def require(ok, message):
    if not ok:
        raise ValueError(message)


def read_json(path):
    def unique_pairs(pairs):
        obj = {}
        for key, value in pairs:
            require(key not in obj, f"duplicate JSON key: {key}")
            obj[key] = value
        return obj
    return json.loads(Path(path).read_text(), object_pairs_hook=unique_pairs,
                      parse_constant=lambda value: (_ for _ in ()).throw(ValueError(f"non-finite JSON value: {value}")))


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def plan_digest(plan):
    return hashlib.sha256(json.dumps(plan, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def sha(value):
    return isinstance(value, str) and len(value) == 64 and all(c in "0123456789abcdef" for c in value)


def number(value):
    return isinstance(value, (int, float)) and not isinstance(value, bool) and math.isfinite(value) and value >= 0


def validate_plan(plan, locked=True):
    require(plan.get("schema") == "pluto.file02.plan.v1", "plan schema")
    require(plan.get("status") in ("DRAFT", "LOCKED"), "plan status")
    if locked:
        require(plan["status"] == "LOCKED", "plan not locked")
    for key in ("run_id", "case_id", "protocol_version", "research_question", "hypothesis",
                "null_hypothesis", "alternative_hypothesis", "analysis_plan", "raw_capture_plan",
                "expected_oracle", "profile", "workload", "regime", "traffic_source",
                "topology", "policy", "actor_capability", "host_inventory", "software_versions", "network_config",
                "safety_config", "seed", "order_index", "phase_seconds", "planned", "matching"):
        require(plan.get(key) is not None, f"missing {key}")
    for key in ("case_id", "protocol_version", "research_question", "hypothesis",
                "null_hypothesis", "alternative_hypothesis", "expected_oracle"):
        require(isinstance(plan[key], str) and plan[key].strip(), f"empty {key}")
    analysis=plan["analysis_plan"]
    require(isinstance(analysis,dict) and analysis.get("independent_unit")=="fresh_four_validator_network" and
            all(isinstance(analysis.get(key),str) and analysis[key].strip() for key in
                ("primary_endpoint","ci_method","exclusion_rule")), "analysis preregistration")
    require(plan["raw_capture_plan"]==["telemetry","transaction_results","validator_blocks_and_apphash",
                                         "loadgen_timeline","host_metrics"], "raw capture plan")
    require(isinstance(plan["run_id"], str) and plan["run_id"].replace("-", "").replace("_", "").isalnum(), "run_id")
    require(plan["profile"] in FACTORS["profiles"], "profile")
    require(plan["workload"] in FACTORS["workloads"] and plan["workload"] not in FACTORS["incompatible"].get(plan["profile"], []), "workload/profile mismatch")
    require(plan["actor_capability"] in ("NONE", "A", "B", "C", "D"), "actor capability")
    require(plan["actor_capability"] in FACTORS["capabilities"][plan["workload"]], "workload/capability mismatch")
    require(plan["actor_capability"] != "D", "multi-source capability not available")
    require(plan["regime"] in FACTORS["regimes"], "regime")
    require(plan["traffic_source"] in FACTORS["origins_available"], "traffic origin not available")
    require(plan["topology"] in FACTORS["topologies_available"], "topology not available")
    require(plan["policy"] in FACTORS["policy_available"], "policy not available")
    require(isinstance(plan["seed"], int) and not isinstance(plan["seed"], bool), "seed")
    require(isinstance(plan["order_index"], int) and plan["order_index"] >= 0, "order_index")
    for key in ("source_commit", "source_tree", "config_sha256", "corpus_sha256", "genesis_sha256"):
        value = plan.get(key)
        require(isinstance(value, str) and len(value) == (40 if key in ("source_commit", "source_tree") else 64)
                and all(c in "0123456789abcdef" for c in value), key)
    for key in ("warmup", "steady", "drain", "cooldown"):
        require(number(plan["phase_seconds"].get(key)), f"phase {key}")
    require(all(plan["phase_seconds"][key] > 0 for key in ("warmup", "steady", "drain", "cooldown")), "phase duration")
    for key in ("valid_tx_s", "invalid_tx_s", "total_bytes_s"):
        require(number(plan["planned"].get(key)), f"planned {key}")
    require(isinstance(plan["host_inventory"], dict) and plan["host_inventory"].get("cpu_model")
            and isinstance(plan["host_inventory"].get("cpu_features"), list)
            and plan["host_inventory"]["cpu_features"]
            and plan["host_inventory"].get("architecture")
            and isinstance(plan["host_inventory"].get("logical_cpus"), int)
            and plan["host_inventory"]["logical_cpus"] > 0, "host inventory")
    require(isinstance(plan["software_versions"], dict) and all(isinstance(plan["software_versions"].get(k), str)
            and plan["software_versions"][k].strip()
            for k in ("go", "cometbft", "crypto", "os")), "software versions")
    require(isinstance(plan["network_config"], dict) and plan["network_config"].get("validator_count") == 4
            and all(k in plan["network_config"] for k in (
                "mempool_type", "mempool_size", "mempool_max_bytes", "mempool_cache_size",
                "mempool_max_tx_bytes", "mempool_recheck", "mempool_broadcast",
                "max_block_bytes", "empty_block_interval_ms", "timeout_commit_ms",
                "network_shape", "endpoint_hosts")), "network config")
    require(plan["network_config"]["endpoint_hosts"] and all(
            host in ("127.0.0.1", "::1") for host in plan["network_config"]["endpoint_hosts"]),
            "non-loopback endpoint")
    require(isinstance(plan["safety_config"], dict) and plan["safety_config"].get("allowed_rpc_host") in ("127.0.0.1", "::1")
            and all(number(plan["safety_config"].get(k)) and plan["safety_config"][k] > 0 for k in (
                "max_tx_per_second", "max_bytes_per_second", "max_outstanding",
                "max_duration_seconds", "request_timeout_ms", "max_cpu_fraction",
                "cpu_breach_seconds", "max_rss_fraction", "min_free_disk_fraction",
                "max_observer_stale_ms")), "safety config")
    require(plan["planned"]["valid_tx_s"] + plan["planned"]["invalid_tx_s"] <= plan["safety_config"]["max_tx_per_second"], "tx rate exceeds safety cap")
    require(plan["planned"]["total_bytes_s"] <= plan["safety_config"]["max_bytes_per_second"], "byte rate exceeds safety cap")
    require(isinstance(plan["matching"], dict) and plan["matching"].get("matched_variable") in ("tx_s", "bytes_s", "resource_pressure", "none")
            and isinstance(plan["matching"].get("unmatched_variables"), list), "matching declaration")
    if plan["regime"] == "matched_pressure":
        require(plan["matching"]["matched_variable"] != "none", "missing matched variable")
    if plan["regime"] == "fixed_legitimate_plus_adversarial":
        require(plan["matching"].get("legitimate_stream_id"), "missing fixed legitimate stream")
    if locked:
        require(plan.get("pilot_reference") and isinstance(plan.get("independent_run_target"), int)
                and plan["independent_run_target"] >= 2 and plan.get("p99_precision_rule"),
                "pilot and precision preregistration required")
    return True


def verify_raw(raw_dir):
    root = Path(raw_dir)
    require(root.is_dir() and not root.is_symlink(), "raw directory")
    require(not (root / "manifest.json").is_symlink(), "raw manifest symlink")
    manifest = read_json(root / "manifest.json")
    require(manifest.get("schema") == "pluto.file02.raw_manifest.v1", "raw manifest schema")
    entries = manifest.get("files")
    require(isinstance(entries, dict) and entries, "raw manifest files")
    seen = set()
    for rel, expected in entries.items():
        part = Path(rel)
        require(not part.is_absolute() and ".." not in part.parts and rel != "manifest.json" and sha(expected), "unsafe manifest entry")
        path = root / part
        require(path.resolve().is_relative_to(root.resolve()) and path.is_file()
                and not path.is_symlink() and digest(path) == expected, f"raw hash mismatch: {rel}")
        seen.add(rel)
    actual = {str(p.relative_to(root)) for p in root.rglob("*") if p.is_file() and p.name != "manifest.json"}
    require(not any(p.is_symlink() for p in root.rglob("*")), "raw symlink")
    require(actual == seen, "unmanifested or missing raw file")
    return digest(root / "manifest.json")


def freeze_raw(raw_dir):
    require(not Path(raw_dir).is_symlink(), "raw directory symlink")
    root = Path(raw_dir).resolve()
    require(root.is_dir() and not (root / "manifest.json").exists(), "raw directory must exist without manifest")
    repository = ROOT.parent.resolve()
    if root.is_relative_to(repository):
        ignored = subprocess.run(["git", "check-ignore", "-q", "--", str(root)], cwd=repository)
        require(ignored.returncode == 0, "raw directory inside repository must be Git-ignored")
    paths = list(root.rglob("*"))
    require(paths and not any(p.is_symlink() for p in paths), "empty or symlinked raw directory")
    files = {str(p.relative_to(root)): digest(p) for p in paths if p.is_file()}
    require(files, "empty raw directory")
    manifest = {"schema": "pluto.file02.raw_manifest.v1", "files": dict(sorted(files.items()))}
    write_once(root / "manifest.json", json.dumps(manifest, indent=2, sort_keys=True) + "\n")
    require(verify_raw(root) == digest(root / "manifest.json"), "freeze verification failed")
    for path in paths:
        path.chmod(0o555 if path.is_dir() else 0o444)
    (root / "manifest.json").chmod(0o444)
    root.chmod(0o555)
    return digest(root / "manifest.json")


def evaluate(plan, observation, raw_dir):
    validate_plan(plan)
    require(observation.get("schema") == "pluto.file02.observation.v1" and observation.get("run_id") == plan["run_id"], "observation identity")
    require(observation.get("plan_sha256") == plan_digest(plan), "plan binding")
    raw_hash = verify_raw(raw_dir)
    require(observation.get("raw_manifest_sha256") == raw_hash, "raw manifest binding")
    runtime = read_json(Path(raw_dir) / "runtime_inventory.json")
    require(runtime.get("schema") == "pluto.file02.runtime_inventory.v1"
            and runtime.get("run_id") == plan["run_id"] and runtime.get("profile") == plan["profile"]
            and runtime.get("source_clean") is True, "runtime inventory identity or dirty source")
    for key in ("source_commit", "source_tree", "config_sha256", "software_versions",
                "host_inventory", "network_config"):
        require(runtime.get(key) == plan[key], f"planned/actual runtime mismatch: {key}")
    require(isinstance(runtime.get("node_configs"), list) and len(runtime["node_configs"]) == 4
            and all(sha(node.get("config_toml_sha256")) for node in runtime["node_configs"]),
            "four actual node config hashes missing")
    require(isinstance(observation.get("qc"), dict) and isinstance(observation.get("achieved"), dict), "observation QC/rates")
    failures = [key for key in FLAGS if observation["qc"].get(key) is not True]
    for key in ("valid_tx_s", "invalid_tx_s", "total_bytes_s"):
        actual = observation.get("achieved", {}).get(key)
        target = plan["planned"][key]
        if not number(actual) or (target == 0 and actual != 0) or (target > 0 and abs(actual - target) / target > 0.05):
            failures.append("RATE_MISS_" + key)
    windows = observation.get("steady_windows")
    require(isinstance(windows, list) and windows, "missing steady windows")
    for index, window in enumerate(windows):
        planned, actual = window.get("planned_count"), window.get("actual_count")
        require(number(planned) and number(actual), "window count")
        if (planned == 0 and actual != 0) or (planned > 0 and actual < 0.9 * planned):
            failures.append(f"RATE_MISS_WINDOW_{index}")
        if plan["regime"] == "fixed_legitimate_plus_adversarial":
            valid_planned, valid_actual = window.get("planned_valid_count"), window.get("actual_valid_count")
            require(number(valid_planned) and number(valid_actual), "fixed legitimate window count")
            if (valid_planned == 0 and valid_actual != 0) or (valid_planned > 0 and valid_actual < 0.9 * valid_planned):
                failures.append(f"LEGITIMATE_RATE_MISS_WINDOW_{index}")
    metric_input = observation.get("metrics", {})
    require(isinstance(metric_input, dict), "metrics")
    require(set(metric_input).issubset(METRICS), "unknown metric group")
    metrics = {}
    for group, names in METRICS.items():
        source_group = metric_input.get(group, {})
        require(isinstance(source_group, dict) and set(source_group).issubset(names), f"metric group {group}")
        require(all(not isinstance(value, float) or math.isfinite(value) for value in source_group.values()), f"non-finite metric {group}")
        for name, value in source_group.items():
            if value is not None and name not in ("rejection_stage", "ingress_source", "finality_measure_defined", "rejected_by_reason", "CI95", "exclude_reason"):
                require(number(value) or (name == "matched_AppHash" and isinstance(value, bool)), f"invalid metric {group}.{name}")
        metrics[group] = {name: source_group.get(name) for name in names}
    require(all(value is None for value in metrics["detection"].values()), "detection is out of scope")
    availability = observation.get("availability", {})
    require(isinstance(availability, dict), "availability")
    for group, values in metrics.items():
        for name, value in values.items():
            if METRIC_REGISTRY["metrics"][group][name]["status"] == "UNAVAILABLE":
                require(value is None, f"unavailable metric must be null: {group}.{name}")
            if value is None:
                require(isinstance(availability.get(name), str) and availability[name].strip(), f"missing availability reason: {name}")
    status = "USABLE" if not failures else "INVALID_FOR_ANALYSIS"
    qc = {"schema": "pluto.file02.qc.v1", "run_id": plan["run_id"], "status": status,
          "exclude_reasons": failures, "raw_manifest_sha256": raw_hash,
          "plan_sha256": plan_digest(plan), "checks": observation["qc"]}
    result = {"schema": "pluto.file02.result_per_run.v1", "run_id": plan["run_id"],
              "source_commit": plan["source_commit"], "source_tree": plan["source_tree"],
              "profile": plan["profile"], "case_id": plan["case_id"], "workload": plan["workload"],
              "actor_capability": plan.get("actor_capability"), "traffic_source": plan["traffic_source"],
              "topology": plan["topology"], "regime": plan["regime"], "seed": plan["seed"],
              "order_index": plan["order_index"], "planned": plan["planned"],
              "achieved": observation["achieved"], "metrics": metrics, "availability": availability,
              "qc_status": status, "exclude_reasons": failures, "raw_manifest_sha256": raw_hash}
    result["plan_sha256"] = plan_digest(plan)
    return qc, result


def write_once(path, content):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, "w") as handle:
        handle.write(content)


def aggregate(results, endpoint):
    require(results, "no results")
    require(all(row.get("schema") == "pluto.file02.result_per_run.v1" for row in results), "result schema")
    identity = ("source_tree", "profile", "case_id", "workload", "traffic_source", "topology", "regime")
    key = tuple(results[0].get(field) for field in identity)
    require(all(tuple(row.get(field) for field in identity) == key for row in results), "mixed comparison strata")
    require(len({row["run_id"] for row in results}) == len(results), "duplicate run ID")
    included = [row for row in results if row.get("qc_status") == "USABLE"]
    values = [row.get("metrics", {}).get("consensus", {}).get(endpoint) for row in included]
    require(endpoint in METRICS["consensus"], "unsupported endpoint")
    require(all(number(v) for v in values), "unmeasured endpoint in usable run")
    n = len(values)
    out = {"schema": "pluto.file02.aggregate.v1", "stratum": dict(zip(identity, key)),
           "endpoint": endpoint, "n_runs_total": len(results), "n_runs_usable": n,
           "included_run_ids": [r["run_id"] for r in included],
           "excluded": {r["run_id"]: r["exclude_reasons"] for r in results if r.get("qc_status") != "USABLE"},
           "mean": statistics.mean(values) if n else None,
           "median": statistics.median(values) if n else None,
           "ci95": None, "ci_status": "INSUFFICIENT_INDEPENDENT_RUNS" if n < 2 else "NOT_COMPUTED_PENDING_PREREGISTERED_METHOD"}
    return out


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    p = sub.add_parser("validate-plan")
    p.add_argument("plan")
    p.add_argument("--allow-draft", action="store_true")
    p = sub.add_parser("freeze-raw")
    p.add_argument("raw_dir")
    p = sub.add_parser("qc-run")
    p.add_argument("plan")
    p.add_argument("observation")
    p.add_argument("raw_dir")
    p.add_argument("output_dir")
    p = sub.add_parser("aggregate")
    p.add_argument("output")
    p.add_argument("endpoint")
    p.add_argument("results", nargs="+")
    args = parser.parse_args()
    try:
        if args.command == "validate-plan":
            validate_plan(read_json(args.plan), locked=not args.allow_draft)
            print("PLAN_OK")
        elif args.command == "freeze-raw":
            print(freeze_raw(args.raw_dir))
        elif args.command == "qc-run":
            qc, result = evaluate(read_json(args.plan), read_json(args.observation), args.raw_dir)
            out = Path(args.output_dir)
            require(not out.exists(), "output directory already exists")
            out.mkdir(parents=True)
            write_once(out / "qc_report.json", json.dumps(qc, indent=2, sort_keys=True) + "\n")
            write_once(out / "result_per_run.json", json.dumps(result, indent=2, sort_keys=True) + "\n")
            with (out / "result_per_run.csv").open("x", newline="") as handle:
                fields = ["run_id", "source_tree", "profile", "case_id", "actor_capability", "traffic_source", "topology", "regime", "seed", "qc_status", "planned_valid_tx_s", "planned_invalid_tx_s", "planned_total_bytes_s", "achieved_valid_tx_s", "achieved_invalid_tx_s", "achieved_total_bytes_s"]
                fields += [f"{group}__{name}" for group, names in METRICS.items() for name in names]
                writer = csv.DictWriter(handle, fieldnames=fields)
                writer.writeheader()
                writer.writerow({**{k: result.get(k) for k in fields},
                                 **{f"planned_{k}": v for k, v in result["planned"].items()},
                                 **{f"achieved_{k}": v for k, v in result["achieved"].items()},
                                 **{f"{group}__{name}": value for group, values in result["metrics"].items() for name, value in values.items()}})
            print(qc["status"])
        else:
            rows = []
            for item in args.results:
                row = read_json(item)
                qc = read_json(Path(item).with_name("qc_report.json"))
                require(qc.get("run_id") == row.get("run_id") and qc.get("status") == row.get("qc_status")
                        and qc.get("raw_manifest_sha256") == row.get("raw_manifest_sha256")
                        and qc.get("plan_sha256") == row.get("plan_sha256"), "QC/result mismatch")
                rows.append(row)
            output = aggregate(rows, args.endpoint)
            write_once(args.output, json.dumps(output, indent=2, sort_keys=True) + "\n")
            print(output["ci_status"])
    except (ValueError, KeyError, OSError) as exc:
        print(f"FILE02_ERROR: {exc}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
