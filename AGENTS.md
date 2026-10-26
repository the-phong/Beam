# Pluto agent execution contract

## Current authority — Phase 04 B1 hybrid authentication (2026-10-26)

The Owner merged accepted Phase 03E PR #8 into `test-mainet` at
`007e827f740c9c741f47bf1f7a51eec98a79cc5a` and authorized one complete
Phase 04 B1 secp256k1 + ML-DSA-44 authentication workflow on
`phase/04-b1-hybrid-authentication`. D-028 records an EVM NO-GO for this
experiment and consolidation of the historical hybrid roadmap activities.
Preserve Phase 02 and D-020–D-027 B0 behavior. Commit and push only the
Phase 04 branch, open a Draft PR against `test-mainet`, and stop for
Controller review. Never publish keys or runtime homes; do not merge,
change protected branches, start another phase, or claim DoS/Byzantine/
physical-power-loss results. The older authority sections below are
historical.

## Current authority — Phase 03E (2026-10-25)

The Owner merged accepted Phase 03D into `test-mainet` at
`c3756843c5e804e3a985d817ab5c83d2dd6798a0` and authorized the complete
Phase 03E four-validator BFT integration on `phase/03e-four-validator-bft`.
Implement and verify the complete scoped workflow, then commit and push only
that branch and open a Draft PR against `test-mainet`. The Controller reviews
technical evidence and the Owner controls acceptance and merge. Preserve
D-020–D-026 and Phase 02 legacy behavior. Do not publish validator keys or
runtime homes; do not merge, change protected branches, start Phase 04, or
claim unsupported PQC, DoS, Byzantine or power-loss results. The older
Phase 03D and earlier authority paragraphs below are historical.

## Current authority — Phase 03D Git closeout (2026-10-25)

The Controller technically accepted and the Owner formally accepted the
Phase 03D classical B0 ABCI++ implementation at staged tree
`7f7002496f38cea99cc3e0b7137e452523cc0b94`. Under D-018, the Owner
authorized a closeout commit on `phase/03d-b0-abci-integration`, a push of
that branch only, and a Draft PR targeting `test-mainet`. The Owner controls
merge. Preserve D-020–D-026 and accepted source behavior. Keep D-026 raw
evidence and unrelated untracked artifacts local unless publication-safe and
explicitly selected. Do not force-push, change `master` or `test-mainet`,
merge, or begin Phase 03E. Stop after verifying the Draft PR.

## Historical authority — Remaining Phase 03D (superseded 2026-10-25)

The Owner accepted and closed the D-025 checkpoint after Controller review of
R-025-01 and R-025-02. Its staged source tree
`7295b641254c91c53b3b80a0c2ec5f6c286e8dd1` is the implementation
baseline for remaining Phase 03D work on `phase/03d-b0-abci-integration`.
The Owner authorized B0 ABCI++ transaction integration, R2 integrity and cost
work, V01–V25 validation, real single-validator execution, M1–M12 recovery
testing and diagnostic benchmarks. D-020–D-025 remain accepted and frozen.
T7 remains PARTIAL; untested interruption boundaries belong to Phase 03D
recovery gates. Keep all work local unless the Owner separately authorizes a
push or PR. Do not invent a Git identity, merge, start Phase 03E, or claim
completion while a required gate lacks evidence. Stop for Controller review.

## Historical D-025 checkpoint authority (superseded by Owner closure)

The Owner merged Phase 03C PR #6 into `test-mainet` at
`b56c81ccce7ea4a0e7f52053c7459c4601c15b38`, authorized Phase 03D, and
accepted D-025. Work on `phase/03d-b0-abci-integration` was then limited
to canonical Genesis V2 JSON/Base64 transport and explicit B0 initialization,
tests, documentation and evidence. D-025 amends only the `InitChain`
transport boundary; D-020–D-024 binary, transaction, state and persistence
rules remain accepted. Stop after this checkpoint for Controller review
before any transaction-processing integration. Do not merge, push to
`test-mainet`, begin Phase 03E, or claim live B0 transfers are operational.

## Historical Phase 03C Owner acceptance closeout (superseded by Phase 03D)

The Owner accepted Phase 03C, Controller R1, the risk addendum and D-024 on
2026-10-24. D-024's single-record Pebble layout is accepted only as an
experimental, correctness-oriented B0 baseline. Phase 03C is complete and
Controller/Owner accepted, but Draft PR #6 is awaiting Owner-controlled
merge into `test-mainet`. The coding agent may update only acceptance
documentation and new closeout evidence on the existing
`phase/03c-b0-state-persistence` branch, commit and push that branch, verify
CI and stop. It must not merge, modify runtime or accepted protocol, activate
B0 transactions or begin Phase 03D. The R2 performance finding, cross-store
matrix M1–M12 and future Phase 03D gates G1–G8 remain open. The Phase 02
ABCI node still rejects every transaction.

