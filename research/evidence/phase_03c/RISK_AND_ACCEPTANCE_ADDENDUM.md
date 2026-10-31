# Phase 03C risk and acceptance addendum — 2026-10-23

**Status:** Documentation for Controller and Owner review. Phase 03C functional
code and the R1 fail-closed read correction passed technical review. This
addendum does **not** accept D-024, merge Draft PR #6, authorize Phase 03D,
or claim that B0 V2 runs in the live ABCI application.

**Reviewed revision:** `e0704b4533aab522900c7d98584435f4c24021f1`
on `phase/03c-b0-state-persistence`; integration baseline
`61bc65fdebf358fb0330cb9ebd4a332ca4a06aca`. The existing Phase 02
runtime remains transaction-free. Historical logs, vectors and accepted
D-020–D-022 specifications are unchanged.

## 1. Evidence labels and pinned references

- **Verified in Phase 03C:** standalone B0 V2 state/storage tests, graceful
  restart, process termination after Prepare or acknowledged Commit, injected
  pre-write/returned-write/post-sync errors, corruption rejection, R1
  fail-closed reads, race checks and independent AppHash vectors. See the
  [Phase 03C report](../../PHASE_REPORTS/PHASE_03C_REPORT.md), original
  [verification log](verification.log) and [R1 log](rework_r1/verification.log).
- **Specified or inspected:** CometBFT v0.38.26 ABCI and replay behavior,
  Pebble v1.1.5 API contract and Pluto's accepted B0 protocol. Source review
  does not constitute an executed V2 cross-store recovery test.
- **Unverified:** every B0 V2/CometBFT cross-store crash row in Section 3,
  interrupted WAL append/fsync, storage-device faults and physical power loss.
- **Analytical only:** O(K) cost analysis and the benchmark plan. No new
  benchmark has been implemented or run.

