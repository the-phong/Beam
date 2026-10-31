# Pluto Phase 03B implementation report — review candidate

## Authority and source

The Owner explicitly authorized Phase 03B after accepting D-020–D-022 and
merging Phase 03A PR #3. `origin/test-mainet` was independently verified at
`54bf720ac4625c17eaa6101a8f398ef7c0663432`; the pre-branch tree was
clean. Work is isolated on `phase/03b-classical-tx-crypto`. The result commit
SHA and Draft PR are recorded in the final agent report and PR because a
commit cannot contain its own SHA. No merge is authorized.

## Implementation and protocol compliance

`internal/tx` is a standalone package. `Parse` and `Encode` implement exactly
`PLUTOTX1 || u16be(1) || u16be(N) || chain[N] || key[33] || receiver[20] ||
u64be(amount) || u64be(nonce) || r[32] || s[32]`, with `1 <= N <= 50` and
146–195 total bytes. Parsing is bounded before indexing and allocates at most
the 50-byte chain string; encoding rejects invalid chain IDs. Decoding and
re-encoding canonical input reproduces the bytes exactly.

`SigningPreimage` uses `PLUTO/B0/SIGN/V1\0 || u32be(len(U)) || U` and
`SigningDigest` takes one SHA-256. `Validate` applies accepted first-failure
precedence: structural Code 1, chain Code 2, zero amount Code 5, scalar Code 4,
compressed point Code 3, ECDSA Code 4, then Code 0 for **stateless success
only**. Scalar overflow/zero and high-S are rejected before library use;
ECDSA verifies the supplied digest without rehashing. Expected chain ID is a
caller argument. The API does not expose account or block validity.

`SenderAddress` validates the compressed point and takes the first 20 bytes
of the accepted domain-separated SHA-256. `TxID` exists only after exact
structural checks and canonical `r,s` checks; wrong chain, zero amount,
invalid key, nonverifying ECDSA and future state failure do not remove
eligibility. `CometHash` is raw-byte SHA-256 for any supplied bytes. The two
hashes have different domains and must be correlated using exact raw bytes
and, for block occurrences, height/index. No RPC feature was added.

The accepted Phase 03A specifications and D-020–D-022 were not changed.
`internal/app`, `internal/node`, `internal/store`, existing tests, Makefile
and workflow source are unchanged. The running Phase 02 node still rejects
every transaction. V2 accounts, genesis, AppHash and ABCI transaction handling
remain future phases.

## Dependency and independent vectors

The direct secp256k1 dependency is `github.com/decred/dcrd/dcrec/secp256k1/v4
v4.4.1`; its minimum Go is 1.17 and its only declared direct dependency was
already in the graph. The exact version upgrade, APIs, checksum and residual
risk are in [DEPENDENCY_REVIEW.md](../evidence/phase_03b/DEPENDENCY_REVIEW.md)
and D-023. D-023 is an implementation choice submitted for review; it does
not change the accepted protocol rules.

[`golden_vectors.json`](../evidence/phase_03b/golden_vectors.json) contains four exact signed fixture records: V01 valid
transfer, V11 signed zero amount, and both chain-length boundaries. All use
public **TEST ONLY** secp256k1 scalars A=1 and B=2, RFC 6979 test signing and
low-S normalization. Each artifact records fields, unsigned payload,
preimage, digest, compressed key, derived addresses, r/s, full wire, TxID,
CometHash and expected stateless code. The separate
`reference_oracle.py` reconstructs these values using Python `hashlib` and
OpenSSL-backed `cryptography`, and checks ECDSA with `Prehashed(SHA256)`.
Its PASS means these four stateless vectors agree with the independent
calculation; it does not prove account or ABCI behavior.

## Verification evidence

Environment: WSL2 Ubuntu/Linux amd64, Go 1.24.1, Python cryptography 3.4.8,
source `54bf720ac4625c17eaa6101a8f398ef7c0663432`. Exact commands,
output, exit codes, start/end UTC times and OS string are in
[`verification.log`](../evidence/phase_03b/verification.log).

