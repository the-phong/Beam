import unittest

import file02_q04 as q04


class Q04Test(unittest.TestCase):
    def test_component_and_windows(self):
        self.assertEqual(q04.component({"scenario": "fixed_legitimate"}), "fixed")
        self.assertEqual(q04.component({"scenario": "benign_background"}), "background")
        windows = q04.counts([(200_000_000, 100, "fixed"),
                              (1_100_000_000, 100, "fixed"),
                              (1_200_000_000, 300, "background")])
        self.assertEqual(windows[0]["total"], {"tx_s": 2, "bytes_s": 400})
        self.assertEqual(windows[0]["fixed"], {"tx_s": 1, "bytes_s": 100})
        self.assertEqual(windows[1]["total"], {"tx_s": 0, "bytes_s": 0})

    def test_rejects_spill_out_of_steady(self):
        with self.assertRaises(ValueError):
            q04.counts([(13_000_000_000, 100, "fixed")])


if __name__ == "__main__":
    unittest.main()
