# PLUTO PHASE 03D — FINAL BOUNDED CLOSURE REPORT

**M7 evidence rework, 2026-10-25:** The later [M7 evidence rework report](PHASE_03D_M7_EVIDENCE_REWORK_REPORT.md) adds explicit no-progress and missing-B0 assertions, a no-SKIP audit, separate raw callback markers, and the D-026 decision amendment. The source IDs and M7 matrix below identify the earlier bounded snapshot.

**2026-10-25. Technical verdict: PARTIAL; STOP for Controller/Owner decision.** The Controller passed the prior independent source review. This bounded pass adds M7 callback evidence and a D-026 **proposal only**. Phase 03D is not accepted; no D-026 code, Phase 03E, commit, push, PR or merge was performed.

## Source identity and changes

Branch `phase/03d-b0-abci-integration`; `HEAD` remains `b56c81ccce7ea4a0e7f52053c7459c4601c15b38`. The prior reviewed implementation tree is `87b504853cb0ff98790976879ea8832db92b83e0`; the prior staged closure tree was `183c54b1a083149563bb9cc9354773cea74d2c1b`. The new staged tree **before adding this report** is `81db54b365256286a9bb020ef924f8d7edc32a42`. No production source or accepted D-020–D-025 rule changed.

Changed source and decision files (SHA-256):

| File | Change | SHA-256 |
|---|---|---|
| `test/integration/b0_asymmetric_test.go` | Added PrepareProposal, ProcessProposal, and post-synced Commit application-kill cases; indexed transaction result and process-state checks | `1cc96a39c5b720ea3b2b4516402743b3abb4972cc37d77559ef00dc6673da451` |
| `test/integration/b0_recovery_test.go` | Added test gate for ProcessProposal and after durable Commit | `be93c553a6793bb0e76b57c52687005ca361f869113ba3cabd39b0d3eed95fcd` |
| `test/integration/b0_m7_callbacks_test.go` | New disposable-process callback probes | See [source manifest](../evidence/phase_03d/final_closure/m7_source_manifest.sha256) |
| [D-026 proposal](../decisions/D-026_EXPLICIT_B0_INITIALIZATION_PROPOSAL.md) | Design only, no implementation | See source manifest |

The source manifest SHA-256 is `de35ef043d7759891c6c9995c741a666e4e2e7a8ef39e11940646e14afbe2872`; all four entries pass `sha256sum -c`. The [evidence manifest](../evidence/phase_03d/final_closure/evidence_manifest.sha256) lists every new raw log and test output (52 entries), SHA-256 `a92a6dba23fa56729cc7d535ac336ffb25a43fa981124ed66404751ab928a4e1`; it also passes `sha256sum -c`. The earlier [Controller-reviewed source bundle](../evidence/phase_03d/closure/source_review_bundle.txt), existing vectors, benchmarks and prior recovery logs remain preserved. This report is a new handoff; it does not alter that reviewed bundle.

## M7 callback coverage

Each new test used a disposable node home and a separate application test process connected to a real pinned CometBFT process. The callback itself wrote a marker; the harness confirmed Comet was alive at that marker, sent **SIGKILL only to the application**, observed Comet independently 1.2 seconds later, and only then stopped/restarted what remained. In all new application-kill runs, Comet had exited and RPC refused connections by that observation. An app-only kill is distinct from prior M8 Comet-only kills and from coordinated shutdown. Test-only proposal omission triggered a real Comet `RECHECK`; it changed no production proposal logic. These process kills do not model physical power loss.

**B/C/A** mean blockstore, durable Comet consensus state, and durable B0 height. `Hn` is the application AppHash at **that height**. Relevant full hashes: H0 `e480c8b300c5ac1d662628f90334aae185ad38ed791c59ab14e3aa42abf083da`; H2 `c0305de0df961f7bb6b7b379a392fb96c3cb7ee1ca6b48c767bd833570d55ae0`; empty H3 `57434e9bf8b6df4fe6c14476a65efe33117efe7b58291158d5e7acf567338c5a`; transfer H3 `b8f217cdbeb9a716851387d5b0fb7319389bca0cb1acfac67ff35ad6e468b743`; transfer H4 `44295da9566bf288667d249e95cd301217abbf9b76b500f594110fd11f9e2ffa`; transfer H5 `ceaf56ac281e653a7ac09ee4d27e6cfd251242def3f14fd5c5f004841a8189b2`. Hashes are compared only at matching heights.

