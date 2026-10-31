# Pluto Phase 03C implementation report — Controller review candidate

## Authority, baseline and scope

The Owner authorized only Phase 03C. The clean pre-branch `origin/test-mainet`
HEAD matched `61bc65fdebf358fb0330cb9ebd4a332ca4a06aca`; the Phase 03B
post-merge GitHub push and pull-request checks both succeeded. This work is on
`phase/03c-b0-state-persistence`. The result commit and Draft PR URL are in
the final agent report and PR; a commit cannot contain its own SHA. D-020–D-023
and the four accepted B0 V1 specifications are unchanged. Phase 03D is not
authorized.

The new `internal/b0state` and `internal/b0store` packages are isolated from
the live Phase 02 application. `internal/app`, `internal/node`, V1
`internal/store`, `cmd/plutod`, current genesis, CometBFT integration and
AppHash V1 are unchanged. The running node still rejects every transaction.
The new code is a standalone state-level simulation and persistence adapter,
not ABCI integration or a claim of a complete B0-Core chain.

## Genesis, logical state and transition API

`EncodeGenesis` and `ParseGenesis` enforce the exact D-021 binary format:
`PLUTOG2! || u32be(2) || u16be(N) || chain_id[N] || u32be(K) ||
K*(address[20] || u64be(balance))`. Chain ID is 1–50 permitted ASCII bytes;
K is 1–65,536; addresses are strictly ascending and unique; every allocation
is positive; the supply sum is checked. Parse rejects wrong chain, version,
length, ordering or trailing bytes before allocating the account array.
Genesis state has height zero and nonce zero for each account. Minimum and
maximum chain-length vectors are recorded.

`State` encapsulates version 2, chain ID, height, supply, sorted accounts and
AppHash; account reads return copies. `NewState` and `DecodeLogical` reject
out-of-range heights, account counts, unsorted/duplicate addresses, supply
overflow or mismatch, wrong chain and mismatched digest. Empty-block advance
checks MaxInt64 and changes only height/hash. Missing accounts are virtual
zero-balance, zero-nonce accounts; successful funding creates an account;
zero-balance senders remain serialized.

`Evaluate` delegates accepted structural/chain/amount/scalar/point/ECDSA
checks to `internal/tx` and then applies nonce/overflow Code 6, sender balance
Code 7, receiver overflow Code 8 and account cap Code 9. It returns Code 0 on
success. It handles self-transfer without intermediate credit and checks
sender sufficiency and nonce first. A failed transaction returns the original
immutable state and consumes no nonce. `SimulateOrdered` applies each success
in input order and continues after failures. The account-cap test uses an
integrity-valid 65,536-account state. Code 8 is a defensive checked branch;
it is mathematically unreachable from a valid conserved-supply state with a
funded different sender. No invalid state was presented as a real ledger
fixture. No ABCI response is fabricated.

## AppHash V2 and independent reference

The exact logical bytes are `u32be(2) || u16be(N) || chain || u64be(height) ||
u64be(total_supply) || u32be(K) || sorted account[20+8+8]`. The commitment is
`SHA-256(ASCII("PLUTO/B0/STATE/V2") || 0x00 || logical_bytes)`. Physical Pebble
layout does not enter this digest. The [vector artifact](../evidence/phase_03c/state_vectors.json)
records complete genesis wires, exact Phase 03B transfer wire, self-transfer
wire, full logical bytes, accounts, negative cases and hashes. Only public
**TEST ONLY** fixture scalars A=1 and B=2 are used. The separate
[Python oracle](../evidence/phase_03c/reference_oracle.py) reconstructs all
genesis and state bytes with `struct`, hashes with `hashlib`, derives fixture
addresses independently and verifies transaction signatures through
OpenSSL-backed `cryptography` with `Prehashed(SHA256)`. It calls no Pluto Go
library. The [Go generator](../evidence/phase_03c/generate_vectors.go) is
reproducible; `go run ... | cmp - state_vectors.json` passed.

| State | Independent AppHash V2 |
|---|---|
| Genesis, chain `pluto-b0-v1` | `e480c8b300c5ac1d662628f90334aae185ad38ed791c59ab14e3aa42abf083da` |
| Post-transfer, height 1 | `c735447cd853aed15e488a349710e96284640994c9da0158478f19fee293982e` |
| Post-self-transfer, height 2 | `9e512d901ea0137a89bfab07a5b03505540f98bf5fe7fc86a680ae90b4f7caec` |
| Post-empty-block, height 3 | `7822225ce3300e3be2e954828c6f7944c669b16f16ecc4ddc149108e75fad024` |
| One-byte chain `c` genesis | `d6c802d654b16be9a8a1d7ef92206c3b0e8f7b77595f4321248c189e97f65b01` |
| Fifty-byte chain `c*50` genesis | `5df773851ea3aa65afc4235fd07fc501b6d2cf08e601a01a90a5414dfb20d488` |

