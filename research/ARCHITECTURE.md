# Pluto architecture — current profiles and historical design record

At `test-mainet` baseline `0bdd869f2c6b3ed59b0ae825954d8667b3c7481e`,
Phase 02, B0 and B1 are selectable local profiles. B0 and B1 have tested
four-validator localhost configurations. Phase 05's evaluation is merged but
bounded; the new four-profile research program is tracked separately in
[`research/checklists/01_CHECKLIST_CHUAN_HOA_MO_RONG_REPO_THUC_NGHIEM.md`](checklists/01_CHECKLIST_CHUAN_HOA_MO_RONG_REPO_THUC_NGHIEM.md).
The historical candidate/authorization wording below describes prior review
points and must not be read as current implementation status.

## S03/S04 profile and correctness boundary

The S03 candidate in `internal/profiles` has one profile-selected canonical
transaction parser, one genesis-bound account/key registry, one transfer
transition and one AppHash rule for C, CP, P and H. The direct-ABCI
`internal/profiles/app` applies the same callbacks for each profile. CP
accepts exactly 2,420 zero padding bytes and matches H's full wire size for
the same chain ID; the padding is excluded from the signed intent. The
[equivalence report](protocol/PROFILE_EQUIVALENCE.md) records
the codec and tests. S04 adds an optional isolated Pebble snapshot to this
same ABCI++ app, canonical JSON/Base64 genesis transport, and a public
correctness corpus. The integration harness starts four CometBFT processes
with distinct validator/P2P identities, four ABCI servers in the test process
and four separate stores on localhost. It checks a transaction against the
corpus oracle, block/AppHash equality at matching heights, four participating
voter identities across commits, account balances/nonces and replay cache.
The [S04 report](evidence/s04/CORRECTNESS_REPORT.md) records
the exact scope. The ordinary `plutod` CLI has no four-profile research selection; the
test harness is the S04 node wiring. B0/B1 code remains intact.

## Phase 05 experimental measurement boundary

`internal/benchmarks/corpus` holds public, hash-checked B0/B1 transaction and
genesis fixtures. The generator uses ephemeral signing keys and publishes
only signed bytes and public bindings. The oracle runs accepted state
evaluators and real ABCI CheckTx/ProcessProposal against disposable stores.
`internal/benchmarks/bench` contains research-only stage/callback timers and Go
microbenchmarks; no production application imports it. The four-validator
benchmark test provisions four fresh homes with the accepted B0/B1 node APIs,
uses localhost RPC/P2P, records per-process diagnostics and checks matched
block/state history. Analysis scripts consume retained raw JSON, using
independent runs as the statistical unit. These components neither change
consensus validity nor install a DoS admission mechanism. Exact boundaries,
confounders and hypotheses are in
[`PHASE_05_EXPERIMENT.md`](specs/PHASE_05_EXPERIMENT.md).

## Historical Phase 04 B1 hybrid design

A separately selected B1 profile uses the existing pinned CometBFT consensus
engine with a new `internal/b1app` ABCI adapter. `init-b1-4v` exclusively
claims a fresh root, generates four independent Comet validator/P2P
identities, installs one canonical Genesis V3 in every home, and writes one
independent B1 H0 Pebble snapshot per validator. `start` validates the B1
record and genesis before launching CometBFT. B0 and Phase 02 stores are
rejected in B1 homes; no migration or fallback exists.

`internal/b1tx` owns bounded canonical transaction decoding, the common
domain-separated signed digest, and the two independent signature checks.
`internal/b1state` binds sender/public keys to an immutable genesis registry
and invokes the accepted B0 balance/nonce transition only after both proofs
pass. `internal/b1store` persists the B1 registry-bound AppHash under a
separate key and record magic. `CheckTx` reads committed state;
`PrepareProposal` and `ProcessProposal` independently simulate ordered
candidate state; `FinalizeBlock` prepares a pending candidate and `Commit`
syncs it. Direct invalid-proposal tests bypass mempool admission. The
application reports the last durable height and AppHash on restart, allowing
the pinned CometBFT handshake/replay path. Four validators on one WSL2 host
are four processes but one physical fault domain. The exact binary profile,
error taxonomy and threat assumptions are in
[`specs/B1_HYBRID_AUTHENTICATION.md`](specs/B1_HYBRID_AUTHENTICATION.md).

