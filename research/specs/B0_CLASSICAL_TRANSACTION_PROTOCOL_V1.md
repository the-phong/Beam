# B0 classical transaction protocol V1 — Owner-accepted Phase 03A specification

**Status: ACCEPTED BY OWNER on 2026-10-23 after Controller rework; not implemented.** This document defines consensus bytes for a minimal, fee-free, account-based value transfer under accepted D-020. The current Phase 02 application still rejects every transaction. Phases 03B–03E need separate authorization. Normative words **MUST**, **MUST NOT**, and **SHOULD** apply to a future B0 implementation only.

## 1. Scope and choices

| Decision | Accepted normative choice | Rationale and trade-off |
|---|---|---|
| Transaction purpose | Transfer indivisible B0 units between 20-byte accounts only. | Small correctness baseline; no fee market, gas, memo, expiry, EVM call or contract data. Spam resistance is absent and must be addressed at a later approved gate, without changing consensus validity using local resources. |
| Version | Magic `PLUTOTX1` plus unsigned big-endian `uint16` version `1`. | Explicit format boundary; unknown versions fail closed. |
| Chain binding | The signed payload carries the Phase 02-compatible chain ID: 1–50 ASCII bytes from `[A-Za-z0-9_.-]`; it MUST equal committed state/genesis chain ID byte-for-byte. | Prevents cross-chain replay, subject to operators using distinct chain IDs. |
| Sender identity | A compressed secp256k1 public key is carried; the sender account address is deterministically derived from that key. No independent sender-address field. | Avoids a second, possibly conflicting identity field. Address collisions remain a generic 160-bit hash assumption. |
| Signature | ECDSA over secp256k1, 64-byte compact `r || s`, low-S only, over a SHA-256 domain-separated digest. | Unique S representative and fixed size; this is neither Ethereum RLP/EIP-155 nor an Ethereum address/signature. |
| Nonce | Per-sender `uint64`, initially zero; a successful transaction requires equality with current account nonce **and current nonce < MaxUint64** before incrementing it by one. | Ordered replay protection without wrap; no future-nonce queue semantics in B0 admission. |
| Transaction ID | Domain-separated SHA-256 of a structurally well-formed wire record with canonical signature scalars, as precisely defined in section 4. | Binds signature bytes independently of later authentication/state checks; differently signed forms of one intent can have different IDs, but account nonce prevents repeated execution. This differs from the CometBFT RPC transaction hash. |

No fee, gas price, memo, expiry, sender-address copy, key version or EVM field is necessary for the B0 value-transfer invariant. Adding any later requires a new protocol version and separate decision; a future EVM adapter must map this protocol explicitly rather than pretend Ethereum wire compatibility.

## 2. Exact wire encoding

All integers are **unsigned**, fixed-width, **big-endian**. `ASCII` denotes exact bytes, not a Unicode or JSON representation. Concatenation `||` has no padding. `N = len(chain_id)` must be 1–50. `wire = unsigned_payload || signature` and has exactly `145 + N` bytes (146–195 inclusive); any shorter, longer or trailing bytes are invalid. No variable-length integer, optional field or alternate encoding exists.

| Offset (zero-based) | Width | Field | Rule |
|---|---:|---|---|
| `0..7` | 8 | magic | ASCII `PLUTOTX1` |
| `8..9` | 2 | version | `0x0001` |
| `10..11` | 2 | chain ID length | `uint16_be(N)`; 1–50 |
| `12..11+N` | N | chain ID | exact permitted ASCII bytes |
| `12+N..44+N` | 33 | sender public key | SEC 1 compressed point, prefix `0x02` or `0x03` |
| `45+N..64+N` | 20 | receiver address | uninterpreted raw 20 bytes, including all-zero if supplied |
| `65+N..72+N` | 8 | amount | `uint64_be`, in indivisible B0 units; must be greater than zero |
| `73+N..80+N` | 8 | nonce | `uint64_be` |
| `81+N..112+N` | 32 | `r` | big-endian ECDSA scalar |
| `113+N..144+N` | 32 | `s` | big-endian ECDSA scalar |

`unsigned_payload` ends after byte `80+N` and has length `81+N`. The receiver has no checksum in wire form. A textual form, if added later, is an RPC/UI concern; hex in documents is a display of these raw bytes and is never signed as text. The wire length bound includes all fields and is checked before allocation or parsing. Re-encoding a decoded transaction MUST reproduce its input bytes exactly.

## 3. Signed intent and signature validity