Serialization, hashing and full snapshot writing cost O(K) for K accounts.
Future B0/B1 comparisons must control account count and execution model.

## Proposed D-024 physical layout and lifecycle

D-024 proposes an explicitly separate B0 database directory and one Pebble
key `pluto/committed-state/v2` containing
`PLUTOS2! || canonical_logical_state || AppHashV2`. It is **proposed, not
Owner-accepted**. `Create` requires an absent directory and valid genesis;
repeat initialization fails. `OpenExisting` requires a full, integrity-valid
snapshot, rejects a V1 key, and performs no migration/reset. One
`pebble.Set(..., pebble.Sync)` replaces the entire committed record. A mutex
serializes committed/pending reads, candidate preparation and commit. An
isolated pending candidate becomes visible as committed only after successful
synchronous persistence. Duplicate/stale Prepare or Commit is rejected.
Details and API contracts are in
[STATE_API_AND_LAYOUT.md](../evidence/phase_03c/STATE_API_AND_LAYOUT.md).

Temporary-directory tests show graceful restart after genesis, transfer and
empty-block commits; pending changes disappear on restart. A disposable
subprocess is killed after Prepare and after acknowledged Commit; reopening
recovers the complete old height-zero and new height-one states respectively.
Fault hooks cover
pre-write failure (old state recovered), Set returning an error without a
write (old state recovered), and a post-sync response error (new complete
state recovered). On Set or post-sync error, in-memory committed stays old
and the store enters an uncertain state requiring close/reopen. This models
both possible complete outcomes; a real I/O error is **not** assumed to leave
old bytes intact. Corruption tests reject wrong magic/version, truncated or
trailing records, wrong hash/chain, tampered balance/nonce, invalid count,
unsorted/duplicate addresses, overflowing supply and supply mismatch, even
when an attacker recomputes the digest for some malformed records. The
actual Phase 02 V1 key is rejected. Tests do not prove physical power-loss
durability, all process-kill timings or CometBFT/application cross-store
replay; no real node home was touched.

## Verification and evidence

Full commands, exact output, exit codes, UTC timestamps, source revision and
host description are in
[verification.log](../evidence/phase_03c/verification.log). The runner is
[run_verification.sh](../evidence/phase_03c/run_verification.sh). Environment:
WSL2 Linux/amd64, Go 1.24.1, Python 3.10.12, `cryptography` 3.4.8, existing
pinned CometBFT v0.38.26, Pebble v1.1.5 and secp256k1 v4.4.1. No dependency
or workflow change was made.
The verbose named state, boundary, corruption and process-kill results are in
[negative_boundary.log](../evidence/phase_03c/negative_boundary.log).

Commands: `go build ./...`, `go test -count=1 ./...`, `go vet ./...`,
`make fmt-check`, `go mod verify`, `make verify`, `make test-integration`,
`go test -race -count=1 ./internal/b0state ./internal/b0store`, separate Go
coverage runs and `go tool cover -func`, three 5-second bounded fuzz targets
(`FuzzGenesis`, `FuzzLogical`, `FuzzRecord`), vector regeneration comparison,
Python oracle, `git diff --check`, and a baseline source-diff check of live
runtime, dependencies, existing tests/workflows and accepted specs. The final
run exited 0 for every command. Statement coverage was 92.2% for `b0state`
and 92.9% for `b0store`. Bounded fuzzing is not an
exhaustive security proof. The first locally drafted fault test deliberately
closed Pebble behind an open Store and panicked when Commit used that invalid
test setup; it was replaced with a controlled returned-write-error seam.
Subsequent package tests and the full verification pass. This initial failed
test was a test-design error, not an observed production-store corruption.

## Review gates and residual risks

Evidence supports C1–C12 and C14 subject to Controller review: exact source,
canonical genesis, invariants, deterministic transitions, nonce/balance/cap,
failure atomicity, independent hash oracle, pending isolation, synced single
record, fail-closed corruption, graceful restart/fault simulation and live
runtime preservation. C13 requires the final hosted CI result in addition to
the local build/test/race/coverage/fuzz record. The final agent report records
that result and each gate explicitly. D-024 needs Owner approval. The
standalone store does not yet participate in ABCI or cross-store replay. Full
state writes are O(K), and no physical power-loss campaign has been run.
Those limits prevent claims about a live transactional chain or
crash-proof durability.