## Historical Phase 03C implementation authority (superseded by Owner acceptance)

The Owner explicitly authorized Phase 03C on
`phase/03c-b0-state-persistence` from verified `test-mainet` SHA
`61bc65fdebf358fb0330cb9ebd4a332ca4a06aca`. The permitted work is an
isolated B0 V2 account-state, genesis, transition, commitment and persistence
subsystem, with tests and evidence. D-020–D-023 and the B0 V1 normative
specifications are accepted and frozen. The Phase 02 ABCI node remains
transaction-free: do not modify its live application, node, V1 store, genesis
or AppHash behavior. Phase 03D and later work are not authorized. Commit and
push only the Phase 03C branch, open a Draft PR against `test-mainet`, report
all gates, and stop for Controller and Owner review. The proposed V2 physical
storage layout requires separate Owner acceptance; this phase cannot accept
it on the Owner's behalf.
Controller R1 rework on existing Draft PR #6 is authorized within Phase 03C:
uncertain commit outcomes must make both committed and pending reads fail
closed until a validated close-and-reopen recovery. No later phase is opened.

## Historical Phase 03B post-merge governance closeout (superseded by Phase 03C authority)

The Owner accepted D-023 on 2026-10-23. Phase 03B PR #4 was merged into
`test-mainet` at `8be07c87abed73763d40252e99bbf0573da487b4`; Phase 03B
is complete and Controller-accepted. The coding agent is authorized only to
record this acceptance in governance documentation on the dedicated
`governance/03b-acceptance-closeout` branch, run the required verification,
commit, push, and open a Draft PR targeting `test-mainet`. The Owner controls
the governance merge. Phase 03C and later work are not authorized. Do not
change runtime, transaction cryptography, dependencies, tests, accepted
specifications, or historical evidence during this closeout.

## Historical Phase 03B implementation authority (superseded by Owner closeout decision)

The Owner accepted D-020–D-022, merged Phase 03A PR #3 into `test-mainet`
at `54bf720ac4625c17eaa6101a8f398ef7c0663432`, and explicitly authorized
Phase 03B on `phase/03b-classical-tx-crypto`. This phase may implement and
test only stateless B0 V1 canonical transaction encoding, secp256k1
authentication, address/ID derivation and reproducible vectors. The Phase 02
ABCI runtime and persisted state remain transaction-free. Phase 03C and later
work, runtime transaction integration, accounts, EVM and hybrid mechanisms
remain unauthorized. D-020–D-022 are frozen; an apparent contradiction
requires Controller review rather than a silent protocol change.

## Historical Phase 03A authority (superseded by Owner Phase 03B decision)

The project owner explicitly authorized **Phase 03A — Classical Transaction
Protocol Specification** after the Phase 02R merge and has now accepted its
B0 V1 protocol and D-020–D-022 following Controller rework. The coding agent
is authorized for Phase 03A documentation closeout only on the existing
`phase/03a-tx-protocol` branch from verified `test-mainet`. It may commit and
push closeout documentation to the existing Draft PR targeting `test-mainet`.
Owner acceptance freezes the specified rules for later separately authorized
work; it does not mean those rules are implemented or tested in the node.
Runtime changes, transaction/crypto implementation, EVM, hybrid mechanisms,
consensus/mempool changes and Phase 03B or later work are **not authorized**.
The implemented Phase 02 architecture remains the transaction-free baseline.

ChatGPT acts as technical controller; the Coding Agent performs engineering
work. The project owner approves phase transitions and performs or explicitly
approves merges. Agent-controller technical communication, source documents
and reports are in English. The controller provides explanations and decision
requests to the owner in Vietnamese.

Every authorized phase uses its own branch from the verified `test-mainet`
baseline. Normal commits and pushes are permitted within that phase's scope;
open a Draft PR against `test-mainet`. Never push directly to `test-mainet` or
`master`, merge automatically, force-push, or start another phase without
separate approval. Preserve unrelated local work. End every execution with a
full report including source/result commit SHAs, branch, PR URL or blocker,
verification and limitations, then STOP. A passing report never grants the
next phase.

The owner approved Master Roadmap v2: finish minimal B0-Core through Phase
03E before the Phase 04 decision on whether sequential EVM integration is
necessary. EVM is optional pending that decision; if approved, execution must
begin sequentially. Phase 03C is merged; Phase 03D is authorized subject to
the current D-025 checkpoint gate, while Phase 03E–14 remain future work until
separately authorized. Do not design admission optimization before Phase 11
profiling.

## Historical Phase 02R authority (superseded by owner Phase 03A decision)

Phase 02R was authorized for independent audit, verification, governance
reconciliation and a Draft PR on `phase/02r-audit-governance`. Its report did
not itself authorize Phase 03A. At that time, Phase 03A implementation was
unapproved. The subsequent explicit owner decision above supersedes that
status for specification work only; the Phase 02R report and evidence remain
preserved.

