# Pluto Phase 02R independent audit and governance report

Date: 2026-10-23 Asia/Bangkok (verification 2026-10-22 UTC). Status: **engineering review ready; controller and owner acceptance pending**. Authorized scope: audit, evidence, governance and Draft PR only. No runtime implementation changed and no later phase started.

## 1. Repository integrity and review basis

The initially clean working tree was on `test-mainet` at `0012ac53666dc3e541fcad431a946005058df1ef`. `git ls-remote origin refs/heads/test-mainet` independently returned the same SHA. Remote: `https://github.com/the-phong/DoS-in-Blockchain-PQC.git`. The audit branch `phase/02r-audit-governance` was created from that remote-tracking baseline after confirming it did not exist locally or remotely. No reset, clean, stash, force-push, merge or integration-branch push was used.

All 23 entries in the Phase 02 final source manifest match the Git objects at the source SHA. The two preserved Phase 01 historical report/evidence files match the Phase 01 baseline manifest. The Phase 01 pre-modification manifest is not expected to match the later Phase 02 working tree. The Phase 02 `EXECUTION_LOG.md` contains a transcription error for the Phase 01 command evidence checksum: its printed value omits `a5ed9577`; the actual file and `BASELINE_SHA256.txt` agree. Historical files were not changed. See [baseline and independent AppHash evidence](../evidence/phase_02r/baseline_and_apphash.txt).

The Phase 02 report and logs were treated as claims. Every Go file, test, Makefile, `go.mod`, `go.sum` policy, CI workflow, node configuration path and the pinned CometBFT v0.38.26 interface/replay code were inspected. Source files reviewed: `cmd/plutod/main.go`, `cmd/plutod/main_test.go`, `internal/app/app.go`, `internal/app/app_test.go`, `internal/node/node.go`, `internal/store/state.go`, `internal/store/state_test.go`, `internal/store/pebble.go`, `internal/store/pebble_test.go`, `internal/version/version.go`, `test/integration/node_test.go`. Build/workflow files: `Makefile`, `.github/workflows/verify.yml`, `go.mod`, `.gitignore`. The direct pins remain CometBFT v0.38.26 and Pebble v1.1.5; no dependency was upgraded. `example.com/pluto` remains a provisional module path.

## 2. Architecture and pinned API assessment

`plutod` dispatches CLI commands. `internal/node` invokes the adjacent CometBFT executable for init and node lifecycle, runs the localhost ABCI socket server, checks genesis/Pluto metadata and opens the application database. CometBFT owns consensus, P2P, RPC, block storage and mempool. `internal/app` implements all methods of the pinned `abci.Application` interface. `internal/store` defines a fixed state encoding and owns a Pebble database. No Cosmos SDK, EVM adapter, account state, transaction authentication or admission optimization is implemented.

The pinned source inspected was `github.com/cometbft/cometbft@v0.38.26/abci/types/application.go`, `consensus/replay.go` and `state/execution.go`, alongside this repository's code. In this API, `ResponseFinalizeBlock.AppHash` is the new application hash; `ResponseCommit` does not return a hash. CometBFT checks one result per block transaction. Zero transactions and zero `TxResults` meet that invariant. The CometBFT replay code has separate app-behind and app-ahead cases and checks state hashes; those paths are an upstream mechanism, not proof that every Pluto power-loss interleaving recovers.

`init` validates chain ID and distinct port numbers, refuses an existing home, uses the bundled CometBFT init to generate genesis and signing keys, then sets initial height 1, empty app state, localhost listeners and one-second empty-block interval. Partial init leaves the home for inspection. `start` checks genesis chain ID/height/app state against Pluto metadata and refuses a missing Pluto database beside CometBFT state. It starts ABCI before the child and, on graceful cancellation, waits for the child before stopping ABCI and closing Pebble. A child exit and several setup failures return errors. Server-stop and database-close errors are ignored on deferred shutdown, and process termination after the ten-second timeout is best effort.

The locality guarantee is configuration dependent: startup does not compare current CometBFT `config.toml` listener/proxy addresses with Pluto metadata or enforce localhost again. Replacing the adjacent `cometbft` binary is likewise not checked against the module pin at runtime. These are operational integrity risks for a development node; do not expose it publicly. Generated key files are excluded from evidence and commits.

## 3. ABCI++ lifecycle and state correctness

| Method | Phase 02 observed implementation | Review assessment |
|---|---|---|
| `Info` | Returns last committed height/hash; before init, height zero and no hash. | Matches handshake use and does not expose pending state. |
| `InitChain` | Requires height 1, empty app state and valid chain ID; sync-writes genesis at height 0; identical retry is accepted. | Retains CometBFT genesis validators/parameters; rejects conflicting committed state. |
| `CheckTx` | Always returns code 1. | Fail-closed admission; no transaction state changes. |
| `PrepareProposal` | Returns an empty tx list at the next height. | Drops even supplied candidate transactions. |
| `ProcessProposal` | Accepts exactly the next height with no txs, otherwise rejects. | Independently checks the proposal; does not rely on local `CheckTx`. |
| `FinalizeBlock` | Rejects nonempty txs, bad heights and duplicate pending blocks; computes next hash in memory. | Does not report unsupported execution as success or persist before `Commit`. |
| `Commit` | Encodes pending state, `Set(..., pebble.Sync)`, then promotes pending to committed. | Correct in-process boundary. Pre-write validation failure leaves committed state unchanged; actual I/O failure was not injected. |
| `Query` | Unsupported code 1. | No false application query success. |
| Snapshot methods | No listed snapshots; offers reject; load errors; apply aborts. | State sync unavailable. |
| Vote extensions | `ExtendVote` errors and verification rejects; not enabled by init. | Consistent with the current genesis, but enabling extensions would require application work. |

