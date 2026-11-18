#!/usr/bin/env python3
"""Analyze matched-rate contention runs; independent runs are the replicate unit."""
import argparse
import csv
import hashlib
import json
from pathlib import Path
import re
import statistics

from analyze import bootstrap_mean_ci, process_metrics, summary
from run_contention import conditions


def clock_discontinuity(execution, log):
    match = re.search(r"--- PASS: TestPhase05Contention \(([0-9.]+)s\)", log)
    return bool(match and execution["wall_seconds"] - float(match.group(1)) > 30)


def one(record, execution):
    if not record.get("complete") or not record.get("all_validators_matched"):
        raise ValueError(f"incomplete or divergent run {execution['run_id']}")
    cfg = record["condition"]
    events = record["events"]
    valid = [e for e in events if e["expected_code"] == 0]
    invalid = [e for e in events if e["expected_code"] != 0]
    if (len(valid), len(invalid)) != (cfg["valid_count"], cfg["invalid_count"]):
        raise ValueError("offered count differs from plan")
    if (record["valid_submitted"], record["valid_included"],
            record["invalid_rejected"]) != (len(valid), len(valid), len(invalid)):
        raise ValueError("incomplete admission or inclusion")
    if (len(set(record["block_hashes"])) != 1 or
            len(set(record["app_hashes"])) != 1 or
            len(set(record["ordered_tx_digests"])) != 1 or
            len(set(record["total_supplies"])) != 1):
        raise ValueError("four validators diverged")
    duration = (record.get("measurement_duration_ns") or
                record["measurement_end_unix_nano"] -
                record["measurement_start_unix_nano"]) / 1e9
    if duration <= 0:
        raise ValueError("invalid measurement duration")
    classes = {}
    for event in invalid:
        classes.setdefault(event["case_id"], []).append(event["admission_latency_ns"] / 1000)
    class_cost = {key: summary(values) for key, values in sorted(classes.items())}
    sufficient = len(valid) >= 300 and len([e for e in valid if e.get("included_height")]) >= 300

    def achieved_offered(items, target):
        if not items or target <= 0:
            return 0.0
        field = "submit_offset_ns" if all("submit_offset_ns" in e for e in items) else "submit_unix_nano"
        times = [e[field] for e in items]
        span = (max(times) - min(times)) / 1e9 + 1 / target
        return len(items) / span if span > 0 else None

    valid_admission = summary([e["admission_latency_ns"] / 1000 for e in valid])
    inclusion = summary([e["inclusion_latency_ns"] / 1e6 for e in valid
                         if e.get("included_height")])
    if not sufficient:
        valid_admission["p99"] = inclusion["p99"] = "INSUFFICIENT SAMPLES"
    return dict(run_id=cfg["run_id"], series=execution["series"], repetition=execution["repetition"],
                profile=cfg["profile"], ratio=execution["ratio"], seed=cfg["seed"],
                source_commit=record["source_commit"], source_tree=record["source_tree"],
                corpus_sha256=record["corpus_sha256"], genesis_sha256=record["genesis_sha256"],
                target_valid_tps=cfg["valid_rate_tps"], target_invalid_tps=cfg["invalid_rate_tps"],
                achieved_valid_tps=achieved_offered(valid, cfg["valid_rate_tps"]),
                achieved_invalid_tps=achieved_offered(invalid, cfg["invalid_rate_tps"]),
                valid_submitted=len(valid), valid_confirmed=record["valid_included"],
                invalid_submitted=len(invalid), invalid_rejected=record["invalid_rejected"],
                valid_admission_us=valid_admission, inclusion_ms=inclusion,
                invalid_admission_us=summary([e["admission_latency_ns"] / 1000 for e in invalid]),
                rejection_cost_by_case_us=class_cost,
                legitimate_goodput_tps=record["valid_included"] / duration,
                duration_seconds=duration, sample_sufficient_for_descriptive_p99=sufficient,
                process=process_metrics(record),
                per_validator_process={str(index): process_metrics({**record,
                    "process_samples": [sample for sample in record["process_samples"]
                                        if sample["validator"] == index]})
                    for index in range(4)},
                matched_height=record["matched_height"],
                app_hash=record["app_hashes"][0], block_hash=record["block_hashes"][0],
                total_supply=record["total_supplies"][0],
                error_count=sum(bool(e.get("error")) for e in events))


