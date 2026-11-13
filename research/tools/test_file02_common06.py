"""Regression checks for counterbalanced scheduling and p99 precision guard."""

import unittest

import file02_common06 as c


class Common06Test(unittest.TestCase):
    def test_counterbalance_replays_and_rejects_changed_order(self):
        value = c.schedule(125711)
        c.validate_schedule(value)
        changed = c.schedule(125711)
        changed["jobs"][0]["profile"] = "H" if changed["jobs"][0]["profile"] != "H" else "C"
        with self.assertRaisesRegex(ValueError, "schedule replay"):
            c.validate_schedule(changed)

    def test_small_sample_cannot_support_p99_rank_claim(self):
        self.assertLess(c.rank_coverage(36), .95)
        self.assertGreaterEqual(c.rank_coverage(2000), .95)


if __name__ == "__main__":
    unittest.main()