## Historical Phase 03E local four-validator extension

`init-b0-4v` creates four independent D-026 homes under one fresh root. It
generates four validator and four P2P identities, writes one shared CometBFT
genesis with four equal-power validators and the unchanged D-025 B0 Genesis
V2 app state, then writes one D-024 H0 snapshot per home. Each home has its
own local ABCI, RPC and P2P endpoint and Pebble database. CometBFT persistent
peers connect the four local P2P instances; the only shared data are public
genesis contents and protocol rules. Production `start` checks that the local
validator signing key matches one member of the genesis validator set before
opening the application. The application performs the same Phase 03D
CheckTx, proposal, FinalizeBlock, Commit and replay logic in each process.
The test harness compares block hashes and AppHashes at matching heights.
All four processes run on one WSL2 host, so host or filesystem failure is a
common-mode fault and no Byzantine or physical power-loss claim follows.

The historical description below reflects the earlier single-validator
implementation and its acceptance sequence.

This section supersedes the historical Phase 01 proposal retained below for
traceability. The historical Phase 02 profile remains a **single-validator,
transaction-free** custom Go ABCI++ application. A separately selected B0 V2
profile now executes classical transactions in a single-validator local
CometBFT v0.38.26 node. Neither profile is EVM or hybrid.
Phase 02R retains this implementation. D-019 requires a complete minimal
classical B0-Core before deciding at Phase 04 whether EVM integration is
needed. Phase 03C's isolated state subsystem was merged at
`b56c81ccce7ea4a0e7f52053c7459c4601c15b38`. Phase 03D began with an
accepted D-025 genesis transport checkpoint. The Owner subsequently closed
that checkpoint and authorized B0 transaction integration; the current
review candidate is described in the [Phase 03D report](PHASE_REPORTS/PHASE_03D_REPORT.md).
The subsequent [recovery closure report](PHASE_REPORTS/PHASE_03D_RECOVERY_CLOSURE_REPORT.md)
records real-node WAL, replay, height/hash and genesis fault evidence; Phase
03D remains unaccepted pending Controller review and residual recovery policy.
The owner has accepted Phase 03A's B0 V1 transaction and V2 logical-state
specifications in `research/specs/` and D-020–D-022 after Controller rework.
The subsequent explicit Phase 03B authorization permits a standalone
`internal/tx` stateless library. It does not change the transaction-free node,
ABCI implementation, Pebble record or AppHash V1.
Phase 03B was completed, Controller-accepted and merged at
`8be07c87abed73763d40252e99bbf0573da487b4`; D-023 is Owner-accepted.
The Owner then authorized Phase 03C from verified `test-mainet` SHA
`61bc65fdebf358fb0330cb9ebd4a332ca4a06aca`. Standalone
`internal/b0state` provides canonical V2 genesis/state, ordered transaction
simulation and AppHash V2. Standalone `internal/b0store` persists one complete
V2 snapshot in a separate Pebble database with a synchronous write and an
isolated pending candidate. D-024 is Owner-accepted on 2026-10-24 strictly as
a correctness-oriented experimental B0 layout. Its full-snapshot O(K) cost,
unverified B0 V2 cross-store recovery and physical-failure limits remain in
the [risk addendum](evidence/phase_03c/RISK_AND_ACCEPTANCE_ADDENDUM.md).
Neither package is imported by the historical `internal/app`; its AppHash
remains the Phase 02 V1 digest. The new B0 profile imports the isolated V2
packages through a separate ABCI adapter.