def normalized(rows):
    baselines = {(r["profile"], r["series"], r["repetition"]): r for r in rows if r["ratio"] == 0}
    out = []
    for row in rows:
        if row["series"] == "invalid_only":
            continue
        baseline = baselines.get((row["profile"], row["series"], row["repetition"]))
        if baseline is None:
            continue
        p99, reference = row["inclusion_ms"]["p99"], baseline["inclusion_ms"]["p99"]
        ratio = p99 / reference if isinstance(p99, (int, float)) and isinstance(reference, (int, float)) and reference else None
        goodput_base = baseline["legitimate_goodput_tps"]
        cpu_base = baseline["process"]["cpu_percent"]
        rss_base = baseline["process"]["peak_sampled_rss_bytes"]
        out.append(dict(run_id=row["run_id"], baseline_run_id=baseline["run_id"],
                        series=row["series"], profile=row["profile"], ratio=row["ratio"],
                        repetition=row["repetition"], inclusion_p99_amplification=ratio,
                        goodput_degradation_fraction=(1-row["legitimate_goodput_tps"]/goodput_base)
                        if goodput_base else None,
                        cpu_percent_delta=row["process"]["cpu_percent"]-cpu_base
                        if cpu_base is not None else None,
                        sampled_rss_delta_bytes=row["process"]["peak_sampled_rss_bytes"]-rss_base))
    return out


def aggregate(rows, normalized_rows):
    groups = {}
    for row in rows:
        groups.setdefault((row["series"], row["profile"], row["ratio"]), []).append(row)
    norms = {(r["series"], r["profile"], r["ratio"], r["repetition"]): r for r in normalized_rows}
    out = []
    for (series, profile, ratio), items in sorted(groups.items()):
        def estimate(path):
            values = [path(item) for item in items]
            values = [v for v in values if isinstance(v, (int, float))]
            return dict(mean=statistics.mean(values) if values else None,
                        ci95=bootstrap_mean_ci(values), run_count=len(values))
        matching = [norms[(series, profile, ratio, r["repetition"])] for r in items
                    if (series, profile, ratio, r["repetition"]) in norms]
        def norm(key):
            values = [r[key] for r in matching if isinstance(r[key], (int, float))]
            return dict(mean=statistics.mean(values) if values else None,
                        ci95=bootstrap_mean_ci(values), run_count=len(values))
        out.append(dict(series=series, profile=profile, ratio=ratio,
                        repetitions=len(items), run_ids=[r["run_id"] for r in items],
                        valid_count_per_run=[r["valid_confirmed"] for r in items],
                        inclusion_p99_ms=estimate(lambda r: r["inclusion_ms"]["p99"]),
                        valid_admission_p99_us=estimate(lambda r: r["valid_admission_us"]["p99"]),
                        goodput_tps=estimate(lambda r: r["legitimate_goodput_tps"]),
                        cpu_percent=estimate(lambda r: r["process"]["cpu_percent"]),
                        peak_sampled_rss_bytes=estimate(lambda r: r["process"]["peak_sampled_rss_bytes"]),
                        inclusion_p99_amplification=norm("inclusion_p99_amplification"),
                        goodput_degradation_fraction=norm("goodput_degradation_fraction"),
                        cpu_percent_delta=norm("cpu_percent_delta"),
                        sampled_rss_delta_bytes=norm("sampled_rss_delta_bytes")))
    return out


