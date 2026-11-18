import unittest

import file02_x16_stage as x16


def events(mode, missing_cpu=False):
    rows = []
    for index in range(241):
        stages = ["structural", "binding", "digest", "secp", "ml"]
        if index < 73:
            stages += ["state", "state_copy"]
        spans = [{"stage": stage, "duration_ns": 100 + index} for stage in stages]
        if mode == "ON":
            for span in spans:
                span["cpu_ns"] = 50
            if missing_cpu and index == 0:
                del spans[0]["cpu_ns"]
        rows.append({"check_type": "new", "stage_cpu_status":
                    "CLOCK_THREAD_CPUTIME_ID_INSTRUMENTED" if mode == "ON" else
                    "UNAVAILABLE_PROFILER_OFF", "spans": spans})
    return rows


class X16StageIntegrityTest(unittest.TestCase):
    def test_rejects_missing_cpu(self):
        with self.assertRaisesRegex(ValueError, "missing stage CPU"):
            x16.stage_statistics(events("ON", missing_cpu=True), "ON")

    def test_rejects_fabricated_off_cpu(self):
        rows = events("OFF")
        rows[0]["spans"][0]["cpu_ns"] = 0
        with self.assertRaisesRegex(ValueError, "OFF mode fabricated"):
            x16.stage_statistics(rows, "OFF")

    def test_rejects_process_identity_drift(self):
        samples = [{"observer_process": {"pid": 5, "start_ticks": 1, "cpu_ns": 10}},
                   {"observer_process": {"pid": 5, "start_ticks": 2, "cpu_ns": 20}}]
        with self.assertRaisesRegex(ValueError, "process identity"):
            x16.process_delta(samples, "observer_process", 5)


if __name__ == "__main__":
    unittest.main()
