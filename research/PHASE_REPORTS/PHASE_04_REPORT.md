# PLUTO Phase 04 — B1 hybrid authentication implementation report

**Status:** implementation candidate for Independent Controller review. Owner authorized one complete Phase 04 B1 workflow on 2026-10-26. Coding Agent completion is not Controller technical acceptance or Owner formal closure. Local implementation checks passed; hosted CI is tracked below. The exact tested implementation commit is distinguished from the later documentation-only handoff commit.

## Scope, baseline and decisions

- Repository: `the-phong/DoS-in-Blockchain-PQC`; local WSL2 repository `/mnt/d/personal/search&develop/blockchain/system/pluto`.
- Phase 03E accepted PR #8 is merged at `007e827f740c9c741f47bf1f7a51eec98a79cc5a`; its accepted implementation commit is `94cd9ae5cd952d86cf3d89c89546ca516e158578`. This merge is the Phase 04 base and the current branch is `phase/04-b1-hybrid-authentication`.
- D-028 records the Owner's consolidation of historical hybrid work from roadmap Phases 05–08 into this complete Phase 04. EVM is NO-GO for the current authentication/correctness experiment; no EVM component was added.
- D-029 records an isolated B1 Genesis V3, transaction V2/profile 1 and committed-record V3. B0 D-020–D-027 and Phase 02 remain unchanged. The only B0 production edit extracts its accepted balance/nonce transition into `ApplyAuthenticatedTrusted`, called after classical authentication as before and by B1 only after both proofs pass.
- The exact envelope, domains, key binding, AppHash, error precedence and threat assumptions are in [`B1_HYBRID_AUTHENTICATION.md`](../specs/B1_HYBRID_AUTHENTICATION.md). The four-validator topology and run command are in [`B1_4V_REPRODUCIBILITY.md`](../specs/B1_4V_REPRODUCIBILITY.md).

## Architecture and cryptographic implementation

B1 uses separate `internal/b1tx`, `b1state`, `b1genesis`, `b1store` and `b1app` packages. A fresh `init-b1-4v` root contains four independent validator identities, P2P identities, Comet states and B1 Pebble stores with one shared byte-identical genesis. B1 metadata records the expected public validator address for each specific home, preventing startup with another genesis member's copied key; the B0 metadata bytes remain unchanged. `start` probes an existing valid B1 snapshot and rejects B0/Phase 02 data, missing data and conflicting genesis. B1 `InitChain` verifies the persisted H0 and supports an identical retry without writing it. `CheckTx` and proposal/finalization paths use one deterministic evaluator but separate committed, candidate and pending states. Every successful B1 transaction requires both signatures over the same signed digest and an exact genesis registry binding. `Commit` syncs a B1-specific single record. No B0 record byte, B0 signed byte, B0 AppHash or CometBFT consensus rule was modified.

The pinned ML-DSA implementation is Cloudflare CIRCL v1.6.4 (`github.com/cloudflare/circl/sign/mldsa/mldsa44`). It declares Go 1.24.0 and provides pure ML-DSA-44 with an explicit context, 1312-byte public keys and 2420-byte signatures. Pluto signs a 32-byte domain-separated canonical intent digest with CIRCL context `PLUTO/B1/HYBRID/V1`. The public-only golden vector verifies that exact profile; CIRCL's own bundled FIPS204 ACVP signature-verification cases are exercised separately. FIPS algorithm compatibility does not imply FIPS module certification. The existing Decred secp256k1/v4 v4.4.1 remains pinned; CometBFT v0.38.26 and Pebble v1.1.5 remain pinned. CIRCL's module graph raised selected transitive `golang.org/x/*` dependencies (exact versions in `go.mod`/`go.sum`); full regressions below exercise the result.

## H01–H24 test and evidence matrix

