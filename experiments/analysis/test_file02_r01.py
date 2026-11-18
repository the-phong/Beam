"""R01 parser and inference guards; synthetic fixtures are never study data."""

import unittest

import file02_r01 as r01


class R01ContractTest(unittest.TestCase):
    def test_parser_requires_both_ordered_positive_cases(self):
        text = ("BenchmarkFile02R01/secp256k1-1  1000  13000 ns/op  0 B/op  0 allocs/op\n"
                "BenchmarkFile02R01/mldsa44-1  1000  95000 ns/op  24 B/op  1 allocs/op\n")
        parsed = r01.parse_benchmark(text, "secp_first")
        self.assertEqual(parsed["mldsa44"]["B_per_op"], 24)
        self.assertEqual(r01.parse_benchmark(text.replace("-1 ", " "), "secp_first"), parsed)
        with self.assertRaises(ValueError):
            r01.parse_benchmark(text, "ml_first")
        with self.assertRaises(ValueError):
            r01.parse_benchmark(text.splitlines()[0], "secp_first")
        with self.assertRaises(ValueError):
            r01.parse_benchmark(text.replace("13000 ns/op", "0 ns/op"), "secp_first")

    def test_paired_ci_uses_process_count_and_fixed_seed(self):
        values = list(range(1, 13))
        self.assertEqual(r01.paired_ci(values), r01.paired_ci(values))
        with self.assertRaises(ValueError):
            r01.paired_ci(values[:-1])
        self.assertEqual(r01.orders().count("secp_first"), 6)
        self.assertEqual(r01.orders().count("ml_first"), 6)


if __name__ == "__main__":
    unittest.main()
