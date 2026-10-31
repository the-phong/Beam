# PLUTO PHASE 03D — RECOVERY CLOSURE REPORT

**2026-10-25. Technical verdict: PARTIAL; STOP for Controller source review.** Phase 03D remains unaccepted. No Phase 03E work, merge, push, PR, Git identity, or commit was performed. The branch is `phase/03d-b0-abci-integration`; `HEAD` remains `b56c81ccce7ea4a0e7f52053c7459c4601c15b38` (Phase 03C merge). There is no Phase 03D result commit. All new work is staged locally. The accepted D-020–D-025 protocol and D-024 physical record format remain unchanged.

## 1. Source identity and Controller upload

Before editing, both expected Git trees existed. `git write-tree` was `494341fca6a952587c16decdf4ebf05d98bf0b36`; prior tested tree `fa28e6f7b02b70020899c8294b74af02c7ab51d0` existed. The **exact 129,294-byte binary diff** between them is [tested_to_final.diff](../evidence/phase_03d/closure/tested_to_final.diff), SHA-256 `42faf8ccd52d5f3073b0b1554f1562f55206f100825b1e503da743693a016b21`. It changes nine paths: README trailing newline; Phase 03D report, status and manifest wording; M7 log, two CPU profiles, the source manifest and final verification log. It changes no implementation source. The prior source manifest SHA-256 `210be2016048643b05b521dcf2af4925556d909a0a36b4ac660381eea2853ee0` passed every listed file check before editing.

The reviewed implementation snapshot is Git tree `87b504853cb0ff98790976879ea8832db92b83e0`. The uploadable [source_review_bundle.txt](../evidence/phase_03d/closure/source_review_bundle.txt) is 404,267 bytes, SHA-256 `94e4c562a1ed43ad92fcf22d65fa5724379e6fc3a74e715be587ac2a6f523916`. It contains the changed-file manifest, actual binary Git patch from accepted D-025 tree `7295b641254c91c53b3b80a0c2ec5f6c286e8dd1`, and verbatim Git blobs for the B0 application, trusted state evaluator, store, node lifecycle, transaction/recovery tests and benchmark scripts. Each source block has its own SHA-256. The [closure source manifest](../evidence/phase_03d/closure/source_manifest.sha256), SHA-256 `f0146d74ebe11561efbdd3c23f4508caf42a69a2a0456035a72a69213f47355c`, passes `sha256sum -c`. The bundle was generated before adding itself and this report to avoid a self-referential tree ID. The final staged evidence tree ID is reported in the operator handoff.

The [evidence manifest](../evidence/phase_03d/closure/evidence_manifest.sha256), SHA-256 `8796dd3d10519e8b197ac913b113cbf388ef18f345310a03f25c8829b9332073`, checks all 54 closure evidence files, including the upload bundle and raw fault logs.

## 2. Remaining implementation changes

- `internal/b0store/store.go`: added test-only filesystem and genesis-stage hooks with zero-option production behavior. `OpenExistingWithOptions` now checks that the directory exists before calling Pebble; an exploratory test showed Pebble could create an absent directory even while returning an error. No key, record byte, sync policy, or accepted layout changed.
- `internal/b0app/app.go`: exposed `OpenWithStoreOptions` for disposable real-node fault tests; the production `Open` passes zero options.
- `test/integration`: added separate-process WAL write/sync I/O-error and SIGKILL probes, Comet's pinned post-Commit failpoint probe, asymmetric application-crash probes, real `InitChain` response loss, and expanded height/hash/pruned-history rejection cases. `internal/b0store/recovery_closure_test.go` covers direct genesis and WAL boundaries.
- `Makefile`: increased only the full integration package timeout from 120 to 180 seconds; the expanded suite took 122.389 seconds in the final run. No dependency changed.

Prior successful V01–V25 vectors, Phase 03B/03C golden files, real transfer tests, benchmark raw samples and CPU profiles were preserved. The earlier [Phase 03D report](PHASE_03D_REPORT.md) is historical pre-closure evidence; this report updates its recovery verdict.

## 3. Fault methods and observed outcomes

**B** is blockstore height, **C** is durable Comet consensus-state height, and **A** is durable Pluto application height. `Hn` below is the **post-execution** application AppHash at height `n`; comparisons are only at matching heights. The public test sender begins at 100/nonce 0 and receiver is absent. Transfer success is Code 0, sender 75/nonce 1, receiver 25/nonce 0. All node homes were disposable. [Verbose real-node results](../evidence/phase_03d/closure/real_node_faults.log), [extra mismatch results](../evidence/phase_03d/closure/mismatch_extra.log), [WAL process-kill results](../evidence/phase_03d/closure/m4_process_kills.log), [store WAL results](../evidence/phase_03d/closure/m4_store_faults.log), [genesis store results](../evidence/phase_03d/closure/m11_store_faults.log) and [raw child/Comet logs](../evidence/phase_03d/closure/raw/) are retained.