## Historical Phase 02 review handoff (superseded by Phase 02R authority)

Phase 02 — CometBFT Foundation is authorized and implemented for independent
review. The repository now contains a real single-validator, empty-block
CometBFT v0.38.26 chain connected to Pluto ABCI++ and Pebble v1.1.5. Phase 03
is **not authorized**. An agent may work only within an explicitly authorized
phase; the Phase 02 report is not approval for Phase 03. No transaction
execution, EVM, hybrid authentication or admission optimization exists.

For Phase 02, the user's local-only decision superseded its original Git
checkpoint: no commits, branches, remotes, Git identity or pushes were
required or created during that phase. D-018 supersedes this policy for newly
authorized phases while preserving that historical fact.
Phase 01 baseline checksums are in `research/evidence/phase_02/` and the
Phase 01 report/evidence must remain preserved. The then-current instruction
recorded further changes in research governance documents and evidence without
claiming a Git revision; D-018 supersedes that instruction for new phases.

For Phase 02 verification run `make verify`, `make test-integration`, and the
relevant race test. The application is transaction-free and all unsupported
transactions must fail closed. `FinalizeBlock` is pending-only; `Commit`
performs the synced persistent write. The Phase 02 AppHash is a fixed digest
of version, chain ID and height, not an Ethereum state root. Generated
validator/node keys must never be included in reports or commits. Do not
broaden node exposure beyond localhost without a new approved decision.

Pluto is a greenfield research project studying resource cost and processing
correctness for hybrid secp256k1–ML-DSA-44 transactions under deliberately
invalid workloads in a custom CometBFT-based EVM application. The repository
currently contains a Phase 02 transaction-free single-validator chain, not an
EVM or hybrid blockchain.

## Historical Phase 02 authority (superseded)

The then-active authorized phase was **02 — CometBFT Foundation**, for review
and authorized Phase 02 rework only. An agent may work only within the
explicitly authorized phase. Phase 03 remains unapproved. Necessary work outside
the active phase must be documented and reported, never silently implemented.
Do not infer approval from a PASS verdict or from roadmap text.

## Architecture and security rules

- CometBFT owns consensus, P2P, block coordination and its mempool. Pluto owns
  application transactions, authentication, account policy, state, EVM
  execution and AppHash. Do not modify CometBFT consensus algorithms.
- The historical target was a custom Go EVM application on CometBFT. D-019
  makes EVM conditional at Phase 04. Do not add Cosmos SDK without an
  explicit recorded decision.
- If EVM is approved, preserve sequential execution as its initial baseline.
  Do not design an admission optimization before Phase 11 profiling justifies it.
- Consensus-critical results must be deterministic for the same relevant
  committed state and proposal. Never use local clocks, queue size, CPU
  budgets, nonreplicated caches, scheduling or mempool arrival order as
  consensus validity inputs.
- Under future `HYBRID_REQUIRED`, both secp256k1 and ML-DSA-44 must verify the
  same canonical intent and bind to registered keys. Fail closed; no fallback.
  Do not invent cryptographic primitives.
- `CheckTx` is an admission filter; proposal validation must independently
  handle transactions that did not pass a local `CheckTx`.
- Do not equate AppHash with an Ethereum state root without a documented
  commitment design and tests.

## Implementation conventions

- Use idiomatic Go, standard library where practical, explicit errors and
  small packages with a current purpose. Avoid speculative interfaces and
  empty future modules.
- Pin exact third-party module versions when first imported. Record source,
  minimum Go version, reason, risks and actual build/test evidence in
  `research/DECISIONS.md` and the manifest. Never silently upgrade an
  accepted version. `go.mod` is the executable dependency source of truth.
- Add meaningful tests for behavior implemented now, including invalid
  inputs and deterministic replay when those become relevant. Run `make verify`
  before declaring a phase done; report every failure and skip.
- Do not claim a blockchain, EVM, hybrid verifier, benchmark result or
  multi-node experiment exists until its implementation and tests are present.

## Documentation and evidence

- Keep `research/ARCHITECTURE.md`, `research/MASTER_PLAN.md`,
  `research/DECISIONS.md`, `research/PHASE_STATUS.md`, and
  `research/config/manifest.yaml` aligned with actual code and approvals.
- Put exact commands, exit codes, relevant output and limitations in a phase
  report and `research/evidence/` as appropriate. Record revision, toolchain,
  workload seeds, hardware and configuration for later experiments.
- A phase report is an engineering claim for independent review, not approval.
  Historical Phase 03A required a STOP and separate Controller/Owner decision
  before Phase 03B. Current Phase 03C closeout also ends with STOP; Phase 03D
  requires separate Owner authorization.
