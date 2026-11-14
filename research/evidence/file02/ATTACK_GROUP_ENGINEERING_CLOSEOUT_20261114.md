# Checklist 02 section 5 — A01–A14 engineering closeout

**Engineering disposition:** all fourteen cases have been processed in
checklist order. Eleven are PASS within their explicitly bounded designs
(A01–A10, A14); A11 and A13 are PARTIAL; A12 is UNSUPPORTED. The three
incomplete boxes remain open. This report submits Gate ATTACK for
Controller/Owner review; it is **not** Owner acceptance or permission to
start Q/D, train a detector or make a DoS claim.

## Evidence by case

| Case | Disposition | Main evidence and limit |
|---|---|---|
| A01 | PASS | [Structural reject](A01_STRUCTURAL_INVALID_AUDIT.md), matched benign control; early reject verified. |
| A02 | PASS | [Binding reject](A02_BINDING_INVALID_AUDIT.md), capability/stage oracle. |
| A03 | PASS | [Bad ECDSA](A03_BAD_ECDSA_AUDIT.md), ML stage not reached. |
| A04 | PASS | [Bad ML-DSA](A04_BAD_MLDSA_AUDIT.md), late crypto stage reached; ~391.31 vs A01 ~7.25 µs/tx app wall service, a mechanism-cost contrast only. |
| A05 | PASS | [State-invalid after signatures](A05_STATE_INVALID_AUDIT.md), actor C and no invalid commit. |
| A06 | PASS | [Replay/duplicate](A06_REPLAY_DUPLICATE_AUDIT.md), 16 networks; valid duplicate cache absorbs 55/56 repeats per block, invalid repeated remains 56/56 app NEW. |
| A07 | PASS | [Valid flood](A07_VALID_FLOOD_AUDIT.md), 20/25 tx/s, 8 networks; no fixed-legitimate loss at 3 s. This is valid-load pressure, not invalid verification waste. |
| A08 | PASS | [Mixed attack](A08_MIXED_ATTACK_AUDIT.md), fixed early/late class ratio and matched benign controls. |
| A09 | PASS | [Matched rate analysis](A09_MATCHED_RATE_AUDIT.md), 72 matched pairs/144 arms from A01–A05/A08. Queue-feature overlap is a candidate only; FPR unavailable without a detector. |
| A10 | PASS | [Matched bytes analysis](A10_MATCHED_BYTES_AUDIT.md), same 72 pairs, 51,360 B/s; stage/admission differences persist after byte matching, without observed service effect to explain. |
| A11 | PARTIAL | [Matched resource disposition](A11_MATCHED_RESOURCE_DISPOSITION.md); no CPU-equivalent benign/attack pair in the available 20-tx/s stratum, so detector separability remains unanswered. |
| A12 | UNSUPPORTED | [Many-peer disposition](A12_MANY_PEER_UNSUPPORTED.md); accepted sender and telemetry do not support multi-peer P2P-origin injection. |
| A13 | PARTIAL | [Time dynamics](A13_TIME_DYNAMICS_AUDIT.md), 8 networks/QC PASS; no degradation at the locked 3-s endpoint, no accepted detector, so lead/false-alert and recovery-from-violation values are unavailable. |
| A14 | PASS | [One-validator process fault](A14_PROCESS_FAULT_AUDIT.md), 8 networks/QC PASS; local process pause/observation measured, with weak four-pair precision and shared-host limits. |

A01–A05 and A08 used twelve matched fixture blocks per case, 24 new
four-validator networks each, fixed six H legitimate tx/s and 20 total
tx/s/51,360 B/s in both arms. Each case's private raw was frozen and
independently verified; attack class/stage oracles passed. Across these six
cases, the preregistered 3,000-ms all-four first-observation endpoint was
72/72 in each arm, paired benign-minus-attack timely fraction 0 and 95%
paired-block bootstrap CI [0, 0]. This is a **negative result at that
endpoint and offered rate**, not proof that no service effect can occur in
the safe range. A07 and A13 likewise observed no fixed-legitimate loss at
their locked endpoints. The A14 local observation-lag effect (+87.53 ms,
95% CI [+0.33, +249.71] ms) is not a service-degradation or DoS result.

## Integrity, definitions and outstanding conditions

All new load remained on loopback, four validators on one physical host,
within the original S06 25 tx/s, 65,536 B/s, 16 outstanding and 60-s/run
caps. Historical raw, accepted protocol and safety configuration were not
edited. Source, locked plans, private immutable raw manifests, per-run
results, QC, aggregate and decisions are linked from each audit. A13's first
derived analysis failed on Python 3.10 nanosecond timestamp parsing; its
rejected QC remains in private `analysis_attempt_1/`. A parser-only commit
reanalyzed the **same frozen raw** and passed independent verify. No failed
experimental pair was replaced.

The A01–A05/A08 raw metadata retains the inherited `R_CONFIRMATORY` scope
string, although the case IDs, locked A plans and RQ1b audits identify the
actual A experiment. This is a metadata-label limitation; frozen raw was
not rewritten. A06/A07/A13/A14 use `A_CONFIRMATORY`.

The Checklist 02 Gate ATTACK rule permits a `service degradation observed`
label only with fixed legitimate matched controls and a reproducible CI.
No Section 5 run meets that positive claim at its locked endpoint. The
unobserved knee remains `NOT_OBSERVED_WITHIN_SAFE_RANGE`. There is no
accepted detection threshold or FPR estimate, and no direct P2P-origin,
many-peer, multi-host or hardware-isolated fault result. `CheckTx` socket
queue wait is one component, not the full transaction queue; first all-four
observation is an upper bound, not exact commit; short eventual counts are
not sustainable throughput.

**Gate recommendation:** accept the eleven bounded technical PASS cases
and the two PARTIAL/one UNSUPPORTED dispositions for review, while keeping
Gate ATTACK open for an unqualified completion claim. Closing it with a
broader scientific scope requires an Owner/Controller decision about A11
resource matching, A12 source/topology capability and A13 detector/SLO
definitions. This task does not open those new experiments or Q/D cases.

## Final verification and publication boundary

From `research/file02-a01-a14`, `make verify` exited 0 (`go test ./...`,
`go vet ./...`, `go mod verify`, build/version); `go test -race
./internal/experiment/load -count=1` exited 0; `go test -tags 'integration
fourval' ./test/integration -run '^$'` compiled all tagged tests and
exited 0. The actual A13 and A14 tagged integration executions each exited
0 with 8/8 new networks. `PYTHONPATH=research/tools python3 -m unittest
discover -s research/tools -p 'test_file02_*.py'` passed 49 tests.
The initial Python unittest invocation omitted `PYTHONPATH` and failed on
imports; it was corrected and did not affect experiment raw or results.
`python3 research/tools/file02_a13.py verify <session>` and the analogous
A14 command both passed frozen-raw integrity, provenance and derived-output
checks. `git diff --check` was clean; `git ls-files research/artifacts`
returned no tracked private artifact. No protected branch was modified.
