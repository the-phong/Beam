# Phase 03D — D-025 genesis transport checkpoint report

**Date:** 2026-10-24 (Asia/Bangkok). **Status:** local checkpoint candidate for independent Controller review; the remainder of Phase 03D is gated. Phase 03E is unauthorized.

## Source and scope

- Integration baseline and local branch parent: `test-mainet` at `b56c81ccce7ea4a0e7f52053c7459c4601c15b38`.
- Working branch: `phase/03d-b0-abci-integration`; the working tree was clean before edits and the remote integration branch matched the required baseline.
- No branch was merged or pushed in this checkpoint. The optional local checkpoint commit was not made: this workspace has no configured authorized Git author identity (`git var GIT_AUTHOR_IDENT` exits 128). All changes remain staged on the existing branch for review.
- D-025 is Owner-accepted. D-020–D-024 remain accepted. The Genesis V2 binary, account semantics, AppHash V2 and D-024 Pebble layout did not change.

## Implementation and transport boundary

`plutod init-b0 --genesis-file <canonical-binary>` is an explicit fresh-home profile. It validates the D-021 binary with `b0state.ParseGenesis`, converts the complete bytes `G` to `json.Marshal(base64.StdEncoding.EncodeToString(G))`, and places that JSON value in CometBFT `genesis.json` as `app_state`. It records the binary's SHA-256 in Pluto metadata for a startup consistency check. Historical `plutod init` remains the Phase 02 profile.

`internal/b0genesis.Decode` accepts a single JSON string with the canonical Go JSON string spelling and standard padded Base64. It permits JSON-defined whitespace around the value but rejects any second value, alternate JSON escapes, alternate Base64 alphabets, bad padding and nonzero pad bits. It bounds the raw input before parsing and re-encodes both the Base64 and JSON spelling to check canonicality. The decoded binary is passed to the existing `b0state.ParseGenesis`; there is no second binary parser. The decoded chain ID must match the `InitChain` request and configured B0 profile.

The maximum D-021 binary size is `8+4+2+50+4+28*65,536 = 1,835,076` bytes. Standard padded Base64 has `4*ceil(1,835,076/3) = 2,446,768` characters. The canonical JSON string is 2,446,770 bytes. The decoder's input cap is 2,446,834 bytes, allowing at most 64 bytes of surrounding JSON whitespace; inner escapes are rejected. Tests construct a maximum-count, maximum-chain-length genesis and confirm these limits.

`internal/b0app` is a separate, genesis-only ABCI++ adapter. `InitChain` durably creates the D-024 B0 store at height zero, returning its V2 AppHash. An identical height-zero retry is idempotent; conflicting binary, chain ID or genesis hash fails. An existing incomplete database fails closed. `Info` reads only committed state. The checkpoint proposes and finalizes empty blocks so pinned CometBFT can start and restart; it rejects all transactions and does not expose B0 account RPC. This temporary checkpoint behavior is **not** the accepted final B0 transaction contract and must not be used as evidence for it.

The B0 startup path refuses a V1 application database and refuses to recreate a missing B0 database when CometBFT persistence is present. It verifies the genesis transport and binary digest before opening the B0 store. Reopening an existing B0 snapshot uses Phase 03C's integrity checks. A failed or incomplete initialization is not automatically cleaned or retried; an operator must inspect the disposable home. There is no shared atomic commit with CometBFT, and this checkpoint does not establish cross-store crash recovery.

## T1–T10 evidence

| Test | Expected | Observed result | Status |
| --- | --- | --- | --- |
| T1 binary ↔ JSON/Base64 | Exact original binary after decoding | `TestGoldenTransportAndState` and `TestMaximumLengthCalculation` passed; independent Python transport length 78 and exact hex matched | PASS |
| T2 pinned real Comet transport | `InitChain.AppStateBytes` equals JSON/Base64 `app_state` bytes | Real `cometbft v0.38.26` child invoked a capturing ABCI adapter; it observed exact 78 bytes, distinct from raw binary; verbose real-node test passed | PASS |
| T3 negative transport and binary | Reject types, extra values, invalid/noncanonical Base64, oversized and malformed binary | `TestTransportRejectsAlternates` and `TestMalformedBinaryThroughTransport` passed, including unknown version, bad length, duplicate/unsorted allocations, zero balance and supply overflow | PASS |
| T4 genesis state | Height 0, balance 100, nonce 0, supply 100, correct AppHash | Go adapter initialization, durable reopen and Python oracle agreed on `e480c8b300c5ac1d662628f90334aae185ad38ed791c59ab14e3aa42abf083da` | PASS |
| T5 independent instances | Same genesis produces same state/hash | Two separate temporary Pebble stores yielded identical states/AppHashes | PASS |
| T6 retry/conflict | Identical height-zero retry succeeds; conflict fails | In-memory and reopened retries passed; changed genesis and request chain/height rejected | PASS |
| T7 interrupted initialization | No silent reset or false success | Simulated incomplete database directory was refused; missing B0 store beside persisted Comet data was refused. Actual process interruption during genesis write was not injected | PARTIAL |
| T8 Phase 02 regression | Historical profile still starts and rejects transactions | `make test-integration` passed the existing Phase 02 real-node suite and the new B0 test | PASS |
| T9 accepted rules/vectors | D-020–D-024 and binary golden artifacts unchanged | Source diff check passed for transaction/state/store/runtime packages, dependencies, historical vectors and two untouched accepted specs; only two explicit D-025 transport amendments were made | PASS |
| T10 research isolation | Transport outside admission/block hot path | `CheckTx` checkpoint test rejected arbitrary bytes without modifying committed state; code inspection shows transport calls only in initialization/startup. No performance measurement was made | PASS within checkpoint scope |

