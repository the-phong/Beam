import unittest

import file02_q06 as q06


class Q06Test(unittest.TestCase):
    def test_stability_and_balance(self):
        result = q06.stability([0, 0, 0, 0], ["benign", "attack", "benign", "attack"])
        self.assertEqual(result["mean_effect_fraction"], 0)
        self.assertEqual(result["leave_one_pair_out_mean_range_fraction"], [0, 0])
        with self.assertRaises(ValueError):
            q06.stability([0]*4, ["benign"]*4)


if __name__ == "__main__":
    unittest.main()
