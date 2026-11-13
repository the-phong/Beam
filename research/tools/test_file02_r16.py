"""R16 locked schedule and paired inference regression."""

import unittest
import tempfile
from pathlib import Path

import file02_r16 as r16


class R16DesignTest(unittest.TestCase):
    def test_schedule_is_complete_and_balanced(self):
        jobs = r16.schedule()["jobs"]
        self.assertEqual(len(jobs), 48)
        self.assertEqual(len({item["run_id"] for item in jobs}), 48)
        for block in range(4):
            selected = [item for item in jobs if item["block"] == block]
            self.assertEqual([item["order"] for item in selected], list(range(12)))
            for profile in r16.PROFILES:
                self.assertEqual({item["arm"] for item in selected if item["profile"] == profile},
                                 set(r16.ARMS))
                self.assertEqual(len({item["seed"] for item in selected}), 1)

    def test_waveforms_hold_total_and_caps(self):
        for counts in r16.WINDOWS.values():
            self.assertEqual(sum(counts), 40)
            self.assertLessEqual(max(counts), 25)
            self.assertLessEqual(max(counts)*2568, 65536)

    def test_bootstrap_preserves_zero_and_sign(self):
        self.assertEqual(r16.bootstrap([0, 0, 0, 0]), [0, 0])
        self.assertEqual(r16.bootstrap([5, 5, 5, 5]), [5, 5])
        low, high = r16.bootstrap([-2, -2, -2, -2])
        self.assertEqual((low, high), (-2, -2))

    def test_csv_integrity_rejects_missing_and_changed_values(self):
        row = {key: 1 for key in r16.CSV_KEYS}
        row["run_id"] = "r16-test"
        row["recovery_upper_ms"] = None
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "result_per_run.csv"
            path.write_text(",".join(r16.CSV_KEYS) + "\n" +
                            ",".join("" if row[k] is None else str(row[k]) for k in r16.CSV_KEYS) + "\n")
            r16.validate_csv(path, [row])
            path.write_text(path.read_text().replace("r16-test", "wrong-id"))
            with self.assertRaisesRegex(ValueError, "content mismatch"):
                r16.validate_csv(path, [row])
            path.write_text("run_id,block\n")
            with self.assertRaisesRegex(ValueError, "schema mismatch"):
                r16.validate_csv(path, [row])


if __name__ == "__main__":
    unittest.main()
