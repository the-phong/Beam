import json
from pathlib import Path
import tempfile
import unittest

import file02_common02 as audit


class Common02Test(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.run = Path(self.tmp.name)
        (self.run / "metadata.json").write_text(json.dumps({"run_id": "qa-C", "profile": "C"}))
        (self.run / "plan.json").write_text(json.dumps({
            "run_id": "qa-C", "warmup_ns": 1_000_000_000, "steady_ns": 4_000_000_000,
            "drain_ns": 3_000_000_000,
            "events": [{"id": "w", "offset_ns": 200_000_000},
                       {"id": "s", "offset_ns": 2_000_000_000}],
        }))
        (self.run / "result.json").write_text(json.dumps({
            "run_id": "qa-C", "actual_send_count": 2,
            "events": [{"id": "w", "planned_offset_ns": 200_000_000, "actual_offset_ns": 201_000_000},
                       {"id": "s", "planned_offset_ns": 2_000_000_000, "actual_offset_ns": 2_001_000_000}],
        }))
        (self.run / "cooldown.json").write_text(json.dumps({
            "schema": "pluto.file02.cooldown_qa.v1", "run_id": "qa-C", "planned_ms": 1000,
            "elapsed_ns": 1_001_000_000, "start_monotonic_ns": 10,
            "end_monotonic_ns": 1_001_000_010, "network_live": True,
        }))
        (self.run / "phases.jsonl").write_text("".join(json.dumps({
            "phase": name, "elapsed_monotonic_ns": stamp,
        }) + "\n" for name, stamp in zip(audit.PHASES, (1, 2, 3, 4, 5, 6, 1_001_000_020, 0, 0))))
        for index in range(4):
            path = self.run / "network" / f"validator-{index}" / "config"
            path.mkdir(parents=True)
            (path / "config.toml").write_text("qa\n")

    def test_disjoint_monotonic_phases_pass(self):
        result = audit.validate_profile(self.run)
        self.assertEqual(result["network_home_count"], 4)
        self.assertEqual((result["warmup_actual_sends"], result["steady_actual_sends"]), (1, 1))

    def test_send_crossing_boundary_fails(self):
        path = self.run / "result.json"
        result = json.loads(path.read_text())
        result["events"][0]["actual_offset_ns"] = 1_000_000_001
        path.write_text(json.dumps(result))
        with self.assertRaisesRegex(ValueError, "crossed warmup/steady"):
            audit.validate_profile(self.run)

    def test_short_cooldown_fails(self):
        path = self.run / "cooldown.json"
        result = json.loads(path.read_text())
        result["elapsed_ns"] = 999_000_000
        path.write_text(json.dumps(result))
        with self.assertRaisesRegex(ValueError, "cooldown duration"):
            audit.validate_profile(self.run)


if __name__ == "__main__":
    unittest.main()
