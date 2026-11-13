"""Every declared fault excludes the run; absent signals never imply PASS."""

import unittest

import file02_common08 as qc


class FaultDecisionTest(unittest.TestCase):
    def test_all_six_faults_exclude_with_reason(self):
        clear = {key: False for key in qc.FAULTS}
        self.assertEqual(qc.classify(clear)["status"], "USABLE_FOR_QA_ONLY")
        for fault in qc.FAULTS:
            signals = clear.copy()
            signals[fault] = True
            result = qc.classify(signals)
            self.assertEqual(result["status"], "INVALID_FOR_ANALYSIS")
            self.assertEqual(result["exclude_reasons"], [fault])

    def test_missing_signal_rejected(self):
        with self.assertRaisesRegex(ValueError, "incomplete fault signals"):
            qc.classify({key: False for key in qc.FAULTS if key != "monitoring_drops"})


if __name__ == "__main__":
    unittest.main()
