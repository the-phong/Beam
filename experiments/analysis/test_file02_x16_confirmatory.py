import json
from pathlib import Path
import tempfile
import unittest

import file02_common as common
import file02_x16_confirmatory as x16


class X16LockTest(unittest.TestCase):
    def test_balanced_deterministic_order(self):
        schedule = x16.schedule()
        self.assertEqual(schedule, x16.schedule())
        self.assertEqual(len(schedule["jobs"]), 48)
        self.assertEqual(schedule["permutations"].count("BI"), 12)
        self.assertEqual(schedule["permutations"].count("IB"), 12)

    def test_rejects_safety_and_schedule_drift(self):
        with tempfile.TemporaryDirectory() as root:
            session = Path(root)
            raw = session / "raw"
            raw.mkdir()
            (raw / "schedule.json").write_text(json.dumps(x16.schedule()) + "\n")
            (raw / "config.json").write_text("{}\n")
            plan = {"schema": "pluto.file02.x16_plan.v1", "status": "LOCKED",
                    "case_id": "X16", "n_blocks": 24, "n_networks": 48,
                    "arms": ["B", "I"], "wire_bytes": 2568, "steady_seconds": 12,
                    "source_commit": "a", "source_tree": "b",
                    "topology": "four_validator_single_physical_host_loopback_RPC_validator0",
                    "safety_caps": [25, 65536, 16, 60], "stage_cpu_mode": "ON",
                    "trace_mode": "ON",
                    "ci_method": "paired_block_percentile_bootstrap_20000_seed_20261116",
                    "exclusion_policy": "failed_arm_excludes_block_no_replacement",
                    "schedule_sha256": common.digest(raw / "schedule.json"),
                    "config_sha256": common.digest(raw / "config.json")}
            x16.validate_plan(session, plan)
            plan["safety_caps"][0] = 26
            with self.assertRaisesRegex(ValueError, "outer plan drift"):
                x16.validate_plan(session, plan)
            plan["safety_caps"][0] = 25
            (raw / "schedule.json").write_text("{}\n")
            with self.assertRaisesRegex(ValueError, "schedule/config drift"):
                x16.validate_plan(session, plan)


if __name__ == "__main__":
    unittest.main()
