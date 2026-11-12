import hashlib
import json
from pathlib import Path
import tempfile
import unittest
import subprocess
import sys

import file02_common as f
import file02_runtime as runtime


H40 = "a" * 40
H64 = "b" * 64


def plan():
    return {
        "schema": "pluto.file02.plan.v1", "status": "LOCKED", "run_id": "qa_001",
        "case_id": "QA_COMMON", "protocol_version": "file02-common-v1",
        "research_question": "QA_ONLY", "hypothesis": "No research inference",
        "null_hypothesis": "Synthetic null", "alternative_hypothesis": "Synthetic alternative",
        "analysis_plan": {"primary_endpoint": "valid_included_count", "independent_unit": "fresh_four_validator_network",
                          "ci_method": "not computed in synthetic fixture", "exclusion_rule": "exclude failed QC"},
        "raw_capture_plan": ["telemetry", "transaction_results", "validator_blocks_and_apphash",
                             "loadgen_timeline", "host_metrics"],
        "expected_oracle": "Four nodes agree", "profile": "H", "workload": "MIXED",
        "regime": "fixed_legitimate_plus_adversarial", "traffic_source": "RPC",
        "topology": "FOUR_VALIDATOR_SINGLE_HOST_DIAGNOSTIC", "policy": "BASELINE_VERIFY_ORDER",
        "actor_capability": "B", "host_inventory": {"cpu_model": "fixture", "cpu_features": ["fixture"], "architecture": "amd64", "logical_cpus": 4},
        "software_versions": {"go": "fixture", "cometbft": "v0.38.26", "crypto": "fixture", "os": "fixture"},
        "network_config": {"validator_count": 4, "mempool_type": "flood", "mempool_size": 128,
                           "mempool_max_bytes": 8388608, "mempool_cache_size": 1024,
                           "mempool_max_tx_bytes": 8192, "mempool_recheck": True,
                           "mempool_broadcast": True, "max_block_bytes": 1048576,
                           "empty_block_interval_ms": 1000, "timeout_commit_ms": 1000,
                           "network_shape": "loopback", "endpoint_hosts": ["127.0.0.1"]},
        "safety_config": {"allowed_rpc_host": "127.0.0.1", "max_tx_per_second": 25,
                          "max_bytes_per_second": 65536, "max_outstanding": 16,
                          "max_duration_seconds": 60, "request_timeout_ms": 2000,
                          "max_cpu_fraction": 0.9, "cpu_breach_seconds": 30,
                          "max_rss_fraction": 0.8, "min_free_disk_fraction": 0.1,
                          "max_observer_stale_ms": 4000},
        "seed": 1, "order_index": 0,
        "phase_seconds": {"warmup": 1, "steady": 4, "drain": 3, "cooldown": 1},
        "planned": {"valid_tx_s": 1, "invalid_tx_s": 1, "total_bytes_s": 5000},
        "matching": {"matched_variable": "none", "unmatched_variables": [], "legitimate_stream_id": "fixed-1"},
        "pilot_reference": "synthetic-test-only", "independent_run_target": 2,
        "p99_precision_rule": "synthetic-test-only", "source_commit": H40, "source_tree": H40,
        "config_sha256": H64, "corpus_sha256": H64, "genesis_sha256": H64,
    }


def observation(raw_hash):
    return {
        "schema": "pluto.file02.observation.v1", "run_id": "qa_001",
        "raw_manifest_sha256": raw_hash, "plan_sha256": f.plan_digest(plan()),
        "qc": {key: True for key in f.FLAGS},
        "achieved": {"valid_tx_s": 1, "invalid_tx_s": 1, "total_bytes_s": 5000},
        "steady_windows": [{"planned_count": 2, "actual_count": 2,
                            "planned_valid_count": 1, "actual_valid_count": 1}],
        "metrics": {"consensus": {"valid_included_count": 4, "matched_AppHash": True}},
        "availability": {name: "synthetic fixture; unmeasured" for names in f.METRICS.values() for name in names},
    }


