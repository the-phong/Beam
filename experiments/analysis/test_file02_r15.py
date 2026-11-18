"""Decision-rule regression for the prospective R15 time alignment."""

import unittest

import file02_r15 as r15


class R15DecisionTest(unittest.TestCase):
    def rows(self, fraction_by_rate):
        return [{"profile": profile, "rate_tx_s": rate,
                 "aligned_timely_fraction": fraction_by_rate[rate],
                 "inclusion_upper_bound_p95_ms": 1800,
                 "checktx_socket_queue_wait_mean_ms": .05,
                 "eventual_goodput_tx_s": rate,
                 "max_accepted_mempool_bytes": 10000,
                 "comet_cpu_seconds": 4.5}
                for profile in r15.PROFILES for rate in r15.RATES for _ in range(3)]

    def test_schedule_unique_and_bounded(self):
        jobs = r15.jobs()
        self.assertEqual(len(jobs), 36)
        self.assertEqual(len({job["run_id"] for job in jobs}), 36)
        self.assertEqual({job["rate"] for job in jobs}, {20, 23, 25})

    def test_negative_with_aligned_goodput(self):
        decisions = r15.decide(self.rows({20: 1, 23: 1, 25: 1}))
        self.assertTrue(all(d["status"] == "NOT_OBSERVED_WITHIN_SAFE_RANGE" for d in decisions.values()))
        self.assertEqual(decisions["C"]["levels"]["25"]["ci95_mean_p95_upper_ms"], [1800, 1800])

    def test_detects_two_adjacent_rate_degradation(self):
        decisions = r15.decide(self.rows({20: 1, 23: .85, 25: .8}))
        self.assertTrue(all(d["status"] == "KNEE_BRACKET_20_TO_23_OBSERVED" for d in decisions.values()))

    def test_reference_failure_is_not_a_knee(self):
        decisions = r15.decide(self.rows({20: .8, 23: .7, 25: .6}))
        self.assertTrue(all(d["status"] == "NOT_IDENTIFIABLE_REFERENCE" for d in decisions.values()))


if __name__ == "__main__":
    unittest.main()
