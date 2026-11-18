import hashlib
import unittest

import file02_x07_queue_v2_results as x07


class X07QueueV2IntegrityTest(unittest.TestCase):
    def test_pinned_one_tx_envelope_encoding(self):
        # Message{Txs{0x00}} in Comet v0.38.26 protobuf wire format.
        self.assertEqual(x07.varint(128), b"\x80\x01")
        self.assertEqual(x07.mempool_envelope_sha256("00"),
                         hashlib.sha256(bytes.fromhex("0a030a0100")).hexdigest())

    def test_overhead_ci_requires_all_locked_blocks(self):
        with self.assertRaisesRegex(ValueError, "eight blocks"):
            x07.ci([1.0] * 7)
        self.assertEqual(x07.ci([2.0] * 8), [2.0, 2.0])


if __name__ == "__main__":
    unittest.main()
