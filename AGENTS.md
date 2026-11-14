# Pluto agent execution contract

## Current authority — Checklist 02 section 5, A01–A14 (2026-11-14)

The Owner confirmed PR #26 is merged and explicitly authorized implementation
and completion of all section 5 attack-workload cases in checklist order.
The engineering disposition now records eleven bounded PASS, two PARTIAL and
one UNSUPPORTED case; Gate ATTACK awaits Controller/Owner review. This
engineering completion does not self-accept the Gate.
This supersedes the former prohibition on opening the A group, but does not
authorize Q/D cases, ML training, a DoS claim, higher S06 safety caps or a
change to accepted RQ1/RQ1b protocol semantics. Keep the fixed legitimate
stream and matched benign controls; lock each case's workload, rates, source,
topology, QC and confidence procedure before confirmatory sends. Execute
only controlled single-host loopback/private targets unless a case has
separately verified source/topology capability. Preserve historical raw,
accepted protocol, private keys and unrelated work. Tick only cases with
verified evidence. Where a required capability is absent or a measurement
cannot be obtained within accepted limits, record PARTIAL/UNSUPPORTED with
specific evidence and leave its box open. Submit changes for Owner review on
a new branch from the PR #26 merged `test-mainet` head; do not merge without
a separate Owner instruction. The Gate RQ1 acceptance below is historical.

## Historical authority — Gate RQ1 bounded Owner acceptance (2026-11-13)

The Owner accepted Checklist 02 R01–R16 with eight PASS, six PARTIAL and
two UNSUPPORTED dispositions after PR #25 merged the R result tree into
`test-mainet` at `d67fef2598f278015968353763357f6aed28250d`. The accepted
H-minus-CP application CheckTx effect is +81.47 microseconds/transaction
only for the tested four-validator single-host RPC-origin 20-tx/s stratum.
No knee was observed within the unchanged safe range; no DoS or distributed
performance conclusion follows. Keep PARTIAL/UNSUPPORTED boxes open and
preserve raw, protocol and safety caps. This acceptance does not authorize
A/Q/D cases, ML training, new R experiments or a phase transition. Record
the acceptance on a governance branch for review; do not push directly to
protected branches or merge the closeout PR without a new Owner instruction.
The R execution authority below is historical.

## Historical authority — File 02 R01–R16 (2026-11-13)

The Owner approved one complete Checklist 02 section 4 task, R01 through R16
in order, using accepted Gate T and the RQ1/RQ1b planning documents on Draft
PR #23. Lock the accepted RQ1 H-minus-CP hypothesis and case plans before
confirmatory runs. RQ1b remains separate and no A/Q/D group is opened.
Implement and verify feasible R cases on four-validator single-host,
RPC-origin topology within unchanged S06 caps. Record unsupported or
unobserved cases with evidence; never fabricate outputs or tick a failed
case. Preserve historical raw, accepted protocol semantics and private data;
measurement-only hooks must leave validation outcomes unchanged and pass
regression tests.
Use a branch from PR #23's planning head and a stacked Draft PR for Gate RQ1
review. Do not merge, train ML or claim DoS. Report only after R01–R16 have
each received an evidenced disposition. The Gate T planning authority below
is historical for this task.

## Historical authority — RQ1/RQ1b planning after Gate T (2026-11-13)

The Owner conditionally accepted Checklist 02 T01–T07 as technically complete
within the verified preflight scope. The permitted next work is preparation
of RQ1 and RQ1b case plans for the four-validator single-host topology, with
explicit scientific limits. Before any official run, each case must separately
lock workload, offered rates, repetitions, endpoints, confidence procedure,
safety, topology and QC. PR #21 merged into `test-mainet` at
`6de41446272520809fbdcb84d3b1155831dc24d5`; PR #22 then merged at
`a88e5d7eb5984f2dcddb5368248be2bff57e2aa4` after four passing checks.
Use `research/file02-rq1-rq1b-planning` from that verified merged head and a
Draft PR for planning-only review.
No R/A/Q/D case execution, ML training, multi-host/P2P-origin claim, DoS
claim or protected-branch direct push is authorized. The T preflight authority
below is historical after these ordered merges.

## Historical authority — File 02 T preflight, T01–T07 (2026-11-13)

The Owner accepted Gate COMMON for bounded pre-case readiness after all four
PR #21 CI checks passed and directed RQ1b to be recorded separately from
the accepted RQ1. D-038 records that decision. The Owner then authorized
Checklist 02 T01–T07 in order. Work on `research/file02-t-preflight` from
the accepted COMMON branch head while PR #21 remains unmerged; preserve the
dependency and use a Draft PR for review. Tick each T item only after its
own verified evidence. Earlier T06 pilots are calibration evidence, not
formal T06 completion. Keep the S06 safety caps, loopback/private targets,
raw evidence privacy and historical bytes. Do not merge PRs, change
protected branches, start R/A/Q/D cases, train ML or claim RQ/DoS results.
The File 02 common foundation authority below is historical.

