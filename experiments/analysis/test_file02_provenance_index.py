import json
import tempfile
import unittest
from pathlib import Path

from file02_provenance_index import build_rows, sha256


class ProvenanceIndexTest(unittest.TestCase):
    def test_hashes_and_run_identity(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            session = root / "a01-fixture"
            (session / "raw").mkdir(parents=True)
            (session / "plan.json").write_text(json.dumps({"case_id": "A01", "source_commit": "abc"}))
            (session / "raw" / "manifest.json").write_text(json.dumps({"schema": "fixture"}))
            (session / "result_per_run.json").write_text(json.dumps({"rows": [{"run_id": "run-1", "qc_status": "PASS"}]}))
            qc = {
                "case_id": "A01", "status": "PASS",
                "plan_sha256": sha256(session / "plan.json"),
                "raw_manifest_sha256": sha256(session / "raw" / "manifest.json"),
            }
            (session / "qc_report.json").write_text(json.dumps(qc))
            rows = build_rows(root)
            self.assertEqual(len(rows), 1)
            self.assertEqual(rows[0]["run_id"], "run-1")
            self.assertEqual(rows[0]["provenance_status"], "QC_PLAN_AND_MANIFEST_HASH_MATCH")

            (session / "raw" / "manifest.json").write_text(json.dumps({"schema": "tampered"}))
            with self.assertRaisesRegex(ValueError, "QC raw manifest hash mismatch"):
                build_rows(root)


if __name__ == "__main__":
    unittest.main()
