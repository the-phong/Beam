# Phase 03B secp256k1 dependency review

Reviewed on 2026-10-23 against the verified source revision
`54bf720ac4625c17eaa6101a8f398ef7c0663432`. The Go module version index
(`go list -m -versions github.com/decred/dcrd/dcrec/secp256k1/v4`) listed
`v4.0.0 v4.0.1 v4.1.0 v4.2.0 v4.3.0 v4.4.0 v4.4.1`; `v4.4.1` was the
latest listed release. [Upstream package API](https://pkg.go.dev/github.com/decred/dcrd/dcrec/secp256k1/v4@v4.4.1)
and its [ECDSA subpackage](https://pkg.go.dev/github.com/decred/dcrd/dcrec/secp256k1/v4@v4.4.1/ecdsa)
document compressed-key parsing and `Verify(hash, pubKey)` over a supplied
digest. This is a maintained Decred implementation; an external audit of its
cryptographic internals was not performed in this phase.

Selected direct module: `github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1`.
The Phase 03A graph already included `v4.4.0` indirectly through CometBFT.
The explicit update to `v4.4.1` is the only selected module version change.
`go list -m -json` reports Go minimum `1.17`, module checksum
`h1:5RVFMOWjMyRy8cARdy79nAmgYw3hK/4HUq48LQ6Wwqo=` and go.mod checksum
`h1:ZXNYxsqcloTdSy/rNShjYzMhyjf0LaoftYK0p+A3h40=`. Pluto uses Go 1.24.1.
The selected module declares `github.com/decred/dcrd/crypto/blake256 v1.1.0`,
already present in the module graph. `go mod tidy` added no new module path.

API and security checks:

- `ParsePubKey` verifies the secp256k1 point but also supports encodings at
  other lengths; Pluto checks exactly 33 bytes and prefix `02` or `03` first.
- `ModNScalar.SetByteSlice` reduces overflow values modulo the group order;
  Pluto checks its overflow result and zero before using either scalar, and
  checks `s.IsOverHalfOrder()` to reject high-S.
- `ecdsa.NewSignature` accepts checked `ModNScalar` values and
  `Signature.Verify` consumes the already-computed digest without another
  hash. Its `Sign` uses RFC 6979 deterministic signing for public test-only
  fixtures. Production signing and key storage are outside Phase 03B.
- The independent Python/OpenSSL checker verifies compressed points,
  signatures with `Prehashed(SHA256)`, addresses, preimages, digests, complete
  wire bytes and both transaction hashes. This cross-check covers the Pluto
  adapter and vector producer, not every possible library defect.

Alternatives considered: keep indirect `v4.4.0` (older API release), add a
second curve library to production (larger dependency surface), or implement
curve arithmetic locally (unacceptable cryptographic risk). No CometBFT,
Pebble, consensus, ABCI or storage code was edited. The module update is
visible in `go.mod` and `go.sum`, and the real-node integration test passed.

Remaining risk: a direct dependency may change behavior available to
CometBFT through Go module selection; the existing integration test checks
single-node operation and restart, not all consensus/network conditions.
