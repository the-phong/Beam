# B1 hybrid authentication profile — Phase 04 candidate

Status: implemented for independent Controller review; Owner acceptance pending. D-028 and D-029 authorize this isolated profile. Accepted B0, D-020–D-027, D-024 record bytes and Phase 02 are unchanged. B1 requires a fresh chain; there is no migration, profile fallback, key rotation, dynamic registration or account creation.

## Transaction envelope and signed intent

All integers use unsigned big-endian. There are no optional fields, nested values, field identifiers or trailing bytes. `N` is the byte length of an ASCII chain ID, 1–50, containing only letters, digits, `_`, `-`, `.`. A transaction is exactly `3897 + N` bytes (3898–3947); parsing checks this bound before slicing or allocating.

| Offset | Width | Field |
|---|---:|---|
| 0 | 8 | ASCII `PLUTOTX2` |
| 8 | 2 | version `0x0002` |
| 10 | 1 | hybrid profile `0x01` |
| 11 | 1 | chain length `N` |
| 12 | N | chain ID |
| 12+N | 20 | sender account address |
| 32+N | 33 | compressed secp256k1 public key |
| 65+N | 1312 | ML-DSA-44 public key |
| 1377+N | 20 | recipient account address |
| 1397+N | 8 | amount |
| 1405+N | 8 | sender nonce |
| 1413+N | 64 | secp256k1 signature, 32-byte R then 32-byte S |
| 1477+N | 2420 | ML-DSA-44 signature |

The canonical unsigned intent `U` is bytes `[0 : 1413+N]`, including both public keys and all transfer fields. Both signatures authenticate the same 32-byte digest:

```
D = SHA-256( ASCII("PLUTO/B1/HYBRID/SIGN/V1") || 0x00 ||
            uint32_be(len(U)) || U )
```

secp256k1 signs/verifies `D` as a prehashed ECDSA message. R and S must be nonzero, in range and low-S. ML-DSA-44 signs/verifies exactly `D` using CIRCL's **pure** ML-DSA API with context `ASCII("PLUTO/B1/HYBRID/V1")`; CIRCL applies the FIPS 204 pure-message framing (`0x00 || uint8(context length) || context || D`). No prehash ML-DSA variant or silent context substitution is accepted. The raw transaction ID is `SHA-256(ASCII("PLUTO/B1/TXID/V1") || 0x00 || uint32_be(len(raw)) || raw)`. This explicitly separates transaction IDs from B0.

For `chain_id = "pluto-b1-golden"` (`N=15`), the first 27 bytes are `504c55544f5458320002010f706c75746f2d62312d676f6c64656e`. The complete public-only Genesis V3, signed transaction, digest, transaction ID, H0 AppHash and post-transfer candidate AppHash are in `internal/b1tx/testdata/hybrid_public_vector.json`. The test recomputes every hash, verifies both proofs, and checks sender 100/0 → 75/1, receiver 0/0 → 25/0, supply 100. Its ephemeral signing keys were never written to the vector or repository. Independently of this Pluto vector, the pinned CIRCL package's bundled ACVP FIPS204 suite is executed with `go test github.com/cloudflare/circl/sign/mldsa/mldsa44 -run '^TestACVP$' -count=1 -v`; that corpus invokes the library's internal primitive and cannot be fed directly into the framed public `Verify` API.

## Account-key binding and state

Genesis V3 bytes are `ASCII("PLUTOG3!") || uint32_be(3) || uint16_be(N) || chain_id || uint32_be(K) || K entries`. Each entry is `address[20] || balance:uint64 || compressed_secp_key[33] || ml_dsa44_public_key[1312]`. `K` is 1–1024, addresses are strictly increasing and unique, and supply is the checked sum of balances and must be nonzero. A registered zero-balance recipient is allowed. The expected address is the first 20 bytes of `SHA-256(ASCII("PLUTO/B1/ADDR/V1") || 0x00 || secp_key || ml_key)`. Both genesis and transaction parsing rederive it. Every sender and recipient must already be registered in the immutable genesis registry. A transaction-supplied key does not establish authorization: its address and both key bytes must match the registered entry before either proof can authorize the account.

