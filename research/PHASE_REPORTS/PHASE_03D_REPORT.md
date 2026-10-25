# Pluto Phase 03D — remaining execution report

**Date:** 2026-10-24 (Asia/Bangkok). **Status:** PARTIAL implementation and evidence candidate for independent Controller review; not Owner accepted or merged. The Owner closed the D-025 checkpoint and authorized the remaining Phase 03D work. Phase 03E and later work remain unauthorized.

## 1. Authority, baseline and source identity

The branch is `phase/03d-b0-abci-integration`, based on `test-mainet` commit `b56c81ccce7ea4a0e7f52053c7459c4601c15b38`. The accepted D-025 staged tree was `7295b641254c91c53b3b80a0c2ec5f6c286e8dd1`; its independently reviewed patch SHA-256 was `908e7c297deab3c574a6903dc63d9f6b45c6abefba1ebef60bab67dddc0fd1b2`. A read-only copy of that patch and its tree manifest were preserved at `/tmp/pluto_phase03d_d025_accepted_checkpoint.patch` and `/tmp/pluto_phase03d_d025_accepted_tree_manifest.txt` before further work. The former retains the accepted SHA-256; the latter has SHA-256 `1cd0eabebf713724afa9e11df9349e96f5ff615e9383ffb5ccd931f9d85c7d65`.

The final tested staged source tree, full staged tree, source manifest hash, and patch handoff are recorded in the final handoff section below. `HEAD` still identifies the Phase 03C merge, not these staged Phase 03D files. No Git author identity was configured, so no commit was fabricated. No push, PR or merge was performed.

## 2. Architecture and runtime selection

The D-025 `plutod init-b0` and `start --profile b0` path remains explicit. `internal/b0genesis` strictly decodes the JSON/Base64 transport and passes the original canonical binary to the accepted Genesis V2 parser. `internal/b0app` now uses `internal/tx` for stateless authentication, `internal/b0state` for account transfers and AppHash V2, and `internal/b0store` for pending/committed snapshots and synchronous Pebble `Set`. CometBFT v0.38.26 retains consensus, P2P, RPC and mempool ownership. The historical Phase 02 profile, V1 store and transaction rejection path remain untouched.

The D-025 account-equality guard is retained only at height zero. Above zero, normal transfers may change account membership and balances; startup requires matching chain ID, self-validating V2 record, conserved genesis total supply and the pinned CometBFT height/hash handshake. These checks detect some one-sided genesis or state changes. They do not prove complete genesis provenance if an adversary coherently replaces the genesis file, application snapshot and CometBFT history. No genesis fingerprint or new persistent field was added.

## 3. Transaction validation pipeline and R2 integrity boundary

The accepted precedence is unchanged: structural Code 1, chain Code 2, zero amount Code 5, scalar Code 4, public key Code 3, ECDSA Code 4, nonce Code 6, balance Code 7, receiver overflow Code 8, account cap Code 9, success Code 0. Code 10 is the over-limit decided-block fallback. Successful response log strings are the accepted names, success is `ok`, and the transaction codespace is `pluto.b0.v1`. Zero gas fields and CheckTx priority are implicit zero in the pinned response types. The query codespace is `pluto.query.v1`.

`b0state.Evaluate` remains the original Phase 03C reference path and fully validates the state before stateless transaction checks. Newly constructed or reloaded states establish an immutable, fully validated trusted snapshot. `EvaluateTrusted` performs bounded stateless rejection first and then uses that trusted candidate for state-dependent checks; successful transitions construct a new state through the accepted validating constructor. The trusted marker is private and cannot be forged through the public API. The store validates complete records and their AppHash on load and commit; a corruption or uncertain write returns an internal error instead of a transaction code. No local resource metric affects validity. Differential direct ABCI tests compare codes, account state and AppHash with the reference evaluator.

## 4. ABCI++ lifecycle

