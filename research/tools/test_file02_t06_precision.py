"""Check counterbalanced replay and precision gate behavior before pilot."""

import unittest

import file02_t06_precision as pilot


class T06PrecisionTest(unittest.TestCase):
    def test_schedule_has_three_of_each_profile_at_each_position(self):
        plan = pilot.schedule(20261112)
        pilot.validate_schedule(plan)
        plan["jobs"][0]["seed"] += 1
        with self.assertRaisesRegex(ValueError, "schedule replay"):
            pilot.validate_schedule(plan)

    def test_cluster_gate_suppresses_small_or_unstable_inputs(self):
        with self.assertRaisesRegex(ValueError, "twelve complete"):
            pilot.cluster_bootstrap([[1.0]*9 for _ in range(12)], 1)
        stable = pilot.cluster_bootstrap([[100.0]*240 for _ in range(12)], 1)
        self.assertEqual(stable["status"], "PRECISION_PASS")
        self.assertEqual(stable["n_transaction_samples"], 2880)

    def test_go_timestamp_with_five_or_nine_fractional_digits(self):
        five = pilot.parse_go_rfc3339("2026-11-12T14:39:39.89815Z")
        nine = pilot.parse_go_rfc3339("2026-11-12T14:39:39.898150000Z")
        self.assertEqual(five, nine)


if __name__ == "__main__":
    unittest.main()