The registry digest hashes `ASCII("PLUTO/B1/REGISTRY/V1") || 0x00 || uint32_be(K) ||` each canonical `address || secp_key || ml_key`. The B1 AppHash hashes `ASCII("PLUTO/B1/STATE/V1") || 0x00 || registry_digest || accepted_B0_logical_state_bytes`. The B0 logical balance/nonce transition is reused after hybrid authentication; B0 AppHash V2 remains unchanged. B1 stores one complete record at Pebble key `pluto/committed-state/b1-v1` with magic `PLUTOS3!`, registry digest, B0 logical bytes, B0 logical hash and B1 AppHash. It uses a separate `pluto-b1-v1` directory and `pebble.Sync` on Commit. B1 `pluto.json` also binds each home to its expected public Comet validator address, so another valid member's copied signing key is refused on startup. This adds no independent initialization marker or D-024 record change. B0/Phase 02 records are refused, and `start` never initializes missing state. The JSON/Base64 Comet `app_state` transport carries the B1 binary under its own strict parser, without changing D-025's B0 transport.

## Deterministic validity and ABCI paths

The first failure code wins: malformed envelope/version/profile/length (1), wrong chain (2), zero amount (5), malformed secp key (3), malformed ML key (12), missing/incorrect registry binding or unregistered recipient (11), invalid secp signature (4), invalid ML signature (13), then accepted B0 nonce (6), funds (7), overflow (8), account-limit (9). Code 10 is the block envelope limit. Code 0 is accepted. CIRCL accepts every fixed-width 1312-byte ML-DSA-44 public-key bit string as a syntactic packing: an unregistered value is code 11 and a wrong proof under a registered key is code 13. Code 12 is defensive if the pinned library ever rejects such a value; truncation or oversize changes the envelope and is code 1. The exact order is implemented by `b1state.EvaluateTrusted`; `CheckTx`, `PrepareProposal`, `ProcessProposal` and `FinalizeBlock` all call it through isolated candidate state. CheckTx NEW and RECHECK only read committed state. ProcessProposal rejects a whole proposal if any sequentially evaluated transaction fails or the block envelope is oversized, regardless of local mempool admission. FinalizeBlock returns deterministic per-transaction results and prepares only pending state; Commit performs the synchronized write. A failed or ambiguous write fails closed until reopen. Account query path is `b1/account/v1`, latest committed state only.

## Threat assumptions and limits

The adversary can submit arbitrary transaction bytes, substitute keys or proofs, replay old transactions, and propose transactions without passing a local CheckTx. Consensus validators are assumed to run the same pinned code and canonical genesis. Local home metadata and validator signing keys are trusted operator-controlled artifacts; the expected-address field detects a copied key when metadata is intact, but an actor able to rewrite both key and metadata is outside this local startup check. The tests cover parser bounds, same-intent proof checks, genesis key binding, cross-chain/profile replay, nonce/funds/supply, direct invalid proposals, real four-validator convergence, crash-stop and replay. ML-DSA permits distinct valid signatures for one intent when signing is randomized; because TxID includes signatures, those variants have different TxIDs. A dedicated test constructs two valid variants and confirms that sequential application accepts only the first under the shared nonce. Multiple variants may still consume precommit admission resources; no DoS protection is claimed. The tests do not establish Byzantine equivocation resistance, multi-host fault isolation, hardware power-loss durability, formal cryptographic proof, FIPS module certification or DoS resistance. The fixed ~3.9 KB transaction and 1 MB application block envelope bound allocations; no admission-pricing policy is introduced. All four validators in tests share one WSL2 host. Key rotation and post-genesis account registration require a later authorized protocol decision.

## Dependency review

NIST [FIPS 204](https://csrc.nist.gov/pubs/fips/204/final) is the ML-DSA algorithm reference. The implementation uses [Cloudflare CIRCL `sign/mldsa/mldsa44` v1.6.4](https://pkg.go.dev/github.com/cloudflare/circl/sign/mldsa/mldsa44), pinned in `go.mod`. CIRCL's public API reports 1312-byte public keys and 2420-byte signatures, pure ML-DSA signing with an explicit context and deterministic signing when `randomized=false`. The package's ACVP test corpus includes ML-DSA-44 signature verification vectors. `go mod verify` and the exact source tests validate the downloaded dependency; no FIPS module certification is claimed. Upgraded transitive `golang.org/x/*` versions are recorded in `go.mod`/`go.sum` and exercised by the full B0/Phase 02 regression suites.