| ID | Scenario | Actual path and assertion | Verdict |
|---|---|---|---|
| H01 | Valid dual proof | Unit evaluator + real four-validator RPC admission/finalization, 100/0 → 75/1 and 0/0 → 25/0 | PASS |
| H02 | Missing classical proof | Zeroed 64-byte proof returns code 4 before ML verify | PASS |
| H03 | Missing ML proof | Zeroed 2420-byte proof returns code 13 | PASS |
| H04 | Invalid classical proof | Mutated R returns code 4 | PASS |
| H05 | Invalid ML proof | Mutated ML signature returns code 13 locally and via real CheckTx | PASS |
| H06 | Classical proof for another intent | Swapped valid secp proof returns code 4 | PASS |
| H07 | ML proof for another intent | Swapped valid ML proof returns code 13 | PASS |
| H08 | Substituted ML key | Unregistered key returns code 11 before proofs | PASS |
| H09 | Substituted secp key | Unregistered key returns code 11 before proofs | PASS |
| H10 | Cross-chain replay | Changed chain returns code 2 | PASS |
| H11 | Profile downgrade | Unsupported profile byte returns code 1; B0 envelope has no fallback | PASS |
| H12 | Unsupported version | Changed version returns code 1 | PASS |
| H13 | Noncanonical encoding | Trailing byte returns code 1 | PASS |
| H14 | Truncated/oversized input | Length-bound failures return code 1; decoder fuzzed | PASS |
| H15 | Wrong nonce | Correctly signed wrong nonce returns code 6 | PASS |
| H16 | Insufficient balance | Correctly signed amount 101 returns code 7 | PASS |
| H17 | Replay | Second application of same wire returns code 6 without state change | PASS |
| H18 | Conflicting sequence | Ordered simulation accepts first, rejects wrong-next-nonce second | PASS |
| H19 | Invalid direct proposal | Direct ProcessProposal bypassing CheckTx rejects on all four stores | PASS |
| H20 | Valid then invalid proposal | Direct ordered proposal rejects atomically on all four stores | PASS |
| H21 | Restart/replay | Already-committed validator restarts; stopped validator catches up; effects once | PASS |
| H22 | Four-validator agreement | Three fresh real P2P networks; valid hybrid transfer with only 3/4 alive; same-height hashes/state | PASS |
| H23 | Application persistence crash | Application-only SIGKILL at pre-write/post-sync Commit gate; inspect B/C/A; recover once | PASS |
| H24 | B0/Phase 02 compatibility | Existing build, 40-case integration, 16-case B0 four-validator and relevant race suites | PASS |

Every test creates disposable homes and records the actual callback/fault marker; a missing marker is a failure. `H01–H18` use `internal/b1state/state_test.go`; the public golden vector is in `internal/b1tx/testdata/hybrid_public_vector.json`. `H19–H23` use `internal/b1app/app_test.go`, `test/integration/b1_four_validator_test.go` and `test/integration/b1_crash_test.go`. Raw test output and callback logs are retained locally under `research/evidence/phase_04/raw/` and excluded by `.gitignore`. A test exit alone is not used as proof of a specific fault boundary.

## Four-validator and recovery observations

The first successful four-validator log records three independent networks. At block H7 (which carries application AppHash for A=6), their matching block/AppHash pairs were `903B99CEBD958776ACA046DD9E1630C7A4BF8EA1D48F88D0C69750B02DB61468` / `46070E568E449DF7DDDDFDCDF8E95D4789CD53BE36E1E07515F6140212CD5ADC`, `A584915E527C7FD58DB71C49FAB061D79EB681650F6B467F9432FF646D4EA68E` / `DE2ABA3AFC09330FCFC3FC4A2F20B995146447AB1DC06B6D85630C121FACECE6`, and `2E2EDBD11D42168A5B0C2EB87F689470E8A06733DDA5D9DE4F69A2D08EFCA363` / `41D7CE56E53DC89DA8EFED466DD0014E2E7112385808032D8A70B242F5F87551`. Every synchronized validator had sender 75/nonce 1, receiver 25/nonce 0 and total supply 100. Different fresh networks have different genesis identities, so their hashes are not expected to match each other.

