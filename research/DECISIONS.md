# Architectural decision log

Current disposition of D-037: the Owner merged [refactor PR #18](https://github.com/the-phong/Beam/pull/18)
into `test-mainet` at `e8f20726f5cea94b47c4dcddb756e39d21207448`.
The Controller conditionally accepted its structural scope and the
[post-merge CI](https://github.com/the-phong/Beam/actions/runs/43146676567)
passed. The [review closeout](evidence/refactor/CONTROLLER_REVIEW_CLOSEOUT.md)
records limits; [branch policy](../docs/BRANCH_POLICY.md) remains an Owner
decision. The original proposal text below is retained for history.

## D-037 — Domain-oriented source layout (2026-11-11; proposed for Controller/Owner review)

The Owner requested one architecture-first structural refactor from the S06
merged `test-mainet` commit `ed6bc8f474c13ed709d5ab5f57b3856edcbf602a`.
The work is isolated on `refactor/domain-oriented-architecture` and a Draft PR;
it does not authorize a research stage, protocol change or merge. Source is
grouped into legacy, classical B0, hybrid B1, experimental profiles/load/
telemetry/corpus/benchmark, and functional CLI entrypoints. The existing
Go package identifiers and executable flags remain to minimize semantic risk.
`docs/REFACTOR_MAPPING.md` records old and new paths and verification. Frozen
corpus and historical evidence bytes remain unchanged; old path spellings in
historical reports are part of their revision context. The active Makefile,
manifest and source navigation use the new paths. No module version changed.
Technical completion and acceptance depend on the full regression report and
independent review of the Draft PR.

## D-036 — S06 single-host pilot and fail-closed offered load (2026-11-11; review candidate)

The Owner merged S05 PR #16 into `test-mainet` at
`ac91250ddff31d07b6dcb19f81878e149256e84f` and authorized S06. The
S06 harness runs four distinct CometBFT v0.38.26 validator processes and
stores on one WSL2 host, with the existing research app servers in the test
process. A separate `profile-load` process sends pre-signed transactions;
four separate `profile-observer` processes collect node samples. The
[`TESTBED_CONFIG`](protocol/TESTBED_CONFIG.md) pins Comet consensus/mempool
settings, the public scenario corpus and all safety caps. This is a pilot in
one fault domain, not four independent machines.

The generator uses absolute monotonic send deadlines and records actual
socket-write timestamps. It fails a run if the planned rate, real writes,
CheckTx oracle or runtime safety checks fail. It accepts only literal
loopback RPC origins, disables proxy/redirects and rejects a claimed direct
P2P source. Comet's P2P relay is observed separately by the accepted S05
instrumentation. The test shuffles seven isolated jobs, exercises all three
regimes, checks four-validator state and P2P relay, and labels missing or
failed sends explicitly. Only bounded pilot evidence is claimed; it cannot
answer RQ1–RQ3, estimate DoS susceptibility or satisfy multi-host timing and
statistical calibration. S07 requires separate Owner authorization.


## D-035 — S05 bounded telemetry without consensus inputs (2026-11-10; review candidate)

The Owner merged S04 PR #15 into `test-mainet` at
`630f2d8c81398f84073796747108c30ab9dee3c8` and authorized S05. The
four-profile application gains an optional bounded recorder for `CheckTx`
result/stage and monotonic secp, ML and state spans. The OFF path invokes
the accepted evaluator without a clock read in the state machine. The ON
path never feeds timing, buffer state or local process measurements into
validation or AppHash. No B0/B1 or CometBFT protocol source changes.

A separate loopback-only proxy observes controlled RPC submissions and
classifies only known response reasons. A process sampler reads native
Comet accepted-mempool RPC, Prometheus mempool and P2P `0x30` channel
counters, and Linux `/proc` CPU/RSS. This avoids treating rejected CheckTx
traffic as accepted occupancy and avoids conflating RPC with P2P relay.
For S05 only, a source-checked, disposable copy of pinned CometBFT v0.38.26
adds aggregate decoded P2P transaction count/bytes at `Reactor.Receive` and
a monotonic CheckTx socket-client enqueue-to-writer-dequeue histogram. The
normal Comet build and its protocol path remain unchanged; the patch adds
only Prometheus observations and uses a temporary modfile under ignored
`build/`. The script fails closed if any of its three pinned source files
changes. The socket queue component ends before transport and the app
callback. Total Comet enqueue-to-app wait and P2P propagation latency remain
`NOT MEASURED`; no subtraction of proxy and application spans is used.
The already pinned Prometheus client v1.21.0 (module minimum Go 1.21) is
promoted from indirect to direct for the hook benchmark and the temporary
Comet build. No version changed; `go mod tidy -diff`, `make verify` and the
four-validator S05 test verify the dependency path. Its cost is measured
only as isolated metric calls, not whole-node overhead.
The [metric dictionary](protocol/METRICS_DICTIONARY.md) records units,
availability, quality, dropped samples, privacy and reproducible commands.
The five S05 checklist items have local technical evidence, subject to
independent review on the Draft PR. This is not Owner acceptance or S06
authorization; S06 remains closed.

## D-034 — S04 bounded correctness network (2026-11-09; technical review candidate)

After the Owner merged S03 PR #14 into `test-mainet` at
`1c8084207adb5da787e34e230dd0dc6d9a3de05f`, the Owner authorized S04
in the original `pluto` folder on `research/s04-bft-correctness`. The S04
testbed runs four independent CometBFT v0.38.26 validator processes and
four separate Pebble v1.1.5 research-v2 snapshots on localhost. The
ABCI++ servers run in the integration test process and use the same
`researchv2app` callback implementation for C, CP, P and H. The public
corpus supplies the signed bytes and common Genesis V4; ephemeral Comet
validator and P2P keys remain in disposable homes. This is an isolated
correctness testbed, not a deployable `plutod` profile or a multi-host trial.

The research-v2 store uses its own key/record magic, binds profile and
SHA-256 of canonical genesis, validates the state hash on reopen, and fails
closed after an uncertain sync response. `FinalizeBlock` prepares a pending
candidate; `Commit` performs a Pebble synced write. Rejected transactions
do not alter account balances or nonces. The optional stage recorder
observes validation steps but supplies no consensus input. The CometBFT
mempool cache can reject replayed bytes before ABCI `CheckTx`; the direct
oracle independently checks nonce/recheck and proposal behavior.

This choice leaves accepted B0/B1 runtime semantics and pinned dependencies
unchanged. Evidence is in the
[S04 correctness report](evidence/s04/CORRECTNESS_REPORT.md).
One WSL2 host is a common failure and resource domain. No loaded relay,
DoS, performance, Byzantine fault, physical power-loss, EVM or S05 result
follows from S04. Controller review and Owner acceptance remain open.

## D-033 — S03 four-profile implementation (2026-11-09; technical review candidate)

The Owner confirmed S02's gate and merge of PR #12 into `test-mainet` at
`2a3dcbceb52b996425d911dde87b691317f56efd`, then authorized S03. The
new `internal/profiles` transaction/state path and `internal/profiles/app`
direct ABCI++ application select C, CP, P or H while keeping one genesis,
account registry, logical transfer, state commitment and callback path. CP
uses canonical zero padding of the ML proof's 2,420-byte size and is accepted
as a valid transfer. Public keys stay in genesis; each profile signs a
profile-separated canonical intent. No accepted B0/B1 behavior or dependency
version changes. The direct S03 app holds state only in memory; durable
storage, real CometBFT node wiring and four-validator proof belong to S04.
The [equivalence report](protocol/PROFILE_EQUIVALENCE.md) and
public vectors record implementation details and limits. This is an S03
engineering candidate for PR review, not a new RQ result or high-load approval.

## D-032 — S02 scientific contract (Owner-confirmed 2026-11-09)

The Owner authorized S02 after the bounded S01 technical gate. The proposed
[`RESEARCH_PROTOCOL.md`](protocol/RESEARCH_PROTOCOL.md),
[`THREAT_MODEL.md`](protocol/THREAT_MODEL.md) and
[`MEASUREMENT_CONTRACT.md`](protocol/MEASUREMENT_CONTRACT.md)
define RQ1–RQ3, actor capabilities, four controlled profiles, observation
points, alert timing, QC and safety limits. CP is a valid transaction padded
to H's exact wire length; all profiles use a new common transfer/state path
and genesis-bound public keys. CP padding has one canonical all-zero encoding
and is excluded from the signed-intent digest, as H's ML proof is excluded
from its secp digest; S03 must test this exact policy. The primary RQ1 H–CP comparison concerns
CheckTx service time at equal bytes, not a B0/B1 historical delta. The 80%
early-recall and 1% FPR thresholds, detector cost caps and calibration
procedure are study design choices. S03 must prove codec/profile equivalence;
S05–S07 must implement and verify instrumentation and reproducibility.
The Owner confirmed Gate S02 after merging PR #12 into `test-mainet` at
`2a3dcbceb52b996425d911dde87b691317f56efd` and authorized S03. No
separate GitHub review/comment is visible; the Owner's confirmation in the
session is the recorded acceptance source. This acceptance does not authorize
high-load experiments or imply RQ1–RQ3 results. Accepted B0/B1 semantics and
D-020–D-030 are unchanged.

## D-031 — New experimental checklist provenance (2026-11-07; in progress)

The Owner requested implementation of the separate reproducible RQ1–RQ3
checklist on a review branch, with only evidenced items checked. The fixed
source baseline is merged `test-mainet` commit
`0bdd869f2c6b3ed59b0ae825954d8667b3c7481e`, tree
`09c24280590aee51d97b53b18c5d9e193dbe6dbe`. Historical Phase 05
measurements remain historical and are not promoted to the new four-profile
study. `research-v2` carries the checklist copy, source/evidence inventory,
and future preregistration and runs. This decision records provenance and
work separation; it does not accept the new research contract, select a
causal endpoint, claim CP/P implementation, or approve a detector. Changes to
accepted B0/B1 behavior require their own explicit decision and tests.
The Owner controls acceptance and merge of the new branch.

Security tooling for D-031: the official `gitleaks/gitleaks` v8.30.1 Linux
x64 release is pinned in CI by archive SHA-256
`551f6fc83ea457d62a0d98237cbad105af8d557003051f41f3e7ca7b3f2470eb`.
Its source release `go.mod` declares Go 1.24.11, but the executable is used
as a separate CI binary and is not imported into Pluto's Go module. It scans
proposed Git commits for known secret patterns; it cannot establish that all
historical logs are privacy safe or replace a publication review. The same
release scanned baseline ancestry and the current worktree locally with no
findings. The full-ref scan found one generic-key false positive in prose on
the unrelated `refactor/unified-architecture` branch; no history rewrite or
automatic allowlist was applied.

## Phase 05 experimental-methodology decision (Owner-authorized 2026-10-26)

**D-030 — Public signed corpus and run-level comparison without production
instrumentation.** Phase 05 retains accepted B0/B1 protocol and consensus
behavior. A public corpus freezes canonical genesis, already signed bytes,
case IDs, SHA-256 and expected first-failure codes; ephemeral signing keys are
discarded. A research-only wrapper measures callback boundaries, and Go
benchmarks measure components. The real four-validator runner uses fresh
localhost homes, bounded rates, matched-height state checks and per-process
sampling. Raw network comparison is not interpreted as ML-DSA-only cost
because B0/B1 differ in transaction size, account registry and persistence.
Five independent runs are used for primary conditions; statistical resampling
uses runs, not individual requests. This choice avoids changes to
consensus-critical code and makes failure history auditable. Its limitations
are short diagnostic windows, shared WSL2 resources, sampled RSS, polling
resolution and the absence of an exactly matched byte-rate network condition
if the safe offered-rate cap prevents one. The preregistered protocol is
[`specs/PHASE_05_EXPERIMENT.md`](specs/PHASE_05_EXPERIMENT.md). Status:
implementation and measurement candidate, not Controller acceptance or a
DoS mechanism authorization.

## Phase 04 B1 hybrid implementation decisions (2026-10-26; Controller review candidate)

**D-028 — Consolidated B1 workflow and EVM NO-GO for this experiment.**
The Owner explicitly authorized one complete Phase 04 implementation that
consolidates the historical roadmap's hybrid design, cryptographic module,
transaction integration, and ABCI safety activities previously placed in
Phases 05–08. This supersedes that scheduling sequence for this B1 experiment;
the historical rows below remain for audit. EVM is NO-GO at this gate: hybrid
authentication, correctness and resource costs can be investigated in the
existing CometBFT application without adding gas, storage or EVM execution
semantics. Future EVM integration requires a new research need and Owner
authorization. The full deliberately invalid workload benchmark remains
future work. This decision does not authorize a DoS mechanism or Phase 05.

**D-029 — Isolated genesis-bound B1 hybrid account profile.** A fresh B1 chain
uses `PLUTOTX2` transaction version 2/profile 1, `PLUTOG3!` Genesis V3 and
`PLUTOS3!` committed record under `pluto/committed-state/b1-v1` in a distinct
`pluto-b1-v1` directory. The B0/Phase 02 formats, keys, genesis, signatures
and AppHashes remain unchanged and are never silently migrated. Each B1
genesis account immutably binds its derived address to one compressed
secp256k1 key and one ML-DSA-44 key. Both proofs sign the same domain-separated
canonical digest; an unregistered or substituted key is rejected before
either signature can authorize state. The registered recipient rule fixes the
account set at genesis; there is no key rotation or account creation in this
profile. A B1 block uses the accepted B0 balance/nonce transition primitive
but a separate registry-bound AppHash. `init-b1-4v` creates four fresh D-026
style H0 homes with one shared B1 genesis. B1's `pluto.json` additionally
records the expected public validator address for that specific home;
`start` refuses a copied key from another valid genesis member. This is
profile metadata, not a new D-024 record or an initialization marker.
`start` requires an existing valid B1 record and rejects foreign B0 state.
This is a scoped candidate for
independent Controller review, not Owner acceptance.

Dependency decision for D-029: `github.com/cloudflare/circl v1.6.4` provides
`sign/mldsa/mldsa44` with pure ML-DSA signatures and a caller-supplied
context. The module declares Go 1.24.0. The pinned module archive checksum is
`h1:pOXuDTCEYyzydgUpQ0CQz3LsinKjiSk6nNP5Lt5K64U=`. The algorithmic
reference is NIST FIPS 204; this does not establish FIPS module certification.
The existing `secp256k1/v4 v4.4.1` implementation remains pinned. CIRCL
v1.6.4 also raises selected indirect `golang.org/x/*` versions; the exact
`go.mod`/`go.sum` delta and regression results govern acceptance. The B1
protocol, threat model and byte offsets are in
[`specs/B1_HYBRID_AUTHENTICATION.md`](specs/B1_HYBRID_AUTHENTICATION.md).

## Phase 03E four-validator provisioning decision (2026-10-25; implementation candidate)

**D-027 — Shared four-validator genesis with independent D-026 H0 homes.**
The Owner authorized Phase 03E after merging Phase 03D into `test-mainet`.
For a fresh local four-validator network, `plutod init-b0-4v` exclusively
claims one root, asks the pinned CometBFT binary to generate four independent
validator and P2P identities, then constructs one byte-identical genesis
containing those four validator public keys at equal power. Each home receives
the same D-025 application genesis transport and an independently synced
D-024 H0 snapshot. The local validator private key must match exactly one
genesis member before start; P2P keys and last-sign state remain private and
independent. Interrupted initialization remains incomplete and cannot be
silently retried over the same root. No consensus, transaction, AppHash or
record-format rule changes. This is a Phase 03E implementation decision for
Controller review, not Owner acceptance of Phase 03E.

The four-node helper is scoped to one host and fresh homes. It does not
import externally provisioned validator keys, migrate existing homes, or
establish independent fault domains. Genesis time is generated once and
shared, so repeated networks have different genesis hashes and must never
reuse validator signing state across them.

## Phase 03D Owner acceptance and Git closeout (2026-10-25)

After independent Controller technical acceptance, the Owner formally
accepted Phase 03D at staged tree
`7f7002496f38cea99cc3e0b7137e452523cc0b94`.
D-020–D-026 remain accepted; this closeout changes no transaction, state,
genesis, persistence or consensus rule. Under D-018, the Owner authorized a
commit and push on `phase/03d-b0-abci-integration` and a Draft PR targeting
`test-mainet`. Merge remains an Owner decision; Phase 03E is unauthorized.
Hosted CI, physical power-loss durability, four-validator BFT, PQC/hybrid
authentication and DoS resistance remain outside the accepted evidence.
The D-026 implementation and conditional-review wording below records its
earlier status and is superseded for execution authority by this closeout.

## Phase 03D explicit initialization decision (2026-10-25; accepted by Owner)

**D-026 — Explicit B0 Initialization and Complete-Data-Loss Detection
(OWNER APPROVED FOR PHASE 03D IMPLEMENTATION on 2026-10-25).** `plutod
init-b0` must durably create and verify the canonical Genesis V2 height-zero
snapshot in a genuinely fresh home. `plutod start` requires an existing,
complete and integrity-valid B0 snapshot and never creates or repairs it.
CometBFT `InitChain` validates the identical persisted H0 and returns its
AppHash; identical H0 retries are idempotent without a write. Missing,
incomplete, conflicting or unexpectedly advanced state fails closed. Older
D-025 homes containing configuration but no B0 snapshot require explicit
operator disposition, not an automatic reset or migration. This decision
changes the initialization actor and order only. The accepted D-024 key,
record bytes, synced write and AppHash V2, and the D-025 canonical JSON/Base64
transport remain unchanged. The local implementation and software fault
evidence are review candidates; Phase 03D remains subject to Controller
review. Approval does not authorize a
commit, push, merge or Phase 03E.

| Prior requirement | D-026 replacement | Affected implementation | Required verification |
|---|---|---|---|
| B0 D-022 `InitChain` durably creates H0 on first start | `init-b0` creates and verifies H0 before success | `internal/node.InitB0`, `internal/b0store` existing Create API | Fresh-home H0, failure windows, duplicate `init-b0` refusal |
| B0 D-022 `InitChain` accepts identical H0 retry | Keep retry, but validate only; never create or rewrite | `internal/b0app.InitChain` | Exact H0 response, retry byte equality, conflicting/missing/H>0 rejection |
| D-025 transport is decoded before H0 creation inside `InitChain` | Keep transport bytes and strict decoding; validate during `init-b0` and again at `InitChain` | `internal/b0genesis`, `internal/node`, `internal/b0app` | Canonical JSON/Base64 vectors and real Comet handshake |
| `start` could allow missing B0 when Comet data was absent | `start` requires a valid existing snapshot regardless of Comet data | `internal/node.runB0`, `internal/b0app.Open` | Missing/incomplete/corrupt home rejection and existing H0/H>0 compatibility |

The [D-026 decision document](decisions/D-026_EXPLICIT_B0_INITIALIZATION_PROPOSAL.md)
records crash windows, alternatives, compatibility and operator recovery.
Historical D-022 and D-025 wording below is preserved as superseded history.

**D-026 conditional source review correction (2026-10-25):** The final
fresh-home directory claim now uses exclusive `os.Mkdir` after any parent
creation; a preliminary existence check is only an early diagnostic.
Successful initialization also parses the generated CometBFT TOML, checks
the configured local addresses and file paths, and validates the generated
node key, validator key/genesis binding and fresh last-sign state. This adds
no persistent marker, changes no D-024 record bytes or D-025 transport, and
does not regenerate an invalid identity. The verifier uses the already pinned
`github.com/spf13/viper v1.19.0` parser path used by CometBFT; its module
version is unchanged and its `go.mod` classification becomes direct. The
Controller's R-D026-01/R-D026-02 findings remain pending final review until
the rework evidence is assessed.

## Phase 03D genesis transport decision (2026-10-24; accepted by Owner)

**D-025 — Canonical Genesis V2 JSON/Base64 transport (ACCEPTED BY OWNER on
2026-10-24).** The D-021 Genesis V2 binary and AppHash V2 remain unchanged.
For the explicit B0 profile, CometBFT `genesis.json` stores `app_state` as a
JSON string equal to `json.Marshal(base64.StdEncoding.EncodeToString(G))`,
where `G` is the complete canonical Genesis V2 binary. Pinned CometBFT
v0.38.26 forwards those JSON value bytes unchanged in
`RequestInitChain.AppStateBytes`. The Pluto B0 adapter accepts only that
canonical JSON string and RFC 4648 standard padded Base64, decodes it, and
passes the original binary to the existing `b0state.ParseGenesis` parser.
The decoded chain ID must match both the ABCI request and configured profile.
The binary alone defines genesis state and AppHash; transport text is never
committed. There is no alternate-format fallback.

This explicitly amends the Phase 03A wording that required raw binary bytes
directly in `InitChain.AppStateBytes`: standard CometBFT genesis files carry
JSON `app_state` values. The historical rule is retained in the amended
specifications. D-020–D-024's transaction, account, state commitment and
physical persistence rules are unchanged. This checkpoint authorizes only
transport and initialization; transaction processing remains gated on
Controller review within Phase 03D. Phase 03E is unauthorized.

**Execution-status addendum, 2026-10-24:** The Owner subsequently accepted
closure of the D-025 checkpoint and authorized the remaining Phase 03D B0
ABCI integration and recovery work. This changes execution authority only;
the accepted D-025 transport rule above is unchanged. T7 remains PARTIAL.

## Phase 03C physical storage decision (2026-10-24; accepted by Owner)

**D-024 — Isolated B0 V2 complete-snapshot persistence (Accepted by Owner on
2026-10-24).** Use a separate Pebble database directory selected by an
explicit caller. Store one complete versioned record at key
`pluto/committed-state/v2` as
`ASCII("PLUTOS2!") || canonical_logical_state_v2 || AppHashV2[32]`, replacing
it with one `DB.Set(key, value, pebble.Sync)` per committed block. Opening
validates the entire record, V2 version, chain context, account invariants
and recomputed hash; the existing V1 key fails closed. Fresh initialization requires an absent
directory and canonical Genesis V2. Alternatives: multi-key atomic batch,
per-account updates, implicit migration of V1 data. A single record makes
the commit boundary and corruption checks simple, at O(K) serialization,
hashing and write cost for K accounts. A returned write error can leave the
durable outcome uncertain; the store does not promote in-memory state and
requires close/reopen validation before another candidate. This decision
does not claim physical power-loss durability from graceful restart tests.
The existing Phase 02 store and live ABCI path are unchanged. **Status:
ACCEPTED BY OWNER on 2026-10-24**, strictly as a correctness-oriented
experimental B0 baseline after Controller acceptance of Phase 03C, the R1
correction and the [risk addendum](evidence/phase_03c/RISK_AND_ACCEPTANCE_ADDENDUM.md).
Full-snapshot serialization, hashing and persistence cost O(K). Pluto and
CometBFT have no shared atomic commit. Production-grade or arbitrary-power-loss
durability has not been established, and B0 V2 cross-store crash recovery
remains unverified. Any later persistence-architecture change requires an
explicit, separately versioned decision and appropriate tests. This decision
does not merge PR #6, activate live B0 transactions or authorize Phase 03D.

## Phase 03B dependency decision (2026-10-23; Accepted by Owner)

**D-023 — Stateless secp256k1 adapter dependency (Accepted by Owner on 2026-10-23).** Select
`github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1` as the direct dependency
for Phase 03B's compressed-point parsing, ECDSA verification over an already
computed 32-byte digest, and deterministic test-only signing. It was already
in the Phase 03A graph indirectly at v4.4.0; this is an explicit, reviewed
update to v4.4.1, not a silent upgrade. The module's minimum Go version is
1.17, below Pluto's 1.24.0. The module declares one direct dependency,
`github.com/decred/dcrd/crypto/blake256 v1.1.0`, already present in Pluto's
module graph; `go mod tidy` introduced no new module path. Module zip checksum:
`h1:5RVFMOWjMyRy8cARdy79nAmgYw3hK/4HUq48LQ6Wwqo=`. Its
`ModNScalar.SetByteSlice` reduces on overflow, so Pluto rejects overflow and
zero **before** using the scalar for ECDSA; it also checks low-S explicitly.
The public-key parser accepts other encodings at other lengths, so Pluto
enforces exactly 33 bytes and prefix 02/03 before parsing. Alternatives:
retain indirect v4.4.0; add a different secp256k1 stack; implement curve
arithmetic locally. v4.4.1 is the latest version listed by the Go module
index at selection time and supports the required prehashed ECDSA API.
This adapter is isolated from the Phase 02 ABCI runtime. D-020–D-022 remain
unchanged and accepted. The Owner accepted this exact dependency selection on
2026-10-23 after Controller acceptance and the merge of Phase 03B PR #4 at
`8be07c87abed73763d40252e99bbf0573da487b4`. This acceptance does not
authorize Phase 03C or change the accepted B0 V1 protocol.

## Phase 03A B0 V1 protocol decisions (2026-10-23; accepted by owner)

The owner authorized specification work and, after Controller R1–R5 rework,
explicitly **accepted D-020, D-021 and D-022 as B0 V1 protocol rules** on
2026-10-23. Their exact rules, alternatives and rationale are in
`research/specs/` and the Phase 03A report. Acceptance does not mean the
protocol is implemented or cryptographically tested. The Owner controls
Draft PR #3's merge; Phase 03B requires separate authorization after merge.

**D-020 — B0 classical transaction V1 (Accepted by Owner).** Select a fixed binary,
versioned, chain-bound value-transfer wire format with compressed secp256k1
public key, `uint64` amount/nonce, compact low-S ECDSA signature, SHA-256
domain-separated signed intent, SHA-256-derived 20-byte account address and
separate domain-separated TxID. Alternatives considered: Ethereum transaction
encoding/address rules, DER signatures, an independent sender-address field,
fees/gas/memo/expiry fields and variable-length encodings. Rationale: a small
unambiguous B0 correctness baseline with one sender identity. Trade-offs:
fee-free operation has no economic spam deterrent; addresses are not Ethereum
compatible; future fields require a new version. The controller rework fixes
cheap-rejection precedence: structural checks, chain match, positive amount
and canonical signature scalars precede point validation/ECDSA. A matching
sender nonce of `MaxUint64` fails Code 6 before balance checks. Pluto TxID
exists exactly for structurally valid V1 wire with in-range low-S scalars,
even if its key, signature, chain or state is invalid; CometBFT RPC instead
uses SHA-256 of raw transaction bytes. Status: **Accepted by Owner on
2026-10-23 for future separately authorized implementation**. Verification:
no transaction bytes or cryptographic vectors were generated or executed in
Phase 03A.

**D-021 — B0 account state and AppHash V2 (Accepted by Owner).** Select checked
`uint64` balances, nonces and supply, sorted 20-byte accounts, sequential
value transfers, no fees/mint/burn, and a SHA-256 commitment to version,
chain ID, height, supply and all accounts. New B0 chains use canonical V2
genesis bytes; Phase 02 V1 homes have no implicit migration. Alternatives:
Merkle/trie state root, account deletion, automatic migration and unordered
storage hashing. Rationale: independently reproducible state transitions
before optional EVM work. Trade-offs: flat commitment has no account proof
and may be costly at scale; physical atomic persistence is future Phase 03C
work. Full-state recomputation is proportional to account count; B0/B1
measurements must control account count and execution model. Status:
**Accepted by Owner on 2026-10-23 for future separately authorized
implementation**. Verification: specification review
and arithmetic/length checks only; no V2 state exists in code.

**D-022 — B0 ABCI validity policy (Accepted by Owner).** Select conservative
committed-state `CheckTx`, deterministic sequential validation of the full
proposal in `ProcessProposal`, and one ordered result per decided transaction
in `FinalizeBlock`, with invalid decided transactions skipped without state
effect. Keep pending state until synced `Commit`. Alternatives: trusting local
admission, accepting all proposals, or aborting finalization on ordinary bad
transaction bytes. Rationale: independent proposer-injection defense and
deterministic fallback without unnecessary application halt. Trade-offs:
strict proposal rejection may cost rounds under bad proposers; the anomalous
decided-block fallback and fixed count/byte limits require implementation and
multi-validator validation before operational claims.
The shared evaluator now explicitly uses cheap structural/amount/scalar checks
before ECDSA, rejects nonce overflow as Code 6, and distinguishes Pluto TxID
eligibility from CometBFT's raw-byte RPC hash. Block-limit fixtures require
otherwise valid signed and funded sequential transactions.
Status: **Accepted by Owner on 2026-10-23 for future separately authorized
implementation**. Verification: pinned v0.38.26 interface review only;
current Phase 02 runtime still rejects all transactions.

## Phase 02R approved governance and roadmap decisions (2026-10-23)

**D-018 — Phase-Based Git Governance.** Context: D-011 correctly records the
owner's local-only exception for Phase 02, but Phase 02R has an explicit new
Git workflow. Alternatives: continue local checksums only; commit directly
to an integration branch; one isolated branch and Draft PR per authorized
phase. Selected: branch from the verified `origin/test-mainet` baseline,
commit and push only approved-scope changes to the phase branch, and open a
Draft PR targeting `test-mainet`. The owner performs or explicitly approves
merges; agents never merge automatically, push directly to `test-mainet` or
`master`, rewrite shared history, or infer a new phase from a PASS verdict.
The Coding Agent reports to the ChatGPT technical controller in English;
the controller communicates decisions to the owner in Vietnamese. Each
execution ends with evidence, a full report and STOP. Rationale: reviewable
history and controlled integration. Risks: branch protection may be absent;
Git authentication and an authorized existing author identity are required.
Status: **Accepted by owner for Phase 02R and future separately authorized
phases**, superseding D-011 prospectively only. D-011 and its checksum
evidence remain historically true. Verification: Phase 02R records the
remote source SHA, branch, resulting commit and Draft PR or exact blocker.

**D-019 — B0-Core First, EVM Decision at Phase 04.** Context: the earlier
14-phase plan assumed EVM was mandatory before hybrid work. Alternatives:
integrate EVM unconditionally before B0-Core; complete B0-Core first and
decide whether EVM is needed at Phase 04. Selected: Phase 03A–03E build and
validate the minimal classical B0-Core, including single-node and
four-validator validation, before Phase 04. Phase 04 records a separate
research and implementation decision; sequential EVM integration occurs
only if approved. A no-go is not a permanent rejection of EVM. Rationale:
establish a smaller correctness baseline and avoid adding unneeded execution
complexity before the research need is known. Risks: a later EVM decision may
require additional state/transaction mapping and migration work. Status:
**Accepted by owner as Master Roadmap v2 planning**; no Phase 03A or Phase 04
implementation is authorized. Verification: roadmap, architecture, status
and manifest state B0-Core-first and conditional EVM; later gate records the
actual go/no-go and evidence. D-004 and D-006 remain historical candidate
and conditional sequential-execution guidance.

## Phase 02 accepted decisions (2026-10-15)

**D-011 — Local baseline replaces Git checkpoint.** Context: user chose local
development without Git identity, commits, branches or remote. Alternatives:
Git baseline; no traceability; SHA-256 file manifest. Selected: checksums of
the 16 existing nonsecret source/config/documentation files in
`research/evidence/phase_02/BASELINE_SHA256.txt`, generated and verified before
Phase 02 edits. Rationale: reproducible Phase 01 checkpoint without invented
identity. Risk: no history/diff machinery; preserve original report/evidence
and use this manifest plus the Phase 02 report. Status: **Accepted** by user.
Verification: `sha256sum -c` passed at baseline; the old commit requirement was
superseded, never fulfilled.

**D-012 — CometBFT v0.38.26.** Context: v1.0.0 was only a historical candidate.
Alternatives: v1.0.0, v0.38.26, v0.40.0. Selected: exact v0.38.26, pinned in
`go.mod` and built as `build/cometbft`; module checksum retained in `go.sum`.
Rationale: v1.0.0 is affected by published ASA-2025-001/002; the 0.38 line
has patches for [ASA-2025-003][asa003] (0.38.19) and
[CSA-2026-001 Tachyon][tachyon] (0.38.21); v0.38.26 is later than these fixes.
v0.40.0 requires Go 1.25, above local Go 1.24.1. v0.38.26 declares Go
1.22.11. Trade-offs: v0.38's ABCI API differs from Phase 01's v1.0.0
reference; security is not proven merely by a patched version or build.
Status: **Accepted for Phase 02 local development**. Verification: module
download, `go build`, actual ABCI integration, live block production and
restart; [release][comet038release], [ASA-2025-001][asa001],
[ASA-2025-002][asa002], [ASA-2025-003][asa003], [Tachyon][tachyon].

**D-013 — Separate pinned CometBFT child and Pluto ABCI socket.** Context:
need a real node with small integration surface. Alternatives: in-process node;
external arbitrary binary; bundled exact-version child. Selected: `plutod`
starts its adjacent `cometbft` binary and a localhost ABCI socket server.
Rationale: CometBFT's own init/node CLI gives valid validator configuration,
while Pluto owns application lifecycle. Trade-offs: two processes and startup
coordination; no public bindings by default. Status: **Accepted**. Verified by
live RPC and restart integration test.

**D-014 — Pebble v1.1.5 and commit boundary.** Context: durable minimal state.
Alternatives: Pebble, flat file atomic rename, SQL. Selected: exact Pebble
v1.1.5, one versioned state key, `Set(..., pebble.Sync)` only in `InitChain`
for genesis and `Commit` for blocks. `FinalizeBlock` remains memory-only.
Rationale: single-key atomicity, synced WAL, exclusive writable directory;
also the provisional version reviewed in Phase 01. Trade-offs: no general
multi-key transaction or arbitrary-power-loss proof; CometBFT/application
cross-store atomicity relies on ABCI replay. Status: **Accepted**. Verified by
reload, finalize-without-commit, exclusive-open and restart tests; storage
version is pinned in `go.mod`.

**D-015 — Canonical state digest.** Context: deterministic Phase 02 AppHash.
Alternatives: JSON hash, Merkle tree, fixed binary SHA-256. Selected: SHA-256
with `PLUTO/APP_STATE/V1` NUL-terminated domain, big-endian version, chain-ID
length and bytes, and height; versioned persisted record with recomputation on
reload. Empty blocks advance height and hash. Rationale: portable and
testable, no unordered maps. Trade-off: not an Ethereum trie root and must be
versioned when adding state. Status: **Accepted**. Verified by canonical
encoding tests and two-instance deterministic replay.

**D-016 — Genesis and node configuration.** Context: reproducible developer
chain. Alternatives: hand-written validator/genesis; CometBFT init. Selected:
bundled `cometbft init`, validated chain ID, initial height 1, empty app
state, generated single validator, localhost listeners and one-second empty
blocks. Init refuses an existing home; start checks Pluto metadata against
genesis. Trade-offs: generated private keys require protection; port collision
and file-system errors still require operator intervention. Status:
**Accepted**. Verified by live init/start, RPC and restart test.

**D-017 — Out-of-scope transactions.** Context: Phase 02 has no transaction
semantics. Alternatives: accept no-op txs; reject all. Selected: `CheckTx`
code 1, empty proposer output, deterministic proposal rejection for nonempty
txs, `FinalizeBlock` error if they somehow reach it. Rationale: fail closed,
never claim execution. Trade-off: no workload submission until Phase 03.
Status: **Accepted**. Verified by ABCI lifecycle tests.

[comet038release]: https://github.com/cometbft/cometbft/releases/tag/v0.38.26
[asa001]: https://github.com/cometbft/cometbft/security/advisories/GHSA-22qq-3xwm-r5x4
[asa002]: https://github.com/cometbft/cometbft/security/advisories/GHSA-r3r4-g7hq-pq4f
[asa003]: https://github.com/cometbft/cometbft/security/advisories/GHSA-hrhf-2vcr-ghch
[tachyon]: https://github.com/cometbft/cometbft/security/advisories/GHSA-c32p-wcqj-j677

## Historical Phase 01 decisions (future candidates superseded where noted)

Decision status describes authorization for this foundation, not experimental
validation. `Accepted` decisions apply in Phase 01; future integration details
remain `Proposed`. Changing an accepted decision requires a new dated entry
with a reason and verification plan. Observations were made on 2026-10-15.

## D-001 — Go application language

- **Context:** A new custom CometBFT application needs a maintainable systems
  language and reproducible modules.
- **Alternatives:** Go; Rust with an ABCI bridge; a framework in another language.
- **Selected:** Go module with a declared Go 1.24.0 minimum; observed local
  toolchain is Go 1.24.1 on linux/amd64.
- **Rationale:** Direct access to candidate CometBFT and geth Go APIs with
  simple build/test tooling.
- **Risks/trade-offs:** The minimum may need adjustment after dependency
  integration; runtime and compiler versions affect later measurements.
- **Status:** Accepted for Phase 01.
- **Verification:** `go version`, `go build`, `go test`, and `go vet` are run in
  Phase 01; no future dependency combination is yet compiled.

## D-002 — CometBFT consensus foundation

- **Context:** The target research needs Byzantine fault tolerant ordering and
  ABCI++ application callbacks.
- **Alternatives:** CometBFT; another BFT engine; custom consensus.
- **Selected:** CometBFT, with v1.0.0 as the initial **provisional integration
  target**, matching the historical plan.
- **Rationale:** The [v1.0.0 ABCI++ methods][abci-methods] and
  [requirements][abci-requirements] explicitly describe the lifecycle under
  study. No consensus algorithm changes are permitted.
- **Risks/trade-offs:** Phase 02 must check API compatibility, security support,
  mempool configuration, liveness and replay behavior at the pinned tag. A
  newer patch line may be preferable after review; it must be recorded.
- **Status:** Accepted engine; exact version Proposed.
- **Verification:** Official release tag and module path
  `github.com/cometbft/cometbft` exist; v1.0.0 declares `go 1.23.1` in
  [go.mod][comet-mod]. No Pluto integration test exists.

## D-003 — Custom application, not Cosmos SDK

- **Context:** The study needs explicit transaction representation and
  application-level validation controls.
- **Alternatives:** Custom ABCI++ application; Cosmos SDK application.
- **Selected:** Custom Go application using CometBFT; no Cosmos SDK dependency.
- **Rationale:** Preserves a clear research boundary and avoids claiming SDK
  semantics or compatibility that do not exist.
- **Risks/trade-offs:** More state, RPC, and EVM integration work is owned by
  Pluto. Adoption of the SDK would require an explicit later decision.
- **Status:** Accepted.
- **Verification:** Phase 01 `go.mod` has no SDK dependency; future behavior is
  architectural, not verified.

## D-004 — EVM library boundary

- **Context:** The target chain requires EVM execution but not a new EVM.
- **Alternatives:** go-ethereum execution components; another EVM library;
  implementing EVM semantics from scratch.
- **Selected:** Propose go-ethereum v1.16.8 behind `internal/evm`, with a
  sequential baseline and explicit transaction-to-message mapping.
- **Rationale:** Historical reference version; official [release][geth-release]
  and [go.mod][geth-mod] are available. The tag declares `go 1.24.0`.
- **Risks/trade-offs:** Exact APIs, state root semantics, receipts, gas, fork
  rules, licensing and compatibility with the ABCI state model need testing.
- **Status:** Proposed for Phase 04; no dependency added in Phase 01.
- **Verification:** Module path `github.com/ethereum/go-ethereum` and Go
  directive observed at tag. Combined build and execution are unverified.

## D-005 — Application storage and commitment

- **Context:** Durable state and deterministic AppHash are needed later.
- **Alternatives:** Pebble; another key/value engine; embedded SQL.
- **Selected:** Propose Pebble v1.1.5 as a candidate storage engine because
  geth v1.16.8 already requires that exact v1 module. Keep AppHash definition
  separate from engine selection.
- **Rationale:** Avoid an avoidable v1/v2 module split during first integration.
  Official [Pebble go.mod][pebble-mod] confirms module path
  `github.com/cockroachdb/pebble` and Go 1.20 directive.
- **Risks/trade-offs:** Pebble v1 lifecycle and geth's storage use need review;
  app-level atomicity, recovery and state commitment are unresolved. Pebble
  bytes alone are not an authenticated state root.
- **Status:** Proposed for later phases.
- **Verification:** Module metadata and geth requirement inspected; no Pluto
  storage code or crash test exists.

## D-006 — Sequential execution baseline

- **Context:** EVM parallelism would confound hybrid-authentication costs.
- **Alternatives:** Sequential execution; parallel scheduling from inception.
- **Selected:** Sequential EVM execution as the initial functional baseline.
- **Rationale:** Establishes an interpretable correctness oracle and comparable
  performance baseline before optimization.
- **Risks/trade-offs:** Limits initial throughput; implementation still needs
  deterministic state ordering and replay checks.
- **Status:** Accepted research constraint.
- **Verification:** No EVM exists yet; not experimentally verified.

## D-007 — Transaction format belongs to Pluto

- **Context:** Standard Ethereum raw transactions have no defined Pluto
  ML-DSA-44 authentication field.
- **Alternatives:** Custom versioned envelope; modify geth transaction types;
  pretend standard raw Ethereum RPC carries both proofs.
- **Selected:** Pluto will specify a versioned canonical application envelope
  and explicit RPC semantics in later protocol work.
- **Rationale:** Makes the signed intent and authentication policy auditable
  without claiming Ethereum wire compatibility.
- **Risks/trade-offs:** Wallet tooling and hash/receipt mapping require work;
  byte encoding, domain, replay protection and account binding are undecided.
- **Status:** Accepted ownership boundary; exact format Proposed.
- **Verification:** No encoding or signature code exists in Phase 01.

## D-008 — Dependency version policy

- **Context:** Experimental results must map to exact source revisions.
- **Alternatives:** Floating latest versions; early unused dependencies;
  pin dependencies when first imported.
- **Selected:** Use Go modules; add exact dependency versions only when used.
  Record candidates separately from actual `go.mod` requirements. Review
  releases, security advisories and toolchain constraints before adoption.
- **Rationale:** Phase 01 builds without network access and avoids a false
  appearance of tested compatibility.
- **Risks/trade-offs:** Candidate versions may change at their phase gate;
  comparisons must record the decision and migration.
- **Status:** Accepted.
- **Verification:** Phase 01 `go.mod` uses only the standard library and has no
  `go.sum`; `go mod verify` is run in the foundation check.

## D-009 — Reproducible research record

- **Context:** Correctness and cost claims require independent reproduction.
- **Alternatives:** Informal notes; versioned plan, manifest, evidence and
  phase reports.
- **Selected:** Version the plan, decisions, architecture, manifest and
  completion report. Later experiments must record commit, toolchain, flags,
  node configuration, hardware, workload seeds and raw results.
- **Rationale:** Separates observed facts from plans and preserves review gates.
- **Risks/trade-offs:** Manual metadata can drift; future tooling should
  generate and validate experiment manifests.
- **Status:** Accepted.
- **Verification:** Required documents and Phase 01 execution evidence will be
  reviewed at the phase gate; no experiment is claimed.

## D-010 — Cryptographic library candidate

- **Context:** Future hybrid authentication needs a FIPS 204-compatible
  ML-DSA-44 implementation and classical secp256k1 verification.
- **Alternatives:** CIRCL; another vetted ML-DSA implementation; new primitive
  implementation (disallowed). Geth's existing secp256k1 stack versus a
  separate library.
- **Selected:** Propose CIRCL v1.6.4 for later evaluation; use geth's existing
  secp256k1 dependency path where its API meets the protocol spec. Neither is
  imported now.
- **Rationale:** CIRCL documents an [ML-DSA-44 package][circl-mldsa], and
  [v1.6.4 go.mod][circl-mod] declares Go 1.24.0. [v1.6.5][circl-newer]
  declares Go 1.25.0,
  exceeding the installed toolchain. Geth v1.16.8 requires
  `github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1`.
- **Risks/trade-offs:** FIPS mode, context handling, known-answer tests,
  serialization and constant-time properties need a later cryptographic
  review. A newer supported release may be selected with a toolchain change.
- **Status:** Proposed for Phase 06.
- **Verification:** The [ML-DSA-44 directory exists at v1.6.4][circl-tag-dir];
  package documentation and module metadata inspected. No crypto API
  compilation, vectors or security assessment performed.

[abci-methods]: https://github.com/cometbft/cometbft/blob/v1.0.0/spec/abci/abci%2B%2B_methods.md
[abci-requirements]: https://github.com/cometbft/cometbft/blob/v1.0.0/spec/abci/abci%2B%2B_app_requirements.md
[comet-mod]: https://github.com/cometbft/cometbft/blob/v1.0.0/go.mod
[geth-release]: https://github.com/ethereum/go-ethereum/releases/tag/v1.16.8
[geth-mod]: https://github.com/ethereum/go-ethereum/blob/v1.16.8/go.mod
[pebble-mod]: https://github.com/cockroachdb/pebble/blob/v1.1.5/go.mod
[circl-mldsa]: https://pkg.go.dev/github.com/cloudflare/circl/sign/mldsa/mldsa44
[circl-mod]: https://github.com/cloudflare/circl/blob/v1.6.4/go.mod
[circl-newer]: https://github.com/cloudflare/circl/blob/v1.6.5/go.mod
[circl-tag-dir]: https://github.com/cloudflare/circl/tree/v1.6.4/sign/mldsa/mldsa44
