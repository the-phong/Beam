# Phase 01 completion report — Pluto

Date: 2026-10-15. Review status: awaiting independent human review.
Evidence: [`research/evidence/phase_01_commands.txt`](../evidence/phase_01_commands.txt).

## 1. Executive Summary

- **Phase:** 01 — Project Foundation & Architecture Bootstrap.
- **Overall engineering status:** **PASS** against G1–G12, subject to
  independent review. PASS is not approval for another phase.
- **Actually accomplished:** A new Git repository and Go module, honest
  metadata-only `plutod` CLI, five behavior subtests, Make/CI checks,
  architecture and decision records, 14-phase plan, manifest, developer
  instructions and observed command evidence.
- **Phase 02 readiness:** The foundation can support a Phase 02 design and
  integration review. Phase 02 is **not authorized**, and no implementation
  of it has begun.
- **Qualifications:** No Git commit exists because no author identity is
  configured. No CometBFT, geth, Pebble or crypto library has been imported
  or compiled together. Docker is unavailable but not required for Phase 01.

## 2. Environment

| Item | Observed value |
|---|---|
| Operating system | WSL2 Ubuntu, Linux 6.18.33.2-microsoft-standard-WSL2, linux/amd64 |
| Go version | `go version go1.24.1 linux/amd64` |
| Git version | 2.34.1 |
| Make version | GNU Make 4.3 |
| Working directory | `/mnt/d/Personal/Search&Develop/Blockchain/system/pluto` |
| Repository state | Newly initialized Git `main`, no commits and no remote; Phase 01 files untracked |
| Docker availability | `docker --version` exit 1: command not found in WSL2 distro |
| Relevant tool limitations | Git `user.name` and `user.email` absent; parent path contains space and `&`, so shell commands must quote it; WSL2-mounted Windows filesystem; no clean-checkout test possible without a commit |

The parent `system` directory contained one pre-existing Vietnamese research
plan and was not a Git repository. It was preserved. A clean `pluto`
subdirectory was initialized as the new repository; the parent plan is not
inside its Git history. `go env` reported `GOMOD=/dev/null` before bootstrap,
`GOPROXY=https://proxy.golang.org,direct`, `GOSUMDB=sum.golang.org`,
`GOOS=linux`, `GOARCH=amd64`, and `GOTOOLCHAIN=auto`. The only verified
assumptions are these observations and the successful Phase 01 commands below.

## 3. Architecture Decisions

The full context, alternatives, rationale, risks, status and verification
record for each item are in [`DECISIONS.md`](../DECISIONS.md). Summary:

| ID | Selected approach; alternatives | Reason and technical risk | Status / verification |
|---|---|---|---|
| D-001 | Go 1.24.0 module; Rust/other languages considered | Direct Go library access; future toolchain compatibility may change | Accepted; Go 1.24.1 build/test verified |
| D-002 | CometBFT, v1.0.0 candidate; other BFT/custom consensus considered | ABCI++ lifecycle suits study; exact API and security support need review | Engine accepted, version provisional; tag/module metadata checked, no integration |
| D-003 | Custom ABCI application; Cosmos SDK considered | Own transaction/security boundary; more application work | Accepted; no SDK import |
| D-004 | geth v1.16.8 future EVM adapter; other EVM/custom engine considered | Historical baseline; semantics and API compatibility untested | Proposed; tag and Go directive checked |
| D-005 | Pebble v1.1.5 candidate; other KV/SQL considered | Matches geth v1.16.8 requirement; AppHash/atomicity unresolved | Proposed; metadata checked, no storage code |
| D-006 | Sequential execution first; parallel first considered | Isolates hybrid cost and correctness; throughput limited initially | Accepted constraint; no EVM exists |
| D-007 | Pluto-owned canonical envelope; geth transaction changes/standard raw RPC considered | Hybrid proof needs explicit representation; wallet compatibility unresolved | Ownership accepted, format provisional; no encoding exists |
| D-008 | Exact pins when used; floating/unused imports considered | Reproducible build without false compatibility claim; future candidate migration may be needed | Accepted; current module has no third-party requires |
| D-009 | Versioned manifest/report/evidence; informal notes considered | Supports independent review; manual metadata may drift | Accepted; Phase 01 records created |
| D-010 | CIRCL v1.6.4 candidate; other libraries/custom primitive considered | Documented ML-DSA-44 API with local-Go-compatible module directive; FIPS mode/vectors still untested | Proposed; module metadata/docs checked, no crypto import |

The target components, ownership boundaries, ABCI lifecycle, determinism
rules, data model, AppHash uncertainty and three diagrams are in
[`ARCHITECTURE.md`](../ARCHITECTURE.md). Framework statements are sourced to
the pinned CometBFT v1.0.0 specification; Pluto internals remain proposed.

## 4. Repository Structure

Actual source tree at report time, excluding `.git/` internals and ignored
`build/plutod`:

```text
pluto/
├── .github/workflows/verify.yml
├── .gitignore
├── AGENTS.md
├── Makefile
├── README.md
├── go.mod
├── cmd/plutod/main.go
├── cmd/plutod/main_test.go
├── internal/version/version.go
└── research/
    ├── ARCHITECTURE.md
    ├── DECISIONS.md
    ├── MASTER_PLAN.md
    ├── PHASE_STATUS.md
    ├── PHASE_REPORTS/PHASE_01_REPORT.md
    ├── config/manifest.yaml
    └── evidence/phase_01_commands.txt
```

`cmd/plutod` owns CLI dispatch; `internal/version` owns the declared
application version and build revision default. `research/` owns governance
and evidence. `.github` describes an equivalent hosted check but was not run
because there is no remote. No empty future packages were created. `go.sum`
does not exist because the executable uses no third-party modules.

## 5. Implemented Functionality

| Functionality | Files / components | Purpose | Test coverage |
|---|---|---|---|
| Default metadata/status and help | `cmd/plutod/main.go`, `run`, `usage` | State honestly that runtime is absent; show commands | `TestRun/default_status`, `help`; executable launch |
| Version/build metadata | `internal/version/version.go`, CLI `version` path | Print Pluto version, Go runtime, platform and revision (unavailable before first commit) | `TestRun/version`; built executable version command |
| Invalid command handling | `run` | Return exit 2 and clear usage on unknown or extra arguments | `TestRun/unknown_command`, `extra_arguments`; manual `start` check |
| Build and verification | `Makefile`, CI workflow, `.gitignore` | Reproducible build/test/vet/format/module/CLI checks; ignore build output | Actual `make verify` exit 0; `git check-ignore` exit 0 |
| Project governance | `AGENTS.md`, `README.md`, `research/*` | Bound future work and document proposed design | Manual content/command alignment review; no automated prose claim |

## 6. Dependency and Version Report

- **Actual Go:** 1.24.1 linux/amd64; `go.mod` declares Go 1.24.0 and the
  provisional module path `example.com/pluto`. `go list -m` resolved it.
- **Current runtime dependencies:** Go standard library only. No external
  module requirements, `go.sum`, or Docker dependency.
- **CometBFT candidate:** `github.com/cometbft/cometbft@v1.0.0` exists;
  official [tagged go.mod][comet-mod] declares Go 1.23.1. The target
  [ABCI++ methods][abci-methods] and [requirements][abci-requirements]
  were inspected. Exact Pluto API integration is unverified.
- **EVM candidate:** `github.com/ethereum/go-ethereum@v1.16.8` exists;
  [tagged go.mod][geth-mod] declares Go 1.24.0 and requires Pebble v1.1.5
  plus Decred secp256k1/v4 v4.0.1. No geth API or EVM build was tested.
- **Storage candidate:** `github.com/cockroachdb/pebble@v1.1.5` has module
  path `github.com/cockroachdb/pebble` and Go 1.20 directive in its
  [tagged go.mod][pebble-mod]. Persistence/AppHash integration is unverified.
- **ML-DSA candidate:** `github.com/cloudflare/circl@v1.6.4` declares Go
  1.24.0 and contains an [ML-DSA-44 package at the tag][circl-tag-dir].
  [v1.6.5][circl-newer] declares Go 1.25.0, beyond the installed compiler. FIPS mode, test vectors
  and integrated API compatibility remain unverified.

Candidate existence and Go directives were checked from official tagged
sources/Go module metadata. They do **not** establish joint dependency
compatibility, semantic correctness, security suitability or support status.
No candidate was added to `go.mod` because none is imported in Phase 01.
Before Phase 02, decide the CometBFT version after reviewing current fixes,
then pin and compile it in a real integration. Migrate the provisional module
path when a repository identity is approved.

## 7. Verification Evidence

Exact observed outputs and environment commands are in
[`phase_01_commands.txt`](../evidence/phase_01_commands.txt). All required
commands below were executed from the repository directory.

| Command / Test | Exit Code | Result | Evidence |
|---|---:|---|---|
| `go version` | 0 | Go 1.24.1 linux/amd64 | Evidence file, mandatory direct checks |
| `go build ./...` | 0 | Built all current packages; silent success | Evidence file, mandatory direct checks |
| `go test ./...` | 0 | `cmd/plutod` passed; `internal/version` has no tests | Evidence file, mandatory direct checks |
| `go vet ./...` | 0 | Silent success | Evidence file, mandatory direct checks |
| `make verify` | 0 | Build, tests, vet, format check, `go mod verify`, CLI version all succeeded | Evidence file, full workflow |
| CLI execution | 0 | No-arg, help and version launched; invalid `start` correctly exited 2 | Evidence file, executable checks |
| Formatting check: `make fmt-check` | 0 | `gofmt -l cmd internal` found no unformatted source | Evidence file, mandatory direct checks |
| `go list -m` | 0 | `example.com/pluto` | Evidence file, mandatory direct checks |
| `git check-ignore -v build/plutod` | 0 | `/build/` rule matched binary | Evidence file, executable checks |

An uncached `go test -v -count=1 ./...` exited 0: one test with **five passing
subtests**, zero failures and zero skips. `internal/version` reported
`[no test files]`. The GitHub Actions workflow was not executed; there is no
remote. A clean-checkout run was not possible before a first commit. `go mod
verify` succeeded but has no external module checksums to validate yet.