**STOP:** No merge, no Phase 03D work. Await Controller review and Owner
acceptance of Phase 03C and proposed D-024.

## Controller R1 rework addendum — 2026-10-23

The Controller reviewed `6459a1ad7ee5a17994f484c04f47658a976d9ea6`
and found that an uncertain Commit blocked Prepare and Commit but allowed
`Committed()` and `Pending()` to expose snapshots. After a post-sync response
error, the durable record may already be at the new height while the
in-memory committed value still represents the old height. Those reads could
therefore mislead a caller.

Both read methods now check `s.uncertain` under the existing read lock and
return `ErrUncertain` with no snapshot while the outcome is unresolved.
Closed-store errors retain their existing precedence. A BeforeSync error
does not set uncertainty, so the old committed snapshot and retryable pending
candidate remain readable. Returned Pebble write errors and AfterSync errors
continue to leave in-memory committed unchanged, block Prepare/Commit, and
require close/reopen. No ambiguous write is retried or inferred successful.

Existing fault tests now assert fail-closed reads for both error boundaries,
normal reads after successful Commit, pre-write readability, unchanged
internal committed/pending values, blocked operations and old/new complete
snapshot recovery on reopen. A new concurrent-reader regression coordinates
24 readers with a post-sync error and checks that no read bypasses the
uncertainty gate; a race-enabled repeated run exercises the same boundary.

The new commands, exit codes, race/coverage/fuzz results and independent
vector recheck are in
[rework_r1/verification.log](../evidence/phase_03c/rework_r1/verification.log).
The final local rework run exited 0 for every command, including targeted
regressions, `make verify`, real-node integration, package race tests, ten
race-enabled concurrent-reader repetitions, three bounded five-second fuzz
targets and the independent oracle. Statement coverage was 92.2% for
`b0state` and 93.1% for `b0store`. The original vector JSON SHA-256 remained
`3d8491b174f44ae51c99796d48c4001ba56e86e008c5dbd08f7d1b68007d69f7`.
The historical Phase 03C verification and vector artifacts above were not
rewritten. Accepted D-020–D-023 rules, Phase 02 runtime, cryptography,
dependencies, Genesis V2 and AppHash V2 remain unchanged. D-024 remains
**Proposed**, and Phase 03D remains unauthorized. The new result commit and
Draft PR #6 CI status are recorded in the Controller rework chat report.

## Controller risk and acceptance addendum — 2026-10-23

At the Owner's request, the [risk and acceptance addendum](../evidence/phase_03c/RISK_AND_ACCEPTANCE_ADDENDUM.md)
records Phase 03C's tested persistence boundary, the pinned CometBFT replay
contract, and a cross-store recovery matrix whose B0 V2 cases remain
unexecuted. It documents R2: `b0state.Evaluate` fully validates and hashes
K accounts before cheap stateless rejection, creating an analytical O(K)
cost per invalid call; no exploit or benchmark result is claimed. It defines
future K=1/64/4096/65536 measurements and mandatory Phase 03D activation
gates without implementing them. D-024 remains **PROPOSED — AWAITING EXPLICIT
OWNER ACCEPTANCE**. This addendum does not alter the historical test record,
accepted protocol specifications or live Phase 02 runtime; PR #6 remains
Draft and Phase 03D remains unauthorized.

## Owner acceptance and pre-merge closeout addendum — 2026-10-24

The Controller accepted the Phase 03C implementation, R1 correction and
[risk addendum](../evidence/phase_03c/RISK_AND_ACCEPTANCE_ADDENDUM.md). The
Owner accepted those deliverables and D-024 on 2026-10-24. D-024's separate
Pebble database and single `PLUTOS2!` complete-state record, replaced with
`DB.Set(key, value, pebble.Sync)`, are accepted only as a correctness-oriented
experimental B0 baseline. This acceptance does not merge Draft PR #6 or
activate B0 transaction handling in the Phase 02 node.

The prior proposal and review-candidate statements above are preserved as
historical execution records. R1's uncertain-commit fail-closed reads remain
required. R2's O(K) validation cost, the unexecuted B0 V2 cross-store matrix
M1–M12, future account-count benchmarks and Phase 03D safety gates G1–G8
remain open. No arbitrary-power-loss or production durability guarantee is
claimed. Phase 03D is not authorized. This closeout changes documentation
only; new command output is recorded separately under
`research/evidence/phase_03c_closeout/`.