**D-025 transport boundary:** `plutod init-b0` reads canonical Genesis V2
binary and writes its standard padded Base64 as one JSON `app_state` string.
Pinned CometBFT forwards those JSON bytes to `internal/b0app.InitChain`.
`internal/b0genesis` strictly decodes them, and `b0state.ParseGenesis` remains
the sole binary parser. Historically `b0app.InitChain` initialized D-024
storage at height zero. Owner-approved D-026 supersedes that creation actor:
`init-b0` creates and verifies H0, `start` requires an existing valid B0
snapshot, and `InitChain` validates the matching persisted H0 without writing;
the B0 application now validates and executes classical transfers through
the accepted `internal/tx`, `internal/b0state` and `internal/b0store` packages.
`CheckTx` reads committed state only; proposal methods use throwaway state;
`FinalizeBlock` prepares pending state; `Commit` performs the synchronized
D-024 snapshot write. The historical Phase 02 `init` profile still uses V1
state and its original empty-block ABCI application.

**D-026 B0 lifecycle (Owner-approved Phase 03D implementation target):**

```mermaid
sequenceDiagram
  participant O as Operator
  participant N as Pluto node
  participant P as B0 Pebble store
  participant C as CometBFT v0.38.26
  O->>N: init-b0 with canonical Genesis V2
  N->>P: Create and sync H0 using D-024 record
  N->>P: Reopen and verify H0
  N-->>O: success after full home verification
  O->>N: start
  N->>P: OpenExisting and verify snapshot
  N->>C: start real node
  C->>N: Info (H0, AppHash V2)
  C->>N: InitChain (D-025 JSON/Base64)
  N->>P: read and validate identical H0
  N-->>C: H0 AppHash, no genesis write
```

The historical Phase 02 sequence below retains its separate V1 `InitChain`
write and is not changed by D-026.

```mermaid
flowchart TB
  Operator[plutod init or init-b0/start] --> Runtime[internal/node]
  Client[HTTP RPC reader] --> Comet[CometBFT: RPC / P2P / consensus / mempool]
  Runtime --> Comet
  Comet -->|historical V1 localhost ABCI| App[internal/app]
  App --> State[internal/store: V1 state and AppHash]
  Comet -->|explicit B0 localhost ABCI| B0[internal/b0app: classical B0 transfers]
  B0 --> V2[internal/b0state + internal/b0store: V2 state and AppHash]
  State --> Pebble[Pebble v1.1.5]
  V2 --> Pebble
  Future[Phase 03E validation / conditional EVM / hybrid] -. not implemented .-> B0
```

CometBFT owns consensus, validator signing, P2P, block construction and the
mempool. Pluto owns application state, ABCI decisions, persistence and AppHash.
`plutod init` invokes the bundled exact-version CometBFT binary to create
genesis, validator and node keys, then sets chain ID, initial height 1, empty
application genesis, localhost ABCI/RPC/P2P listeners and one-second
empty-block interval. `plutod start` checks genesis against Pluto metadata,
opens the exclusive Pebble database, starts the ABCI server and launches the
CometBFT child. SIGINT/SIGTERM stops the child and closes server/database.
`plutod state` reads the database only after node stop. Generated private
validator and node keys must remain private, outside research evidence.

## Implemented ABCI++ lifecycle

```mermaid
sequenceDiagram
  participant C as CometBFT v0.38.26
  participant A as Pluto application
  participant P as Pebble
  C->>A: Info
  A-->>C: last durable height and AppHash
  C->>A: InitChain (new chain only)
  A->>P: sync-write genesis record at height 0
  A-->>C: genesis AppHash; no validator overrides
  C->>A: PrepareProposal (proposer)
  A-->>C: empty tx list
  C->>A: ProcessProposal (all validators)
  A-->>C: ACCEPT only empty tx list and next height
  C->>A: FinalizeBlock (decided block)
  A-->>C: pending next-height AppHash (not durable)
  C->>A: Commit
  A->>P: sync-write entire pending record atomically
  A-->>C: ResponseCommit acknowledgment
```

`CheckTx` returns code 1 for every transaction: user transactions do not
exist in Phase 02. It does not change state. Proposal validation independently
rejects nonempty transaction lists; transactions in proposals need not have
passed every validator's `CheckTx`. A correct proposer drops all input txs.
`FinalizeBlock` errors on nonempty txs, incorrect heights or duplicate
pending finalization; it never reports successful unsupported execution.
Snapshot listing is empty, offers are rejected, queries unsupported, and vote
extensions are not enabled. State sync is not advertised.