| Command | Exit code | Result |
|---|---:|---|
| `go version` | 0 | `go1.24.1 linux/amd64` |
| `go build ./...` | 0 | all packages built |
| `go test -count=1 ./...` | 0 | existing and new unit tests passed |
| `go vet ./...` | 0 | no diagnostics |
| `make fmt-check` | 0 | Go formatting clean |
| `go mod verify` | 0 | all modules verified |
| `make verify` | 0 | build, tests, vet, format, module and version checks passed |
| `make test-integration` | 0 | existing real-node single-validator/restart test passed |
| `go test -race -count=1 ./internal/tx` | 0 | no race reported |
| `go test -coverprofile=research/evidence/phase_03b/coverage.out ./internal/tx` | 0 | 96.7% statement coverage |
| `go tool cover -func=research/evidence/phase_03b/coverage.out` | 0 | per-function coverage recorded |
| `go test ./internal/tx -run '^$' -fuzz '^FuzzParse$' -fuzztime 10s -parallel 2` | 0 | 407,953 executions; no failure |
| `go test ./internal/tx -run '^$' -fuzz '^FuzzCodec$' -fuzztime 10s -parallel 2` | 0 | 416,469 executions; no failure |
| `python3 research/evidence/phase_03b/reference_oracle.py` | 0 | four vectors independently verified |
| `git diff --check` | 0 | whitespace clean |
| `git diff --exit-code 54bf720ac4625c17eaa6101a8f398ef7c0663432 -- cmd internal/app internal/node internal/store test .github Makefile` | 0 | existing runtime, tests, build and workflows unchanged |

The first verification run found an unformatted new `internal/tx/tx_test.go`:
`make fmt-check` and `make verify` exited 2. `gofmt -w` corrected it, and the
full rerun above passed. The original exact failure output is preserved in
[`verification_initial.log`](../evidence/phase_03b/verification_initial.log).
The 10-second fuzz runs are bounded smoke/property checks, not exhaustive
security proofs. Their transient coverage corpus is in the Go fuzz cache. The
generated `coverage.out` (ignored by repository policy) was copied
byte-for-byte to [`coverage.profile`](../evidence/phase_03b/coverage.profile).

Negative tests cover malformed magic/version/length/ASCII, wrong chain,
zero amount and precedence, zero/out-of-range/high-S scalars, invalid point,
nonverifying signatures, changed signed fields, key substitution, a
double-hash check, TxID eligibility and identity distinction. Property tests
cover canonical parse/encode round trips, deterministic evaluation and
bounded arbitrary input. No production keys are present.

## Gates, limitations and next action

Phase 03B gates B1–B10 and B12 have local evidence above; B11 has local build,
test and static PASS while hosted CI remains pending until the Draft PR runs.
No state-dependent or ABCI transaction vectors were run: **NOT RUN — RESERVED
FOR PHASES 03C–03E**. No multi-validator B0, V2 state, crash-recovery or DoS
performance result is claimed. A short fuzz run and four golden vectors do
not establish exhaustive cryptographic assurance. D-023 and this report await
independent Controller/Owner review. Phase 03C is not authorized.

## Acceptance addendum — 2026-10-23

This addendum records decisions made after the execution report above. The
Controller accepted Phase 03B, and PR #4 was merged into `test-mainet` at
`8be07c87abed73763d40252e99bbf0573da487b4`. The Owner explicitly
accepted D-023, selecting `github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1`
for B0-Core. The post-merge [push verification](https://github.com/the-phong/DoS-in-Blockchain-PQC/actions/runs/39892775154)
and [pull-request verification](https://github.com/the-phong/DoS-in-Blockchain-PQC/actions/runs/39892779874)
both completed successfully. Earlier pending-review statements above remain
the historical execution snapshot; they do not describe the current status.
This addendum changes no normative protocol rule, prior test result, vector,
runtime behavior or dependency version. Phase 03C remains unauthorized.

The documentation-only closeout was verified on the merged source baseline
`8be07c87abed73763d40252e99bbf0573da487b4`: `git diff --check`,
`make verify`, `make test-integration`, and the source-diff check for runtime,
cryptographic code, dependencies, tests, accepted specifications, and golden
vectors all exited 0. Exact output and environment are in the new
[closeout verification log](../evidence/phase_03b_closeout/verification.log).
The governance closeout branch and resulting commit are recorded in its
Draft PR and final agent report; this addendum is not an authorization for
Phase 03C.
