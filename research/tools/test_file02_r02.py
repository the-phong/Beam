"""R02 schema and stage QC guards using synthetic records only."""

import copy
import unittest

import file02_r02 as r02


def fake_process(order):
    profiles = []
    for name in order:
        spans = [{"stage": stage, "duration_ns": 20} for stage in sorted(r02.STAGES[name])]
        profiles.append({"profile": name, "code": {"C": 1, "CP": 2, "P": 3, "H": 4}[name],
                         "genesis_sha256": "a" * 64,
                         "expected_count": r02.SAMPLES, "recorder_drops": 0,
                         "process_cpu_ns": 1000, "total_alloc_bytes": 1200,
                         "total_mallocs": 240, "elapsed_ns": 2000,
                         "samples": [{"slot": i + 1, "service_ns": 1000,
                                      "app_lock_wait_ns": 0, "wire_bytes": r02.WIRE[name],
                                      "spans": spans} for i in range(r02.SAMPLES)]})
    return {"schema": "pluto.file02.r02.raw.v1", "order": order, "profiles": profiles}


class R02ContractTest(unittest.TestCase):
    def test_platform_alias_is_explicit(self):
        self.assertEqual(r02.go_arch("x86_64"), "amd64")
        self.assertEqual(r02.go_arch("aarch64"), "arm64")

    def test_schedule_is_counterbalanced(self):
        jobs = r02.orders()
        self.assertEqual(len(jobs), 12)
        for position in range(4):
            self.assertEqual({p: sum(job[position] == p for job in jobs) for p in r02.PROFILES},
                             {p: 3 for p in r02.PROFILES})

    def test_stage_and_wire_qc_fail_closed(self):
        order = list(r02.PROFILES)
        raw = fake_process(order)
        rows, groups = r02.parse_process(raw, order)
        self.assertEqual(len(rows), 4)
        self.assertEqual(len(groups["H"]), 240)
        bad = copy.deepcopy(raw)
        bad["profiles"][3]["samples"][0]["wire_bytes"] += 1
        with self.assertRaises(ValueError):
            r02.parse_process(bad, order)
        bad = copy.deepcopy(raw)
        bad["profiles"][3]["samples"][0]["spans"].pop()
        with self.assertRaises(ValueError):
            r02.parse_process(bad, order)


if __name__ == "__main__":
    unittest.main()