`H23` pre-write is application-process SIGKILL before the B1 record write: B/C/A=`3/2/2`, AppHash(A)=`620aba9acea6c93966c38d830064600f55d7e540c8d20a6ba7824da59adaaf53`, sender 100/0 and receiver 0/0 before recovery. `H23` post-sync is SIGKILL after the synced record write but before successful ABCI Commit response: B/C/A=`4/3/4`, AppHash(A)=`1a7ff8d81a23e6bc612145c093bfa143a1da88a39217535ca0e6d126bd1f07e2`, sender 75/1 and receiver 25/0. CometBFT is a distinct process and was confirmed alive at each callback marker. It had exited by the end of the outage observation because the app connection was lost; the parent observed no additional committed height before coordinated restart. Production restart converged to sender 75/1 and receiver 25/0 on all four validators at matching height, with one transaction effect. Raw logs retain the exact process IDs, markers and replay block hashes. A separate store test injects pre-write error and post-sync response loss; those are simulated software faults, not process kills. No test here proves physical power-loss durability.

The corrected-source rerun after the hosted cleanup finding also passed all 15 named B1 integration cases with zero SKIP. Its three fresh networks matched at block H7/AppHash(A=6): `BD27F54F6ED68527CC72F1B078EC8937186AD8B14A54B8424BFF8C2010C465A4` / `C7F3746EE01938B6FFADF6BCCA6F31FF6E28ECE571896EF2ABB9BBF559045E83`; `12B035CA3C4230E312E7AA126840AA321938F4E23C7173F5F7CA1B99BBF4E1F8` / `F239876A20D946ECE37B5262337FD51025AB7EDBD8574FFA2CE72FDE8EB1C61C`; `0B2BCE4D2A1C6D41C6E6DA0246A36ED438EB5DF5CA857B74183164B2D333AA52` / `AA932E6C172285F8215B8269A37B5BF8EC0375AEC4ADB0090C08E11A021D0CEA`. All retained sender 75/nonce 1, receiver 25/nonce 0 and supply 100. Its H23 pre-write marker recorded B/C/A=`4/3/3`, AppHash(A)=`b772d2c4d9d72cba80bd00baa18e68b019e124d77c00f2883c69592fad1c851b`, sender 100/0, receiver 0/0; post-sync recorded B/C/A=`3/2/3`, AppHash(A)=`e3f281069ca77d931ca55ac9637bc2cdcd6f7d35276ed7cf7d9b3f10a289cdcc`, sender 75/1, receiver 25/0. Comet was alive at both callback markers and recovered state applied the transfer once. The raw source for these corrected-run observations is `raw/b1_integration_cleanup.log`.

## Security review

The parser bounds input to 3947 bytes before slicing or allocation and enforces exact magic/version/profile/chain length/field order, rejecting trailing data. The signed digest includes both public keys, chain, sender, recipient, amount and nonce under a B1-specific domain. Genesis registry lookup plus address rederivation prevents proof/key substitution; both signature verifiers must succeed. Account nonce and checked balances prevent replay, overspend and overflow. A test creates two distinct valid randomized ML-DSA signatures over one intent: their TxIDs differ, but the same nonce permits only one state effect. This is transaction-ID malleability with precommit admission-resource implications, not a double-spend in the tested transition. Consensus validity uses no clock, local queue, resource budget or mempool arrival history. ProcessProposal validates every transaction independently of CheckTx; invalid ordered proposals cannot mutate committed state. A pending FinalizeBlock candidate is not durable until Commit. Startup rejects missing, corrupt or foreign B1 state. Key rotation and dynamic account creation are deliberately absent. Tests establish the listed software behavior under pinned dependencies; they do not prove cryptographic security beyond the selected algorithms, a FIPS-certified module, Byzantine behavior, distributed fault independence or DoS resistance.