## Historical authority — File 02 common foundation, sections 0–2 (2026-11-12)

The Owner explicitly requested autonomous setup, implementation and completion
of sections 0, 1 and 2 of the outside Checklist 02. Work in the original
Pluto folder on `research/file02-common-foundation` from merged `test-mainet`
commit `e0bcd9e0f903a2e807a3d28adc5554dffa48e1f1`. Implement the bounded
research contract, common-run controls, factor/metric registry, plan and QC
artifact foundation. Tick only verified items. The S07 smoke may be rerun as
QA; it is not confirmatory File 02 data. Preserve accepted S01–S07 behavior,
public corpus, historical evidence bytes, ignored private artifacts, unrelated
D026 files and local stash. Push only this branch and open a Draft PR against
`test-mainet`; stop for Controller/Owner review. Do not merge, alter protected
branches, start T/R/A/Q/D cases, build ML detection or claim RQ1–RQ3/DoS
results. The S07 authority below is historical.

## Current authority — S07 reproducibility and QA (2026-11-12)

The Owner merged governance PR #19 into `test-mainet` at
`49facc28ca29102e3e7295546f29b74fe3503a00` and explicitly authorized
completion of S07.1–S07.5 in checklist order. Work only in the original
`pluto` folder on `research/s07-reproducibility` from that merge commit.
Beam is the GitHub repository name only; the application and provisional
Go module remain Pluto. Build a traceable bounded four-profile smoke,
immutable private raw evidence, integrity checks, per-run analyzer and quick
CI. Tick only verified items. Preserve accepted S01–S06 behavior, public
corpus and historical evidence bytes, ignored artifacts, unrelated D026
files and the local stash. Push only this S07 branch and open a Draft PR
against `test-mainet`; stop for Controller/Owner review. Do not merge,
change protected branches, start File 02 confirmatory runs, build ML
detection or claim RQ1–RQ3/DoS results. The dated sections below are history.

## Historical status and authority — post-refactor governance (2026-11-12)

The Owner merged domain-oriented refactor PR #18 into `test-mainet` at
`e8f20726f5cea94b47c4dcddb756e39d21207448`. The post-merge `verify`
workflow succeeded. The Controller conditionally accepted the structural
refactor; this is not acceptance of S07 or RQ1–RQ3 results. This governance
closeout records that review, current paths and branch status on
`governance/refactor-acceptance-closeout` from the merge commit. Preserve
historical evidence, local ignored artifacts, unrelated D026 files and stash.
Submit a Draft PR to `test-mainet` for review. Do not merge, change the
default or protected branches, resolve PR #13 by merging, begin S07, run
File 02 experiments or claim new research results. The sections below record
authority at their original dates and are historical, not current grants.

## Historical authority — domain-oriented structural refactor (2026-11-11)

The Owner merged S06 PR #17 into `test-mainet` at
`ed6bc8f474c13ed709d5ab5f57b3856edcbf602a` and requested one complete
architecture-first structural refactor from that commit on
`refactor/domain-oriented-architecture`. Preserve accepted protocol,
corpus/evidence bytes, ignored artifacts, unrelated D026 files and the local
stash. Run the full regression suite, push only the refactor branch, open one
Draft PR targeting `test-mainet` and stop for Controller/Owner review. Do not
merge, change protected branches, begin S07 or claim loaded RQ1–RQ3 results.
The older S06 authority below is historical.

## Historical authority — S06 safe testbed and open-loop load (2026-11-11)

The Owner merged S05 PR #16 into `test-mainet` at
`ac91250ddff31d07b6dcb19f81878e149256e84f` and explicitly authorized
S06. Work only in the original `pluto` folder on
`research/s06-safe-testbed` from that merge commit. Complete S06.1–S06.5
in checklist order with pinned testbed configuration, a bounded open-loop
generator, three offered-load regimes, capability-correct workload corpus,
and safety abort evidence. Tick only verified items. A four-validator
single-host run is a pilot, not independent infrastructure. Keep all load
on controlled loopback/private endpoints, preserve accepted B0/B1/S03–S05,
the refactor stash, ignored artifacts and unrelated D026 files. Commit/push
only the S06 branch, open a Draft PR into `test-mainet`, and stop for Owner
review. Do not merge, touch protected branches, start S07 or claim loaded
RQ1–RQ3/DoS results.

## Historical authority — S05 telemetry (2026-11-10)