## Independent reference values

The Python standard-library calculation in `research/evidence/phase_03d/d025/reference_transport.py` parses the preserved Phase 03C vector bytes independently of Pluto Go code and computes the D-025 wrapper and V2 logical-state hash. It reported binary SHA-256 `d41a0d212d2af0b572b4bed1190a43f50a4e21034d8eaa4927e4c1c7251e46e8`, transport length 78 and AppHash V2 `e480c8b300c5ac1d662628f90334aae185ad38ed791c59ab14e3aa42abf083da`. The preserved `state_vectors.json` SHA-256 was `3d8491b174f44ae51c99796d48c4001ba56e86e008c5dbd08f7d1b68007d69f7`. The Phase 03C independent oracle also passed all its existing state vectors.

## Verification

Full exact command output, exit codes, environment, versions, branch and source revision are in [D-025 verification.log](../evidence/phase_03d/d025/verification.log); the reproducible runner is [run_verification.sh](../evidence/phase_03d/d025/run_verification.sh). On WSL2 Linux/amd64, Go 1.24.1 and Python 3.10.12, these commands all returned exit code 0:

- `go build ./...`
- `go test -count=1 ./...`
- `go vet ./...`
- `make fmt-check`
- `go mod verify`
- `make verify`
- `make test-integration`
- `go test -tags integration ./test/integration -run TestPinnedCometGenesisTransportAndB0Restart -timeout 120s -count=1 -v`
- `go test -count=1 -v ./internal/b0genesis ./internal/b0app`
- `go test -race -count=1 ./internal/b0genesis ./internal/b0app ./internal/node ./cmd/plutod`
- Both Python reference scripts, preserved-vector SHA-256, `git diff --check` and the source preservation diff check.

Pinned dependencies remained CometBFT `v0.38.26`, Pebble `v1.1.5` and secp256k1 `v4.4.1`; `go.mod` and `go.sum` did not change. Hosted CI was not run for this local-only checkpoint. No new benchmark, power-loss, WAL-interruption or B0 transaction execution claim is made.

The staged diff passed `git diff --cached --check`. The Git identity check and staged status are recorded in [git_checkpoint.log](../evidence/phase_03d/d025/git_checkpoint.log). There is no result commit SHA or PR for this local-only checkpoint; the branch HEAD remains the verified source SHA.

## Remaining risks and next gate

T7 covers controlled incomplete-state and missing-store conditions, not all interruption boundaries. This checkpoint does not test B0 V2 cross-store recovery, M1–M12, R2 resource costs, transaction validity, duplicate-execution protection or four-validator behavior. The existing Phase 03C risk addendum and Phase 03D G1–G8 gates remain open. Controller acceptance of D-025 transport and initialization is required before the separately gated remainder of Phase 03D proceeds. Phase 03E remains unauthorized.

## Controller R-025 rework addendum — 2026-10-24

This addendum records the bounded rework after Controller source inspection. The original T1–T10 table and verification log above remain historical evidence for the first staged snapshot; the new results are in the external `/tmp/pluto_phase03d_d025_rework_verification.log`, produced by the staged `research/evidence/phase_03d/d025/run_rework_verification.sh`. The final staged tree ID and patch checksum are recorded in the external Controller handoff because storing either inside the staged patch would change its identity.

### R-025-01 and R-025-02

The D-025 decoder now checks `len(raw)-len(trimJSONWhitespace(raw)) <= 64` in addition to the total byte cap. JSON whitespace remains exactly SP, HTAB, LF and CR. Small and maximum-size Genesis V2 fixtures each test 64 accepted and 65 rejected surrounding whitespace bytes. All canonical JSON/Base64 and binary parser checks remain in place.