## Verification, source and evidence audit

The initially attempted direct use of an ACVP internal-message vector through CIRCL's framed public `Verify` API failed because the APIs sign different message representations; that exploratory failure is preserved locally. The final check executes CIRCL's actual ACVP suite and tests Pluto's public B1 golden vector separately. An initialization negative test initially used a different *valid* genesis validator key, which could start because it remained a member of the set. That exploratory run is retained. The correction binds each B1 home to its expected public validator address in metadata and makes the copied-key test a real startup refusal; no B0 metadata or record was changed. The final negative test also imposes a bounded startup timeout. An exploratory test expected all-`0xff` fixed-width ML-DSA public-key bytes to fail decoding, but CIRCL treats them as a syntactic packing; the value is unregistered and correctly fails binding with code 11. The failed raw log and corrected test are retained. Code 12 remains defensive for a future library parser rejection.

| Exact command | Exit | Raw log |
|---|---:|---|
| `make verify` | 0 | `raw/make_verify_closure.log` |
| `make test-integration` | 0, 40 named, zero SKIP | `raw/make_test_integration_final.log` |
| `make test-integration-4v` | 0, 16 named, zero SKIP | `raw/make_test_integration_4v_final.log` |
| `PLUTO_4V_ARTIFACTS="$PWD/research/evidence/phase_04/raw/b1_4v_final" make test-integration-b1-4v` | 0, 15 named, zero SKIP | `raw/make_test_integration_b1_4v_final_source.log` |
| `go build ./...` | 0 | `raw/go_build_closure.log` |
| `go test -count=1 ./...` | 0 | `raw/go_test_closure.log` |
| `go vet ./...` | 0 | `raw/go_vet_closure.log` |
| `go mod verify` | 0 | `raw/go_mod_verify_closure.log` |
| `go test -race -count=1 ./internal/b0app ./internal/b0state ./internal/b0store ./internal/b1app ./internal/b1state ./internal/b1store ./internal/b1tx ./internal/tx ./cmd/plutod` | 0 | `raw/race_core_closure.log` |
| `go test -race -tags 'integration fourval' ./test/integration -run '^TestB1Hybrid' -count=1 -v -timeout=900s` | 0, 15 named, zero SKIP | `raw/race_b1_4v_closure.log` |
| `go test -race -tags integration ./internal/node -run '^TestD026' -count=1 -v -timeout=300s` | 0, 28 named, zero SKIP | `raw/race_d026.log` |
| B0 four-validator/M7 race regression | 0, 27 named, zero SKIP | `raw/race_b0_4v_m7.log` |
| `go test github.com/cloudflare/circl/sign/mldsa/mldsa44 -run '^TestACVP$' -count=1 -v` | 0 | `raw/circl_acvp_closure.log` |
| `go test ./internal/b1state -run '^$' -fuzz '^FuzzHybridDecode$' -fuzztime=15s -parallel=2` | 0 | `raw/fuzz_decode_closure.log` |
| `go test ./internal/b1state -run '^$' -fuzz '^FuzzHybridEvaluate$' -fuzztime=15s -parallel=2` | 0 | `raw/fuzz_evaluate_closure.log` |
| `git diff --check`, `git diff --cached --check` | 0 | `raw/diff_check_closure.log`, `raw/diff_cached_check_closure.log` |
| `go test -count=100 -run '^TestHybridABCIProposalAndCommit$' ./internal/b1app` | 0 after CI cleanup correction | `raw/b1app_cleanup_100x.log` |
| `go test -race -count=20 -run '^TestHybridABCIProposalAndCommit$' ./internal/b1app` | 0 after correction | `raw/b1app_cleanup_race_20x.log` |
| `make verify`; `go test -count=1 ./...`; `make test-integration-b1-4v` | 0 each after correction; 15 B1 named, zero SKIP | `raw/make_verify_cleanup.log`, `raw/go_test_cleanup.log`, `raw/b1_integration_cleanup.log` |