| Method | Implemented contract and evidence |
|---|---|
| `Info` | Reports only durable committed height/AppHash; uncertain store errors fail closed. |
| `InitChain` | Retains D-025 strict transport, exact genesis and height-zero synchronous initialization/idempotency. |
| `CheckTx` | NEW and RECHECK use the latest committed state only; no nonce reservation or pending mutation. Wrong nonce 1 remains Code 6 while nonce 0 is only in the mempool. |
| `PrepareProposal` | Under the application read lock, simulates supplied candidates in order on an isolated state, skips invalid/over-budget candidates, continues scanning, enforces 6,144 transactions and 1,048,576 aggregate bytes, and returns empty when `MaxTxBytes <= 0`. |
| `ProcessProposal` | Independently rejects any invalid transaction or envelope violation using a throwaway committed-state candidate; it does not trust CheckTx or a proposal cache. |
| `FinalizeBlock` | Holds the application write lock through final height validation and pending preparation. In-limit blocks return one result per input, apply successes in order and preserve state on failures. Over-limit blocks return Code 10 for every input, apply none, and still advance height once. |
| `Commit` | Requires pending state, calls D-024 synchronous complete-record write, promotes in memory only after success, and retains R1 fail-closed uncertainty handling. Pinned `ResponseCommit` has no AppHash. |
| `Query` | Only `b0/account/v1`, 20-byte address, height zero and no proof; existing account is `01 || balance:u64be || nonce:u64be`, missing account `00`, unsupported query Code 1/`unsupported_query`. Only committed state is visible. |

Direct tests cover order, first-failure results, pending isolation, same-block nonces, anomalous decided blocks, stale/duplicate heights, queries, and ambiguous Commit. The real-node test covers transaction submission, inclusion, finalization, committed queries and restart. Direct ABCI tests are not presented as real consensus observations.

## 5. Accepted vector execution V01–V25

The exact wire hex, TxID eligibility, Pluto TxID, Comet raw hash, response code and AppHash for individual cases are emitted by `TestB0AcceptedSingleTransactionVectors` in the verbose verification log. Fixture secrets are public **nonproduction** scalar IDs 1 and 2. All comparisons use the accepted chain and canonical wire rules. The `b0state.Evaluate` reference supplies independent logical state comparison; Phase 03B and 03C golden files and Python oracles remain unchanged.

| Vector | Observed result / test boundary |
|---|---|
| V01 | Valid A→B 25, Code 0; A 75/nonce 1, B 25, supply 100. Direct ABCI and real CometBFT. |
| V02–V03 | Zero `r` and high-S: Code 4, no protocol TxID. Direct ABCI. |
| V04–V06 | Tampered amount Code 4; other chain Code 2; 33-byte invalid key Code 3. Direct ABCI. |
| V07 | Bad magic/version/length/ASCII, short/trailing bytes and 65-byte-key shape: Code 1. Direct ABCI. |
| V08–V11 | Future nonce Code 6, replay Code 6 after Commit, insufficient balance Code 7, zero amount Code 5. Direct ABCI; replay and admission negatives also real node. |
| V12–V16 | Exact balance, duplicate, sequential nonces, proposer injection and sender/key binding match accepted codes and state effects. Direct ABCI. |
| V17–V19 | Max nonce Code 6, overflowing Genesis V2 rejected before state creation, and account cap Code 9. V17/V19 use complete integrity-valid synthetic persisted states, not a live consensus fixture. |
| V20 count | 6,145 valid signed, funded, sequential-nonce 146-byte wires on one-byte chain; 897,170 bytes. `ProcessProposal` rejects, anomalous `FinalizeBlock` returns 6,145 Code 10 results and unchanged accounts at next height. Direct ABCI. |
| V20 bytes | 5,378 valid signed, funded, sequential-nonce 195-byte wires on 50-byte chain; 1,048,710 bytes. Same rejection/fallback, isolated byte limit. Direct ABCI. |
| V21–V22 | Failed nonce-0 transaction followed by valid nonce-0 transfer yields `[7,0]`; self-transfer increments nonce without changing balance. Direct ABCI. |
| V23–V25 | Modified persisted logical byte with old digest is rejected on open; zero amount outranks canonical-but-bad signature; nonce error outranks insufficient balance. V25 uses an integrity-valid synthetic state. |