| Case / exact point | Method and pre-restart result | Restart and transaction effect |
|---|---|---|
| M4 before WAL write | ErrorFS returns an error from `.log` `Write`; separately, SIGKILL while that call is blocked. Pebble's returned-error path exits its child fatally. B=3,C=2,A=2, H2=`c0305de0df961f7bb6b7b379a392fb96c3cb7ee1ca6b48c767bd833570d55ae0`; sender 100/0. | Actual Comet plus application replay of stored H3 block. B=C=A=4, H4=`44295da9566bf288667d249e95cd301217abbf9b76b500f594110fd11f9e2ffa`; sender 75/1, receiver 25/0. |
| M4 WAL sync entry | Test FS rejects `.log` `SyncData`; separately, SIGKILL while `SyncData` is blocked. Pebble error exits its child. B=3,C=2,A=3, H3=`b8f217cdbeb9a716851387d5b0fb7319389bca0cb1acfac67ff35ad6e468b743`; sender 75/1. | Actual Comet mock replay, no second transfer. B=C=A=4, H4 as above; sender 75/1, receiver 25/0. The valid record after an injected sync error demonstrates OS/page-cache survival in this run, not power-loss durability. |
| M6 after successful Commit response, before Comet state save | Pinned Comet `FAIL_TEST_INDEX=7` at `state/execution.go` post-Commit `fail.Fail`; log shows `committed state` then `*** fail-test 7 ***`. This is an injected process exit, not SIGKILL. The decided H1 transfer has stored Finalize Code 0 and matching H1 AppHash. B=1,C=0,A=1; H1=`c735447cd853aed15e488a349710e96284640994c9da0158478f19fee293982e`; sender 75/1. | Actual Comet mock replay/save. B=C=A=1 with the same H1 hash; sender 75/1, receiver 25/0. Empty blocks were disabled in this disposable home to make H1 the transaction block. |
| M7 before transaction FinalizeBlock; before transaction Commit | SIGKILL only the application subprocess at each marked callback. Comet was independently running at the kill point and its RPC refused connection 1.2 seconds later, a fail-closed exit. At coordinated stop B=3,C=2,A=2, H2 as above; sender 100/0. | Actual Comet replay of the decided block. B=C=A=4, H4 as above; sender 75/1, receiver 25/0. The earlier post-Commit app-parent SIGKILL run also retained sender 75/1 after restart. |
| M9 app ahead / consensus ahead / missing block / pruned base | Integrity-valid synthetic snapshots and Comet store manipulations in separate homes. App ahead: B=C=2,A=3, H3=`57434e9bf8b6df4fe6c14476a65efe33117efe7b58291158d5e7acf567338c5a`. Consensus ahead: B=A=2,C=3. Missing latest block: B=A=1,C=2. Actual `PruneBlocks(2)` with genesis A=0: B=C=2, blockstore base=2. | All real pinned restarts refused startup before another height. Pruned case reported `app block height (0) is too far below block store base (2)`. No transaction was executed in these fixtures; sender remained 100/0. |
| M10 same-height alternate app state; tampered consensus AppHash; replay conflict | First fixture B=C=A=2 but internally valid Pluto H2=`e2434048376271e68674d9c56137d510c6b85a38514235dacdf0920a8aabcc90` conflicts with Comet's H2=`c0305de0df961f7bb6b7b379a392fb96c3cb7ee1ca6b48c767bd833570d55ae0`. Second flips Comet's expected H2 to `c1305de0df961f7bb6b7b379a392fb96c3cb7ee1ca6b48c767bd833570d55ae0`. Third starts A=1 with internally valid alternate H1=`e6cdbe4347b271cf3193ca9d63b196ebe2f73784e3ef3487479070ec72057dac`, B=C=2, and replays available H2. | Each real restart rejected the matching-height hash conflict. The replay fixture logs `Applying block height=2`, then an AppHash mismatch. These are synthetic integrity-valid states, not ordinary transfers. |
| M11/T7 before mkdir, after mkdir, before genesis record, WAL write/sync, post-sync response loss | Direct disposable store hooks return errors or SIGKILL a subprocess while `.log` `Write`/`SyncData` is blocked. Before mkdir leaves no B0 directory; after mkdir/before-record/write leave an incomplete store that fails closed. Sync entry may leave a complete H0 record. A real Comet `InitChain` response is withheld after synced H0 and Comet is SIGKILLed: B=C=A=0, H0=`e480c8b300c5ac1d662628f90334aae185ad38ed791c59ab14e3aa42abf083da`; sender 100/0. | A fresh pre-mkdir retry creates H0. Incomplete directories are never silently reset. Complete H0 reopens; real Comet retries identical JSON/Base64 `InitChain`, then B=C=A=2, H2=`c030…`, sender 100/0. No transfer was requested. |