`FinalizeBlock` now holds the application write lock while reading the committed height, checking the requested next height and calling the store's pending-state `Prepare`. It does not call an application method that reacquires the RWMutex. Tests cover invalid, stale and skipped heights, a duplicate pending finalization, pending invisibility before Commit, advancement after Commit, eight concurrent duplicate callbacks and both valid orderings of competing FinalizeBlock/Commit callbacks. The concurrency tests are evidence of callback behavior; the source-level lock scope supplies the logical atomicity argument.

### R-025-03 — Genesis identity and current trust boundaries

At B0 startup, Pluto metadata contains SHA-256 of the canonical genesis binary; `runB0` compares it with the currently configured `genesis.json` transport after strict decoding. Both files are mutable local configuration, so this comparison detects one-sided change, not coherent replacement of both. A real-node negative test changes only the genesis allocation after blocks have committed and observes startup rejection from the metadata digest check.

`b0store.OpenExisting` validates the complete V2 snapshot and its AppHash. `Info` reports that durably committed height/hash. AppHash V2 commits the logical state at its height; it does not commit the original genesis bytes, CometBFT validator set or consensus parameters. At height zero, `b0app.Open` compares the recovered hash with the configured genesis hash. At height greater than zero, this **genesis-only, empty-block** checkpoint can compare recovered accounts and total supply with genesis because no transfers are possible; a negative test confirms rejection of a different same-chain allocation at height one. This guard must be re-evaluated before transaction execution is activated, when balances can legitimately differ from genesis.

Pinned CometBFT v0.38.26's handshake reads `Info`, invokes `InitChain` if the application reports height zero, and compares or replays application hashes against its persisted consensus/block state according to `consensus/replay.go`. The current tests observe normal restart and a one-sided genesis-file mismatch. They do **not** execute all height/hash mismatch or cross-store recovery paths. Neither chain ID nor a later AppHash alone proves genesis provenance. Coherent replacement of config, metadata and stores, validator/consensus-genesis changes with unchanged allocations, and later transfer histories remain outside this checkpoint's verified identity boundary.

### R-025-04 — Missing-data behavior

Fresh `init-b0` creates a new home with canonical config/genesis but no B0 state record; first start calls `InitChain` and persists height zero. A missing B0 database beside `state.db`, `blockstore.db` or `cs.wal` is rejected before the ABCI app starts; the real-node test verifies this after committed blocks. In a disposable-home test, removing the entire application and CometBFT `data/` directory under retained config/genesis caused pinned CometBFT to exit because `priv_validator_state.json` was absent; no new B0 snapshot was created. **The Pluto missing-store guard itself cannot distinguish that complete loss from first startup.** CometBFT's observed file error is not a durable B0 recovery policy and does not prove that every variant of complete data loss fails closed.

Proposed separately reviewable policy: initialize and synchronously validate the B0 height-zero record during the explicit `init-b0` operation, then require that record on every `start`; `InitChain` would verify the identical record idempotently. This would change the initialization boundary and requires Controller/Owner review and crash-boundary tests before implementation. No persistent layout, fingerprint or consensus rule was changed in this rework.

### T7 interruption matrix after rework

| Boundary | Injection and observed state | Result |
| --- | --- | --- |
| Before a B0 record exists | Create an empty B0 database directory, then attempt open; no process killed and no complete record existed | Fail closed; no height-zero recovery |
| After `InitChain` returns from D-024 synced genesis Set in a standalone test subprocess | Subprocess writes a marker after `InitChain` returns; parent sends SIGKILL while Pebble remains open; parent reopens the store and retries identical `InitChain`. No CometBFT connection or ABCI network response participates. | Complete validated height-zero state recovered; retry succeeded within this test boundary |
| Existing committed chain, B0 database path removed | Gracefully stop node, move B0 database aside, then start pinned CometBFT with consensus data retained | Startup refused implicit reset; no recovery attempted |
| All local `data/` lost under retained config/genesis | Gracefully stop, move entire disposable data directory, then start | CometBFT exited for missing `priv_validator_state.json`; no B0 snapshot created; B0 first-start ambiguity remains |
| During Pebble directory creation, WAL append, fsync or any physical power-loss point | No deterministic fault hook or hardware interruption used | **NOT RUN** |
| CometBFT/application cross-store response loss, mismatch or replay | Outside this checkpoint; assigned to remaining Phase 03D M1–M12 and G3–G6 gates | **NOT RUN** |

T7 remains **PARTIAL**. SIGKILL after a completed synchronized write does not establish behavior during the write, arbitrary power-loss durability or atomicity across Pluto and CometBFT. Deferral of originally mandatory T7 interruption windows requires explicit Controller/Owner acceptance; this report does not grant it.