### Hosted CI correction retained

The initial source commit `e0b33751e5cca0337ee8e623979f09ac52ccc872` passed both complete hosted push/PR runs. After the first documentation-only commit, [PR run 40268983466](https://github.com/the-phong/DoS-in-Blockchain-PQC/actions/runs/40268983466) failed during `make verify`: `TestHybridABCIProposalAndCommit` reported `TempDir RemoveAll cleanup: ... directory not empty`; later integration steps were SKIPPED and are **not** counted as passes. The test had `defer a.Close()` before reassigning `a` to a reopened Pebble-backed application, so the deferred call closed the original object again and leaked the reopened one until temporary-directory cleanup. The correction registers a cleanup closure that closes the current reopened app and checks the close error. No production code changed. The failed log is retained at `raw/hosted_doc_pr_failed.log`; the correction passed 100 targeted normal runs, 20 targeted race runs, `make verify`, all default-tag tests, and the full B1 four-validator integration suite locally. Final hosted CI must run on the corrected HEAD before Controller acceptance.

## Git closeout and Controller handoff

- Corrected tested source commit: `fa5c95ba6a2a0f4489796c4248e6265f27cbcc89`; tree: `3045db9db16837687ec2e55c0db4689797900c22`. Its behavior delta from the initial source commit is only the B1 test cleanup and refreshed source SHA-256 manifest.
- [Changed-file manifest](../evidence/phase_04/CHANGED_FILES.txt) and [source-review patch text bundle](../evidence/phase_04/SOURCE_REVIEW.patch.b64) connect base `007e827f740c9c741f47bf1f7a51eec98a79cc5a` to the corrected source/test/spec/governance paths. Decode with `base64 -d research/evidence/phase_04/SOURCE_REVIEW.patch.b64 > /tmp/pluto-phase04-source.patch`; the restored patch SHA-256 is `0fb7c0ff3cd7508bdac5e477b42213f96d69a07a3114ceea793076f0bef956e0`.
- [Changed-source SHA-256 manifest](../evidence/phase_04/SOURCE_SHA256.txt): `55bb7c3d5b49e9609a0170554b744e9db7859e401b3671653afc32e15113ceb5`. The governance manifest's source-commit field and closeout evidence are excluded from this changed-source file-hash list; the exact corrected Git tree authenticates their contents at that commit.
- [Publication-safe evidence audit](../evidence/phase_04/EVIDENCE_AUDIT.md) and [raw-evidence SHA-256 manifest](../evidence/phase_04/RAW_SHA256.txt) identify 338 locally retained files without publishing runtime homes or private keys. Raw manifest SHA-256: `2cad13dd205caabb9473cd0b223401f7a968d6cbecc11c054b2bb39e95a1cfc4`; `sha256sum -c` returned 0.
- Independent hosted CI on initial source commit `e0b3375…`: [push run 40267930032](https://github.com/the-phong/DoS-in-Blockchain-PQC/actions/runs/40267930032) and [Draft PR run 40267954738](https://github.com/the-phong/DoS-in-Blockchain-PQC/actions/runs/40267954738) both completed **SUCCESS**. Each ran `make verify`, `make test-integration`, `make test-integration-4v`, and `make test-integration-b1-4v` successfully, with 71 `=== RUN` entries and no unintended SKIP/FAIL/data race. The subsequent PR failure and correction are disclosed above. Corrected final HEAD hosted CI is pending.
- Draft PR: [#9](https://github.com/the-phong/DoS-in-Blockchain-PQC/pull/9), DRAFT, base `test-mainet`, head `phase/04-b1-hybrid-authentication`.
- The corrected source commit changes test cleanup and its source checksum only. The final documentation/evidence commit and tree are reported in the Controller handoff and PR because a commit cannot contain its own hash.
- No merge or Phase 05 work is authorized. STOP after Controller handoff; Owner controls acceptance and merge.
