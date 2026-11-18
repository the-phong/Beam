import json
from pathlib import Path
import tempfile
import unittest

import file02_x07_trace as x07


class X07TraceIntegrityTest(unittest.TestCase):
    def test_rejects_missing_marker(self):
        with self.assertRaisesRegex(ValueError, "missing socket_dequeue"):
            x07.first_stage([{"event": "socket_queue_attempt", "monotonic_ns": 1}],
                            "socket_dequeue")

    def test_rejects_partial_file_and_bad_sequence(self):
        with tempfile.TemporaryDirectory() as root:
            path = Path(root) / "trace.jsonl"
            row = {"schema": x07.SCHEMA, "event": "rpc_sync_entry",
                   "tx_sha256": "a" * 64, "wire_bytes": 2568, "code": -1,
                   "pid": 123, "sequence": 2, "monotonic_ns": 1,
                   "host_monotonic_ns": 10, "utc": "2026-11-16T08:22:40.504593925Z"}
            path.write_text(json.dumps(row))
            with self.assertRaisesRegex(ValueError, "partial/empty"):
                x07.trace_by_tx(path, 123)
            path.write_text(json.dumps(row) + "\n")
            with self.assertRaisesRegex(ValueError, "trace schema/order/PID"):
                x07.trace_by_tx(path, 123)

    def test_nanosecond_utc_and_bad_clock(self):
        self.assertGreater(x07.utc_ns("2026-11-16T08:22:40.504593925Z"), 0)
        with self.assertRaisesRegex(ValueError, "invalid UTC"):
            x07.utc_ns("2026-11-16T08:22:40")


if __name__ == "__main__":
    unittest.main()
