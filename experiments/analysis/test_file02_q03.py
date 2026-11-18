import unittest

import file02_q03 as q03


class Q03Test(unittest.TestCase):
    def test_rank_gate_keeps_inferential_p99_null(self):
        report = q03.precision([[float(i+1) for i in range(72)] for _ in range(12)], 72)
        self.assertEqual(report["status"], "INSUFFICIENT_SAMPLES")
        self.assertIsNone(report["inferential_p99_upper_ms"])
        self.assertIsNone(report["ci95_ms"])
        self.assertLess(report["rank_band_coverage_iid_optimistic"], .95)

    def test_rejects_missing_or_invalid_included_samples(self):
        with self.assertRaises(ValueError):
            q03.precision([[1.0, 2.0], [1.0]], 2)
        with self.assertRaises(ValueError):
            q03.nearest_rank([1.0, float("nan")])

    def test_nearest_rank(self):
        self.assertEqual(q03.nearest_rank(list(range(1, 101))), 99)


if __name__ == "__main__":
    unittest.main()
