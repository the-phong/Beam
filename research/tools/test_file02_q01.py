"""Q01 pair membership and endpoint safeguards."""

import unittest

import file02_q01 as q


class PairGuardTest(unittest.TestCase):
    def pair(self):
        base = {"qc_status": "PASS", "genesis_sha256": "g",
                "fixed_hashes": ["h"], "pair": 0, "seed": 9,
                "windows": [{"fixed": 6, "background": 14, "bytes": 51360}]*12,
                "fixed_eligible_count": 72, "fixed_timely_count": 72}
        benign = dict(base, run_id="b")
        attack = dict(base, run_id="a")
        return {"benign": benign, "attack": attack}

    def test_matched_pair_is_one_unit(self):
        row = q.paired_row("A04", 0, self.pair(), "raw")
        self.assertEqual(row["eligible_per_arm"], 72)
        self.assertEqual(row["benign_minus_attack_fraction"], 0)

    def test_mismatched_legitimate_stream_is_rejected(self):
        pair = self.pair()
        pair["attack"]["fixed_hashes"] = ["other"]
        with self.assertRaises(ValueError):
            q.paired_row("A04", 0, pair, "raw")

    def test_missing_arm_or_failed_qc_is_rejected(self):
        pair = self.pair()
        with self.assertRaises(ValueError):
            q.paired_row("A04", 0, {"benign": pair["benign"]}, "raw")
        pair["attack"]["qc_status"] = "FAIL"
        with self.assertRaises(ValueError):
            q.paired_row("A04", 0, pair, "raw")

    def test_rate_drift_is_rejected(self):
        pair = self.pair()
        pair["attack"]["windows"] = [{"fixed": 5, "background": 15, "bytes": 51360}]*12
        with self.assertRaises(ValueError):
            q.paired_row("A04", 0, pair, "raw")


if __name__ == "__main__":
    unittest.main()