The application keeps **committed** and **pending** states separately.
`FinalizeBlock` computes pending state only in memory. `Commit` replaces one
Pebble key with `pebble.Sync`, which requests WAL synchronization before
success, then promotes pending to committed. A failed Save returns an ABCI
error and leaves committed state unchanged. Pebble's lock prevents two writable
opens. On startup `Info` reports the validated persisted record; CometBFT
performs its app/block-store handshake and may replay blocks after a crash.
The integration test covers graceful restart, not arbitrary power-loss
durability or every replay edge case.

## Phase 02 state and AppHash contract

Consensus state is `(format_version uint32=1, chain_id ASCII string,
height int64>=0)`. Chain IDs are 1–50 bytes of ASCII letters, digits, `_`,
`-` or `.`. Genesis is height 0. Every committed empty block increments height
and changes AppHash. The digest is SHA-256 of this canonical byte sequence:

```text
ASCII("PLUTO/APP_STATE/V1") || 0x00 || uint32_be(1) ||
uint16_be(chain_id_byte_length) || chain_id_bytes || uint64_be(height)
```

The persistence record is `ASCII("PLUTO02!") || version || chain_id_length ||
chain_id || height || 32-byte AppHash`, using the same field encodings. Reload
validates exact length, version, chain ID, nonnegative height and recomputed
digest. This is neither a Merkle tree nor an Ethereum trie state root, and it
covers only Phase 02 state. Additional state requires a versioned commitment
and migration decision.

In CometBFT v0.38.26, `ResponseFinalizeBlock` returns the newly finalized
AppHash; `ResponseCommit` has no AppHash. Block header H's `app_hash` reflects
the state after H-1, not after H. `/abci_info` reports the application's
latest committed height and AppHash. These same-numbered RPC fields must not
be compared as though they referred to the same state height.

## Implemented and future module boundaries

```mermaid
flowchart LR
  CLI[cmd/plutod] --> Node[internal/node]
  Node --> App[internal/app]
  Node --> Comet[CometBFT binary and config]
  App --> Store[internal/store]
  App --> Types[CometBFT ABCI types]
  Store --> Pebble[Pebble]
  Library[internal/tx: standalone B0 V1 codec and authentication]
  B0[internal/b0state: isolated V2 state and transitions] --> Library
  B0Store[internal/b0store: isolated V2 synced snapshot] --> B0
  CLI --> Version[internal/version]
  Hybrid[future internal/hybrid] -. canonical authentication .-> App
  EVM[future internal/evm] -. sequential execution .-> App
  RPC[future application RPC] -. tx bytes .-> App
```

Future Pluto transaction handling must separate canonical bytes, secp256k1
and ML-DSA-44 authentication of the same intent, account/key binding,
state-dependent validity, persistent state and AppHash. If Phase 04 approves
EVM, its first execution model is sequential. `HYBRID_REQUIRED` must fail
closed. Consensus-critical results may not depend on local clocks, CPU
budgets, queue occupancy, caches, scheduling or mempool arrival order. No
hybrid or EVM modules are implemented. `internal/tx` is standalone Phase 03B;
`internal/b0state` and `internal/b0store` are standalone Phase 03C packages.
None is imported by `internal/app`.

The node is localhost-only by default but has signing keys and is not hardened
for hostile networks. Genesis and Pluto metadata are checked on startup;
deleting or modifying one side's database must not trigger an implicit reset.
No arbitrary crash campaign, state sync, migration or multi-validator test has
been done. `example.com/pluto` remains a provisional local module path.
Phase 02R observed that `Run` does not compare the current CometBFT
`config.toml` listener/proxy addresses with the Pluto metadata generated at
`init`; operators must protect and review that file before each run. The
existing checks do not prove cross-store crash recovery or power-loss safety.

## Historical Phase 01 proposal (superseded by implementation and D-019 where applicable)

# Pluto target architecture (Phase 01)