V20 count concatenated-wire SHA-256 is `243156b61f976efa18a0d909af3d1d4f22085cbf35f57c6b876f9604006d9b15`; byte fixture is `e80839653e1b18044d13db7514fc4a3150c58a1cda81c9dfc32f78212680e07b`. Tests assert every constituent transaction's stateless and sequential state validity before applying the envelope test. Neither V20 block can be produced by a correct proposer.

## 6. Real pinned single-validator observation and AppHash alignment

The disposable-home `TestB0RealTransactionCommitAndRestart` broadcasts V01, observes CheckTx Code 0 and execution Code 0 at height 3, then queries A=75/nonce1 and B=25/nonce0 with supply 100. The CometBFT RPC raw-byte hash is `4e55c31771244d6421ef46dd931c4465ae82e09a51bbc510d66a53d30813b26a`; the separate Pluto protocol TxID is `9b5275a89d6668e558459a94a22407812fdde6d4abe051c93e87cb1e9b480a9c`. RPC correlation uses exact wire bytes and block position. After restart, the state persists, a replay is Code 6, and a new valid transaction yields A=65/nonce2 and B=35. Real CheckTx negatives include malformed Code 1, wrong chain Code 2, invalid signature Code 4, future nonce Code 6 and insufficient balance Code 7.

At height H, block-header AppHash is the preceding application's state hash, while `FinalizeBlock` produces AppHash after H. `Info` reports the durable result after `Commit`; block H+1's header should carry the H result. The real-node test checks the matching-height relationship, including following-header hash `b8f217cdbeb9a716851387d5b0fb7319389bca0cb1acfac67ff35ad6e468b743` for the transfer fixture. The independent Phase 03C Python oracle confirms genesis `e480c8b300c5ac1d662628f90334aae185ad38ed791c59ab14e3aa42abf083da`, transfer H1 `c735447cd853aed15e488a349710e96284640994c9da0158478f19fee293982e`, self H2 `9e512d901ea0137a89bfab07a5b03505540f98bf5fe7fc86a680ae90b4f7caec`, and empty H3 `7822225ce3300e3be2e954828c6f7944c669b16f16ecc4ddc149108e75fad024` for its separate fixture.

## 7. Crash and replay matrix M1–M12

Here **B** is CometBFT blockstore height, **C** is persisted CometBFT consensus-state height, and **A** is durable Pluto application height. The observations below come from real pinned CometBFT child processes where stated. The temporary test homes are disposable; no Owner home or validator key is included in evidence. AppHashes refer to the *post-execution state at the specified A*, never an unrelated header. The complete command and test logs are in `research/evidence/phase_03d/remaining/verification.log`.

| Case | Controlled observation | Assessment |
|---|---|
| M1 | SIGKILL Comet during `PrepareProposal` before block persistence: B=C=A=2, no transfer; explicit rebroadcast later applied once. | PASS for this point. |
| M2 | Gate before `FinalizeBlock`: B=3,C=A=2; SIGKILL Comet, restart used real-app replay; final B=C=A=4, A balance75/nonce1. | PASS for this point. |
| M3 | Gate before `Commit`, after persisted Finalize response: B=3,C=A=2; stored response Code0 and expected post-H AppHash verified; restart used real-app replay once. | PASS for this point. |
| M4 | Phase 03C prewrite/returned-write/post-sync hooks cover old/new complete records; M3/M5 bracket the live call. No process interruption inside WAL append/fsync and no real cross-store returned Pebble `Set` error was injected. | **PARTIAL**. |
| M5 | Withhold Commit acknowledgment after synced Pluto write; SIGKILL Comet: B=3,C=2,A=3; persisted Finalize response/hash verified. Restart used mock replay; A balance75/nonce1 unchanged. | PASS for this point. |
| M6 | Normal C=A restart and M5 C=A−1 mock replay observed. A targeted kill immediately after successful Commit acknowledgment but before CometBFT state save was not isolated. | **PARTIAL**. |
| M7 | SIGKILL only application parent after committed transfer while Comet child remained alive briefly; `m7_verbose.log` records pre-restart B=C=A=3 and post-restart B=C=A=4, sender balance75/nonce1. Prolonged live-child behavior at every ABCI boundary was not tested. | **PARTIAL**. |
| M8 | Comet alone was killed while the application server remained alive in M2/M3/M5, followed by coordinated restart and correct replay. | PASS for tested boundaries. |
| M9 | App ahead of available blockstore (B=C=2,A=3) refused startup: `app block height (3) is higher than core (2)`. Other impossible/pruned relations were not tested. | **PARTIAL**. |
| M10 | Integrity-valid but different app snapshot at the same height was refused by pinned handshake: `state.AppHash does not match AppHash after replay`. Other mismatch types were not exhaustively tested. | **PARTIAL**. |
| M11 | D-025 standalone subprocess SIGKILL after synced genesis recovers complete height-zero record and identical retry. Incomplete directory and missing B0 beside persisted Comet data fail closed. Kill during directory creation or WAL append/fsync was not run. | **PARTIAL / T7 PARTIAL**. |
| M12 | Duplicate `FinalizeBlock` at committed height fails; M5 mock replay leaves transfer and nonce applied exactly once. | PASS for tested replay. |