The canonical state is `(version=1, ASCII chain ID 1–50 bytes, nonnegative int64 height)`. AppHash is SHA-256 of `PLUTO/APP_STATE/V1\0 || uint32_be(1) || uint16_be(chain_id_length) || chain_id || uint64_be(height)`. The persisted record contains `PLUTO02!`, the same ordered fields and the 32-byte digest. Decode checks exact length, magic, version, chain ID, nonnegative height and recomputed hash. `Next` rejects `MaxInt64` overflow. The independent Python calculation in the evidence file reproduces heights 0–4, including the observed real-node hashes. This is a fixed Phase 02 digest, not an Ethereum state root or general account commitment.

A block header at height H records the AppHash of state after H-1. The test's height-2 header has the height-1 hash `2feedf…c8a4`, while `/abci_info` at application height 2 has `eeb423…f8232`. This is the expected protocol relationship. The two-instance unit test checks identical ordered empty-block transitions. It is not a four-validator or Byzantine test. The height comparison `committed.Height+1` can wrap at `MaxInt64`; `FinalizeBlock` itself rejects overflow through `Next`, but proposal methods do not guard this unreachable-in-practice boundary explicitly. Record for future state-transition review; no runtime patch is authorized in 02R.

Phase 03 work will require a specified canonical transaction, independent proposal validation, one `TxResult` per transaction in `FinalizeBlock`, deterministic account/nonce/balance transitions, a versioned AppHash/state migration and tests for replay and error semantics. `CheckTx`, `PrepareProposal`, `ProcessProposal`, `FinalizeBlock`, `Commit`, `Query`, state encoding and the test harness will need revision under separate authorization. `CheckTx` cannot be the sole validity gate.

## 4. Persistence, recovery and operations

Pebble uses one key for the complete committed record and an exclusive writable database lock. A synchronous WAL write is requested before `Commit` reports success. `FinalizeBlock` without `Commit` is absent after reopen. Graceful restart was observed through the real child process, ABCI handshake, independent state read and later blocks. The test also verifies that a missing Pluto database beside CometBFT state is rejected. It uses temporary homes only.

This does not establish crash safety under arbitrary power loss, torn or corrupted CometBFT/application stores, failed filesystem sync, every block replay ordering, or a database fault. The unit Save-failure test corrupts pending state so validation fails **before** Pebble writes. CometBFT and Pluto have separate durable stores and rely on replay/handshake to reconcile; an application-ahead or data-loss case may halt. No state sync or snapshots exist. The node is single-validator, so it establishes neither multi-validator AppHash agreement nor hostile-network security.

## 5. Independently observed verification

Environment: Go 1.24.1 linux/amd64, GNU Make 4.3, Git 2.34.1, WSL2 Ubuntu kernel `6.18.33.2-microsoft-standard-WSL2`, mounted Windows filesystem. Exact fresh command output, UTC timestamps and exit codes are in [verification.txt](../evidence/phase_02r/verification.txt). The commands ran at source SHA `0012ac53666dc3e541fcad431a946005058df1ef` on the audit branch before documentation changes.

| Exact command | Exit | Relevant observation |
|---|---:|---|
| `go version` | 0 | `go version go1.24.1 linux/amd64` |
| `go build ./...` | 0 | All Pluto packages built. |
| `go test -count=1 ./...` | 0 | CLI, app and store passed; node/version have no direct tests. |
| `go vet ./...` | 0 | No diagnostics. |
| `make fmt-check` | 0 | No unformatted Go source. |
| `go mod verify` | 0 | `all modules verified`. |
| `make verify` | 0 | Built both executables; unit, vet, format, module and CLI version checks passed. |
| `make test-integration` | 0 | Real node initialization, blocks and graceful restart passed. |
| `go test -race ./internal/app ./internal/store` | 0 | Both packages passed under race detector. |
| `go test -cover ./internal/app ./internal/store` | 0 | App 77.5%; store 88.2% statement coverage. |
| `go test -tags integration ./test/integration -timeout 120s -count=1 -v` | 0 | One real-node test passed; one validator, zero txs, height 2 then recovered height 2 and height 4. |

No command in this list was skipped. The test contains a conditional skip only for an ephemeral-port collision; it did not occur in the observed run. After documentation edits, `git diff --check`, `make verify`, `make test-integration` and a runtime-source unchanged check all exited 0; see [post-documentation verification](../evidence/phase_02r/post_documentation_verify.txt). GitHub's push and PR `verify` checks on the first audit commit both completed successfully; the workflow ran `make verify` and `make test-integration`. Workload seeds and benchmark hardware measurements are not applicable: no workload or benchmark was run. Test timing and coverage are diagnostic, not performance or correctness proofs.