The raw child logs distinguish `pebble: fatal commit error: injected error`, the Comet failpoint exit, and SIGKILL. The earlier M3/M5 pre-write/post-sync controls are not relabeled as WAL faults. No test models physical power removal or storage-device cache loss. The WAL sync observations on WSL2/9p cannot establish hardware durability.

## 4. Updated M1–M12 matrix

| Case | Status | Evidence / remaining limit |
|---|---|---|
| M1 | PASS for tested point | Prior proposal-stage Comet SIGKILL; no block persisted; explicit rebroadcast once. |
| M2 | PASS | Prior before-Finalize real replay. |
| M3 | PASS | Prior before-Commit real replay and persisted Finalize result. |
| M4 | PASS for controlled software faults | Pre-write, returned WAL write/sync errors, blocked-call SIGKILL, post-sync; old or complete new record and correct real/mock replay. Physical power loss remains untested. |
| M5 | PASS | Prior synced app write with withheld Commit response; mock replay. |
| M6 | PASS | Pinned post-ack/pre-Comet-save failpoint, B=1,C=0,A=1; mock replay, exactly once. |
| M7 | PARTIAL | Added app-only kills before Finalize and Commit; prior post-Commit kill. Comet exited fail closed. App-only interruption at every other ABCI callback is not tested. |
| M8 | PASS for tested boundaries | Prior Comet-only kills while app remained alive. |
| M9 | PASS for specified relations | App ahead, C ahead B, missing latest block, genuine pruned base all rejected. Normal permitted replay is covered by M2/M3/M4. |
| M10 | PASS for specified conflicts | Internally valid app mismatch, tampered Comet expected hash, and available-history replay mismatch all rejected. |
| M11 | PASS for feasible software windows | Before/after mkdir, before record, WAL write/sync error and kill, post-sync loss, actual Comet `InitChain` retry. Complete loss of all local chain data under retained config still needs an initialization-policy decision. |
| M12 | PASS for tested replay | Prior duplicate Finalize height rejection; M4/M5/M6/M7 show one transfer effect. |

## 5. T7 matrix

| Initialization boundary | Status and observation |
|---|---|
| Before B0 directory creation | PASS: injected error leaves no directory; fresh retry creates valid H0. |
| After directory creation, before record | PASS fail closed: existing incomplete path is not reset and cannot open. |
| During genesis WAL write | PASS software fault: returned error or SIGKILL at marked write, incomplete store rejected. |
| During genesis WAL sync | PASS software fault: returned error or SIGKILL at marked `SyncData`; reopened complete H0 in observed runs. |
| After synced record, before response | PASS: direct post-sync response error and real Comet response withholding; identical retry accepted. |
| Conflicting retry | PASS: prior direct adapter test rejects it. |
| Missing B0 beside persisted Comet data | PASS fail closed in prior real-node test. |
| Complete loss of both stores under retained config | **UNRESOLVED** initialization policy; prior pinned Comet refused missing `priv_validator_state.json`, but the B0 missing-store guard alone cannot identify a new chain. No policy/layout change was made. |
| Physical power loss / device-cache loss | **NOT RUN** hardware limitation, not inferred from process kills. |

**T7 remains PARTIAL** because the complete-data-loss policy is unresolved. The feasible software interruption windows above were reached; hardware power-loss safety remains unverified.

## 6. G1–G8 and D1–D18

| Gate | Updated result |
|---|---|
| G1 authentication precedence | PASS for V01–V25 and tested invalid inputs. |
| G2 integrity boundary | PASS for complete-record load, integrity-valid mismatch checks and trusted-state invariants; coherent replacement of all stores is outside the claim. |
| G3 cross-store recovery | **PARTIAL**: M7's remaining callback classes and T7 complete-loss policy remain. |
| G4 duplicate execution | PASS for tested real/mock replay; balances and nonces change once. |
| G5 height/AppHash | PASS for matching-height real node and mismatch rejection. |
| G6 failure-atomic persistence | PASS for tested software write/sync/error/kill envelope; hardware power loss unverified. |
| G7 resource diagnostics | PASS as diagnostic measurement only. |
| G8 historical compatibility | PASS: Phase 02 integration and frozen protocol/source checks. |