## 8. Acceptance Gate Results

| Gate | Result | Supporting evidence and explanation |
|---|---|---|
| G1 | PASS | New `main` Git repository, `go.mod`, `go list -m` resolves provisional path; no commit/remote yet. |
| G2 | PASS | `go build ./...` and `make build` within `make verify` exited 0. |
| G3 | PASS | `./build/plutod`, `help`, `version` launched and printed expected status/metadata. |
| G4 | PASS | Five behavior subtests passed; invalid CLI path also manually checked. |
| G5 | PASS | `go vet ./...` and non-mutating `make fmt-check` exited 0. |
| G6 | PASS | Architecture defines CometBFT/Pluto/research ownership, lifecycle, determinism, hybrid boundary, AppHash uncertainty and three diagrams. |
| G7 | PASS | D-002, D-004, D-005, D-008, D-010 record candidate versions, official metadata checks and unverified integration. |
| G8 | PASS | Master plan, decisions, phase status and YAML manifest exist. |
| G9 | PASS | README commands match executed Make targets and CLI behavior. |
| G10 | PASS | Source contains only metadata CLI/version; no ABCI, EVM, transaction, hybrid or admission implementation. |
| G11 | PASS | Evidence file records observed commands, exit codes and relevant output; test count reported. |
| G12 | PASS | Status/agent contract explicitly require human approval; Phase 02 remains unapproved. |

## 9. Issues and Risks

- **Outstanding defects in implemented Phase 01 code:** None found by the
  executed checks. CI execution and clean-checkout verification remain
  unobserved, not silently counted as passing.
- **Missing prerequisites for later phases:** No hosted repository identity,
  Git author identity, Docker integration, or chosen final CometBFT release.
  None blocked Phase 01. Docker will matter only if a later deployment design
  requires it; local Go verification did not.
- **Compatibility uncertainty:** CometBFT/geth/Pebble/CIRCL candidates were
  checked separately, never built together. Library APIs, patch support and
  security advisories must be reviewed at their adoption gates.
- **Architecture uncertainty:** AppHash commitment, atomic persistence,
  rejected proposal and invalid decided-block policy, EVM transaction mapping,
  and cryptographic envelope/key binding require specifications and tests.
- **Determinism/security concern:** Later local admission signals must never
  alter consensus-critical validity. Hybrid mode must fail closed.
- **Intentional technical debt:** The `example.com/pluto` path must migrate to
  an approved repository URL; evidence/manifest revision becomes a commit hash
  only after the first authorized commit.

## 10. Scope Compliance

Confirmed by source inventory and build: no functioning blockchain is claimed
or implemented; no CometBFT node/ABCI handler, EVM implementation, transaction
execution, secp256k1 or ML-DSA integration, hybrid encoder, admission control,
consensus change or multi-validator experiment exists. No benchmark result
was generated or fabricated. No Phase 02 work or phase transition occurred.

## 11. Next-Phase Readiness

Phase 02 can build on the Go module, documented ABCI ownership/lifecycle,
decision log, verification workflow and phase gates. Its first design tasks
are to choose and pin the actual CometBFT tag, verify API and Go compatibility,
define a minimal durable state/AppHash contract, and test a real single-node
ABCI++ lifecycle including restart/replay. A repository identity and module
path should be settled before publishing or sharing imports. Human approval
is required before any Phase 02 implementation.

## 12. Final Verdict

```text
PHASE: 01
STATUS: PASS
BASE_REVISION: not_applicable_no_prior_repository
RESULT_REVISION: uncommitted_no_git_identity
BUILD: PASS (go build ./... and make build)
TESTS: PASS (1 test, 5 passing subtests, 0 failed, 0 skipped)
STATIC_ANALYSIS: PASS (go vet ./... and make fmt-check)
DOCUMENTATION: PASS (architecture, decisions, plan, status, manifest, README, agent contract)
ACCEPTANCE_GATES_PASSED: 12/12
KNOWN_BLOCKERS: none_for_phase_01; Phase_02_requires_human_approval
READY_FOR_REVIEW: YES
READY_FOR_PHASE_02: NO
```

[abci-methods]: https://github.com/cometbft/cometbft/blob/v1.0.0/spec/abci/abci%2B%2B_methods.md
[abci-requirements]: https://github.com/cometbft/cometbft/blob/v1.0.0/spec/abci/abci%2B%2B_app_requirements.md
[comet-mod]: https://github.com/cometbft/cometbft/blob/v1.0.0/go.mod
[geth-mod]: https://github.com/ethereum/go-ethereum/blob/v1.16.8/go.mod
[pebble-mod]: https://github.com/cockroachdb/pebble/blob/v1.1.5/go.mod
[circl-mldsa]: https://pkg.go.dev/github.com/cloudflare/circl/sign/mldsa/mldsa44
[circl-newer]: https://github.com/cloudflare/circl/blob/v1.6.5/go.mod
[circl-tag-dir]: https://github.com/cloudflare/circl/tree/v1.6.4/sign/mldsa/mldsa44
