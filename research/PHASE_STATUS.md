# Phase status — 2026-11-13

## Current File 02 sections 0–2 foundation candidate

The Owner merged S07 PR #20 into `test-mainet` at
`e0bcd9e0f903a2e807a3d28adc5554dffa48e1f1` and explicitly authorized
sections 0–2 of the outside Checklist 02. Work is on
`research/file02-common-foundation` from that merge. The
[common contract](protocol/FILE02_COMMON_CONTRACT.md),
[artifact workflow](protocol/FILE02_ARTIFACTS.md), factor registry and QC
tooling are an implementation candidate. A new bounded S07 four-profile QA
smoke passed, and its private raw is ignored. COMMON-01–09 bounded QA is
evidenced; T06 calibration and precision pilots passed their registered
gates. A separate bounded knee search tested 20/23/25 tx/s without
confirming a knee. Section 2 factor, 63-metric and artifact contracts are
ready for review; future per-case output remains `PENDING_CASE_EXECUTION`.
Numerical knee levels, per-case plans, multi-host/P2P origin and the full
COMMON gate remain open. The [section 2 report](evidence/file02/SECTION2_READINESS_REPORT.md)
records the RQ1 wording issue. No confirmatory case or RQ1–RQ3 result is claimed.
The common contract now records the negative bounded knee finding without
requiring a further search. A [separate legitimate-service question](evidence/file02/SECTION2_SERVICE_DEGRADATION_PROPOSAL.md)
has a decision-ready proposal; the accepted RQ1 hypothesis is unchanged.
The seven per-case outputs in Checklist 02 section 2.3 remain unchecked.
Controller/Owner review remains separate. The sections below retain their
historical wording.

## Current S07 implementation candidate

