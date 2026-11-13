# Pluto research master plan

**Current File 02 status (2026-11-13):** [D-041](DECISIONS.md) records
Owner acceptance of the bounded R01–R16 disposition, integrated by
[PR #25](https://github.com/the-phong/Beam/pull/25). Eight cases are PASS,
six PARTIAL and two UNSUPPORTED. The [Gate record](evidence/file02/RQ1_GATE_BOUNDED_ACCEPTANCE_20261113.md)
accepts H-minus-CP +81.47 microseconds/transaction only in the tested
single-host RPC stratum; the RQ1 hypothesis is unchanged and RQ1b remains
separate. No defensible knee was observed within 25 tx/s. No DoS,
distributed-network behavior or sustainable throughput conclusion is
accepted. A/Q/D cases and ML training require separate authorization.
Later paragraphs retain their original historical status.

**Historical authorization (2026-11-12):** After the Owner merged S07 PR #20
into `test-mainet` at `e0bcd9e0f903a2e807a3d28adc5554dffa48e1f1`, the
Owner authorized only sections 0–2 of the outside Checklist 02. The
[File 02 common contract](protocol/FILE02_COMMON_CONTRACT.md) and
[artifact workflow](protocol/FILE02_ARTIFACTS.md) are a review candidate on
`research/file02-common-foundation`. Numeric load levels, independent-run
precision and confirmatory T/R/A/Q/D cases remain open. Later historical
authority paragraphs below are retained for audit.

**Historical status (2026-11-11):** Phase 05 PR #10 is merged into `test-mainet`
at `0bdd869f2c6b3ed59b0ae825954d8667b3c7481e`. The Owner requested a
new, separately evidenced RQ1–RQ3 program in the
[research checklist](checklists/01_CHECKLIST_CHUAN_HOA_MO_RONG_REPO_THUC_NGHIEM.md).
S01 and S02 are accepted; S03 four-profile source and direct-ABCI work was
merged in PR #14 at `1c8084207adb5da787e34e230dd0dc6d9a3de05f`.
The Owner merged S04 PR #15 at
`630f2d8c81398f84073796747108c30ab9dee3c8` and authorized S05. Its bounded implementation, directly measured
socket queue component and remaining quality limits are in the
[metric dictionary](protocol/METRICS_DICTIONARY.md).
S05 PR #16 was merged at `ac91250ddff31d07b6dcb19f81878e149256e84f`;
the Owner authorized S06 on `research/s06-safe-testbed`. Its controlled
single-host four-validator [pilot](evidence/s06/TESTBED_REPORT.md) has local
technical evidence. Multi-host measurements, detector comparison and S07
are not authorized. D-031–D-036 record provenance and decisions. Earlier phase schedules and
authority paragraphs below are historical and do not establish checklist
completion.

**Historical Phase 05 experimental authority (2026-10-26):** The Owner technically
accepted Phase 04 B1, merged PR #9 at
`b861479982b0114be2ad60f1e7727362075a7f73`, and authorized one complete
adversarial-corpus, correctness-oracle, benchmark and research-decision
workflow on `phase/05-b1-experimental-evaluation`. This scheduling amendment
consolidates the original roadmap's adversarial testing, benchmark
infrastructure and baseline measurement objectives (historical Phases 09–11)
into the active Phase 05. The historical rows below remain unchanged for
audit. EVM remains NO-GO under D-028; no DoS mechanism or admission
optimization is authorized. See D-030 and the preregistered
[`Phase 05 experiment`](specs/PHASE_05_EXPERIMENT.md). Controller review and
Owner acceptance remain separate from Coding Agent implementation.

**Owner Phase 04 authorization and roadmap amendment (2026-10-26):** The
accepted Phase 03E PR #8 is merged into `test-mainet` at
`007e827f740c9c741f47bf1f7a51eec98a79cc5a`. The Owner authorized one
complete B1 hybrid authentication implementation on
`phase/04-b1-hybrid-authentication`. D-028 consolidates the historical
Phase 05–08 hybrid design, cryptography, integration and ABCI safety work into
this Phase 04. EVM is NO-GO for the present authentication/correctness
experiment. The historical roadmap rows remain below as the original plan,
not current Phase 04 authority. The implementation and tests require
independent Controller review and Owner acceptance; no later phase is opened.

**Phase 03E authorization addendum (2026-10-25):** The Owner merged the
accepted Phase 03D closeout into `test-mainet` at
`c3756843c5e804e3a985d817ab5c83d2dd6798a0` and authorized one complete
Phase 03E four-validator BFT implementation on
`phase/03e-four-validator-bft`. The required local multi-validator tests and
Draft PR are one engineering workflow. Controller review and Owner acceptance
remain separate; Phase 04 and later work remain unauthorized. The older
current-authority paragraph below records the earlier Phase 03D closeout.

Project question: What are the resource costs and correctness properties of
hybrid secp256k1–ML-DSA-44 transactions in a custom CometBFT-based
application under deliberately invalid workloads, and does an EVM add value
to that investigation? No finding or publication
outcome is presumed. Every phase requires a separate human authorization;
finishing one phase does not authorize the next.

**Historical execution authority (2026-10-25):** Phase 03C is merged at
`b56c81ccce7ea4a0e7f52053c7459c4601c15b38`. The Controller technically
accepted Phase 03D, and the Owner formally accepted its classical B0 ABCI++
implementation at staged tree `7f7002496f38cea99cc3e0b7137e452523cc0b94`.
D-020–D-026 remain accepted. The Owner authorized only D-018 Git closeout:
commit and push `phase/03d-b0-abci-integration` and open a Draft PR against
`test-mainet`. The Owner controls merge. Hosted CI is not fully verified;
physical power-loss durability and four-validator BFT were not tested.
PQC/hybrid authentication is not implemented and DoS resistance is not
established. Phase 03E and later phases remain unauthorized. The conditional
Phase 04 EVM gate is unchanged.

The method is: protocol specification → correctness oracle → functional
implementation → adversarial testing → instrumentation → baseline measurement
→ root-cause analysis → conditional mechanism design → comparative evaluation
→ reproducible artifact. Each phase records a report, exact revision, executed
checks, limitations and a review decision.

For locally developed Phase 02, the user explicitly replaced the Git commit
checkpoint with a SHA-256 Phase 01 baseline manifest. `BASE_COMMIT` and
`RESULT_COMMIT` were therefore not applicable at that time. D-018 restores
branch-based Git governance for Phase 02R and later authorized phases. This
does not authorize Phase 03 or relax any blockchain/correctness gate.

## Approved Master Roadmap v2 sequence (planning, not phase authorization)

The owner approved **B0-Core first** (D-019). Phase 02R audits the existing
Phase 02 foundation and changes governance only. Phase 03 is split into five
separately gated steps: **03A** classical transaction protocol specification;
**03B** canonical encoding and secp256k1 verification; **03C** account state,
nonce, balance, persistence and state commitment; **03D** ABCI transaction
integration; **03E** single-node and four-validator B0-Core validation. B0-Core
must be complete before Phase 04. **Phase 04 is a decision gate**: evaluate
whether sequential EVM integration is necessary for the research and integrate
only after a separate approval. A no-go decision permits later phases to use
the approved B0-Core execution model. Phases 05–08 cover hybrid protocol,
cryptography, integration and safety; 09–11 cover adversarial corpus,
benchmark infrastructure and baseline measurements; 12 is a conditional DoS
detection or mitigation contribution; 13 expands multi-node experiments;
14 delivers the reproducible artifact and manuscript. Each step requires
separate authorization; this sequence alone grants none.

The table below preserves the original 14-phase research objectives. D-019
reconciles the historical mandatory-EVM wording in its Phase 04/05 rows.

| Phase | Objective | Main deliverables | Dependencies | Acceptance criteria | Scope restrictions |
|---|---|---|---|---|---|
| **01 — Project Foundation & Architecture Bootstrap** | Establish a reproducible, honest foundation. | Go module, metadata CLI, workflow, architecture, decisions, manifest, report. | None. | Build, tests, vet, formatting, CLI, documents and G1–G12 pass with evidence. | No blockchain, ABCI, EVM, hybrid crypto, admission control or benchmarks. |
| **02 — CometBFT Foundation** | Run a minimal real ABCI++ application against a pinned CometBFT version. | Node startup/configuration, lifecycle integration, durable minimal state and deterministic AppHash contract, replay tests. | Approved 01; final CometBFT version and API review. | Real single-node smoke test, restart/replay and state hash agreement at the tested scope. | No EVM or hybrid transaction claims. Do not alter consensus algorithms. |
| **03 — Classical Transaction Core** | Complete B0-Core through separately gated 03A specification, 03B encoding/secp256k1, 03C accounts/persistence/commitment, 03D ABCI integration and 03E one/four-validator validation. | Versioned transaction bytes, classical authentication, account/nonce/balance rules, state commitment, ABCI behavior and positive/negative vectors. | Accepted Phase 02R review and separate approval for each subphase. | Exact signed bytes, binding, replay rejection, deterministic validity and B0-Core validation verified. | No ML-DSA, hybrid fallback or EVM assumption yet. |
| **04 — Sequential EVM decision gate** | Decide whether EVM is needed after B0-Core; if approved separately, integrate sequentially with selected components. | Recorded go/no-go decision; conditional adapter, gas/receipt/state mapping, persistence and differential tests. | Completed 03E and separate owner authorization. | Decision justified by research needs; if integrated, deterministic execution, crash/replay safety and selected semantic comparisons pass. | No automatic EVM integration, parallel executor or performance optimization. |
| **05 — Hybrid Protocol Design** | Freeze the hybrid security profile before code. | Canonical envelope spec, key policy, signed-intent domain, state rules, rejection oracle and test-vector design. | Phase 04 decision recorded; FIPS 204 and library review; separate authorization. | Independent review finds unambiguous encoding, same-intent authentication and fail-closed policy. | Specification only; no implementation or admission mechanism. |
| **06 — Hybrid Cryptographic Module** | Implement and validate both proof checks and binding primitives. | Library adapters, constrained parsing, vectors, negative tests and crypto review record. | Approved 05; pinned crypto libraries. | Known-answer/interoperability vectors and malformed-input tests pass. | No custom cryptographic primitive and no transaction integration claim. |
| **07 — Hybrid Transaction Integration** | Apply hybrid policy to application transactions. | End-to-end account binding, canonical decode, state validation, conditional EVM mapping and fail-closed tests. | Approved 06. | Hybrid-valid tx executes under the approved model; either missing/invalid proof rejects; state transitions replay identically. | No optimization or benchmark conclusion. |
| **08 — ABCI Lifecycle & Safety** | Validate the hybrid policy across real ABCI++ paths. | CheckTx/recheck, proposal and finalize invariants; adversarial proposal tests; AppHash consistency. | Approved 07. | Valid proposals remain acceptable; malicious input cannot bypass checks or split deterministic state. | Do not treat local admission as consensus proof. |
| **09 — Adversarial Testing** | Build an oracle-backed invalid-input corpus. | Malformed, one-proof-invalid, binding, state, replay and proposer-injection cases; fuzz/property tests. | Approved 08. | Every case has input bytes, expected path/outcome and reproducible seed; no safety regression. | No performance or novelty claim from functional tests alone. |
| **10 — Benchmark Infrastructure** | Measure costs by phase without changing validity. | Workload generator, instrumentation, profiles, environment/seed manifests and clean controls. | Approved 09. | Repeatable runs capture CPU, allocations, latency, throughput and resource context with validation. | No admission-control optimization. |
| **11 — Baseline Evaluation** | Quantify and diagnose invalid-input effects. | Single-node and appropriate baseline measurements, statistical analysis, profiles and root-cause report. | Approved 10. | Raw data, scripts, controls, uncertainty and interpretation are independently reviewable. | No invented improvement or assumption that a mechanism is needed. |
| **12 — Research Mechanism P (conditional)** | If 11 identifies an addressable bottleneck, design and test a targeted mechanism. | Explicit go/no-go decision, safety proof obligations, implementation, ablation and comparison. | Approved 11 **plus** documented research decision that a mechanism is justified. | If authorized: correctness preserved and measured benefit/trade-offs shown against strong baselines; if not: record no-go and skip. | Never implement P merely to follow the roadmap; no consensus-critical local nondeterminism. |
| **13 — Multi-Node Experiments** | Test correctness and costs across validators and hosts. | Repeatable local and multi-host deployments, workload runs, AppHash agreement and network-resource observations. | Approved 11, and 12 only if pursued. | Independent validators agree on state; environment, topology and raw results are recorded. | Local multi-process tests must not be misrepresented as multi-host results. |
| **14 — Research Artifact & Manuscript** | Package a reproducible, evidence-limited research contribution. | Pinned source/data, scripts, artifact instructions, threat model, limitations and manuscript. | Approved 13 and complete review of prior phases. | Independent reproduction of supported claims and clear distinction between observation and hypothesis. | No unsupported novelty, vulnerability or venue acceptance claim. |

If Phase 11 finds no meaningful addressable cost, Phase 12 is skipped or
reframed only after a documented decision. The research can still contribute
a rigorous correctness and cost characterization. Gate decisions and accepted
changes are recorded in `DECISIONS.md` and `PHASE_STATUS.md`.
