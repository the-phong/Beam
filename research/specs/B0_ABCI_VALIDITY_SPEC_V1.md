# B0 ABCI++ validity contract V1 — Owner-accepted Phase 03A specification

**Status: ACCEPTED BY OWNER on 2026-10-23; InitChain transport explicitly amended by Owner-accepted D-025 on 2026-10-24.** This contract records accepted D-022 and targets the pinned [CometBFT v0.38.26 `abci.Application` interface](https://github.com/cometbft/cometbft/blob/v0.38.26/abci/types/application.go) and its [ABCI++ application requirements](https://github.com/cometbft/cometbft/blob/v0.38.26/spec/abci/abci%2B%2B_app_requirements.md). It must not be substituted with the historical Phase 01 v1.0.0 reference. Current Phase 02 code continues to reject all transactions.

## 1. Shared evaluator and limits

All future transaction paths use the same ordered decode, authentication and transition evaluator specified in [transaction V1](B0_CLASSICAL_TRANSACTION_PROTOCOL_V1.md) and [state V1](B0_STATE_TRANSITION_SPEC_V1.md). They may use different state snapshots, but MUST assign the same code to the same transaction and candidate state. The evaluator returns `(code, next_state)`; on any nonzero code, `next_state` equals input state exactly. No local cache result is authoritative for consensus.

The B0 application-level block limit is **6,144 transactions** and **1,048,576 aggregate raw transaction bytes**, summed with checked unsigned arithmetic. Each transaction is at most 195 bytes. Both bounds are independently reachable with **otherwise valid, signed, funded, sequential-nonce transactions**: 6,145 records with a 1-byte chain ID total 897,170 bytes and isolate count, while 5,378 records with a 50-byte chain ID total 1,048,710 bytes and isolate aggregate size. Exact fixtures and preconditions are in [vector V20](B0_TEST_VECTOR_PLAN_V1.md#2-required-vector-matrix). These limits are fixed protocol rules for `ProcessProposal` and `FinalizeBlock`, independent of node resources. CometBFT may enforce stricter consensus block limits; `PrepareProposal` additionally obeys `RequestPrepareProposal.MaxTxBytes`. Proposing fewer transactions due to local mempool contents is allowed; accepting or rejecting the same complete proposal based on CPU, wall clock, queue occupancy, mempool arrival or nonreplicated cache is forbidden.

The following result codes and exact `Log` strings are specified for both `ResponseCheckTx` and `ExecTxResult`. `Codespace` is `pluto.b0.v1`; `Data` and events are empty, `GasWanted`/`GasUsed` are zero, and `Priority` in `CheckTx` is zero. Codes are determined by the first failing stage in the transaction specification. Code 10 is a block-level envelope failure and never a per-transaction validity result from `CheckTx`.

| Code | Exact `Log` | Meaning |
|---:|---|---|
| 0 | `ok` | Valid against the evaluated state; transfer applied only in finalization. |
| 1 | `malformed` | Invalid length, magic, version, chain-ID syntax or noncanonical field/encoding. |
| 2 | `wrong_chain` | Well-formed chain ID differs from committed/candidate chain ID. |
| 3 | `invalid_key` | Compressed public key fails secp256k1 point validation, including its prefix. |
| 4 | `invalid_signature` | Scalar range/low-S canonicality or ECDSA verification failure. |
| 5 | `invalid_amount` | Amount is zero; checked before scalar, point or ECDSA work. |
| 6 | `invalid_nonce` | Wire nonce differs from candidate sender nonce, or matching candidate nonce is `MaxUint64` and cannot increment. |
| 7 | `insufficient_balance` | Sender lacks amount, including absent sender. |
| 8 | `balance_overflow` | Receiver credit would overflow `uint64`. |
| 9 | `account_limit` | A new receiver would exceed 65,536 accounts. |
| 10 | `block_limit` | Decided block exceeds app count/byte limit; every tx fails and none executes. |

No gas, fee, validator update or consensus-parameter update is returned by this protocol. A signature failure, bad nonce or insufficient balance is an application rejection, not an ABCI transport error. A malformed or state-invalid transaction never mutates state, consumes a nonce or reduces total supply.

For every per-transaction evaluation, first-failure precedence is **Code 1 structural → Code 2 chain mismatch → Code 5 zero amount → Code 4 noncanonical scalar → Code 3 invalid compressed point → Code 4 failed ECDSA → Code 6 nonce mismatch or `MaxUint64` → Code 7 insufficient balance → Code 8 receiver overflow → Code 9 account limit → Code 0 success**. This order applies identically in `CheckTx`, `PrepareProposal`, `ProcessProposal` and in-limit `FinalizeBlock`; only the evaluated account snapshot differs. Code 10 overrides all per-transaction evaluation for an over-limit decided block. Structural and amount rejection precede expensive curve/ECDSA work; none of these stages depends on local resource measurements. The transaction specification's independent TxID eligibility predicate requires only structural checks and canonical scalar range/low-S, regardless of this first-failure precedence: Code 1 and scalar-canonicality Code 4 have no Pluto TxID; Code 2, 3, 5, ECDSA-verification Code 4, and state-dependent Codes 6–9 can have one. CometBFT RPC uses its own raw-byte `SHA-256(wire)` hash for transaction lookup; tests correlate both identifiers by exact raw bytes and, for included duplicates, block height/index as described in [transaction V1](B0_CLASSICAL_TRANSACTION_PROTOCOL_V1.md#4-address-and-transaction-id).

## 2. Method-by-method contract

### `Info` and `InitChain`

`Info` reports the last **durably committed** B0 height and V2 AppHash; pending state is invisible. On a new B0 chain, `InitChain` checks `InitialHeight==1`, matching chain ID and exact canonical V2 genesis bytes, writes genesis height 0 durably, and returns V2 AppHash while retaining CometBFT genesis validators/consensus parameters. An identical height-0 retry may be idempotent; conflicting genesis or an existing Phase 02 V1 record is an error, never an implicit migration. Phase 03C must test interrupted initialization and CometBFT handshake before activation.

**D-025 amendment (2026-10-24):** The historical phrase "exact canonical V2
genesis bytes" above described raw `InitChain.AppStateBytes`. For the
CometBFT v0.38.26 JSON genesis path, `AppStateBytes` actually contains one
canonical JSON string with standard padded Base64 of the unchanged Genesis
V2 binary. The B0 adapter strictly decodes the transport, then validates
the resulting binary with `b0state.ParseGenesis` and exact request/configured
chain IDs before durable height-zero initialization. Only that binary
determines state or AppHash; alternate transport representations are rejected.

**D-026 lifecycle supersession (Owner approved 2026-10-25):** The historical
sentence above requiring B0 `InitChain` to write Genesis H0 is superseded.
`plutod init-b0` durably creates and verifies the canonical H0 snapshot in a
fresh home before returning success. `plutod start` requires a complete,
existing snapshot and never creates H0. B0 `InitChain` still checks
`InitialHeight==1`, chain ID and the unchanged D-025 canonical JSON/Base64
transport, but validates the identical persisted H0 and returns its AppHash
without writing. Identical H0 retries return the same result; missing,
incomplete, conflicting or advanced application state fails closed. D-024
record bytes, AppHash V2 and all transaction validity rules are unchanged.

### `CheckTx`

Decode and evaluate each submitted transaction against the latest **committed** state only, for both NEW and RECHECK calls. Return Code 0 or the stable first-failure code above; never mutate committed or pending state. A locally accepted transaction is not promised inclusion or future validity. A transaction with nonce `n+1` may be rejected until nonce `n` commits, even if both would be valid sequentially in one proposed block. This conservative admission rule is intentional for B0. `CheckTx` success is not a consensus-validity proof, and a proposal may include bytes never seen by local `CheckTx`.

### `PrepareProposal`

Require the next height. Starting from a **throwaway copy** of committed state, visit `RequestPrepareProposal.Txs` in supplied order. Omit any noncanonical, unauthenticated or state-invalid candidate. Include a valid candidate only if adding it stays within the fixed count/byte bounds and the nonnegative `MaxTxBytes` budget; advance the throwaway state only for included candidates. Continue scanning later candidates, so a skipped large candidate need not exclude a later fitting candidate. If `MaxTxBytes <= 0`, return an empty list. Return no state update. This is proposer-local selection; the input list may depend on mempool arrival, but consensus validity is checked independently below.

### `ProcessProposal`

Require the next height and fixed block limits. Starting from committed state, decode and evaluate **every supplied transaction in order** on a throwaway state. Return `REJECT` if any transaction fails or if the block exceeds the count/byte limits; otherwise return `ACCEPT`. Never trust `CheckTx`, `PrepareProposal`, a local mempool or a cached proposal result. Never mutate committed/pending state. Correct validators with the same committed state and full proposal must return the same status, including for proposer-injected bytes. No wall clock, resource budget, vote extension or local queue may affect the status. A correct proposer applying the previous method produces an acceptable proposal under the same state.

### `FinalizeBlock`

Require the next height and no already pending block. Re-run the same evaluator from committed state over the **decided** transaction sequence; do not assume this node previously ran `ProcessProposal` or that local `CheckTx` saw the transactions. For an in-limit block, emit exactly one `ExecTxResult` per input transaction in the same order. Apply each successful transfer to the candidate state. For each invalid transaction, emit its first-failure code and leave the candidate state unchanged, then continue with later transactions. If the decided block exceeds the fixed app count/byte limit, emit Code 10 for **every** transaction, apply none, and still advance the block height. In both cases, compute a pending V2 AppHash after advancing height once. Return no successful execution claim for any unsupported or invalid transaction. `TxResults` length always equals input tx count; `ValidatorUpdates` is empty and `ConsensusParamUpdates` is nil.

This fallback defines deterministic behavior for an anomalous decided block that a correct `ProcessProposal` would have rejected. It avoids treating ordinary invalid transaction bytes as a fatal ABCI error. For every proposal accepted by a correct `ProcessProposal`, all `FinalizeBlock` transaction results are Code 0 and its state is exactly the state simulated during proposal validation, plus the height increment. Wrong height, missing initialized state or broken internal state integrity is a fatal application error, not a transaction result; no pending state is exposed. The block-limit fallback is a safety rule, not permission for a correct proposer to exceed limits.

### `Commit`

Atomically persist the complete pending account state, height and V2 commitment with a synchronous durability boundary before acknowledging `ResponseCommit`; then promote pending to committed. `ResponseCommit` in v0.38.26 does **not** carry AppHash. On persistence failure return an ABCI error and do not promote or clear pending as if successful. Crash/replay and physical storage details are Phase 03C work. A duplicate or pre-finalization `Commit` is an error.

### `Query`

Queries are non-consensus reads of committed state only. The specified exact account endpoint is `Path="b0/account/v1"`, `Data=address[20]`, `Height=0` (latest only), `Prove=false`. Return `ResponseQuery.Code=0`, `Height=committed height`, and `Value=0x01 || uint64_be(balance) || uint64_be(nonce)` for a present account, or the single byte `0x00` for an absent account. For any other path, malformed address length, nonzero requested height or `Prove=true`, return Code 1, `Codespace="pluto.query.v1"`, empty Value and `Log="unsupported_query"`. No Merkle or historical proof is claimed. Callers must account for local-node staleness.

### Other methods

Snapshot/state sync stays unsupported and vote extensions stay disabled as in Phase 02. Enabling either requires a separate design and test gate. A future implementation must continue to provide all pinned interface methods and preserve the Phase 02 single-validator node until each B0 step is approved.

## 3. Outcome separation and safety invariants

| Path | Meaning of rejection/failure | State effect |
|---|---|---|
| `CheckTx` nonzero Code | Local admission refusal against committed state. | None; does not decide consensus. |
| `ProcessProposal` `REJECT` | Consensus validation of the complete proposal failed. | None; correct validators do not vote for that proposal. |
| `FinalizeBlock` nonzero `ExecTxResult` | An anomalous decided block contains a failed transaction. | That tx has no effect; later txs evaluate against prior successes; block height still advances. |
| `Commit` error | Durability or internal failure. | Do not claim commit; operator/replay intervention may be required. |

The first three outcomes are not interchangeable. Proposal validation and final execution share bytes, validation order and state-transition rules. A transaction is never made valid by having passed local admission, nor made invalid by resource pressure. Every correct validator processing the same committed state and decided block must produce identical ordered results, resulting state and AppHash. See the pinned [v0.38.26 ABCI++ specification](https://github.com/cometbft/cometbft/blob/v0.38.26/spec/abci/abci%2B%2B_app_requirements.md) for the surrounding engine contract.

## 4. Implementation validation and future changes

The Owner accepted the deterministic invalid-decided-block fallback, fixed 6,144/1,048,576 application bounds and latest-only account query as V1 rules. Later implementation must test their interaction with CometBFT genesis block parameters, four-validator liveness, replay and persistence. A different fallback, limit or query policy requires a new reviewed decision; none is changed by this closeout. No runtime policy is implemented now.