## 6. Findings and corrective work

- **High operational risk:** `config.toml` may be edited after init to broaden RPC/P2P exposure or change `proxy_app`; startup does not validate it against Pluto metadata. Reproduction: inspect `internal/node/node.go:Run` and compare checks of `genesis.json` and `pluto.json` with the absent check of current `config.toml`. Proposed future correction: validate effective listeners/proxy and bundled binary identity at startup, or document an explicit operator trust boundary. Do not run with unreviewed config.
- **High governance risk:** GitHub's branch-protection API returned HTTP 404, `Branch not protected`, for `test-mainet` on 2026-10-23. Phase discipline currently relies on operator behavior. The owner should require PR reviews and verification checks, restrict direct/force pushes and retain merge authority. No repository administration setting was changed in 02R.
- **Medium recovery uncertainty:** no injected actual Pebble I/O failure, forced-kill or power-loss interleaving test. Proposed separately authorized work: isolated temporary-home fault/replay matrix across application and Comet stores, with exact expected handshake and halt behavior.
- **Medium lifecycle uncertainty:** deferred ABCI stop and Pebble close errors are not surfaced; partial initialization is not rolled back. Proposed correction after authorization: aggregate shutdown errors and improve actionable diagnostics without destructive cleanup.
- **Medium security limit:** generated validator/node keys and a single validator make this a localhost research node only. Multi-validator agreement, state sync and public hardening are absent.
- **Low edge case:** proposal height addition has no explicit `MaxInt64` guard; `Next` rejects at finalization. Address alongside Phase 03 state rules if authorization permits.
- **Low evidence defect:** the historical Phase 02 execution log transcribes one Phase 01 checksum incorrectly. The baseline manifest and preserved file match; retain the original log and cite the correction here.
- **Low dependency risk:** pins are reproducible but the review did not establish ongoing advisory freedom or audit all transitive dependencies. Reassess before public deployment or new dependency adoption.

No confirmed Phase 02 empty-block runtime defect was changed in this audit. The local test evidence supports correctness only for the exercised protocol and graceful-restart paths.

## 7. Governance and acceptance gates

D-018 prospectively replaces Phase 02's local-only workflow with one branch and Draft PR per authorized phase, owner-controlled merges and separate phase approval. D-019 makes B0-Core completion precede a conditional Phase 04 EVM decision. `AGENTS.md`, `MASTER_PLAN.md`, `DECISIONS.md`, `PHASE_STATUS.md`, `ARCHITECTURE.md`, `README.md` and the manifest were reconciled. Historical Phase 01/02 reports and original evidence are unchanged. The owner should configure the currently unprotected `test-mainet` branch; no repository administration change was made in this audit.

| Gate | Status | Evidence / limit |
|---|---|---|
| G1 repository integrity | PASS | Clean source SHA, remote SHA and 23/23 source manifest match; historical files preserved. |
| G2 implementation review | PASS | Every Go source/test plus pinned ABCI/replay and build/config paths inspected. |
| G3 build | PASS | `go build ./...` and `make verify` exit 0. |
| G4 unit/static | PASS | Uncached unit tests, vet, format and module verification exit 0. |
| G5 runtime | PASS at tested scope | Real single-node restart test exits 0; power-loss recovery unverified. |
| G6 state correctness | PASS at tested scope | Encoding review, independent hashes, pending/committed unit behavior. |
| G7 security assessment | PASS | Localhost/key/configuration/dependency/multi-validator risks documented; no unsupported assurance. |
| G8 architecture preservation | PASS | No runtime/dependency/consensus/transaction/EVM/PQC source changes. |
| G9 governance | PASS | D-018, D-019 and current authority/roadmap reconciled. |
| G10 reproducibility | PASS | Exact source SHA, versions, environment, commands, outputs and limits in evidence. |
| G11 Git isolation | PASS | Branch pushed; Draft PR #2 targets `test-mainet`; integration branches unchanged. No merge. |
| G12 phase discipline | PASS | No Phase 03A implementation; explicit STOP. |

The gates make Phase 02R **eligible for independent controller review**, not owner acceptance. A future critical failure or repository rule rejection must revise the verdict. No approval for Phase 03A or Phase 04 follows from these results.

## 8. Git publication and final disposition

Working branch: `phase/02r-audit-governance`; source commit: `0012ac53666dc3e541fcad431a946005058df1ef`; first audit commit: `587a5164fba8847e2b7e03bf56fa7c6919faba9f`. Push: **SUCCESS**, branch only. Draft PR: <https://github.com/the-phong/DoS-in-Blockchain-PQC/pull/2>, base `test-mainet`, open and unmerged. The final HEAD SHA is recorded in the Phase 02R chat report and PR after this publication-metadata update; a commit cannot embed its own final SHA in its contents. Merge: **NO**. Controller/owner review is the next action. Phase 03A authorization: **NOT GRANTED**. Execution stops after the final report.
