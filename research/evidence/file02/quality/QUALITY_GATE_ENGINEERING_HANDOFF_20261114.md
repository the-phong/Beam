# Checklist 02 section 6 — Gate QUALITY engineering handoff

Date: 2026-11-14. **Engineering disposition complete for Q01–Q07; Gate QUALITY awaits Controller/Owner review.** Work is on `research/file02-q01-q07` from `test-mainet` merge commit `2b453b3c14c410545d5699e03cff90d0eb21d63e` after bounded Gate ATTACK Owner acceptance [D-043](../../../DECISIONS.md). The tested topology is four validators on one physical host, validator-0 loopback RPC. Safety caps, historical raw bytes, accepted RQ1 H−CP protocol and the distinct RQ1b question were not changed.

| Case | Disposition | Primary evidence and limit |
|---|---|---|
| Q01 | PASS; checked | [Matched-service audit](Q01_MATCHED_VALID_SERVICE_AUDIT.md), 76 A pairs/152 fresh networks, exact fixed legitimate stream and rate/byte controls within pairs. Every original case effect at the 3-s endpoint was 0 with [0,0] CI; ceiling-limited. |
| Q02 | PASS; checked | [Corrected order-balanced audit](Q02_ORDER_BALANCED_DECOMPOSITION_AUDIT.md), 96 new networks/12 blocks, six/six arm order per profile, raw/QC PASS. Structural-invalid H–C DiD 0 proportion points, 95% CI [0,0]. C/P cross-profile bytes differ; no ML-specific attribution. The first complete v1 session was withdrawn for order confounding, not concealed. |
| Q03 | PASS for measurement discipline; checked | [Valid-included p99 audit](Q03_VALID_INCLUDED_P99_AUDIT.md), 22 separated groups using current Q02 v2. All inferential p99/CI are `null`/`INSUFFICIENT_SAMPLES` under the preregistered rank gate; descriptive values are inclusion upper bounds. |
| Q04 | PASS; checked | [Rate-fidelity audit](Q04_RATE_FIDELITY_AUDIT.md), 248 networks/2,976 one-second windows using actual socket-write tx/s and bytes/s. Zero ±5% violations/exclusions; valid goodput was never substituted for offered load. |
| Q05 | PARTIAL; unchecked | [Bottleneck disposition](Q05_BOTTLENECK_DISPOSITION.md). R15 found no knee through unchanged 25 tx/s. Comet CPU, socket queue, sampled mempool, relay envelope and generator metrics exist; crypto CPU, full transaction queue and matched peer latency do not. No natural saturated bottleneck can be named. |
| Q06 | PARTIAL; unchecked | [Replication disposition](Q06_REPLICATION_DISPOSITION.md), seeded and arm-order-balanced A pairs plus a separate order-balanced Q02-H structural replication; CI/leave-one-out and all aborted/superseded sessions logged. No independent physical-host replication; zero binary endpoint is ceiling-limited. |
| Q07 | PASS; checked | [No-invalid-accepted audit](Q07_NO_INVALID_ACCEPTED_AUDIT.md), 248/248 runs, 38,600 valid exact-once, 20,496 invalid absent, full genesis replay with 482→482 supply and matching AppHash; live oracle checked four-validator block/state agreement. |

## RQ1b interpretation

For the preregistered **fraction of fixed legitimate transactions first observed included on all four validators within 3,000 ms of monotonic socket write**, the matched A01–A05/A08/A13 and corrected structural Q02 strata show no reproducible positive loss at the tested 20-tx/s on phase. The original A families each have a separate paired CI [0,0]; Q02's 12-block DiD is 0 [0,0]. This is a **bounded negative result at a ceiling-limited binary endpoint**, not an equivalence test or proof of no smaller timing change. A13 lacks an accepted detector and observed service onset. No practical minimum important effect or production SLO was preregistered, so this report does not claim universal “no material impact.” RQ1's accepted H−CP +81.47 µs/tx valid CheckTx effect remains separate.

The existing measurement cannot support a p99 latency difference, sustained capacity, saturation bottleneck, new knee, multi-host or P2P-origin conclusion. It also cannot establish DoS. Structural-invalid Q02 is a profile interaction diagnostic; invalid inputs stop before crypto. No RQ2/RQ3, D cases, ML training or scientific figures were produced. The public Q CSV/JSON files contain traceable case/run IDs and source raw manifest hashes; the private raw remains ignored. No untraceable result plot exists in this handoff. The Q02 private `figures/status.json` remains pending review rather than implying a scientific plot was made.

## Provenance corrections and required review

The first Q02 session stopped at an unchanged stale-observer safety cap before CP load (`INVALID_FOR_INFERENCE`). The first complete 96-network session passed technical QC, but Q06 exposed a `schedule.v1` parity bug: C/P always benign-first, CP/H always attack-first. Its cross-profile inference was withdrawn, Q02's checklist box was reopened, and a source-committed `schedule.v2` reran the entire 96-network plan with six/six order balance per profile. The original raw and output remain unchanged. Q03/Q04 initial analyses were kept as historical revisions; their current V2 outputs use corrected Q02 raw. Q07's first analyzer falsely excluded eight A13 runs by assuming every funded sender sent; its diagnostic was preserved and the final full-genesis replay fixed that assumption before Q07 was checked. These corrections are part of the scientific audit trail.

Controller/Owner may review the five checked bounded PASS dispositions and two open PARTIAL items. Closing Gate QUALITY without further experiments would require explicit acceptance that Q05 saturation attribution and Q06 physical-host replication are unavailable in the current safe/single-host scope, and that the RQ1b negative claim stays limited to the registered endpoint and rate. Any future host replication or saturation study needs its own prospective plan and authority. This engineering report is **not** Gate QUALITY acceptance or permission to begin Checklist 02 section 7.

## Verification

| Command | Exit | Result |
|---|---:|---|
| `PYTHONPATH=research/tools python3 -m unittest discover -s research/tools -p 'test_file02*.py'` | 0 | 64 File 02 Python tests passed, including Q02 order balance, Q03 precision, Q04 windows, Q06 balance and Q07 invalid-commit rejection. |
| `make verify` | 0 | Go build/tests (including Q07 replay corruption test), vet, formatting, module verification and version passed at source/result commit `c188023`. |
| `make test-s07-schema` | 0 | Three schema/integrity tests passed, including tampered raw rejection. |
| `go test -tags 'integration fourval' ./test/integration -run '^$' -count=1` | 0 | All four-validator integration tests compiled after Q07. |
| `python3 research/tools/file02_q02.py verify research/artifacts/file02/q02-20261114T044327Z-e7f236e2` | 0 | 96/96 corrected networks; raw manifest `58265e00a8c0fc08a37cb4e2935633d85304a1e70d32142df5d9eaa4d1180f3f`. |
| `file02_q01.py verify`, `file02_q03.py verify`, `file02_q04.py verify`, `file02_q06.py verify`, `file02_q07.py verify` | 0 each | Published Q outputs independently rederived from frozen source sessions; Q06 correctly reports PARTIAL host replication. |
| `git diff --check` | 0 | No whitespace errors. |

Historical R15's current verifier reports `R15 plan drift` because it compares its locked historical `go.mod` hash against today's module bytes. The locked hash matches `go.mod` at R15's source commit and `file02_common.verify_raw` passed; [Q05](Q05_BOTTLENECK_DISPOSITION.md) discloses this current-verifier limitation. It is not counted as a passing current R15 full verify.
