"""Integrity failures must be visible before a run can be analyzed."""

import json
from pathlib import Path
import tempfile
import unittest

import s07_orchestrate as s07


class S07IntegrityTests(unittest.TestCase):
    def setUp(self):
        s07.ARTIFACTS.mkdir(parents=True, exist_ok=True)
        self.temp = tempfile.TemporaryDirectory(dir=s07.ARTIFACTS, prefix="test-s07-")
        self.addCleanup(self.temp.cleanup)
        self.session = Path(self.temp.name)
        (self.session / "config.json").write_bytes(s07.CONFIG.read_bytes())
        (self.session / "corpus.json").write_bytes(s07.CORPUS.read_bytes())
        s07.write_json(self.session / "session.json", {
            "schema": "pluto.s07.session.v1", "source_commit": "a" * 40,
            "config_sha256": s07.sha256(s07.CONFIG), "corpus_sha256": s07.sha256(s07.CORPUS),
        })

    def test_tampered_raw_is_rejected(self):
        s07.freeze(self.session, "a" * 40, [],
                   [{"profile": p, "status": "INVALID_FOR_ANALYSIS", "reason": "test failure"}
                    for p in s07.PROFILES])
        s07.verify(self.session)
        raw = self.session / "config.json"
        raw.chmod(0o600)
        raw.write_bytes(b"tampered")
        with self.assertRaisesRegex(ValueError, "modified frozen file"):
            s07.verify(self.session)

    def test_existing_run_file_cannot_be_reused(self):
        target = self.session / "result.json"
        s07.create_exclusive(target, b"first")
        with self.assertRaises(FileExistsError):
            s07.create_exclusive(target, b"second")
        self.assertEqual(target.read_bytes(), b"first")

    def test_invalid_telemetry_is_rejected(self):
        path = self.session / "observer-0.jsonl"
        path.write_text(json.dumps({"schema": "pluto.telemetry.v1", "kind": "node_sample",
                                    "dropped_samples": 1, "observed_utc": s07.utc_now()}) + "\n")
        with self.assertRaisesRegex(ValueError, "incomplete observer"):
            s07.sample_stream(path)


if __name__ == "__main__":
    unittest.main()