| D gate | Updated result |
|---|---|
| D1 baseline/authority | PASS: verified source and branch; local-only. |
| D2 explicit B0 genesis/runtime | PASS. |
| D3 result codes/precedence | PASS for tested vectors. |
| D4 committed-only CheckTx | PASS. |
| D5 ordered PrepareProposal | PASS. |
| D6 independent ProcessProposal | PASS. |
| D7 Finalize/fallback | PASS. |
| D8 Commit/uncertainty | PASS for tested software faults and fail-closed reopen. |
| D9 committed account query | PASS. |
| D10 AppHash/height | PASS for observed matching-height and mismatch paths. |
| D11 R2 diagnostics | PASS as measurement evidence, not a DoS conclusion. |
| D12 crash/replay | **PARTIAL** because M7 callback coverage and T7 policy remain. |
| D13 accepted vectors | PASS for direct V01–V25; synthetic anomalous cases remain labeled. |
| D14 real single validator | PASS for the tested classical transfer, restart and fault cases. |
| D15 historical Phase 02 | PASS. |
| D16 reproducibility/CI | **PARTIAL**: local commands, raw evidence and source manifest pass; hosted CI NOT RUN. |
| D17 G1–G8 | **PARTIAL** because G3 remains. |
| D18 phase discipline | PASS: no normative decision change, merge, push or Phase 03E work. |

## 7. Evaluator and benchmark review

`EvaluateTrusted` accepts only `State` values with an unexported trusted bit created by `NewState`, genesis parsing, logical decoding or a prior transition. Creation/reload validates ordering, supply, height, chain, canonical bytes and stored digest; `Accounts()` returns a copy. The fast path skips repeated whole-state validation, so its integrity assumption is that package code does not mutate the private account slice or falsely set the trusted bit. `Evaluate` remains the Phase 03C reference entry point but calls `EvaluateTrusted` after `Validate`; their differential test establishes fast-path equivalence under the shared transition implementation, **not independent correctness**. Golden vectors and the Phase 03C Python oracle provide separate checks with finite coverage.

The accepted stateless validation runs before state checks; nonce precedes balance, balance precedes receiver overflow and account cap. Block envelope limits precede per-transaction evaluation. CheckTx, PrepareProposal and ProcessProposal read committed state; ProcessProposal validates independently of local CheckTx; FinalizeBlock produces pending state and one result per transaction; Query sees committed state. Direct tests cover pending isolation, malformed/invalid precedence, failed-tx continuation, over-limit fallback and uncertain-commit read closure. The review found no reason to alter a frozen validation code or D-020–D-025 rule.

The prior raw benchmark samples and three five-iteration trials remain in [verification.log](../evidence/phase_03d/remaining/verification.log). K=`1,64,4096,65536` fixtures and 30 sync samples per K yield descriptive median/p95/p99 only; 30 samples cannot support reliable tail-latency or population confidence claims. Go `ns/op` is elapsed wall-clock time, **not measured CPU time**. `-cpuprofile` provides sampled CPU attribution; the trusted malformed profile contains only 10 ms of samples and cannot support fine-grained attribution. Full-record writes, compaction, filesystem cache, WSL2 and the repository's Windows 9p mount confound latency and durability extrapolation. These results are Phase 03D engineering diagnostics, not final PQC, DoS, production-throughput or hardware-durability findings.

## 8. Verification, unresolved conditions and STOP

`make verify` exited 0 in [make_verify.log](../evidence/phase_03d/closure/make_verify.log); `make test-integration` exited 0 in [make_test_integration_final.log](../evidence/phase_03d/closure/make_test_integration_final.log) after 122.389 seconds; the targeted real-node fault command exited 0 in `real_node_faults.log`; [m4_process_kills.log](../evidence/phase_03d/closure/m4_process_kills.log), `mismatch_extra.log`, `m4_store_faults.log` and `m11_store_faults.log` each exited 0. The relevant `go test -race -count=1 ./internal/b0app ./internal/b0state ./internal/b0store ./internal/b0genesis` result is in `race_final.log`. Final statement coverage was B0 app 84.8%, B0 state 88.0% and B0 store 82.9% in `coverage.log`, below the aspirational 90% target. Exact commands, exit codes, toolchain, branch, tree IDs and SHA-256 values are in the closure evidence. Failed exploratory `make verify` and build attempts are retained in `exploratory_make_verify.log` and `exploratory_m11_build_failure.log`; the failure led to the absent-path correction and missing test import, respectively.

Unverified conditions are M7 application-only interruption at all remaining ABCI callbacks, complete loss of both stores under retained configuration, physical power loss and device-cache behavior, hosted CI, and independent Controller source review. The single-validator WSL2/9p runs do not establish multi-validator or hardware behavior. The Owner has not accepted Phase 03D or authorized Phase 03E.

**Final technical verdict: PARTIAL. Submit the source bundle and this evidence to the Controller; STOP.**