M1–M3 and M5 provide controlled process-crash evidence, not arbitrary power-loss evidence. M4, M6, M7, M9, M10 and M11 retain unexecuted subconditions; therefore cross-store recovery is not fully established. A commit error with uncertain outcome blocks all ABCI reads and transitions until a validated close/reopen; the new adapter regression verifies this but does not inject a real Pebble `Set` error into a running Comet node. No shared CometBFT/Pebble atomic commit exists.

## 8. T7 carryover boundary matrix

| Genesis window | Tested mechanism and outcome |
|---|---|
| Before record creation / existing incomplete directory | Synthetic incomplete disposable B0 directory; initialization/open failed closed; no process killed at this exact point. |
| After synchronized genesis write, before response | Standalone child writes marker after `InitChain`; parent SIGKILLs it; complete height-zero snapshot reopens and identical retry succeeds. No Comet connection in this test. |
| Conflicting/identical retry | Direct adapter tests: identical height-zero retry idempotent, conflicting genesis rejected. |
| Missing B0 beside persisted Comet data | Real pinned startup rejects implicit reset. |
| All local data lost under retained config | Pinned Comet exits because `priv_validator_state.json` is missing; Pluto's missing-store guard alone cannot distinguish complete loss from first start. A separately approved initialization policy is needed for a complete-data-loss guarantee. |
| During directory creation, WAL append, fsync, or physical power failure | **NOT RUN** at these precise boundaries. Post-sync SIGKILL and simulated hooks do not substitute for them. |

T7 remains **PARTIAL**. The unexecuted software interruption windows and the complete-data-loss ambiguity remain Controller/Owner review items; no silent change to D-024 initialization or physical layout was made.

## 9. Resource-cost diagnostics and R2

The benchmark harness uses K=`1,64,4096,65536`, fixed chain/keys/account distribution and signed fixtures; K=1 distinct transfer creates the receiver, whereas larger K transfers to an existing receiver. It measures malformed, wrong-chain, zero amount, noncanonical scalar, canonical ECDSA-invalid, valid self and distinct transfers, empty advancement, full-state hash/serialization and synchronized complete-record persistence. The raw output, three 5-iteration benchmark trials, one warm-up sync sequence and 30 independent sync samples per K are retained in the verification log; `summarize_sync.py` reports median/p95/p99. The `logical-bytes/op` label is the potential canonical state size, **not** bytes hashed by a cheap-invalid trusted evaluation. For chain length 11, full logical sizes are 73, 2,341, 147,493 and 2,359,333 bytes. D-024 persisted record adds 40 bytes.

Preliminary run results are preserved separately in `verification_initial.log` and `verification_pre_real_verbose.log`; the final rerun is authoritative. On K=65,536, 100-iteration CPU-profile benchmark runs measured malformed reference evaluation at 3,437,935 ns/op, 4,735,179 B/op, 2 allocations; trusted evaluation at 33.36 ns/op, 0 B/op, 0 allocations. Three 5-iteration integrated CheckTx trials measured malformed input at 232.6–443.8 ns/op and 128 B/op, canonical-but-invalid ECDSA at 155,892–156,958 ns/op and 1,200 B/op, and valid distinct transfer at 4,710,095–5,295,306 ns/op and roughly 9.98 MB/op. The latter still pays O(K) complete-state reconstruction and hashing. The 30 raw synchronized persistence samples per K yield:

