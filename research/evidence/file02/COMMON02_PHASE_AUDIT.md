# COMMON-02 — fresh network and monotonic phase audit, 2026-11-12

Source commit `76d9507ab7639939f547fb5603c3c5cc08e024a3`, on
`research/file02-common-foundation`. This is a bounded single-host QA test,
not a confirmatory File 02 experiment.

| Command | Exit | Result |
| --- | ---: | --- |
| `make verify` | 0 | Go build/unit/vet/format/module checks passed after the optional cooldown instrumentation. |
| `make test-file02-common` | 0 | Fourteen focused tests, including late send crossing warm-up/steady and short cooldown refusal. |
| `make test-s07-schema` | 0 | Three historical integrity tests remain passing. |
| `make test-file02-common02` | 0 | Four new C/CP/P/H networks; zero S07 exclusions and four COMMON-02 audit rows. |

S07 QA session `s07-20261112T131539Z-1bc803c4` has frozen manifest
SHA-256 `b562f5b87f901ee85aa3b1ef26f7be6c82a89f4d395a2b1dab158881d39c0507`.
The separate ignored COMMON-02 audit has manifest SHA-256
`d91261975746bd6aed9b28c1844f686593e285676870a4f5a7bf04214d83c958`.
The audit reran the S07 raw hash/oracle verifier, checked four distinct run
IDs and four validator home directories per run, and validated the
generator's monotonic actual-send offsets against the frozen warm-up and
steady boundaries. Sends in drain or cooldown, a send crossing the
warm-up/steady boundary, missing phase records, nonmonotonic phase markers,
short cooldown, or a dead validator during cooldown fail the audit.

| Profile | Warm-up sends | Steady sends | Drain sends | Cooldown sends | Cooldown elapsed ns |
| --- | ---: | ---: | ---: | ---: | ---: |
| C | 1 | 8 | 0 | 0 | 1,002,295,651 |
| CP | 1 | 8 | 0 | 0 | 1,002,788,678 |
| P | 1 | 8 | 0 | 0 | 1,001,892,979 |
| H | 1 | 8 | 0 | 0 | 1,002,571,228 |

The S07 harness creates each private run and validator home with exclusive
directory creation and a new genesis/validator key set. The optional File 02
QA mode adds a real one-second cooldown while the network is still live;
ordinary S07 smoke keeps its accepted schedule. The process-local phase
markers and generator offsets are monotonic. UTC timestamps remain only for
correlation. This supports COMMON-02 for the available single-host QA path.
It does not establish multi-host clock uncertainty, independent machines,
precision, capacity or any RQ result. Raw homes and keys remain private.
