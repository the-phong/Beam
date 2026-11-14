"""Attack case schedule and artifact validation regression checks."""

import csv
import io
import hashlib
import json
from pathlib import Path
import tempfile
import unittest

import file02_attack as attack


class AttackContractTest(unittest.TestCase):
    def test_schedule_balanced_unique_and_case_bound(self):
        for case in attack.CASES:
            schedule = attack.schedule(case)
            jobs = schedule["jobs"]
            self.assertEqual(len(jobs), 24)
            self.assertEqual(len({job["run_id"] for job in jobs}), 24)
            self.assertEqual(sum(job["order"] == 0 and job["condition"] == "benign"
                                 for job in jobs), 6)
            for pair in range(12):
                group = jobs[2*pair:2*pair+2]
                self.assertEqual({job["condition"] for job in group}, {"benign", "attack"})
                self.assertEqual([job["order"] for job in group], [0, 1])
                self.assertEqual([job["pair"] for job in group], [pair, pair])
        with self.assertRaises(ValueError):
            attack.schedule("A12")

    def test_csv_and_decision_depend_on_values(self):
        row = {key: value for key, value in zip(
            ("run_id", "pair", "order", "condition", "seed", "genesis_sha256",
             "valid_committed", "invalid_rejected", "fixed_timely_count", "fixed_eligible_count",
             "fixed_timely_fraction", "invalid_mean_checktx_service_ns", "qc_status"),
            ("A01-p00-benign", 0, 0, "benign", 2026111300, "a"*64,
             241, 0, 72, 72, 1.0, None, "PASS"))}
        raw = attack.csv_text([row])
        parsed = list(csv.DictReader(io.StringIO(raw)))
        self.assertEqual(parsed[0]["fixed_timely_count"], "72")
        changed = dict(row, fixed_timely_count=71)
        self.assertNotEqual(raw, attack.csv_text([changed]))
        text = attack.decision_text({"case_id": "A01", "status": "PASS", "n_valid": 24},
                                    {"decision": "INCONCLUSIVE"})
        self.assertIn("INCONCLUSIVE", text)
        self.assertIn("no DoS", text)

    def test_runtime_plan_binds_endpoints_and_rejects_public_target(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            network = root / "network"
            run = root / "run"
            run.mkdir()
            for index in range(4):
                path = network / f"validator-{index}" / "config" / "config.toml"
                path.parent.mkdir(parents=True)
                path.write_text(f"validator = {index}\n")
            config = (attack.ROOT / "research/config/s06_pilot.json").read_text()
            (run / "config.json").write_text(config)
            (run / "genesis.bin").write_bytes(b"test-only")
            source = attack.r01.git("rev-parse", "HEAD")
            tree = attack.r01.git("rev-parse", "HEAD^{tree}")
            data = {"run_id": "A01-p00-attack", "profile": 4, "events": [{}]*241,
                    "warmup_ns": 1_000_000_000, "steady_ns": 12_000_000_000,
                    "drain_ns": 5_000_000_000, "seed": 2026111300,
                    "regime": "fixed_legitimate_plus_adversarial",
                    "genesis_sha256": hashlib.sha256(b"test-only").hexdigest()}
            (run / "plan.json").write_text(json.dumps(data))
            meta = {"case_id": "A01", "condition": "attack", "scope": "R_CONFIRMATORY",
                    "run_id": data["run_id"], "config_sha256": hashlib.sha256(config.encode()).hexdigest(),
                    "seed": data["seed"], "source_commit": source, "source_tree": tree,
                    "order": 0, "pair": 0}
            (run / "metadata.json").write_text(json.dumps(meta))
            nodes = [{"rpc": f"http://127.0.0.1:{26657+index}"} for index in range(4)]
            (run / "nodes.json").write_text(json.dumps(nodes))
            attack.bind_run_plan(run, network)
            bound = json.loads((run / "research_plan/plan.json").read_text())
            self.assertEqual(bound["planned"], {"valid_tx_s": 6, "invalid_tx_s": 14,
                                                  "total_bytes_s": 51360})
            self.assertEqual(bound["network_config"]["rpc_endpoints"], [n["rpc"] for n in nodes])
            (run / "research_plan/plan.json").unlink()
            nodes[0]["rpc"] = "http://example.com:26657"
            (run / "nodes.json").write_text(json.dumps(nodes))
            with self.assertRaises(ValueError):
                attack.bind_run_plan(run, network)


if __name__ == "__main__":
    unittest.main()
