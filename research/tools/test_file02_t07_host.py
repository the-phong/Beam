"""Clock continuity checks for the T07 host survey."""

import unittest

import file02_t07_host as host


def samples(jump_ns=0):
    origin = 1_000_000_000_000
    return [{"index": i, "mono_before_ns": origin + 50_000_000 * i,
             "realtime_ns": origin + 50_000_000 * i + 1_000 + (jump_ns if i >= 100 else 0),
             "mono_after_ns": origin + 50_000_000 * i + 200}
            for i in range(host.SAMPLE_COUNT)]


class HostClockTests(unittest.TestCase):
    def test_continuous_clock_and_unavailable_throttle(self):
        result = host.summarize(samples(), None, None)
        self.assertTrue(result["clock_continuous_within_10ms"])
        self.assertEqual(result["same_host_mapping_uncertainty_ns"], 100)
        self.assertIsNone(result["throttle_count_delta"])
        self.assertIsNone(result["cross_host_offset_ns"])

    def test_wall_clock_step_fails(self):
        result = host.summarize(samples(11_000_000), 3, 4)
        self.assertFalse(result["clock_continuous_within_10ms"])
        self.assertEqual(result["throttle_count_delta"], 1)


if __name__ == "__main__":
    unittest.main()
