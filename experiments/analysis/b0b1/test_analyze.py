import unittest
import os
import subprocess
import sys
import tempfile
from pathlib import Path

import analyze
import analyze_micro
import run_matrix


class AnalysisTests(unittest.TestCase):
    def test_small_sample_has_no_p99(self):
        result = analyze.summary([1, 2, 3, 4, 5])
        self.assertEqual(result["p50"], 3)
        self.assertEqual(result["p99"], "INSUFFICIENT SAMPLES")

    def test_bootstrap_uses_run_values(self):
        self.assertIsNone(analyze.bootstrap_mean_ci([1, 2]))
        self.assertEqual(analyze.bootstrap_mean_ci([7, 7, 7]), [7, 7])

    def test_paired_comparison_needs_matching_seed(self):
        base = dict(invalid_percent=50, rate_tps=8, seed=9,
                    admission_us={"p50": 100}, valid_committed_tps=2)
        b0 = dict(base, profile="b0", run_id="b0")
        b1 = dict(base, profile="b1", run_id="b1",
                  admission_us={"p50": 150}, valid_committed_tps=1)
        pairs = analyze.paired([b0, b1])
        self.assertEqual(len(pairs), 1)
        self.assertEqual(pairs[0]["median_admission_us_difference"], 50)
        self.assertEqual(pairs[0]["valid_committed_tps_difference"], -1)
        self.assertEqual(analyze.paired([b0, dict(b1, seed=10)]), [])

    def test_micro_parser_rejects_missing_measurements(self):
        with self.assertRaises(ValueError):
            analyze_micro.parse("PASS\n")
        groups=analyze_micro.parse("BenchmarkPhase05Crypto/b1_ml_verify-20 100 44.1 ns/op 23034 B/op 7 allocs/op\n")
        self.assertEqual(groups["BenchmarkPhase05Crypto/b1_ml_verify"][0]["iterations"],100)
        interleaved=analyze_micro.parse("BenchmarkPhase05FinalizeCommit/b0_Commit-20 2026/10/26 WAL replay\n10 123 ns/op 4 B/op 1 allocs/op\n")
        self.assertEqual(interleaved["BenchmarkPhase05FinalizeCommit/b0_Commit"][0]["ns_per_op"],123)

    def test_process_cpu_split_and_legacy_records(self):
        rate = os.sysconf("SC_CLK_TCK")
        first = dict(unix_nano=1, validator=0, role="plutod", pid=7,
                     cpu_ticks=50, user_cpu_ticks=30,
                     system_cpu_ticks=20, rss_bytes=1024)
        last = dict(first, unix_nano=1_000_000_001, cpu_ticks=80,
                    user_cpu_ticks=50, system_cpu_ticks=30, rss_bytes=2048)
        record = dict(measurement_start_unix_nano=1,
                      measurement_end_unix_nano=1_000_000_001,
                      process_samples=[first, last])
        split = analyze.process_metrics(record)
        self.assertEqual(split["user_cpu_seconds"], 20 / rate)
        self.assertEqual(split["system_cpu_seconds"], 10 / rate)
        self.assertEqual(split["cpu_seconds"], 30 / rate)
        legacy = dict(record, process_samples=[
            {key: value for key, value in sample.items()
             if key not in ("user_cpu_ticks", "system_cpu_ticks")}
            for sample in (first, last)])
        old = analyze.process_metrics(legacy)
        self.assertIsNone(old["user_cpu_seconds"])
        self.assertIsNone(old["system_cpu_seconds"])

    def test_timeout_stops_only_owned_process_tree(self):
        sentinel = subprocess.Popen(["sleep", "30"])
        try:
            with tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                script = ("import subprocess,sys,time; from pathlib import Path; "
                          "child=subprocess.Popen(['sleep','30']); "
                          "Path(sys.argv[1]).write_text(str(child.pid)); time.sleep(30)")
                with (root / "bounded.log").open("wb") as log:
                    code, stopped = run_matrix.run_bounded(
                        [sys.executable, "-c", script, str(root / "child.pid")],
                        log, os.environ.copy(), 1)
                child = int((root / "child.pid").read_text())
                self.assertEqual(code, 124)
                self.assertIn(child, stopped)
                self.assertIsNone(sentinel.poll(), "unrelated process was stopped")
                try:
                    state = Path(f"/proc/{child}/stat").read_text().split(") ", 1)[1][0]
                except FileNotFoundError:
                    state = "Z"
                self.assertEqual(state, "Z", "owned child is still running")
        finally:
            sentinel.terminate()
            sentinel.wait(timeout=5)


if __name__ == "__main__":
    unittest.main()