Pinned primary sources: [CometBFT v0.38.26 application requirements](https://github.com/cometbft/cometbft/blob/v0.38.26/spec/abci/abci%2B%2B_app_requirements.md),
[consensus block persistence](https://github.com/cometbft/cometbft/blob/v0.38.26/consensus/state.go),
[block execution and Commit](https://github.com/cometbft/cometbft/blob/v0.38.26/state/execution.go),
[handshake and replay](https://github.com/cometbft/cometbft/blob/v0.38.26/consensus/replay.go),
[FinalizeBlock-response persistence](https://github.com/cometbft/cometbft/blob/v0.38.26/state/store.go),
[Pebble v1.1.5 `Sync` contract](https://github.com/cockroachdb/pebble/blob/v1.1.5/options.go)
and [`DB.Set`](https://github.com/cockroachdb/pebble/blob/v1.1.5/db.go).
The module versions were checked against `go.mod` and the pinned sources in
the local Go module cache; no later CometBFT or Pebble API was substituted.

## 2. Exact atomicity and durability boundary

`b0store` encodes one complete logical V2 snapshot and its AppHash in one
versioned value at key `pluto/committed-state/v2`. `Commit` calls Pebble
`DB.Set(key, value, pebble.Sync)` and promotes the pending in-memory state
only after that call and any post-sync hook return successfully. The value
is a single application-level record replacement, not a transaction across
Pebble, CometBFT's blockstore, its state store and the ABCI transport.
`pebble.Sync` requests a synchronized WAL write under Pebble v1.1.5; it is
not proof that a filesystem or device honored the request under all failures.

An explicitly injected `BeforeSync` error occurs before `DB.Set`: old
committed state remains readable and pending remains available for retry.
A returned `DB.Set` error has an **uncertain** durable outcome. A post-sync
response error demonstrates that a new complete snapshot may persist even
though the caller received an error. R1 makes both `Committed()` and
`Pending()` return `ErrUncertain` with no snapshot while this state is open;
`Prepare()` and `Commit()` also reject work. The store neither promotes the
old in-memory state nor guesses the durable outcome. Close/reopen validates
the complete record, version, chain context, canonical account invariants
and recomputed AppHash before exposing a recovered state. A corrupt or
missing record fails closed; it is not reset or migrated.

These boundaries support different claims:

| Boundary | Phase 03C evidence | Limit |
|---|---|---|
| Application-level record replacement | One full `DB.Set` value; old/new complete snapshots observed in injected recovery cases | No shared atomic commit with CometBFT |
| Synchronous WAL request | Production path uses `pebble.Sync`; code/API reviewed | Storage-stack compliance under arbitrary faults untested |
| Graceful restart | New isolated database reopened after genesis, transfer and empty block | Does not exercise sudden hardware failure |
| OS process termination | Disposable child killed after pending state and after acknowledged Commit; old/new full records reloaded | Does not interrupt WAL append or fsync |
| Crash during WAL append/fsync | **NOT RUN** | Old/new/corrupt/error outcome must be established by controlled future tests; never infer old bytes from a returned error |
| Filesystem/device failure | **NOT RUN** | Torn, missing or reordered durable writes and directory metadata may defeat assumptions; loader must reject invalid snapshots |
| Physical power loss | **NOT RUN** | No hardware durability claim |
| CometBFT/application cross-store recovery | **NOT RUN for B0 V2** | The live Phase 02 empty-block integration restart is a different implementation |

Fresh B0 `Create` requires a nonexistent directory. An interrupted genesis
creation can leave an existing but incomplete directory; automatic retries
currently fail rather than reset it. Future `InitChain` integration needs an
explicit, tested recovery/idempotency policy compatible with D-021.

### Pinned CometBFT checkpoint order

For decided block **H**, pinned code first saves the block in the CometBFT
blockstore, calls `FinalizeBlock`, saves its response (including AppHash H),
calls application `Commit`, and then saves CometBFT's updated consensus
state. The v0.38.26 requirements summarize three crash checkpoints through
application persistence; `state/execution.go` exposes the additional
post-Commit consensus-state save. These are distinct stores and are not a
shared transaction. `ResponseFinalizeBlock.AppHash` is state **after H**;
block header H carries the prior application hash, while CometBFT's updated
state at H carries the new hash. A later block header H+1 carries the hash
after H.

At startup, the pinned handshaker calls application `Info` for committed
height/hash, compares them with blockstore and consensus-state heights, and
selects real-app replay when the app is behind. When the app reports H but
CometBFT's consensus state remains H-1 with block H stored, it uses the
persisted FinalizeBlock response with a mock app to advance CometBFT state
without reapplying the application transfer. An app ahead of the blockstore
is an error; several inconsistent height/hash cases panic or fail startup.
The pinned requirements describe deterministic reexecution and comparison
after a pre-Commit crash; the inspected replay path reexecutes through the
real app. Phase 03D must test the exact comparison and failure behavior for
Pluto rather than assuming that every response field is compared by code.

## 3. Cross-store recovery matrix

Notation: **B** = CometBFT blockstore height, **C** = persisted CometBFT
consensus-state height, **A** = durable application height reported by `Info`.
The rows below describe one target block H. Every B0 V2 cross-store case is
**NOT RUN in Phase 03C**. A future row is **PASS only if** its stated outcome
is demonstrated with exact heights, account balances/nonces, FinalizeBlock
results and correctly related AppHashes; it is **FAIL** if a duplicate state
change, silent reset, divergent result, invalid accepted snapshot or
unreported mismatch occurs. Failure to produce a required observation is
**UNVERIFIED**, not PASS.

| Case | Expected pinned behavior | Phase 03C observation / gap | Required future test and objective PASS condition | Owner |
|---|---|---|---|---|
| M1. Crash before block persistence | No H block is replayable; app and stored consensus state remain H-1. Consensus may later propose H again. | Standalone pending isolation tested; no V2 ABCI node. **NOT RUN.** | Kill before `SaveBlock`; reopen, confirm no H transfer committed and next H proceeds once. | 03D |
| M2. Block persisted, before application execution | B=H, C=A=H-1; handshake replays H through the real app. | Standalone ordered replay tested only. **NOT RUN.** | Kill after `SaveBlock` before `FinalizeBlock`; restart and confirm exactly one transfer, one nonce increment and AppHash H. | 03D |
| M3. After FinalizeBlock/results persistence, before Commit | B=H, C=A=H-1; real-app reexecution is needed and should reproduce the stored result/hash. | Pending is non-durable; Comet result persistence untested. **NOT RUN.** | Kill after response save; compare exact reexecution results and hash, confirm one Commit and no discrepancy silently accepted. | 03D |
| M4. During application Commit, including WAL append/fsync | Durable A may remain H-1, reach H, or fail integrity validation. Old valid state takes real replay; new valid state takes mock replay when C=H-1. | Pre-write, returned-error and post-sync simulations cover old/new records, but no interrupted WAL/fsync or cross-store handshake. **NOT RUN.** | Fault at controlled write boundaries; reopen both stores. PASS only for complete valid old/new plus correct replay path, or explicit fail-closed error for corrupt storage. | 03D; real hardware campaign only if separately authorized |
| M5. Durable app Commit, before acknowledgment | B=H, A=H, C=H-1; handshake uses stored FinalizeBlock response and mock app, without reapplying H. | Standalone post-sync response error recovers H; no ABCI ack test. **NOT RUN.** | Kill or drop acknowledgment after durable write; verify mock path, unchanged balance/nonce and Comet state reaches H. | 03D |
| M6. After successful Commit | If C=H-1, mock replay completes the later consensus-state save; if C=H, matching `Info` state needs no replay. | Standalone successful restart tested; no cross-store checkpoints. **NOT RUN.** | Kill on both sides of Comet state save; verify no second transfer and correct H hash in both cases. | 03D |
| M7. Application crash while CometBFT remains alive | The pinned requirements assume coordinated crashes; connection loss cannot safely imply Commit failure or success. Progress must stop until validated restart/handshake. | No live B0 V2 app. **NOT RUN.** | Terminate only app process at each ABCI boundary; require no new committed height before coordinated recovery and exact reopened state. | 03D |
| M8. CometBFT crash while application remains alive | Recovery depends on B,C,A. If A is ahead of B, pinned replay returns a height error; if B=H,A=H,C=H-1, mock replay is expected. | No live B0 V2 app. **NOT RUN.** | Terminate only CometBFT process before/after Commit; require valid handshake or explicit startup failure, never implicit app rollback. | 03D |
| M9. Mismatched application/consensus heights | App behind available blocks may replay; app ahead of blockstore, consensus state ahead of blockstore, unavailable pruned history or impossible gaps fail startup. | V2 loader validates its own height, not cross-store height. **NOT RUN.** | Construct each height relation in isolated homes; PASS only for contract-permitted replay or explicit fail-closed error/panic before serving. | 03D |
| M10. Mismatched expected AppHash | Equal-height app/Comet state comparison and replay hash checks must detect mismatch; header H uses state H-1, so compare matching state heights. | V2 record self-hash validation tested; external Comet hash comparison absent. **NOT RUN.** | Use integrity-valid but different app state and tampered expected hashes; require startup or replay failure before serving or committing more blocks. | 03D |
| M11. Interrupted genesis initialization | With app `Info` height 0, Comet may call `InitChain` again. V2 `Create` currently refuses an existing path, so incomplete directory needs explicit handling; no reset/migration. | Fresh/duplicate Create behavior tested only. **NOT RUN.** | Kill before/after directory creation and genesis sync; require either validated identical height-0 retry or explicit operator-visible failure, never silent reset. | 03D |
| M12. Duplicate replay/reexecution attempt | App at H must not execute H again; app at H-1 may execute H once. Block replay must be height-aware, not rely only on a later nonce error. | Standalone duplicate tx fails nonce; no block replay integration. **NOT RUN.** | Repeat H around Commit/ack/state-save crashes; PASS only if transfer and nonce apply exactly once and response/hash agree. | 03D; expand to four validators in 03E |

M4's future simulated fault test is not a substitute for a separately
authorized physical power-loss campaign. An application crash while the
CometBFT child lives, or vice versa, explicitly tests outside the pinned
co-crash assumption. No matrix row is marked PASS from source inspection.

## 4. R2 — state validation resource amplification

The current `b0state.Evaluate` calls full `State.Validate()` **before**
`tx.Validate`. `State.Validate` traverses every account, serializes all K
records and recomputes AppHash. Consequently, even malformed wire bytes,
wrong-chain transactions, zero amounts and invalid signatures cause O(K)
state work in this standalone evaluator. `SimulateOrdered` also validates
the initial state once and calls `Evaluate` for each transaction. With T
cheap-invalid transactions, the current path therefore has O(TK) state
validation work before counting the fixed-size transaction checks.

For a successful transfer, `Evaluate` additionally copies accounts and
`NewState` copies/traverses/hashes them again. `AdvanceBlock` validates and
rehashes; `Commit` validates, serializes and writes the full record. This
can increase transaction latency, memory allocation, garbage-collection
pressure, full-state hashing time and synchronous persistence latency as K
grows. If connected unchanged to future `CheckTx`, malformed and
invalid-signature admission workloads could spend state work before cheap
rejection, obscuring cryptographic-cost comparisons and amplifying invalid
input resource cost. Multi-transaction blocks multiply this work. The live
Phase 02 `CheckTx` does **not** call B0 state code; this is a future
integration risk, **not a measured or demonstrated exploitable DoS**.

For chain ID length N, canonical logical bytes are `26 + N + 36K`; with
`N=11` (`pluto-b0-v1`), this is 73, 2,341, 147,493 and 2,359,333 bytes at
K=1, 64, 4,096 and 65,536 respectively. These are analytical serialized
sizes, not measured allocations or latency. The AppHash accessor returns a
cached digest; benchmarks must time recomputation via a state transition or
explicit serialization/hash operation, not mistake accessor cost for hash
cost.

## 5. Mandatory future microbenchmark design — plan only

No new benchmark was implemented or executed. After separate authorization,
construct immutable, integrity-valid base states at K=`1,64,4096,65536`.
Use `pluto-b0-v1`, public Phase 03B test keys A=1 and B=2, and sorted unique
addresses: A and, for K>=64, B; fill remaining slots with the fixed
20-byte sequence `16 zero bytes || uint32_be(i)` for i starting at zero.
Give every account 1,000,000 units and nonce zero; checked supply is
`K*1,000,000`. Use fixed, signed wire fixtures (amount 25, nonce 0 where
applicable), assert the expected code/hash after every trial, and reset to
the same base snapshot before each iteration. K=1 has no existing distinct
receiver: its A-to-B transfer creates B and ends at K=2. Label that case
**account creation** and do not pool it with K>=64 existing-receiver results.

| Workload | Required result or operation |
|---|---|
| Malformed wire | Code 1, unchanged state |
| Canonical wrong-chain wire | Code 2, unchanged state |
| Signed zero amount | Code 5, unchanged state |
| Noncanonical signature scalar | Code 4, unchanged state |
| Canonical scalars but invalid ECDSA | Code 4, unchanged state |
| Valid self-transfer | Code 0, nonce +1, unchanged balance/supply |
| Valid distinct transfer | Code 0; K=1 creates receiver, K>=64 credits existing receiver |
| Empty-block advancement | Height +1 and changed AppHash, no account change |
| Canonical serialization and AppHash recomputation | Exact bytes/hash; separate from cached `AppHash()` accessor |
| Synchronous full-snapshot persistence | `pebble.Sync` complete-record Commit in disposable databases |

For every K/workload report `ns/op`, `B/op`, `allocs/op`, CPU time, logical
bytes serialized, bytes hashed and the validation stage/result. For repeated
persistence trials report latency distribution (at least median, p95, p99
when sample size supports those quantiles), WAL/compaction state and full
record size. Keep storage medium, filesystem and fsync settings fixed and
reported; separate warm and cold database trials rather than mixing them.
Record exact Go version, OS/kernel, CPU, memory, `GOMAXPROCS`, GC settings,
storage device, source SHA, fixture bytes, test command, warm-up and measured
trial counts. Plan at least three warm-up passes and five independent
measured trials per point; retain raw samples and confidence/variance
summaries. Do not infer CPU time from wall `ns/op`; use an explicit CPU
profiler or process CPU counters with their measurement limits recorded.

A separately authorized comparison may evaluate a design that verifies a
committed snapshot at load/commit boundaries and avoids repeated full-state
validation for cheap-invalid bytes. Hold accepted codes, state hashes,
account fixtures and execution model constant, and report any change in
integrity-detection boundary. This is a measurement comparator, **not** an
implemented optimization or a new admission policy. No throughput or DoS
result is claimed, and no performance threshold is invented here.

## 6. Phase 03D safety gates before live B0 activation

All gates below are **NOT RUN / NOT SATISFIED in Phase 03C**. Phase 03D is
not authorized; ownership below is a planning assignment only. A gate fails
if evidence is missing or its pass condition is violated. The Owner and
Controller decide acceptance.

| Gate | Owner | Required evidence | Objective PASS condition |
|---|---|---|---|
| G1 Authentication precedence | 03D | Real ABCI `CheckTx`, proposal and finalization tests with Code 1/2/5/4/3/6–9 precedence, plus cost trace or profiler for cheap-invalid paths | Cheap stateless rejection does not perform unnecessary full-state validation; codes and deterministic consensus decisions match accepted D-020–D-022 |
| G2 State integrity boundaries | 03D | Documented trust model, load/commit integrity checks, immutable-state tests, corruption and race injection | Corrupt committed state is detected before use; cheap-invalid transactions need no repeated O(K) rehash; successful transitions preserve invariants |
| G3 Crash-recovery handshake | 03D | Disposable real-node fault runs for matrix M1–M11, exact `Info`/`FinalizeBlock`/`Commit` traces and persisted heights/results | Every tested boundary replays or fails closed as specified, with no unsupported success claim |
| G4 Duplicate execution protection | 03D | M5/M6/M12 replay tests with balances, nonces and result comparison | Each committed transfer is applied exactly once across restart/ack ambiguity |
| G5 Height and AppHash consistency | 03D | Recorded application `Info`, FinalizeBlock response, Comet consensus state and block headers at adjacent heights | Compare matching state heights; H header uses app state H-1, updated Comet state H and H+1 header use app hash H; any mismatch blocks progress |
| G6 Failure-atomic persistence | 03D | B0 ABCI wiring tests for pre-write, returned-write, post-sync error and reopen; preserve R1 assertions | No ambiguous state exposed or retried, no in-memory promotion on failure, complete valid recovered record only |
| G7 Resource-cost measurement | 03D before live admission claims | Reproducible K=1/64/4096/65536 results for Section 5 workloads, raw samples and environment | Costs and confounders are reported separately for state validation, authentication, hashing and fsync; no unsupported performance claim; no unapproved threshold |
| G8 Existing semantics | 03D | Diff against Phase 02 baseline and accepted specs; real-node positive/negative vectors | No silent D-020–D-022 change or implicit V1 migration; Phase 02 behavior remains preserved until separately authorized B0 activation |

Phase 03E owns expanded four-validator agreement and recovery tests after
separate authorization. It cannot substitute for the Phase 03D single-node
gates. No gate above authorizes implementation by itself.

## 7. D-024 decision boundary and recommendation

**D-024 status: PROPOSED — AWAITING EXPLICIT OWNER ACCEPTANCE.** The
Controller recommends the single-record Pebble snapshot as a
correctness-oriented **experimental B0 baseline**, subject to its O(K)
copy/hash/write cost, no historical account proofs, unverified physical
power-loss behavior and untested V2 cross-store recovery. It is neither
production-grade nor performance-optimized. If future controlled
measurements justify a different storage strategy, that change requires a
separately versioned design, explicit decision, migration/compatibility
policy and tests. This addendum does not alter the selected D-024 layout or
any accepted protocol rule.