| K | Median ns | p95 ns | p99 ns | Complete-record bytes |
|---:|---:|---:|---:|---:|
| 1 | 1,838,511 | 2,331,752 | 2,437,626 | 113 |
| 64 | 1,990,150 | 2,715,296 | 2,870,994 | 2,381 |
| 4,096 | 3,136,989 | 9,672,342 | 10,229,337 | 147,533 |
| 65,536 | 13,762,282 | 16,559,950 | 18,293,218 | 2,359,373 |

These p95/p99 values are descriptive order statistics from only 30 local samples. Reference `Evaluate` shows O(K) full validation for malformed input while trusted evaluation returns before account serialization and hashing. The large-K CPU profile attributes substantial reference samples to SHA-256 and account ordering; the trusted profile has only 10 ms of samples, too few for reliable attribution. Wall-clock benchmark time is not CPU time; only the two explicit `-cpuprofile` runs provide sampled CPU attribution. The host was WSL2 Linux 6.18.33.2 on an Intel i7-12700H, with the repository on a Windows drive exposed as 9p; Go used default GOMAXPROCS and GOGC. These are diagnostic local engineering measurements, not a DoS exploit, mitigation result, production throughput claim, or Phase 11 experiment. The mounted filesystem and sync semantics limit latency generalization.

## 10. Verification and preservation

`research/evidence/phase_03d/remaining/run_verification.sh` records each exact command, output and exit code, toolchain/environment, staged tree, source manifest, vectors, fuzz counts, benchmark raw data and profile summaries. Its final run began at `2026-10-24T12:22:04Z` against staged tree `fa28e6f7b02b70020899c8294b74af02c7ab51d0` and ended with `verification_failures=0` and the same tree. Its **37 commands** included `go build ./...`, `go test -count=1 ./...`, `go vet ./...`, `make fmt-check`, `go mod verify`, `make verify`, `make test-integration`, race and coverage suites, targeted ABCI and verbose real-node tests, Python oracles, bounded fuzz targets, benchmarks, `git diff --check`, `git diff --cached --check`, a protected-source/specification diff and source manifest recheck; each exited **0**. A separate verbose M7 integration command also exited **0**, with its complete output and source tree identity in `m7_verbose.log`. Coverage was `b0app` **84.8%**, `b0state` **88.0%**, and `b0store` **80.1%** of statements, below the aspirational 90% target. Five fuzz targets ran for 2–3 seconds and exited 0, totaling 34,057, 33,502, 10,408, 24,019 and 35,685 executions respectively; they are not exhaustive security evidence. An earlier exploratory integration run failed with exit 2 due test-harness callback timing and an RPC timeout, was corrected, and is documented separately. The final rerun is not a claim of hosted CI; no PR or push was authorized. Because no Phase 03D commit exists, `plutod version` embeds the unchanged `HEAD` prefix `b56c81c`; the staged tree and manifest identify the tested implementation.

The accepted D-020–D-024 rules, accepted D-025 transport amendments, Phase 02 `internal/app`/V1 `internal/store`, Phase 03B `internal/tx`, `cmd/plutod`, `internal/node`, `go.mod`, `go.sum`, `Makefile`, workflows, old vectors and D-025 evidence remain unchanged against the accepted D-025 tree. The explicit B0 path is additional. The historical Phase 02 integration test still passes.

## 11. Safety gates G1–G8

