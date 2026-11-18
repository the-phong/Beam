import hashlib
import importlib.util
import json
import unittest
from pathlib import Path

import file02_x16_results as x16


class X16ResultsIntegrityTest(unittest.TestCase):
    def test_counter_ignores_unavailable_start_but_rejects_reset(self):
        self.assertEqual(x16.counter_delta([{"v": None}, {"v": 7}, {"v": 10}], "v"), 3)
        with self.assertRaisesRegex(ValueError, "missing/reset"):
            x16.counter_delta([{"v": 10}, {"v": 7}], "v")

    def test_paired_ci_is_deterministic(self):
        values = [1.0, 2.0, 3.0, 4.0]
        first = x16.paired_ci(values)
        self.assertEqual(first, x16.paired_ci(values))
        self.assertLessEqual(first[0], 2.5)
        self.assertGreaterEqual(first[1], 2.5)

    def test_figures_render_only_analyzed_data(self):
        self.assertEqual(x16.figure_bytes({"status": "WITHHELD_QC_FAILURE"}), {})
        if importlib.util.find_spec("matplotlib") is None:
            self.skipTest("matplotlib is optional for figure rendering")
        stages = ("structural", "binding", "digest", "secp", "ml", "state", "state_copy")
        row = {"sample_index": 0, "accepted_mempool_bytes": 100,
               "app_cpu_cumulative_s": 0.1, "comet_cpu_cumulative_s": 0.2,
               "observer_cpu_cumulative_s": 0.01}
        stage = {name: {"mean": 1000, "ci95": [900, 1100]} for name in stages}
        aggregate = {"status": "ANALYZED", "validator0_resource_curves":
                     {"B": [row], "I": [row]},
                     "stage_thread_cpu_ns_by_arm": {"B": stage, "I": stage}}
        pictures = x16.figure_bytes(aggregate)
        self.assertEqual(set(pictures), {"resource_curve.png", "stage_thread_cpu.png"})
        self.assertTrue(all(raw.startswith(b"\x89PNG") for raw in pictures.values()))
        self.assertEqual(pictures, x16.figure_bytes(aggregate))

    def test_public_figure_manifest_integrity(self):
        evidence = Path(__file__).resolve().parents[2] / "research" / "evidence" / "file02" / "mechanism"
        manifest = json.loads((evidence / "X16_CONFIRMATORY_PUBLIC_MANIFEST.json").read_text())
        self.assertEqual(manifest["qc"], "PASS_48_RUNS_24_PAIRS")
        for name, expected_sha in manifest["files"].items():
            raw = (evidence / name).read_bytes()
            self.assertEqual(hashlib.sha256(raw).hexdigest(), expected_sha, name)
            if name.endswith(".png"):
                self.assertTrue(raw.startswith(b"\x89PNG\r\n\x1a\n"), name)


if __name__ == "__main__":
    unittest.main()