The Owner merged governance [PR #19](https://github.com/the-phong/Beam/pull/19)
into `test-mainet` at `49facc28ca29102e3e7295546f29b74fe3503a00`, then
authorized S07 reproducibility and QA. Branch `research/s07-reproducibility`
starts from that merge commit in the original Pluto folder. Beam is only the
repository name; the application and Go module remain Pluto. The
[S07 run instructions](../REPRODUCE.md) and
[readiness report](evidence/s07/RUN_READINESS_REPORT.md) document a bounded
C/CP/P/H smoke, immutable private raw artifacts, analyzer and quick CI.
All five S07 checklist tasks have local evidence; hosted quick CI and secret
scan passed on [Draft PR #20](https://github.com/the-phong/Beam/pull/20).
Both full regression CI runs passed on source/evidence commit `24edcb4`.
Gate S07 is technically OK within the bounded smoke scope; final PR-head CI
and independent Controller/Owner acceptance remain separate. File 02
confirmatory experiments, ML Detection and RQ1–RQ3 result
claims remain closed. Historical status follows.

## Historical status after structural refactor merge

The Owner merged [PR #18](https://github.com/the-phong/Beam/pull/18) into
`test-mainet` at `e8f20726f5cea94b47c4dcddb756e39d21207448`.
[Post-merge CI](https://github.com/the-phong/Beam/actions/runs/43146676567)
completed successfully. The Controller conditionally accepted the structural
refactor; [review closeout](evidence/refactor/CONTROLLER_REVIEW_CLOSEOUT.md)
records evidence and limits. S07 remains unstarted and unauthorized; File 02
experiments are unopened. Current branch roles and PR #13's conflict are in
[branch policy](../docs/BRANCH_POLICY.md). The paragraphs below are dated
historical checkpoints, not current authorization.

## Historical `test-mainet` baseline and new research program

Phase 05 PR #10 is merged into `test-mainet` at
`0bdd869f2c6b3ed59b0ae825954d8667b3c7481e`. The Phase 05 B0/B1
corpus, four-validator localhost runner, analysis and bounded NO-GO
recommendation are historical evidence. Their 78 usable corrected conditions
and earlier exploratory matrix do not answer the new four-profile RQ1–RQ3
contract. The Owner requested the new checklist program on 2026-11-07;
S06 was merged in PR #17 into `test-mainet` at
`ed6bc8f474c13ed709d5ab5f57b3856edcbf602a`. The source-only
architecture refactor now proceeds on `refactor/domain-oriented-architecture`
from that merge commit in the original `pluto` folder.
The [S01 baseline audit](evidence/s01/BASELINE_AUDIT.md) fixes
the source tree, records the reruns, and marks the bounded local technical
Gate S01 OK. The Owner confirmed Gate S02 after PR #12 merged into
`test-mainet` at `2a3dcbceb52b996425d911dde87b691317f56efd`. S03 has
a bounded local technical Gate OK for the four-profile source and direct
ABCI++ tests, with [equivalence documentation](protocol/PROFILE_EQUIVALENCE.md)
and [audit evidence](evidence/s03/PROFILE_AUDIT.md). PR #14
was merged by the Owner on 2026-11-09. The Owner then authorized S04.
The S04 candidate adds a public four-profile correctness corpus, a stage
recorder used only for oracle checks, a profile/genesis-bound Pebble snapshot,
and a real four-validator localhost test harness. The
[correctness report](evidence/s04/CORRECTNESS_REPORT.md)
records the bounded result and verification. The Owner merged
[PR #15](https://github.com/the-phong/DoS-in-Blockchain-PQC/pull/15) and
authorized S05 telemetry. The [S05 metric dictionary](protocol/METRICS_DICTIONARY.md)
records directly observed and unavailable fields. S05.1–S05.5 have bounded
local technical evidence, including an S05-only source-checked Comet build
for P2P ingress and CheckTx socket queue. The Owner merged S05 PR #16 and
authorized S06. The [S06 testbed contract](protocol/TESTBED_CONFIG.md) and
[pilot report](evidence/s06/TESTBED_REPORT.md) cover a bounded, randomized
seven-job four-validator loopback pilot. The merged pilot is bounded; S07 is
closed. The refactor is a separate Draft PR candidate, not an accepted
research result.
No new RQ1–RQ3 result is accepted. The paragraphs below record historical
authorization and candidate status at their original dates.

## Historical Phase 05 experimental implementation candidate

The Owner technically accepted and merged Phase 04 PR #9 at
`b861479982b0114be2ad60f1e7727362075a7f73`. Accepted B1 hybrid
authentication remains within its Phase 04 limitations. The Owner authorized
one consolidated Phase 05 invalid-input corpus, B0/B1 correctness and
resource-cost evaluation on `phase/05-b1-experimental-evaluation`.
The executable experiment is an implementation candidate. The Controller
requested a matched-legitimate-rate correction on Draft PR #10 at reviewed
commit `3758fb7fb7df973d1003a5a49ff9071351a19dcf`. The prior 122-run
matrix is retained as exploratory: its 12-request runs had insufficient
per-run p99 and varying legitimate workload share. The supplementary
78-condition four-validator matrix completed with zero command failures;
one wall-clock-disrupted B0 secondary baseline was retained and replaced by
a fresh monotonic-timed run. The corrected analysis has 78 usable conditions,
including 300 confirmed valid transfers in every valid-bearing run.
The 22-command local regression ledger and both hosted checks on correction
implementation commit `547190a9f6b945cf327f8b6855704b887ad78bda`
passed. A final documentation/evidence-only closeout commit requires its own
hosted rerun before Controller review. The earlier hosted checks passed on
implementation commit `ea5b2ce802dbb315b7b4c6718d83f698a11a112b`;
those checks do not cover this correction.
Controller review and Owner acceptance remain separate gates.
No DoS mechanism, admission policy, EVM or subsequent phase is authorized. The
current research recommendation is NO-GO for a new mechanism on the bounded
data, while broader DoS susceptibility remains INCONCLUSIVE. D-028's
EVM NO-GO continues. The Phase 04 paragraphs below are historical.

## Historical Phase 04 B1 candidate (Owner authorized 2026-10-26)

The Owner merged accepted Phase 03E PR #8 at
`007e827f740c9c741f47bf1f7a51eec98a79cc5a` and authorized one complete
Phase 04 B1 hybrid secp256k1 + ML-DSA-44 implementation. The working branch
is `phase/04-b1-hybrid-authentication`. D-028 records the consolidated
hybrid workflow and current EVM NO-GO; D-029 records an isolated,
genesis-bound B1 account profile. Source, four-validator tests, fault
evidence, and the Phase 04 report are implementation candidates until
Independent Controller review and Owner acceptance. No DoS resistance,
Byzantine fault tolerance, multi-host result or physical durability is
claimed. The historical status paragraphs below retain their original
context; their earlier authorization limits are superseded for this phase.

## Historical Phase 03E implementation candidate

The Owner approved one complete Phase 03E four-validator BFT integration
workflow after merging the accepted Phase 03D branch into `test-mainet` at
`c3756843c5e804e3a985d817ab5c83d2dd6798a0`. The implementation branch
is `phase/03e-four-validator-bft`. D-027 records the bounded shared-genesis
provisioning choice. Four real local CometBFT validators, transaction/state
agreement, quorum and crash recovery are under implementation and test. The
Phase 03E report and exact verification evidence govern claims. Independent
Controller technical acceptance and Owner formal closure remain pending.
Phase 04 and later phases are unauthorized.

The Phase 03D authority statements below are historical and superseded by
the Owner's Phase 03E execution contract; they remain for audit traceability.

## Phase 03D formal acceptance and Git closeout

The Controller technically accepted the classical B0 ABCI++ Phase 03D
implementation, including D-026 and the R-D026-01/R-D026-02 corrections, at
accepted staged tree `7f7002496f38cea99cc3e0b7137e452523cc0b94`.
The Owner formally accepted Phase 03D on 2026-10-25 and authorized a local
closeout commit, push of `phase/03d-b0-abci-integration`, and a Draft PR
targeting `test-mainet` under D-018. The Owner retains merge control.
Phase 03E remains unauthorized.

Acceptance covers classical secp256k1 transactions in the single-validator
CometBFT B0 ABCI++ runtime, explicit D-026 H0 initialization, and the tested
software-fault recovery boundaries. Hosted CI is not fully verified; physical
power-loss durability and four-validator BFT were not tested. PQC/hybrid
transaction authentication is not implemented, and DoS resistance is not
established. Diagnostic benchmarks do not establish those later results.
The pre-acceptance review statuses below are retained as historical records.
Detailed D-026 raw evidence and the final source-review package remain local
for Controller inspection and are not part of this public Git closeout.

## Historical D-026 conditional source review

**D-026 conditional source pass rework:** Controller findings R-D026-01
(atomic fresh-home claim) and R-D026-02 (Comet initialization artifact
verification) are being addressed locally. The previous tested tree
`cdce2bb2ed79abe9f5768d77d5278495e9e5b61c` and final staged tree
`51830c4b63506a73018297a27a7333ac7643de99` remain preserved as review
baselines. The corrections require fresh evidence and Controller final review;
Phase 03D remains PARTIAL.

## Historical Phase 03D implementation candidate

**D-026 checkpoint authorization, 2026-10-25:** The Owner approved the
explicit B0 initialization lifecycle for implementation within Phase 03D.
The local candidate now has `init-b0` create and verify H0, `start` require a
complete existing snapshot, and B0 `InitChain` validate identical persisted
H0 without a write. D-024 record bytes and D-025 transport remain frozen.
Software-fault evidence is summarized in the
[Phase 03D report](PHASE_REPORTS/PHASE_03D_REPORT.md).
This is implementation evidence, not Controller acceptance. Hardware power
loss remains unverified; Phase 03D remains PARTIAL pending review. Phase 03E
remains unauthorized. See [D-026](DECISIONS.md) and its
[decision rationale](decisions/D-026_EXPLICIT_B0_INITIALIZATION_PROPOSAL.md).

The Owner closed the D-025 checkpoint after Controller review and authorized
remaining Phase 03D work on `phase/03d-b0-abci-integration`. The accepted
checkpoint staged tree is `7295b641254c91c53b3b80a0c2ec5f6c286e8dd1`.
The branch now contains an explicit B0 V2 runtime with transaction admission,
proposal validation, sequential finalization, durable V2 Commit and latest
committed-state account queries. Local direct ABCI and real pinned
single-validator tests exercise transfers, restart and controlled recovery
boundaries. The [recovery closure report](PHASE_REPORTS/PHASE_03D_RECOVERY_CLOSURE_REPORT.md)
supersedes the earlier Phase 03D partial matrix: M4, M6, M9, M10 and the
feasible M11 software windows now have direct evidence. Phase 03D is **NOT
OWNER-ACCEPTED OR MERGED** and its technical verdict remains **PARTIAL**:
The earlier M7 callback coverage and T7 policy limits are historical; M7
software callback evidence has since been strengthened and the D-026 policy
has now been Owner-approved for implementation. Physical power-loss and four-validator behavior were
not tested. Phase 03E and later phases are **NOT AUTHORIZED**. The
historical Phase 02 runtime remains transaction-free.

## Historical D-025 genesis transport checkpoint (closed by Owner)

Phase 03C PR #6 was Owner-merged into `test-mainet` at
`b56c81ccce7ea4a0e7f52053c7459c4601c15b38`; both post-merge checks
succeeded. The Owner authorized Phase 03D and accepted D-025 on 2026-10-24.
The then-current checkpoint permitted only strict JSON/Base64 transport for the
unchanged canonical Genesis V2 binary and explicit B0 initialization. The
separately selected B0 profile was genesis-only and continued to reject
transactions. Full Phase 03D transaction integration, R2 work, resource
measurements and M1–M12 crash testing wait for Controller checkpoint review.
Controller R-025 rework is staged for source review. T7 remains PARTIAL;
missing-data and cross-store recovery limitations are retained in the Phase
03D D-025 report addendum, not marked accepted or complete.
Phase 03E and later work are **NOT AUTHORIZED**. The Phase 02 profile remains
available and transaction-free.

## Historical Phase 03C Owner acceptance and pre-merge closeout

**Phase 03C — COMPLETED, CONTROLLER ACCEPTED, OWNER ACCEPTED, AWAITING
OWNER-CONTROLLED MERGE OF PR #6.** The Owner accepted the implementation,
R1 fail-closed read correction, risk addendum and D-024 on 2026-10-24.
D-024 is accepted only as a correctness-oriented experimental B0 persistence
baseline. The reviewed branch commit is
`3484c0f3a47dffa24ddcc0dbad2d95ca23ce31fc`; PR #6 remains open and
Draft against `test-mainet`. This acceptance is not a merge.

`internal/b0state` and `internal/b0store` remain standalone. The live Phase 02
ABCI application still rejects every transaction and uses V1 state and
AppHash. The [risk addendum](evidence/phase_03c/RISK_AND_ACCEPTANCE_ADDENDUM.md)
retains R2, cross-store matrix M1–M12 (**NOT RUN** for B0 V2), full-state O(K)
benchmark requirements and Phase 03D gates G1–G8 (**NOT SATISFIED**).
No Phase 03D or later implementation is authorized; the Owner controls PR #6's
merge. The prior review-candidate status below is preserved as history.

## Historical Phase 03C implementation candidate (superseded by Owner acceptance)

The Controller reviewed commit `6459a1ad7ee5a17994f484c04f47658a976d9ea6`
and requested R1 rework on existing Draft PR #6: committed and pending reads
must return `ErrUncertain` after an ambiguous write result. The correction at
`e0704b4533aab522900c7d98584435f4c24021f1` and the functional Phase 03C
implementation passed technical review. The Owner requested a
[documentation-only risk and acceptance addendum](evidence/phase_03c/RISK_AND_ACCEPTANCE_ADDENDUM.md)
before deciding acceptance. This is not Phase 03C Owner acceptance; D-024 is
**PROPOSED — AWAITING EXPLICIT OWNER ACCEPTANCE**, Draft PR #6 remains open,
and Phase 03D remains unauthorized.

The Owner authorized Phase 03C after the Phase 03B governance closeout merged
into `test-mainet` at `61bc65fdebf358fb0330cb9ebd4a332ca4a06aca`.
Work on `phase/03c-b0-state-persistence` adds standalone V2 account state,
genesis parsing, sequential simulation, AppHash and a separate Pebble store.
The Phase 02 ABCI runtime still rejects every transaction and uses its V1
record and AppHash. D-020–D-023 remain accepted and unchanged. D-024 proposes
the physical V2 storage layout for Controller and Owner review; it is not yet
accepted. Phase 03C completion is subject to evidence, CI and independent
review. Phase 03D and all later phases are **NOT AUTHORIZED**.

## Historical governance closeout after Phase 03B merge

Phase 03B is **completed, Controller-accepted, and merged** via PR #4 into
`test-mainet` at `8be07c87abed73763d40252e99bbf0573da487b4`. The Owner
accepted D-023 on 2026-10-23; D-020–D-022 remain accepted and unchanged.
Both post-merge GitHub verification runs passed. The only active authorization
is documentation-only acceptance closeout on
`governance/03b-acceptance-closeout`, with a Draft PR and Owner-controlled
merge. Phase 03C is **NOT AUTHORIZED**. The running node remains the Phase 02
transaction-free application; `internal/tx` is a standalone stateless library.
Account state, AppHash V2 and ABCI transaction handling remain unimplemented.

## Historical Phase 03B execution status (superseded by Controller acceptance and merge)

The Owner accepted D-020–D-022 and merged Phase 03A PR #3 into `test-mainet`
at `54bf720ac4625c17eaa6101a8f398ef7c0663432`. The Owner separately
authorized Phase 03B on `phase/03b-classical-tx-crypto`. This phase implements
and verifies a standalone stateless transaction library and public test
vectors. The Phase 02 node still rejects all transactions; account state,
AppHash V2 and ABCI integration are reserved for Phases 03C–03E. Phase 03C is
not authorized. Completion and acceptance remain subject to Controller/Owner
review of the Phase 03B report and Draft PR.

## Historical Phase 03A closeout status (superseded by merger and 03B authorization)

- **Active authorization:** Phase 03A documentation closeout only. The Owner
  authorized Phase 03A after Phase 02R was merged into `test-mainet` at
  `ab1d1746fdd70bc0bbf00f20a186663f3cacf2ab`, then accepted the B0 V1
  protocol and D-020–D-022 after Controller R1–R5 rework on 2026-10-23.
- **Engineering status:** Four B0 protocol documents and the Phase 03A report
  specify canonical transaction bytes, secp256k1 signed intent, account/state
  transitions, AppHash V2, ABCI validity and negative-vector plans. The Owner
  accepted the normative rules after the Controller's R1–R5 rework; they are
  **accepted as protocol, not implemented**. PR #3 remains a Draft awaiting
  owner-controlled merge. The runtime remains the Phase 02 empty-block
  application.
- **Verification:** `git diff --check`, `make verify` and
  `make test-integration` exited 0 on the Phase 03A branch; exact commands,
  output, toolchain and limits are in `research/evidence/phase_03a/` and
  `research/PHASE_REPORTS/PHASE_03A_REPORT.md`. The same required checks and
  a runtime/dependency/test source-diff check exited 0 during controller
  rework; see `controller_rework_verification.txt`. Documentation closeout
  checks also exited 0; see `closeout_verification.txt`. No cryptographic
  vector was generated or executed in this specification phase.
- **Historical baseline:** Phase 02/02R architecture, reports and evidence
  remain preserved. Phase 02 V1 AppHash and `PLUTO02!` storage are still the
  only implemented state format. Accepted V2 requires a fresh B0 chain; no
  automatic migration is specified or implemented.
- **Governance:** D-018 branch/PR/owner-merge control applies. D-019 requires
  B0-Core before the conditional Phase 04 EVM decision. D-020–D-022 are
  **Accepted by Owner** as protocol requirements for later authorized work.
- **Known limits:** Current tests exercise only the Phase 02 runtime. No B0
  transaction execution, account persistence, secp256k1 verification, V2
  commitment, four-validator agreement or security/performance result exists.
- **Next action:** Await the Owner-controlled merge of Draft PR #3 and a
  separate Phase 03B decision after documentation closeout.
  **Phase 03B is not authorized.** No merge by the agent.
