# B0 V2 state API and physical layout — Phase 03C candidate

Source baseline: `61bc65fdebf358fb0330cb9ebd4a332ca4a06aca`.
Decision D-024 is proposed for Controller and Owner review.

## Logical state and genesis

`b0state.ParseGenesis(wire, expectedChainID)` validates the exact accepted
`PLUTOG2!` bytes and produces a height-zero `State`. `EncodeGenesis` emits
canonical bytes from already ordered positive allocations. `NewState` accepts
only a sorted, unique, supply-conserving snapshot. Each account contains
`address[20]`, `uint64` balance and `uint64` nonce. Account count is 1–65,536;
missing accounts act as virtual zero-balance/zero-nonce accounts and are not
serialized. `Accounts()` returns a copy. `LogicalBytes()` and `AppHash()`
expose the D-021 canonical state commitment; `DecodeLogical` requires exact
bytes, an expected digest and an explicit chain context. Height advance is
checked at MaxInt64.

`Evaluate(state, raw)` calls the accepted Phase 03B stateless validator,
then checks nonce/increment, sender balance, receiver overflow and account cap
in that order. It returns an unchanged state for each transaction rejection;
internal state errors use Go `error`. `SimulateOrdered` applies each successful
transfer to the next candidate and continues after failed transactions.
Neither function creates ABCI responses. The receiver-overflow check is
defensive and unreachable from an integrity-valid conserved-supply state
when a different sender has sufficient balance.

## Store lifecycle

`b0store.Create(dir, genesis, chain)` requires a nonexistent directory and
explicit canonical genesis. `OpenExisting(dir, chain)` requires a complete
valid V2 record and fails if the V1 key is present; neither call migrates or
resets data. `Committed()` and `Pending()` return read-only state values.
`Prepare(raws)` simulates an ordered block and advances height exactly once,
but writes nothing. Only `Commit()` calls Pebble `Set(..., pebble.Sync)` and
promotes pending after a successful response. A second Prepare or Commit
fails with explicit lifecycle errors. A pre-sync injected failure retains
pending for retry. A returned Set error or post-sync response error leaves
in-memory committed unchanged and marks the store uncertain; it must close
and reopen to learn which complete state persisted. A mutex protects state
and Pebble access. `Close()` releases the lock.

Physical key: ASCII `pluto/committed-state/v2`. Physical value:

```text
ASCII("PLUTOS2!") || logical_state_v2 || AppHashV2[32]
```

The V1 key is ASCII `pluto/committed-state/v1`; V1 uses `PLUTO02!`. The
complete logical state and digest are validated before accepting a reopened
database. One record and one synchronous Pebble write provide an atomic
application record replacement. This does not prove physical power-loss
durability or CometBFT cross-store replay behavior. Serialization, hashing
and full-record writes cost O(K) for K accounts; B0/B1 performance comparisons
must control account count and execution model.

All tests create disposable directories with `t.TempDir()`. Fault hooks are
controlled test seams; they do not model every operating-system failure. A
subprocess test also terminates a disposable writer after Prepare or after a
successful Commit, then reopens and validates the old or new complete
snapshot respectively. It is an OS process-kill test, not a power-loss test.

## Controller R1 addendum — 2026-10-23

After a returned Pebble write error or post-sync response error, both
`Committed()` and `Pending()` return `ErrUncertain` and no snapshot. Reads
remain available after an explicitly pre-write failure because that hook ran
before any write attempt. `Prepare()` and `Commit()` continue to reject
uncertain state. Closing and reopening validates the complete persisted
record and resolves whether the old or new snapshot became durable. The
in-memory committed value is not promoted and pending is not discarded on a
failed Commit response. See the separately appended R1 evidence; earlier
execution logs are historical and unchanged.