| Gate | Evidence and observed outcome | Status |
|---|---|---|
| G1 authentication precedence | V01–V25 direct codes, reference differential tests and constant-cost malformed CheckTx measurements. | PASS for tested paths. |
| G2 integrity boundary | Full validation on creation/reload/commit, immutable trusted states, corruption rejection, byte-identical AppHash comparison. | PASS for tested paths; coherent multi-store replacement remains outside this guarantee. |
| G3 cross-store recovery | M1–M3, M5/M8 execute, but M4/M6/M7/M9/M10/M11 have missing required subconditions. | **PARTIAL**. |
| G4 no duplicate execution | M5 mock replay, M12 duplicate height rejection, real restart/replay Code6. | PASS for tested paths; untested crash windows remain under G3. |
| G5 height/AppHash | Real H/H+1 header alignment, `Info` and persisted Finalize response; mismatch startup rejected. | PASS for tested paths. |
| G6 failure-atomic persistence | D-024 single synchronized record, R1 adapter uncertainty fail-closed, M5 post-sync response loss; real mid-WAL fault absent. | **PARTIAL**. |
| G7 resource cost | Full K/workload benchmark matrix, raw samples, profiles and disclosed limits. | PASS as diagnostic measurement, not performance threshold. |
| G8 historical compatibility | Protected diff check and Phase 02 integration regression. | PASS. |

## 12. Acceptance matrix D1–D18

| Gate | Assessment |
|---|---|
| D1 baseline and authority | PASS: verified base, accepted D-025 tree and local-only branch. |
| D2 explicit B0 genesis/runtime | PASS: D-025 tests and explicit profile. |
| D3 result codes/precedence | PASS: direct vectors and accepted log/codespace. |
| D4 committed-only CheckTx | PASS: tests for NEW/RECHECK and pending isolation. |
| D5 ordered PrepareProposal | PASS: budget, skip and sequential tests. |
| D6 independent ProcessProposal | PASS: proposer-injected and vector tests. |
| D7 Finalize/fallback | PASS: result-per-input, failed-tx continuation, both V20 limits. |
| D8 Commit/uncertainty | PARTIAL: R1 adapter and M5 pass; mid-WAL/cross-store returned-error point untested. |
| D9 committed account query | PASS: direct and real-node tests. |
| D10 AppHash/height | PASS for observed real and direct paths. |
| D11 R2/measurements | PASS as diagnostic evidence, with full snapshot cost retained. |
| D12 crash/replay | PARTIAL: M4/M6/M7/M9/M10/M11 subconditions remain. |
| D13 vectors | PASS for direct V01–V25; anomalous cases explicitly synthetic. |
| D14 real single validator | PASS: transaction, query, restart, next transfer, replay rejection. |
| D15 historical Phase 02 | PASS: source diff and existing integration. |
| D16 reproducibility/CI | PARTIAL: local checks and source manifest; hosted CI NOT RUN. |
| D17 G1–G8 | PARTIAL: G3/G6 outstanding. |
| D18 governance/phase discipline | PASS: no merge, push, new phase or normative change. |

## 13. Limitations, verdict and review request

The result is **PARTIAL** and is **not eligible for a full Phase 03D completion claim**. Priority review items are controlled mid-Pebble-WAL interruption/returned-write cross-store outcomes, a targeted post-acknowledgment/pre-Comet-state-save interruption, broader asymmetric process failures and height/hash mismatch classes, remaining T7 genesis windows, and complete-data-loss initialization ambiguity. Physical power-loss, device failure, multi-validator agreement, EVM, PQC, DoS mitigation and production suitability were not tested or claimed. Coverage percentages and short fuzz runs do not prove correctness.

The Controller is asked to review the exact staged source patch, direct and real-node evidence, replay hooks and the unresolved mandatory gates. The Owner controls any acceptance, persistent-policy decision, push or merge. Phase 03E is not authorized.

## 14. Final source-review handoff

Final tested staged tree before the last report/evidence additions: `fa28e6f7b02b70020899c8294b74af02c7ab51d0`; its source manifest SHA-256 is `210be2016048643b05b521dcf2af4925556d909a0a36b4ac660381eea2853ee0`. The final staged tree ID, full local patch path/size/SHA-256 and delta from accepted D-025 are recorded externally in `/home/ntp/pluto_phase03d_remaining_handoff.txt`, because embedding the final tree and patch checksum in the patch would be self-referential. Source files are unchanged between the tested tree and that handoff; a final `sha256sum -c` confirms this. No commit, push, PR or merge was performed for this local execution.

**STOP — Phase 03D implementation and partial recovery evidence submitted for independent Controller review. Phase 03E and all subsequent phases are NOT AUTHORIZED. No merge has been performed.**