Historical Phase 01 status: **proposed application design**. At that time, the only implemented component was the
`plutod` metadata CLI. Framework behavior below is taken from the CometBFT
v1.0.0 [ABCI++ methods][methods] and [application requirements][requirements].
Those documents define the target contract; no Pluto ABCI compatibility test has
yet been run.

## Components and ownership

```mermaid
flowchart TB
  W[Research client and workload generator] --> R[RPC boundary]
  R --> C[CometBFT node: RPC, P2P, mempool, consensus]
  C -->|ABCI++ calls| A[Pluto application adapter]
  A --> T[Canonical transaction parsing and validation]
  T --> H[Hybrid authentication and account binding]
  T --> S[State-dependent checks]
  S --> E[Sequential EVM execution]
  E --> P[Persistent application state and AppHash]
  A --> I[Research instrumentation]
  C --> I
```

CometBFT owns validator consensus, block coordination, P2P dissemination,
its configured mempool and its RPC. Pluto owns transaction representation,
authentication policy, account/key registry, state rules, EVM semantics,
application persistence, deterministic results and AppHash. Hybrid signature
checks belong in Pluto, never inside CometBFT's consensus algorithm.
Research code owns workload generation, instrumentation, correctness oracles,
benchmarks and analysis. It may observe CometBFT, but must not make
consensus-critical application decisions depend on measurements.

The client will submit application transaction bytes through a chosen RPC
route. Standard Ethereum raw transactions do not inherently carry Pluto's
ML-DSA proof. A future Pluto RPC boundary may therefore be needed; its exact
API and transaction format are not specified in Phase 01. CometBFT's P2P and
mempool carry transaction bytes, while Pluto interprets them. An RPC response
is not proof of eventual inclusion or execution.

## Intended transaction lifecycle

```mermaid
sequenceDiagram
  participant Client
  participant C as CometBFT
  participant A as Pluto application
  Client->>C: Submit transaction bytes by RPC
  C->>A: CheckTx (new admission)
  A-->>C: Admit or reject against local committed state
  Note over C,A: Mempool may later request CheckTx recheck after Commit
  C->>A: PrepareProposal (proposer; preliminary tx list)
  A-->>C: Proposed tx list within size rules
  C->>A: ProcessProposal (each validator, including proposer)
  A-->>C: Deterministic accept or reject
  C->>A: FinalizeBlock (decided block)
  A-->>C: Deterministic tx results and AppHash
  C->>A: Commit
  A-->>C: Persisted state acknowledged
```

This is a conceptual flow, not a claim that every node handles each transaction
in this order. Per [v1.0.0 methods][methods], `CheckTx` is an admission filter
and does not apply state changes. Recheck can revisit outstanding mempool
transactions after a commit. `PrepareProposal` is proposer-local and may
modify the preliminary list, subject to size and application coherence rules.
It need not be deterministic, but a correct proposer must prepare a block
other correct applications accept. `ProcessProposal` is consensus-critical:
its accept/reject decision must be deterministic for the relevant state and
proposal. `FinalizeBlock` processes the decided block and supplies execution
results plus AppHash; `Commit` persists the resulting state. The precise state
transition and crash-recovery order will be specified and tested in Phase 02
against [v1.0.0 requirements][requirements].

Never assume a proposed transaction passed `CheckTx` on every validating node.
The proposer can add transactions in `PrepareProposal`, and malicious proposers
can submit arbitrary proposal contents. Also, state may differ from the
admission snapshot when a later proposal is checked. Pluto must independently
apply consensus-critical validity rules to the complete proposal, including
ordering and duplicate handling. Its `FinalizeBlock` behavior must remain
coherent with `ProcessProposal`; a policy for invalid transactions in decided
blocks needs a formal specification before implementation. The v1.0.0
requirements explicitly require coherence between correct `PrepareProposal`
and `ProcessProposal` implementations.

## Data, state and deterministic behavior

The proposed data path separates six concerns:

1. **Canonical transaction bytes:** one bounded, versioned representation and
   unambiguous signed intent; malformed encodings are rejected.