| Boundary reached and kill method | Before restart | Recovery and transaction outcome | M7 result |
|---|---|---|---|
| Initial `Info`, app SIGKILL before `InitChain` | B/C=0/0, A absent, no account or AppHash | Production `start` explicitly refused missing B0 beside Comet data; no transaction requested | PASS fail closed |
| `InitChain` before B0 write, app SIGKILL | B/C=0/0, A absent | Same operator-visible refusal; no implicit reset or transaction | PASS fail closed |
| `InitChain` after synced H0, before response, app SIGKILL | B/C/A=0/0/0, H0, sender 100/0 | Fresh app and real Comet retried identical genesis; B/C/A=2/2/2, H2, sender 100/0; no transfer | PASS |
| `CheckTx` NEW before response, app SIGKILL | B/C/A=2/2/2, H2, sender 100/0; admission response lost | Explicit readmission; indexed execution Code 0, B/C/A=4/4/4, H4, sender 75/1, receiver 25/0 once | PASS |
| Comet-issued `CheckTx` RECHECK before response, app SIGKILL | B/C/A=3/3/3, empty H3, sender 100/0. Test proposer omitted one admitted tx for one block | Real Comet restart; indexed execution Code 0, B/C/A=5/5/5, transfer H5, sender 75/1, receiver 25/0 once | PASS |
| `PrepareProposal` before handling admitted tx, app SIGKILL | B/C/A=2/2/2, H2, sender 100/0 | No decided block; explicit rebroadcast, indexed Code 0, B/C/A=4/4/4, H4, transfer once | PASS |
| `ProcessProposal` before response, app SIGKILL | B/C/A=2/2/2, H2, sender 100/0 | Comet retained tx; indexed Code 0 after restart, B/C/A=4/4/4, H4, transfer once | PASS |
| `FinalizeBlock` before handling tx, app SIGKILL (preserved prior case) | B/C/A=3/2/2, H2, sender 100/0 | Actual Comet replay, indexed Code 0, B/C/A=4/4/4, H4, transfer once | PASS |
| `Commit` before B0 write, app SIGKILL (preserved prior case) | B/C/A=3/2/2, H2, sender 100/0 | Actual Comet replay, indexed Code 0, B/C/A=4/4/4, H4, transfer once | PASS |
| `Commit` during WAL `Write` or `SyncData`, app SIGKILL (prior M4) | Write: B/C/A=3/2/2, H2, sender 100/0. Sync entry: 3/2/3, transfer H3, sender 75/1 | Actual replay or mock replay respectively; B/C/A=4/4/4, H4, transfer once. Entry to `SyncData` is not proof of device durability | PASS for reached software fault |
| `Commit` after synced B0 write but before response, app SIGKILL | B/C/A=3/2/3, transfer H3, sender 75/1, receiver 25/0 | Actual Comet mock replay, indexed Code 0, B/C/A=4/4/4, H4, transfer once | PASS |
| After successful Commit response, application parent SIGKILL (preserved prior case) | B/C/A=3/3/3, sender 75/1; Comet child confirmed alive immediately after parent kill | Coordinated stop followed by real restart, B/C/A=4/4/4, sender 75/1; no duplicate effect | PASS for tested post-ack point |
| Read-only `Info` and `Query`, app SIGKILL at each callback | Each B/C/A=2/2/2, H2, sender 100/0 | Real restart B/C/A=3/3/3, empty H3, sender 100/0, receiver absent; no transaction requested | PASS |

The raw outputs also show timing variants in which a previously in-flight empty block left B=2,C=A=1 when a read-only or CheckTx callback was killed; no transfer had occurred, and restart converged. The final matrix uses the final successful run's observed heights. No callback is called PASS merely because a marker was configured; every listed marker was reached. For the initial `Info` and pre-write `InitChain` windows, the accepted fail-closed outcome requires operator recovery and was not relabeled as a successful in-place replay.

## D-026, gates and remaining limits

The separate [D-026 proposal](../decisions/D-026_EXPLICIT_B0_INITIALIZATION_PROPOSAL.md) recommends making `init-b0` create and verify the canonical H0 snapshot, requiring it on `start`, and making `InitChain` validate identical persisted genesis idempotently. It compares an independent marker alternative, crash ordering, D-025 transport, Comet handshake, repeated initialization, missing data and operator recovery. **It is not implemented or accepted.** D-024 record bytes stay unchanged in the proposal.

**M7: PASS for the enumerated software callback boundaries and fail-closed startup case.** **G3: PARTIAL** because T7 complete-data-loss policy remains unresolved. **D12: PARTIAL** for the same unresolved recovery policy despite M7 callback closure. **D17: PARTIAL** because G3 remains partial. T7 remains **PARTIAL**: feasible prior software interruption tests pass, but loss of both stores under retained configuration still needs an approved initialization policy; physical power loss and device-cache behavior remain unverified hardware conditions. No Phase 03D completion is claimed.

Verification: [M7 normal](../evidence/phase_03d/final_closure/m7_targeted_integration_final.log) passed for the callback harness and preserved post-Commit case; [M7 race](../evidence/phase_03d/final_closure/m7_targeted_race.log) passed for the new callback harness, without rerunning the preserved post-Commit case under race. [Final proposal/Commit normal](../evidence/phase_03d/final_closure/m7_asymmetric_final.log) and [race](../evidence/phase_03d/final_closure/m7_asymmetric_race_final.log) passed after indexed-result additions; [pre-write InitChain normal](../evidence/phase_03d/final_closure/init_pre_write.log) and [race](../evidence/phase_03d/final_closure/init_pre_write_race.log) passed; [make verify](../evidence/phase_03d/final_closure/make_verify_final.log) passed after final source edits. The [exploratory failure record](../evidence/phase_03d/final_closure/EXPLORATORY_FAILURES.md) retains the failed RECHECK indexing probe and the earlier ProcessProposal rebroadcast assumption. Raw successful child and Comet logs are in [raw](../evidence/phase_03d/final_closure/raw/). Hosted CI and physical power-loss tests were not run in this bounded pass.

**STOP — Await Controller/Owner decision. Do not implement D-026 without explicit Owner approval. Do not merge or begin Phase 03E.**
