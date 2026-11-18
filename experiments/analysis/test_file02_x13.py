"""Unit checks for X13 locked scheduling and caps."""

import unittest

import file02_x13 as x13


class X13ScheduleTest(unittest.TestCase):
    def test_balanced_complete_order(self):
        schedule = x13.schedule()
        self.assertEqual(len(schedule["jobs"]), 36)
        counts = {}
        for block in range(12):
            jobs = schedule["jobs"][3*block:3*block+3]
            self.assertEqual({j["rate"] for j in jobs}, {20, 23, 25})
            self.assertEqual([j["order"] for j in jobs], [0, 1, 2])
            for rate in x13.RATES:
                self.assertLessEqual(rate*x13.WIRE, 65536)
            key = tuple(j["rate"] for j in jobs)
            counts[key] = counts.get(key, 0)+1
        self.assertEqual(len(counts), 6)
        self.assertEqual(set(counts.values()), {2})


if __name__ == "__main__":
    unittest.main()
