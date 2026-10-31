# B0 account state and transitions V1 — Owner-accepted Phase 03A specification

**Status: ACCEPTED BY OWNER on 2026-10-23; Genesis V2 transport explicitly amended by Owner-accepted D-025 on 2026-10-24.** This document defines logical consensus state under accepted D-021. It preserves the existing Phase 02 `PLUTO02!` record and height-only V1 AppHash as a historical baseline. No Phase 02 database is migrated, reset or rewritten by this specification.

## 1. Logical state and invariants

A B0 state is `(format_version=2, chain_id, height, total_supply, accounts)`. `chain_id` follows the [transaction specification](B0_CLASSICAL_TRANSACTION_PROTOCOL_V1.md) and the existing Phase 02 chain-ID character/length rule. `height` is an integer in `[0, MaxInt64]`; genesis state is height 0 and the first block is height 1. `total_supply` is an unsigned 64-bit count of indivisible B0 units. No mint, burn, fee or reward transition exists in this protocol.

Each account is `(address[20], balance uint64, nonce uint64)`. Address is the exact 20-byte account key, compared lexicographically as unsigned bytes. A missing account has *virtual* balance 0 and nonce 0 for validation, but is not serialized until it receives value. Genesis accounts have nonce 0. A successful transfer can leave a zero-balance sender account; it MUST remain present to preserve its nonce. Accounts are never automatically deleted or pruned. Every serialized state MUST have `1 <= account_count <= 65,536`, strictly increasing unique addresses, and `sum(balance) == total_supply`, with sum computed in a width that cannot overflow `uint64` before the bound is checked. A valid chain can start with one or more positive allocations; it cannot add accounts after reaching the cap. The account cap is a consensus rule, not a local resource signal.
The B0 account does not store a public key: each spending transaction validates its included key and derives the sender address from it. It can debit only the account at that derived address. Any later registered-key or hybrid policy needs a new approved state/protocol version.

The complete committed state and chain ID MUST be sufficient to replay the same ordered block to the same result. Wall clocks, CPU time, queue occupancy, caches, mempool arrival order and process scheduling MUST NOT enter state validity or AppHash. All arithmetic is exact unsigned arithmetic with explicit checked bounds; wrapping is invalid.

## 2. Canonical genesis application bytes

For a **new B0 chain**, the authoritative decoded Genesis V2 binary `G` is exactly:

```text
ASCII("PLUTOG2!") || uint32_be(2) || uint16_be(N) || chain_id[N] ||
uint32_be(K) || K * (address[20] || uint64_be(balance))
```

`N` is 1–50, with the permitted ASCII chain ID. `K` is 1–65,536. Entries are strictly increasing by raw address bytes, unique, and each genesis balance is greater than zero. No nonce is encoded because every genesis nonce is zero. Exact input length is `8 + 4 + 2 + N + 4 + 28*K` bytes; trailing bytes, unknown magic/version, noncanonical order, duplicate addresses, zero balances or sum exceeding `MaxUint64` fail genesis validation. `InitChain.ChainId` MUST equal this encoded chain ID. `total_supply` is the checked sum of allocations. The resulting application state is height 0 and uses the V2 AppHash below.

**D-025 transport amendment (2026-10-24):** The original Phase 03A wording
required these raw binary bytes directly in `InitChain.AppStateBytes`.
CometBFT v0.38.26 instead loads genesis `app_state` as a JSON value and
forwards its JSON bytes unchanged. The accepted B0 transport is precisely
`app_state = json.Marshal(base64.StdEncoding.EncodeToString(G))`: one JSON
string containing RFC 4648 standard padded Base64. Thus the actual
`RequestInitChain.AppStateBytes` carries that JSON string's bytes, **not**
`G` directly. The Pluto B0 adapter MUST reject other JSON types, trailing
values, noncanonical JSON/Base64, malformed or oversized input, then decode
to `G` and call the existing canonical binary parser. Both the request and
configured profile chain IDs MUST match the decoded `G` chain ID. The binary
format, height-zero state and AppHash V2 computation above are unchanged;
JSON/Base64 bytes are never part of the state commitment. No fallback
transport or implicit migration is permitted.

The existing Phase 02 genesis uses empty app state and the V1 state record. A V2-capable implementation MUST NOT silently read that record as V2, infer allocations, reset either store or change AppHash. The accepted B0 V1 path is fresh genesis only unless a separately authorized, explicit, tested migration is specified. Operators must preserve Phase 02 homes and use a new home for a fresh B0 chain. This is a protocol compatibility boundary, not a current runtime behavior.

## 3. Sequential transfer function