2. **Cryptographic authentication:** both signatures attest the same intent,
   with domain separation and registered keys.
3. **State-dependent validity:** account policy, key version, nonce, balance,
   fees and block constraints against the correct state snapshot.
4. **Sequential EVM semantics:** one ordered execution baseline, including gas,
   receipts, errors and logs, before experimental performance changes.
5. **Persistent application state:** durable account/EVM data and the metadata
   needed for safe recovery.
6. **Committed AppHash:** a deterministic commitment to the application's
   defined state at a height, returned through ABCI++ and reproduced on replay.

The AppHash definition is **unresolved**. It must commit to all
consensus-relevant application state and be recoverable after crashes. A
Pebble database is a storage engine, not itself a Merkle commitment. Ethereum's
trie state root, if used inside the EVM state model, is not automatically the
CometBFT AppHash. Phase 02 must define a commitment scheme and persistence
protocol before calling the application safe. Never hash process-local
iteration order or nondurable state.

For identical committed state and proposal input, consensus-critical decisions
must not depend on local wall-clock time, queue occupancy, CPU budgets,
nonreplicated caches, nondeterministic scheduling or local mempool arrival
order. Local resource signals may inform future admission policy in `CheckTx`
where the ABCI contract permits it; they cannot change `ProcessProposal`,
`FinalizeBlock` or AppHash validity. Proposal fields supplied by CometBFT may
be inputs only according to the pinned ABCI specification.

## Future hybrid boundary

The future `internal/hybrid` module will parse and canonically encode the
application envelope, construct a domain-separated intent, bind account ID
and key version to keys in committed state, and verify secp256k1 and ML-DSA-44
with established libraries. Account policy is applied in state validation.
`HYBRID_REQUIRED` must fail closed if either proof is absent or invalid; no
ECDSA-only fallback is allowed in that profile. Exact bytes, key lifecycle,
FIPS 204 signing mode/context and rejection codes belong in a later protocol
specification and correctness oracle, before implementation. None exists now.

## Intended module dependencies

```mermaid
flowchart LR
  CMD[cmd/plutod] --> V[internal/version]
  CMD -. future .-> APP[internal/app]
  APP -.-> ABCI[internal/abci]
  ABCI -. adapter .-> COMET[CometBFT]
  APP -.-> HYB[internal/hybrid]
  APP -.-> EVM[internal/evm]
  APP -.-> STORE[internal/store]
  EVM -. library .-> GETH[go-ethereum]
  STORE -. engine .-> PEBBLE[Pebble]
  RPC[internal/rpc] -.-> APP
  TEST[test/hybrid and bench/hybrid] -.-> APP
```

Solid arrows represent Phase 01 imports. Dashed arrows are proposed boundaries,
not implemented dependencies. `internal/app` will coordinate lifecycle and
policy; `internal/abci` will translate pinned CometBFT types; `internal/evm`
will wrap selected geth execution APIs; `internal/store` will own durable state;
`internal/rpc` will expose application-specific submission; `internal/hybrid`
will own encoding and verification. `test/hybrid` and `bench/hybrid` will hold
oracles and measurements. `deploy/research`, `results` and `paper` are future
experiment deployment, output and publication areas. These directories are not
created in Phase 01.

## Integration uncertainties before implementation

- Verify exact CometBFT v1.0.0 ABCI Go types and lifecycle with a minimal real
  application in Phase 02; no API compile test exists yet.
- Define rejected proposal and invalid decided-block policies without harming
  liveness or validator state consistency.
- Specify a durable AppHash that covers the chosen EVM and account state.
- Reconcile geth's Pebble v1 dependency with any direct storage use; avoid a
  separate v2 path without an explicit migration decision.
- Define stable, portable transaction bytes, key-binding rules, crypto library
  mode and test vectors before hybrid implementation.

[methods]: https://github.com/cometbft/cometbft/blob/v1.0.0/spec/abci/abci%2B%2B_methods.md
[requirements]: https://github.com/cometbft/cometbft/blob/v1.0.0/spec/abci/abci%2B%2B_app_requirements.md
