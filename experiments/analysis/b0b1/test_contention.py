import unittest

import analyze_contention


class ContentionAnalysisTests(unittest.TestCase):
    def record(self, count):
        events = []
        for index in range(count):
            events.append(dict(expected_code=0, code=0, case_id=f"V{index:03d}",
                               admission_latency_ns=1000, inclusion_latency_ns=1_000_000,
                               included_height=10, submit_unix_nano=index*666_666_667,
                               bytes=100, error=""))
        return dict(complete=True, all_validators_matched=True,
                    condition=dict(run_id="test", profile="b0", valid_rate_tps=1.5,
                                   invalid_rate_tps=0, valid_count=count, invalid_count=0, seed=1),
                    events=events, valid_submitted=count, valid_included=count,
                    invalid_rejected=0, measurement_start_unix_nano=1,
                    measurement_end_unix_nano=200_000_000_001,
                    process_samples=[], block_hashes=["a"] * 4,
                    app_hashes=["b"] * 4, ordered_tx_digests=["c"] * 4,
                    total_supplies=[6401] * 4, source_commit="c", source_tree="t",
                    corpus_sha256="x", genesis_sha256="g", matched_height=10)

    def test_p99_requires_300_valid_confirmed_in_one_run(self):
        meta = dict(run_id="test", series="primary", repetition=1, ratio=0)
        short = analyze_contention.one(self.record(299), meta)
        self.assertFalse(short["sample_sufficient_for_descriptive_p99"])
        self.assertEqual(short["inclusion_ms"]["p99"], "INSUFFICIENT SAMPLES")
        enough = analyze_contention.one(self.record(300), meta)
        self.assertTrue(enough["sample_sufficient_for_descriptive_p99"])
        self.assertEqual(enough["inclusion_ms"]["p99"], 1)

    def test_matched_normalization_uses_own_baseline(self):
        def row(ratio, p99, goodput):
            return dict(profile="b1", series="primary", repetition=2, ratio=ratio,
                        run_id=str(ratio), inclusion_ms={"p99": p99},
                        legitimate_goodput_tps=goodput,
                        process={"cpu_percent": 12, "peak_sampled_rss_bytes": 100})
        values = analyze_contention.normalized([row(0, 100, 1.5), row(90, 200, 1.2)])
        self.assertEqual(values[1]["inclusion_p99_amplification"], 2)
        self.assertAlmostEqual(values[1]["goodput_degradation_fraction"], .2)

    def test_wall_clock_jump_is_excluded_and_monotonic_duration_is_used(self):
        log = "--- PASS: TestPhase05Contention (83.85s)\n"
        self.assertTrue(analyze_contention.clock_discontinuity(
            {"wall_seconds": 2463.8}, log))
        self.assertFalse(analyze_contention.clock_discontinuity(
            {"wall_seconds": 87.5}, log))
        record = self.record(300)
        record["measurement_end_unix_nano"] += 2_300_000_000_000
        record["measurement_duration_ns"] = 200_000_000_000
        for i, event in enumerate(record["events"]):
            event["submit_offset_ns"] = i * 666_666_667
        row = analyze_contention.one(record, dict(run_id="test", series="primary",
                                                  repetition=1, ratio=0))
        self.assertEqual(row["duration_seconds"], 200)
        self.assertAlmostEqual(row["achieved_valid_tps"], 1.5, places=4)


if __name__ == "__main__":
    unittest.main()