For each transaction in block order, run the [transaction protocol's validation order](B0_CLASSICAL_TRANSACTION_PROTOCOL_V1.md#5-validation-order-and-error-precedence) against a temporary state containing all **previous successful** transactions of that block. A failure leaves every balance, nonce, account membership and total supply unchanged; no fee or nonce is consumed. A success applies exactly one of these cases:

1. **Distinct sender and receiver:** require `amount > 0`, sender nonce equals wire nonce and is `< MaxUint64`, sender balance `>= amount`, checked `receiver.balance + amount <= MaxUint64`, and, if receiver is absent, `account_count < 65,536`, in the full evaluator order linked above. Then subtract amount from sender, add it to receiver (creating absent receiver with nonce 0), increment sender nonce by one, retain both accounts, and leave total supply unchanged.
2. **Self-transfer:** require the same amount, nonce and sender-balance checks in that order. Increment sender nonce by one; leave its balance, account count and total supply unchanged. The debit and credit are a single net-zero transition, with no intermediate overflow.

A newly funded recipient can subsequently be a sender in the same block if its derived key signs a later transaction with nonce 0. Multiple transactions by the same sender are valid in one block only when their wire nonces equal the sender's successive candidate nonces (`0,1,...` for a genesis nonce-0 sender) and each transfer remains funded. A failed transaction does not advance nonce, so a later transaction with the same expected nonce may still succeed. A second successful execution of identical canonical bytes fails nonce validation. **Every successful transaction MUST be able to increment its sender nonce without overflowing `uint64`: the candidate sender nonce MUST be strictly less than `MaxUint64`.** Once structural, chain, amount and cryptographic checks pass, an incorrect nonce or a matching `MaxUint64` nonce returns Code 6 (`invalid_nonce`) before balance, receiver-overflow or account-cap checks, and leaves state unchanged. An empty block changes height and AppHash but not accounts or total supply.

After processing all block transactions, increment height exactly once, rejecting `MaxInt64` overflow. Recompute the canonical state hash, then hold the entire result as **pending** until `Commit`. `FinalizeBlock` never writes it durably. `Commit` must make the complete state and its height/hash durable as one application-level atomic boundary before acknowledging. A failed commit cannot promote pending state as committed. A crash/replay design for a physical multi-key store is Phase 03C work; this specification does not assert one exists today.

## 4. Canonical V2 state commitment

Let `STATE_DOMAIN = ASCII("PLUTO/B0/STATE/V2") || 0x00`. Define the logical serialization `S` as:

```text
uint32_be(2) || uint16_be(N) || chain_id[N] || uint64_be(height) ||
uint64_be(total_supply) || uint32_be(K) ||
for each account in strictly ascending address order:
    address[20] || uint64_be(balance) || uint64_be(nonce)
```

`AppHashV2 = SHA-256(STATE_DOMAIN || S)`, exactly 32 bytes. Empty blocks alter `height` and therefore the hash. The serialization includes every consensus-relevant B0 account, including zero-balance accounts with used nonces, and the total supply. A loader MUST verify version, chain ID, height bound, account count/order/uniqueness, total-supply sum, exact lengths and the recomputed digest before returning state. Storage internals may use another physical layout only if they reconstruct this exact logical `S` and atomically persist a consistent snapshot. The V2 hash is a flat digest, not a Merkle proof, Ethereum trie root or state-sync format; arbitrary historical account proofs are unavailable. **Full-state recomputation reads, encodes and hashes all `K` accounts, so bytes hashed and the unavoidable work are proportional to account count `K`, including for an empty block; reconstructing canonical order may add sorting cost if the physical layout is unordered.** Future B0/B1 experiments MUST record and control account count, state layout and execution model when comparing transaction cost; otherwise AppHash work can confound authentication measurements. This is a measurement requirement, not an implemented optimization.

The Phase 02 V1 digest is `SHA-256("PLUTO/APP_STATE/V1\0" || version || chain ID || height)` and commits to no accounts. V2 uses a different domain and format version. Existing Phase 02 state MUST NOT be reinterpreted or upgraded in place without a separate approved migration design, test vectors and explicit operator action. In CometBFT v0.38.26, `ResponseFinalizeBlock.AppHash` is the hash of state **after** block H; block header H contains the prior committed application's hash. `ResponseCommit` carries no new AppHash.

## 5. Invariant and boundary matrix

| Condition | Required outcome |
|---|---|
| Transfer from absent or unfunded sender | Insufficient balance; no state change. |
| Incorrect, reused or future nonce | Nonce error; no state change. |
| Amount zero | Amount error; no state change. |
| Sender balance exactly amount | Success; sender remains present with zero balance and incremented nonce. |
| Receiver absent with account count below cap | Success; create balance=`amount`, nonce=0. |
| Receiver absent at cap | Account-limit error; no state change. |
| Receiver balance addition overflows | Balance-overflow error; no state change. This is unreachable from a valid supply-bounded state with a funded distinct sender, but must be checked defensively. |
| Sender nonce `MaxUint64` | Code 6 `invalid_nonce` after earlier state-independent stages, before balance checks; no wrap. |
| Genesis allocation sum exceeds `MaxUint64` | Reject genesis; no initialized B0 state. |
| State height `MaxInt64` | Refuse next block; no wrap or commit. |
| Any successful block | Sum of account balances equals unchanged total supply; accounts remain sorted and unique. |

## 6. Implementation follow-ups

The Owner accepted the fee-free correctness baseline, 65,536-account cap and fresh-genesis V1 path. The cap's deployment suitability still needs testing; any later increase requires a new protocol version and decision. A Phase 02-home migration would require a separately approved specification and operator action. Future Phase 03C work must select an atomic physical storage layout and fault-injection matrix under its own authorization. None of these follow-ups changes the accepted logical rules here.