Let `SIGN_DOMAIN = ASCII("PLUTO/B0/SIGN/V1") || 0x00`. Let `U` be the exact `unsigned_payload` bytes above. The 32-byte signing digest is:

```text
D = SHA-256(SIGN_DOMAIN || uint32_be(len(U)) || U)
```

The chain ID, sender key, receiver, amount and nonce are all covered by `D`. No signature byte is included. The domain and version prevent interpreting the same bytes as a different Pluto operation or future hybrid intent. The ECDSA message representative is the unsigned big-endian integer represented by the 32-byte `D`; secp256k1 ECDSA scalar arithmetic is modulo `n` as in SEC 1. The verifier supplies exactly `D` to a prehashed-message ECDSA API and MUST NOT SHA-256 it a second time. RFC 6979 reference signing, when used, takes this same 32-byte digest as its hashed-message input.

Curve parameters are secp256k1 from [SEC 2 v2.0, section 2.4.1](https://www.secg.org/sec2-v2.pdf): field prime `p = FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F` and group order `n = FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141` (hex). A 33-byte public key MUST have prefix `02` or `03`; its 32-byte x coordinate MUST be `< p`. Point decompression MUST produce a finite point on `y² = x³ + 7 (mod p)` with the prefix's y parity. The infinity, off-curve, non-canonical x, uncompressed (`04`), hybrid and trailing-byte encodings MUST be rejected. The group has cofactor 1; complete on-curve validation implies subgroup membership for secp256k1. See [SEC 1 v2.0, sections 2.3.3 and 3.2.2](https://www.secg.org/sec1-v2.pdf).

The 32-byte `r` and `s` values MUST each be in `[1,n-1]`; `s` MUST additionally satisfy `s <= floor(n/2)`. Values with leading zero bytes are valid only as part of the required fixed 32-byte scalar width; DER, shorter integers, recovery IDs and high-S alternatives are invalid. After scalar checks, verify ECDSA `(r,s)` against the validated public key and `D` using SEC 1. A signer SHOULD use [RFC 6979](https://www.rfc-editor.org/rfc/rfc6979) with HMAC-SHA-256 to avoid fragile signing randomness, then normalize `s := min(s, n-s)`. Verification cannot and does not infer the signer's nonce-generation method; any valid low-S signature is accepted.

## 4. Address and transaction ID

Let `ADDR_DOMAIN = ASCII("PLUTO/B0/ADDR/V1") || 0x00`. For a validated compressed public key `P`, the sender address is the **first 20 bytes** of `SHA-256(ADDR_DOMAIN || P)`, in digest byte order. No Keccak hash, EIP-55 checksum or Ethereum address rule is used. A receiver address is any raw 20-byte value; funds sent to an address without a known key may be unspendable, so clients SHOULD derive and verify recipients before submission.

Let `ID_DOMAIN = ASCII("PLUTO/B0/TXID/V1") || 0x00`. **TxID eligibility** is a byte-only predicate, independent of the current chain and account state: (a) the complete wire passes every exact-length, magic, version, chain-length and permitted-ASCII check in validation stage 1; and (b) its fixed-width `r,s` integers satisfy `1 <= r < n` and `1 <= s <= floor(n/2)`. The 33 public-key bytes are an opaque fixed-width field for this predicate; prefix, coordinate and curve-point validation are deliberately excluded. Amount positivity, chain-ID equality, ECDSA verification, nonce, balance and account limits are also excluded. If and only if (a) and (b) hold, `TxID = SHA-256(ID_DOMAIN || uint32_be(len(wire)) || wire)` exists. This rule gives no TxID to malformed wire bytes or out-of-range/high-S scalars, but gives one to an invalid compressed key (including a bad prefix in the 33-byte slot), a nonverifying ECDSA signature with canonical scalars, a wrong-chain transaction and a nonce/balance-rejected transaction. Existence of a TxID says nothing about admission or execution validity. Its 32 bytes are normally displayed as lowercase hex without altering the bytes.

In pinned CometBFT v0.38.26, `types.Tx(wire).Hash()` is `tmhash.Sum(wire) = SHA-256(wire)`; its RPC transaction hash has **no Pluto domain prefix or length prefix** and is defined even for bytes that have no Pluto TxID. The two hashes MUST NOT be substituted for one another. Future tests and benchmarks SHOULD record the exact raw transaction bytes (or their captured byte-for-byte copy), `CometHash = SHA-256(wire)`, and, only when eligible, Pluto TxID; use CometHash to locate `broadcast_tx_*`, `tx` and `tx_search` RPC records, then verify the returned raw `tx` bytes and correlate by `(chain_id, block_height, transaction_index, raw_bytes)` plus Pluto TxID where it exists. Duplicate identical bytes have the same two hashes, so occurrence identity additionally requires height/index; failed admission may have no block height/index or indexed RPC record. No Pluto-specific RPC endpoint is proposed in this phase. Source: pinned [CometBFT `types.Tx.Hash`](https://github.com/cometbft/cometbft/blob/v0.38.26/types/tx.go), [`tmhash.Sum`](https://github.com/cometbft/cometbft/blob/v0.38.26/crypto/tmhash/hash.go) and [RPC `Tx`/`TxSearch`](https://github.com/cometbft/cometbft/blob/v0.38.26/rpc/core/tx.go).

## 5. Validation order and error precedence

The future common evaluator MUST apply these stages in order and stop at the first failure, using the codes in the [ABCI validity specification](B0_ABCI_VALIDITY_SPEC_V1.md):

1. **Structural, state-independent:** bound input length to 146–195, then check magic, version, chain length/characters, exact calculated total length and no trailing data (Code 1). Reject unknown versions; never reinterpret them as V1. Read fixed-width fields but do not yet validate key or scalar mathematics.
2. **Chain context, state-independent:** compare wire chain ID to the current committed/candidate chain ID (Code 2).
3. **Cheap semantic, state-independent:** require `amount > 0` (Code 5).
4. **Cryptographic encoding, state-independent:** interpret fixed-width `r,s` as unsigned big-endian integers; require `1 <= r < n` and `1 <= s <= floor(n/2)` (Code 4). These checks do not perform ECDSA verification.
5. **Cryptographic key, state-independent:** validate the 33-byte compressed point, including prefix, coordinate bound, decompression and curve membership (Code 3).
6. **Cryptographic authentication, state-independent:** compute `D` and verify ECDSA with the validated key and canonical scalars (Code 4).
7. **Account state-dependent:** derive sender address; require wire nonce equals its current nonce **and current nonce < MaxUint64** (Code 6). A matching `MaxUint64` wire nonce still fails Code 6; this precedes balance checks.
8. **Account state-dependent:** require sender balance at least amount (Code 7).
9. **Account state-dependent:** if sender and receiver differ, require receiver balance plus amount to fit `uint64` (Code 8); if receiver is absent, require room under the account-count cap (Code 9).
10. Only then apply the state transition defined in [B0 state transitions](B0_STATE_TRANSITION_SPEC_V1.md).

The first failing stage determines the code. Cheap structural, chain, amount and scalar checks precede point decompression and ECDSA to avoid needless expensive work on common invalid inputs. This fixes a deterministic error precedence across `CheckTx`, proposal evaluation and final execution; it adds no admission mechanism or local-resource-dependent rule. In a resource-cost experiment, the invalid-input mix and rejection stage MUST be recorded because a zero-amount transaction no longer incurs ECDSA verification, while a nonzero amount with canonical but invalid signature can still exercise that cost. Fair B0/B1 comparisons must use the same declared validity stages and workloads, while separately measuring any inherently different authentication work. All stages through 6 are independent of account state (stage 2 reads the fixed chain identity); stages 7–9 use the sequential candidate account state. TxID eligibility is the separate predicate in section 4 and does not change this first-failure order. An all-zero receiver address is a regular address, not a burn address. There is no fallback to another signature scheme.

## 6. Implementation follow-ups and references

The Owner accepted the fee-free B0 unit rule, 20-byte SHA-256-derived addresses, and the stated account/block limits. Future implementation may choose a human display scale without changing wire units; any change to accepted protocol rules requires a new Owner-approved decision. A future capacity upgrade or Phase 02-to-V2 migration would likewise require a separate specification, approval and tests. Neither is included in this accepted V1 protocol. Phase 03B remains unauthorized.

Primary references: [SEC 1 v2.0](https://www.secg.org/sec1-v2.pdf), [SEC 2 v2.0](https://www.secg.org/sec2-v2.pdf), [RFC 6979](https://www.rfc-editor.org/rfc/rfc6979), and the pinned [CometBFT v0.38.26 ABCI application requirements](https://github.com/cometbft/cometbft/blob/v0.38.26/spec/abci/abci%2B%2B_app_requirements.md). None of these sources makes the Pluto wire format a standard.