class File02CommonTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.raw = Path(self.tmp.name) / "raw"
        self.raw.mkdir()
        (self.raw / "events.json").write_text("{}\n")
        p = plan()
        runtime = {"schema": "pluto.file02.runtime_inventory.v1", "run_id": p["run_id"],
                   "profile": p["profile"], "source_clean": True,
                   "node_configs": [{"config_toml_sha256": H64} for _ in range(4)]}
        runtime.update({key: p[key] for key in ("source_commit", "source_tree", "config_sha256",
                                             "software_versions", "host_inventory", "network_config")})
        (self.raw / "runtime_inventory.json").write_text(json.dumps(runtime))
        (self.raw / "manifest.json").write_text(json.dumps({
            "schema": "pluto.file02.raw_manifest.v1",
            "files": {name: f.digest(self.raw / name) for name in ("events.json", "runtime_inventory.json")},
        }))

    def test_plan_rejects_unavailable_and_incompatible_factors(self):
        p = plan()
        self.assertTrue(f.validate_plan(p))
        p["traffic_source"] = "P2P"
        with self.assertRaisesRegex(ValueError, "origin"):
            f.validate_plan(p)
        p["traffic_source"] = "RPC"
        p["profile"] = "P"
        p["workload"] = "ECDSA_INVALID"
        with self.assertRaisesRegex(ValueError, "workload/profile"):
            f.validate_plan(p)

    def test_plan_rejects_public_target_and_unsafe_rate(self):
        p = plan()
        p["network_config"]["endpoint_hosts"] = ["example.org"]
        with self.assertRaisesRegex(ValueError, "non-loopback"):
            f.validate_plan(p)
        p = plan()
        p["planned"]["total_bytes_s"] = 70000
        with self.assertRaisesRegex(ValueError, "byte rate exceeds safety cap"):
            f.validate_plan(p)

    def test_qc_rate_and_safety_exclude_but_retain_raw(self):
        o = observation(f.digest(self.raw / "manifest.json"))
        qc, result = f.evaluate(plan(), o, self.raw)
        self.assertEqual(qc["status"], "USABLE")
        self.assertIsNone(result["metrics"]["relay"]["propagation_p99_ms"])
        o["qc"]["apphash_agreement"] = False
        o["achieved"]["valid_tx_s"] = 0.7
        qc, _ = f.evaluate(plan(), o, self.raw)
        self.assertEqual(qc["status"], "INVALID_FOR_ANALYSIS")
        self.assertIn("apphash_agreement", qc["exclude_reasons"])
        self.assertIn("RATE_MISS_valid_tx_s", qc["exclude_reasons"])
        self.assertTrue((self.raw / "events.json").exists())

    def test_tamper_and_extra_raw_file_fail(self):
        o = observation(f.digest(self.raw / "manifest.json"))
        (self.raw / "events.json").write_text("tampered")
        with self.assertRaisesRegex(ValueError, "raw hash mismatch"):
            f.evaluate(plan(), o, self.raw)
        (self.raw / "events.json").write_text("{}\n")
        (self.raw / "extra").write_text("unmanifested")
        with self.assertRaisesRegex(ValueError, "unmanifested"):
            f.evaluate(plan(), o, self.raw)

    def test_plan_change_after_observation_fails(self):
        o = observation(f.digest(self.raw / "manifest.json"))
        changed = plan()
        changed["seed"] = 2
        with self.assertRaisesRegex(ValueError, "plan binding"):
            f.evaluate(changed, o, self.raw)

    def test_actual_runtime_mismatch_fails(self):
        runtime_path = self.raw / "runtime_inventory.json"
        runtime = json.loads(runtime_path.read_text())
        runtime["network_config"]["mempool_size"] = 999
        runtime_path.write_text(json.dumps(runtime))
        manifest = json.loads((self.raw / "manifest.json").read_text())
        manifest["files"]["runtime_inventory.json"] = f.digest(runtime_path)
        (self.raw / "manifest.json").write_text(json.dumps(manifest))
        o = observation(f.digest(self.raw / "manifest.json"))
        with self.assertRaisesRegex(ValueError, "planned/actual runtime mismatch"):
            f.evaluate(plan(), o, self.raw)

    def test_actual_comet_config_values_are_checked(self):
        net = f.read_json(f.ROOT / "config/s06_pilot.json")["network"]
        config_path = Path(self.tmp.name) / "config.toml"
        config_path.write_text('''proxy_app = "tcp://127.0.0.1:10000"
[rpc]
laddr = "tcp://127.0.0.1:10001"
[p2p]
laddr = "tcp://127.0.0.1:10002"
persistent_peers = "a@127.0.0.1:1,b@127.0.0.1:2,c@127.0.0.1:3"
[mempool]
type = "flood"
size = 128
max_txs_bytes = 8388608
cache_size = 1024
max_tx_bytes = 8192
recheck = true
broadcast = true
[consensus]
create_empty_blocks = true
create_empty_blocks_interval = "1s"
timeout_commit = "1s"
[instrumentation]
prometheus = true
prometheus_listen_addr = "127.0.0.1:10003"
''')
        self.assertEqual(runtime.require_node_config(config_path, net)["config_toml_sha256"], f.digest(config_path))
        config_path.write_text(config_path.read_text().replace("size = 128", "size = 129"))
        with self.assertRaisesRegex(ValueError, "actual Comet config mismatch"):
            runtime.require_node_config(config_path, net)

    def test_capability_and_missing_availability_fail(self):
        changed = plan()
        changed["workload"] = "STATE_INVALID"
        with self.assertRaisesRegex(ValueError, "workload/capability"):
            f.validate_plan(changed)
        o = observation(f.digest(self.raw / "manifest.json"))
        del o["availability"]["CPU_crypto"]
        with self.assertRaisesRegex(ValueError, "missing availability"):
            f.evaluate(plan(), o, self.raw)
        o = observation(f.digest(self.raw / "manifest.json"))
        o["steady_windows"][0]["actual_valid_count"] = 0
        qc, _ = f.evaluate(plan(), o, self.raw)
        self.assertIn("LEGITIMATE_RATE_MISS_WINDOW_0", qc["exclude_reasons"])
        o = observation(f.digest(self.raw / "manifest.json"))
        o["metrics"]["node"] = {"CPU_total": -1}
        with self.assertRaisesRegex(ValueError, "invalid metric"):
            f.evaluate(plan(), o, self.raw)

    def test_aggregate_uses_runs_and_keeps_ci_open(self):
        o = observation(f.digest(self.raw / "manifest.json"))
        _, result = f.evaluate(plan(), o, self.raw)
        out = f.aggregate([result], "valid_included_count")
        self.assertEqual(out["ci_status"], "INSUFFICIENT_INDEPENDENT_RUNS")
        self.assertIsNone(out["ci95"])
        second = json.loads(json.dumps(result))
        second["run_id"] = "qa_002"
        out = f.aggregate([result, second], "valid_included_count")
        self.assertEqual(out["n_runs_usable"], 2)
        self.assertEqual(out["ci_status"], "NOT_COMPUTED_PENDING_PREREGISTERED_METHOD")
        second["profile"] = "C"
        with self.assertRaisesRegex(ValueError, "mixed comparison"):
            f.aggregate([result, second], "valid_included_count")

    def test_cli_writes_once_and_aggregate_checks_qc(self):
        plan_path = Path(self.tmp.name) / "plan.json"
        obs_path = Path(self.tmp.name) / "observation.json"
        plan_path.write_text(json.dumps(plan()))
        obs_path.write_text(json.dumps(observation(f.digest(self.raw / "manifest.json"))))
        out = Path(self.tmp.name) / "derived"
        cmd = [sys.executable, str(Path(f.__file__)), "qc-run", str(plan_path), str(obs_path), str(self.raw), str(out)]
        done = subprocess.run(cmd, capture_output=True, text=True)
        self.assertEqual(done.returncode, 0, done.stderr)
        self.assertEqual(done.stdout.strip(), "USABLE")
        self.assertIn("consensus__valid_included_count", (out / "result_per_run.csv").read_text().splitlines()[0])
        again = subprocess.run(cmd, capture_output=True, text=True)
        self.assertEqual(again.returncode, 2)
        agg_path = out / "result_aggregate.json"
        done = subprocess.run([sys.executable, str(Path(f.__file__)), "aggregate", str(agg_path),
                               "valid_included_count", str(out / "result_per_run.json")], capture_output=True, text=True)
        self.assertEqual(done.returncode, 0, done.stderr)
        self.assertEqual(json.loads(agg_path.read_text())["n_runs_usable"], 1)
        qc = json.loads((out / "qc_report.json").read_text())
        qc["status"] = "INVALID_FOR_ANALYSIS"
        (out / "qc_report.json").write_text(json.dumps(qc))
        mismatch = subprocess.run([sys.executable, str(Path(f.__file__)), "aggregate", str(out / "mismatch.json"),
                                   "valid_included_count", str(out / "result_per_run.json")], capture_output=True, text=True)
        self.assertEqual(mismatch.returncode, 2)

    def test_freeze_raw_is_write_once_and_hash_bound(self):
        raw = Path(self.tmp.name) / "fresh"
        raw.mkdir()
        (raw / "events.jsonl").write_text("one\n")
        manifest_hash = f.freeze_raw(raw)
        self.assertEqual(f.verify_raw(raw), manifest_hash)
        with self.assertRaisesRegex(ValueError, "without manifest"):
            f.freeze_raw(raw)
        (raw / "events.jsonl").chmod(0o644)
        (raw / "events.jsonl").write_text("two\n")
        with self.assertRaisesRegex(ValueError, "raw hash mismatch"):
            f.verify_raw(raw)


if __name__ == "__main__":
    unittest.main()