The Owner merged S04 Draft PR #15 into `test-mainet` at
`630f2d8c81398f84073796747108c30ab9dee3c8` and explicitly authorized
S05. Work only in the original `pluto` folder on
`research/s05-observability` from that merge commit. Complete S05.1–S05.5
with a metric dictionary, bounded collector and structured events, real
isolated four-validator checks, ON/OFF and overhead evidence. Tick only
verified items. Keep unavailable capture points labeled `NOT MEASURED` and
do not infer queue wait or P2P transaction ingress from RPC or ABCI data.
Preserve accepted B0/B1/S03/S04 behavior, public corpus bytes, local stash,
ignored private artifacts and unrelated D026 files. Commit and push only
the S05 branch, open a Draft PR against `test-mainet`, and stop for Owner
review. Do not start S06, claim DoS/resource study results, merge or touch
protected branches.

## Historical directory authority — one research tree (2026-11-10)

The Owner requested one research directory with no separate versioned tree.
Keep the S01–S04 checklist, protocol, corpus, evidence and ignored private
artifacts under `research/` in the original `pluto` folder. Historical reports
may retain their original command spelling; use the current paths for reruns.
Current C/CP/P/H code lives under `internal/profiles/`; its command folders
are `cmd/profile-corpus/` and `cmd/profile-vectors/`.
Historical B0/B1 experiment code is grouped by purpose under
`internal/benchmarks/`, `cmd/b0b1-corpus/` and `research/benchmarks/b0b1/`.
Preserve corpus, manifest and raw-log bytes, the local refactor stash and
unrelated D026 files; document any report link repairs.
This organization alone did not approve S05 or change S04's review gate.
The later Owner merge and S05 authorization above supersede that status.

## Historical authority — S04 four-validator correctness (2026-11-09)

The Owner merged S03 PR #14 into `test-mainet` at
`1c8084207adb5da787e34e230dd0dc6d9a3de05f` and explicitly authorized
S04. Work in the original `pluto` folder only, on
`research/s04-bft-correctness` from that merge commit. Complete S04.1–S04.4
with a public four-profile correctness corpus, direct ABCI oracle, real
isolated four-validator evidence, cache/dedup/recheck and recorder ON/OFF
checks. Tick only evidenced items and record the bounded gate. Preserve
accepted B0/B1/S03 behavior, the local refactor stash and ignored private
artifacts. Commit/push only the S04 branch, open a Draft PR against
`test-mainet` and stop for Owner review. Do not start S05, make resource/DoS
claims, merge or touch protected branches.

## Historical authority — S03 controlled profiles (2026-11-09)

The Owner confirmed S02 review/acceptance and merged PR #12 into
`test-mainet` at `2a3dcbceb52b996425d911dde87b691317f56efd`, then
authorized S03. Work only in the original `pluto` project folder on
`research/s03-controlled-profiles`, based on that merge commit. Do not create
a new sibling folder or worktree per checklist stage. Preserve the local
preexisting refactor state in stash commit `7ec52bd` and the copied historical
private artifacts; never publish them. Complete S03 with verified C/CP/P/H
source, equivalence report, public vectors and tests; update the outside and
repo checklist item by item. Push only this research branch and open a Draft
PR into `test-mainet`; the Owner controls merge. S04 and later stages remain
closed until S03's own gate and subsequent authorization.

## Historical authority — research-v2 experimental checklist (2026-11-07)

The Owner requested completion of the separate
`research/checklists/01_CHECKLIST_CHUAN_HOA_MO_RONG_REPO_THUC_NGHIEM.md`
program, with each item checked only after verifiable completion. Work from
the merged `test-mainet` baseline `0bdd869f2c6b3ed59b0ae825954d8667b3c7481e`
on `research/reproducible-testbed`, preserve unrelated local work, and seek
review through a PR targeting `test-mainet` (the existing branch matching the
Owner's “main-test” wording). The Owner controls merge. Keep historical
evidence separate from new RQ1–RQ3 runs; use isolated test networks only;
never publish private keys, runtime homes, or unsanitized raw logs. An item
marked complete requires a reproducible artifact and test evidence. Scientific
contract, experiment and final-result gates still require independent review;
the agent cannot self-approve them. No EVM or admission/DoS mechanism is
inferred from this checklist request. The older authority sections below are
historical.

## Historical authority — Phase 05 B1 experimental evaluation (2026-10-26)

The Owner technically accepted and merged Phase 04 PR #9 into `test-mainet`
at `b861479982b0114be2ad60f1e7727362075a7f73` and authorized one complete
Phase 05 adversarial-corpus, correctness-oracle, benchmark and research
decision workflow on `phase/05-b1-experimental-evaluation`. Preserve accepted
B0/B1/Phase 02 semantics. Tests and workloads target only isolated localhost
homes. Publish no private keys, runtime homes or unsanitized raw evidence.
Commit/push only the Phase 05 branch, open a Draft PR against `test-mainet`,
and stop for Controller review. No DoS mechanism, admission optimization,
EVM, protected-branch change, merge or subsequent phase is authorized. The
older authority sections below are historical.

## Historical authority — Phase 04 B1 hybrid authentication (2026-10-26)

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

## Historical authority — Phase 03E (2026-10-25)

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

## Historical authority — Phase 03D Git closeout (2026-10-25)

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
