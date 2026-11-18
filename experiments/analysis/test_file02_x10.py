import sys
from pathlib import Path
import unittest

sys.path.insert(0, str(Path(__file__).parent))
import file02_x10 as x10


class X10DecisionTest(unittest.TestCase):
    def test_positive_but_inside_registered_margin(self):
        self.assertEqual(x10.decide_mean_effect([39.5, 81.6], True),
                         "NO_RESEARCH_RELEVANT_MEAN_EFFECT_WITHIN_REGISTERED_MARGIN")

    def test_positive_crossing_margin_is_inconclusive_size(self):
        self.assertEqual(x10.decide_mean_effect([80, 120], True),
                         "POSITIVE_DIRECTION_PRACTICAL_SIZE_INCONCLUSIVE")

    def test_relevant_effect_requires_lower_above_margin(self):
        self.assertEqual(x10.decide_mean_effect([101, 130], True),
                         "RESEARCH_RELEVANT_LATENCY_EFFECT")

    def test_precision_and_binary_conflict(self):
        self.assertEqual(x10.decide_mean_effect([30, 120], True),
                         "INCONCLUSIVE_PRECISION_TARGET_MISSED")
        self.assertEqual(x10.decide_mean_effect([-20, 60], False),
                         "INCONCLUSIVE_CONTINUOUS_EFFECT")

    def test_balanced_order(self):
        self.assertEqual(sorted(x10.schedule()["permutations"]),
                         sorted(list(x10.PERMUTATIONS) * 4))


if __name__ == "__main__":
    unittest.main()
