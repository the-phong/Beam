"""COMMON-07 records must be bound to measured wire bytes and capability."""

import copy
import unittest

import file02_common07 as audit


class RunRecordTest(unittest.TestCase):
    def setUp(self):
        self.meta = {"run_id": "r1", "profile": "C", "scope": "QA_ONLY",
                     "source_commit": "a"*40, "source_tree": "b"*40}
        self.plan = {"run_id": "r1", "regime": "equal_tx_s", "warmup_ns": 1_000_000_000,
                     "steady_ns": 12_000_000_000, "events": [
                         {"id": str(i), "capability": "C_registered_key_holder",
                          "traffic": "legitimate", "source": "rpc", "bytes_hex": "abcd",
                          "offset_ns": 1_000_000_000+i*50_000_000} for i in range(240)]}
        self.result = {"run_id": "r1", "events": [
            {"id": str(i), "traffic": "legitimate", "source": "rpc", "wire_bytes": 2,
             "actual_offset_ns": 1_000_000_000+i*50_000_000} for i in range(240)]}

    def test_rates_and_identity(self):
        row = audit.record(self.meta, self.plan, self.result)
        self.assertEqual((row["planned_tx_s"], row["achieved_tx_s"]), (20, 20))
        self.assertEqual((row["planned_bytes_s"], row["achieved_bytes_s"]), (40, 40))
        self.assertEqual((row["case_id"], row["actor_capability"]),
                         ("T06_BOUNDED_PRECISION_PILOT", "C"))

    def test_rejects_wire_or_capability_mismatch(self):
        bad = copy.deepcopy(self.result)
        bad["events"][0]["wire_bytes"] = 3
        with self.assertRaisesRegex(ValueError, "wire size mismatch"):
            audit.record(self.meta, self.plan, bad)
        bad_plan = copy.deepcopy(self.plan)
        bad_plan["events"][0]["capability"] = "A_arbitrary_bytes"
        with self.assertRaisesRegex(ValueError, "unsupported case/capability"):
            audit.record(self.meta, bad_plan, self.result)


if __name__ == "__main__":
    unittest.main()