def load(root, require_complete=False, repair_root=None):
    manifest = root / "execution.jsonl"
    executions = [json.loads(line) for line in manifest.read_text().splitlines() if line.strip()]
    ids = [x["run_id"] for x in executions]
    if len(ids) != len(set(ids)):
        raise ValueError("duplicate run ID")
    expected = []
    for index, original in enumerate(conditions("full"), 1):
        spec = original.copy()
        klass = spec.pop("class_")
        series = spec.pop("series")
        ratio = spec.pop("ratio")
        repetition = spec.pop("repetition")
        run_id = f"{index:03d}-{series}-{spec['profile']}-p{ratio}-r{repetition}"
        expected.append(dict(run_id=run_id, series=series, ratio=ratio,
                             repetition=repetition,
                             config={**spec, "class": klass, "run_id": run_id}))
    if require_complete and len(executions) != 78:
        raise ValueError(f"expected 78 full experiment runs, found {len(executions)}")
    if require_complete and any(any(item[key] != plan[key] for key in plan)
                                for item, plan in zip(executions, expected)):
        raise ValueError("full experiment condition or ordering mismatch")
    repairs = {}
    if repair_root is not None:
        repair_manifest = repair_root / "execution.jsonl"
        repair_items = [json.loads(line) for line in repair_manifest.read_text().splitlines() if line.strip()]
        if len(repair_items) != 1 or repair_items[0]["run_id"] != "034-secondary-b0-p0-r2":
            raise ValueError("repair ledger must contain only the affected condition")
        repairs[repair_items[0]["run_id"]] = (repair_root, repair_items[0])
    rows, failures, excluded = [], [], []
    for item in executions:
        data_path = root / "runs" / (item["run_id"] + ".json")
        log_path = root / "logs" / (item["run_id"] + ".log")
        if not log_path.is_file() or hashlib.sha256(log_path.read_bytes()).hexdigest() != item["log_sha256"]:
            raise ValueError(f"log digest mismatch {item['run_id']}")
        if data_path.is_file() and hashlib.sha256(data_path.read_bytes()).hexdigest() != item.get("data_sha256"):
            raise ValueError(f"result digest mismatch {item['run_id']}")
        log = log_path.read_text()
        match = re.search(r"--- PASS: TestPhase05Contention \(([0-9.]+)s\)", log)
        anomaly = clock_discontinuity(item, log)
        if anomaly:
            excluded.append(dict(run_id=item["run_id"], reason="wall-clock discontinuity between outer ledger and Go monotonic test duration",
                                 outer_wall_seconds=item["wall_seconds"], go_test_seconds=float(match.group(1)),
                                 log_sha256=item["log_sha256"], data_sha256=item.get("data_sha256")))
            if item["run_id"] not in repairs:
                failures.append({**item, "analysis_failure": "clock discontinuity; repair required"})
                continue
            replacement_root, replacement = repairs[item["run_id"]]
            if any(replacement[key] != item[key] for key in ("run_id", "series", "ratio", "repetition", "config")):
                raise ValueError("repair condition differs from superseded run")
            item = replacement
            data_path = replacement_root / "runs" / (item["run_id"] + ".json")
            log_path = replacement_root / "logs" / (item["run_id"] + ".log")
            if not log_path.is_file() or hashlib.sha256(log_path.read_bytes()).hexdigest() != item["log_sha256"]:
                raise ValueError("repair log digest mismatch")
            repair_log = log_path.read_text()
            repair_match = re.search(r"--- PASS: TestPhase05Contention \(([0-9.]+)s\)", repair_log)
            if not repair_match or clock_discontinuity(item, repair_log):
                raise ValueError("repair still has a clock anomaly")
        if item["exit_code"] != 0 or not data_path.is_file():
            failures.append(item)
            continue
        if hashlib.sha256(data_path.read_bytes()).hexdigest() != item["data_sha256"]:
            raise ValueError(f"result digest mismatch {item['run_id']}")
        rows.append(one(json.loads(data_path.read_text()), item))
    if require_complete and (failures or len(rows) != 78):
        raise ValueError(f"{len(failures)} failed or excluded runs; {len(rows)}/78 usable")
    return rows, failures, excluded


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("raw_root", type=Path)
    parser.add_argument("output_dir", type=Path)
    parser.add_argument("--require-complete", action="store_true")
    parser.add_argument("--repair-root", type=Path)
    args = parser.parse_args()
    rows, failures, excluded = load(args.raw_root, args.require_complete, args.repair_root)
    normal = normalized(rows)
    grouped = aggregate(rows, normal)
    args.output_dir.mkdir(parents=True, exist_ok=True)
    for name, value in (("per_run.json", rows), ("normalized.json", normal),
                        ("aggregate.json", grouped), ("failed_runs.json", failures),
                        ("excluded_runs.json", excluded)):
        (args.output_dir / name).write_text(json.dumps(value, indent=2, sort_keys=True)+"\n")
    with (args.output_dir / "per_run.csv").open("w", newline="") as stream:
        fields = ("run_id", "series", "profile", "ratio", "repetition", "target_valid_tps",
                  "achieved_valid_tps", "target_invalid_tps", "achieved_invalid_tps",
                  "valid_submitted", "valid_confirmed", "invalid_submitted", "invalid_rejected",
                  "valid_admission_p50_us", "valid_admission_p95_us", "valid_admission_p99_us",
                  "inclusion_p50_ms", "inclusion_p95_ms", "inclusion_p99_ms", "goodput_tps",
                  "cpu_percent", "sampled_peak_rss_bytes", "duration_seconds", "p99_sufficient",
                  "matched_height", "app_hash", "source_tree", "corpus_sha256",
                  "validator_0_cpu_percent", "validator_1_cpu_percent", "validator_2_cpu_percent", "validator_3_cpu_percent",
                  "validator_0_rss_bytes", "validator_1_rss_bytes", "validator_2_rss_bytes", "validator_3_rss_bytes")
        writer = csv.DictWriter(stream, fieldnames=fields, lineterminator="\n");writer.writeheader()
        for r in rows:
            row=dict(run_id=r["run_id"], series=r["series"], profile=r["profile"],
                                 ratio=r["ratio"], repetition=r["repetition"],
                                 target_valid_tps=r["target_valid_tps"], achieved_valid_tps=r["achieved_valid_tps"],
                                 target_invalid_tps=r["target_invalid_tps"], achieved_invalid_tps=r["achieved_invalid_tps"],
                                 valid_submitted=r["valid_submitted"], valid_confirmed=r["valid_confirmed"],
                                 invalid_submitted=r["invalid_submitted"], invalid_rejected=r["invalid_rejected"],
                                 valid_admission_p50_us=r["valid_admission_us"]["p50"],
                                 valid_admission_p95_us=r["valid_admission_us"]["p95"],
                                 valid_admission_p99_us=r["valid_admission_us"]["p99"],
                                 inclusion_p50_ms=r["inclusion_ms"]["p50"],
                                 inclusion_p95_ms=r["inclusion_ms"]["p95"],
                                 inclusion_p99_ms=r["inclusion_ms"]["p99"],
                                 goodput_tps=r["legitimate_goodput_tps"],
                                 cpu_percent=r["process"]["cpu_percent"],
                                 sampled_peak_rss_bytes=r["process"]["peak_sampled_rss_bytes"],
                                 duration_seconds=r["duration_seconds"],
                                 p99_sufficient=r["sample_sufficient_for_descriptive_p99"],
                                 matched_height=r["matched_height"], app_hash=r["app_hash"],
                                 source_tree=r["source_tree"], corpus_sha256=r["corpus_sha256"])
            for index in range(4):
                row[f"validator_{index}_cpu_percent"] = r["per_validator_process"][str(index)]["cpu_percent"]
                row[f"validator_{index}_rss_bytes"] = r["per_validator_process"][str(index)]["peak_sampled_rss_bytes"]
            writer.writerow(row)
    print(f"completed={len(rows)} failures={len(failures)} excluded_originals={len(excluded)} output={args.output_dir}")
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
