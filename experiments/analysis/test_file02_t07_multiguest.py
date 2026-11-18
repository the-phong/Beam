"""Causal clock bounds and failure checks for T07 multi-guest survey."""

import unittest

import file02_t07_multiguest as multi


def probe(remote_shift=0):
    return {"controller_before_wall_ns": 1_000_000,
            "controller_after_wall_ns": 1_004_000,
            "controller_before_mono_ns": 2_000_000,
            "controller_after_mono_ns": 2_004_000,
            "remote_before_ns": 1_001_000+remote_shift,
            "remote_after_ns": 1_001_100+remote_shift}


class MultiGuestBoundsTests(unittest.TestCase):
    def test_causal_bounds_and_cross_guest_interval(self):
        a = multi.probe_interval(probe())
        b = multi.probe_interval(probe(5_000))
        self.assertEqual((a["lower_ns"], a["upper_ns"]), (-3_000, 1_100))
        self.assertEqual(multi.cross_guest(
            [a["lower_ns"], a["upper_ns"]],
            [b["lower_ns"], b["upper_ns"]]), [900, 9_100])
        self.assertEqual(multi.boundary([probe()]*multi.PROBES_PER_BOUNDARY)[
            "narrowest_interval_width_ns"], 4_100)

    def test_missing_probe_and_clock_reversal_fail(self):
        with self.assertRaisesRegex(ValueError, "missing offset probes"):
            multi.boundary([probe()])
        bad = probe()
        bad["remote_after_ns"] = bad["remote_before_ns"]-1
        with self.assertRaisesRegex(ValueError, "malformed round-trip probe"):
            multi.probe_interval(bad)


if __name__ == "__main__":
    unittest.main()
