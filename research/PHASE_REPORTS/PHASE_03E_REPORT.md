# Phase 03E implementation report — four-validator classical B0

Status: implementation candidate for Independent Controller review. Phase 03E
has not been Controller accepted or Owner closed. The Owner authorized the
complete Phase 03E workflow on 2026-10-25; Phase 04 and later work remain
unauthorized. No merge is authorized for the Coding Agent.

## Source identity and scope

- Repository: `the-phong/DoS-in-Blockchain-PQC`; local path
  `/mnt/d/personal/search&develop/blockchain/system/pluto`.
- Phase 03E branch: `phase/03e-four-validator-bft`; base is the Owner's
  Phase 03D merge in `test-mainet`,
  `c3756843c5e804e3a985d817ab5c83d2dd6798a0`, tree
  `39dd15ed276ffa49297cf659b6dcca9f78d942c2`.
- Accepted Phase 03D source tree:
  `7f7002496f38cea99cc3e0b7137e452523cc0b94`; Phase 03D closeout
  commit `de6ab2267b1feddbaa7e8c3251907bf41028209d` is an ancestor
  of the merge.
- Pre-report Phase 03E staged source/document checkpoint, excluding this
  report and the evidence manifest:
  `5ef7519ee8e9aca17e9654699658b70adf7b49a2`. Its only change from
  the fully passing `post_fix_4v.log` code/test snapshot is one diagnostic
  log call that prints all 20 latency samples; the diagnostic case is rerun
  on this checkpoint. The final commit and tree are reported in the PR and
  Controller handoff.
- The 15-path source SHA-256 manifest is
  `research/evidence/phase_03e/SOURCE_SHA256.txt` (SHA-256
  `f4f235ebc7f80a2374a77b81f1b3cb2b426a0915dce005b4b7d666285020dec8`).
  The verbatim source delta is `SOURCE_REVIEW.patch` (SHA-256
  `90cf65d25ae808b3ecb161972074c844d899d35225ac802a62826fea62f8be35`).
- Dependencies remain Go 1.24.1 observed (`go.mod` minimum 1.24.0),
  CometBFT v0.38.26, Pebble v1.1.5 and secp256k1/v4 v4.4.1. `go.mod`
  and `go.sum` are unchanged.

The changed source is `cmd/plutod/main.go`, `internal/node/b0.go`,
`internal/node/b0_init_verify.go`, new `internal/node/b0_network.go`,
`test/integration/b0_four_validator_test.go`, and `Makefile`. Governance
and reproduction changes are `.gitignore`, `AGENTS.md`,
`research/ARCHITECTURE.md`, `research/DECISIONS.md`,
`research/MASTER_PLAN.md`, `research/PHASE_STATUS.md`,
`research/config/manifest.yaml`,
`research/specs/B0_FOUR_VALIDATOR_INTEGRATION_V1.md`, this report and
`research/evidence/phase_03e/`. There are no changes under `internal/tx`,
`internal/b0app`, `internal/b0state`, `internal/b0store` or
`internal/b0genesis`.

## Architecture and D-026 compatibility

The `init-b0-4v` CLI consumes the unchanged canonical binary Genesis V2.
`InitB0FourValidators` atomically claims one new root, creates four distinct
Comet validator/P2P identities, writes one byte-identical shared genesis
with equal voting power 10 and the D-025 transport, and creates one
independently synced D-024 H0 snapshot per home. Each home has separate
Pluto, CometBFT, Pebble, ABCI, P2P and RPC state. The four Comet processes
connect through local persistent peers. `start` checks for a valid B0
snapshot first, then configuration, genesis membership and signing-key
integrity before opening the live application or starting Comet. D-027 is
the bounded provisioning decision; Controller review remains pending.
The identifier D-027 was unused in the merged base (`git grep D-027 HEAD`
returned no hits) before assignment.
Repeated or interrupted initialization fails closed. Neither the D-024
record bytes nor D-025 transport nor transaction/consensus validity changed.
Phase 02 legacy `init` and runtime remain separate.

The test harness uses temporary homes and retains only harness-owned PIDs.
It starts real `plutod` and pinned `cometbft` processes, submits RPC
transactions, inspects matching-height blocks and ABCI queries, and stops
only its own process groups. For malformed proposals, the harness closes
the real Comet nodes and calls the same `b0app.ProcessProposal` directly on
each persisted application at its own committed state; this tests the
proposal-validation oracle without a mock consensus coordinator. No key or
last-sign file is copied between validators.

## Correctness and fault matrix

The reference transfer is amount 25 from sender 100/nonce 0 to receiver
0/nonce 0. Expected and observed settled state is sender 75/nonce 1,
receiver 25/nonce 0, total supply 100. Empty blocks advance AppHash V2 by
height, so every block hash and AppHash comparison uses one selected height.
Comet block header H+1 carries the application AppHash returned by Commit
at H; crash tests use that mapping explicitly.

| ID | Setup and oracle | Observed result | Verdict |
|---|---|---|---|
| T01 | Fresh root, four unique validator and P2P keys, identical genesis/H0 | Four unique identities, byte-identical genesis, four H0 snapshots | PASS |
| T02 | Four active Comet nodes and votes from all four identities | Commit range through H6 contains votes from all four | PASS |
| T03 | Real signed transfer through RPC/ABCI | Transfer committed H7, Code 0; sender 75/1, receiver 25/0 | PASS |
| T04 | Invalid CheckTx and direct invalid ProcessProposal oracle | Malformed, noncanonical, chain, signature, nonce, balance and replay rejection; all four proposal oracles reject malformed/signature/replay/sequence cases without state mutation | PASS |
| T05 | Matching-height block, AppHash, ordered tx, balances, nonce and supply | All four block H8 hash `4510E5C94BD7F50C736251E8141C2015E1F7BEC79A640D6EED93C09A99AF75FB`, AppHash `6DD82DC6F6FF4A5B60216421AA0F54CF4A971ED566EE4885F17FB1A7AB95DC6D`; H7 contains the single ordered transfer; supply 100 | PASS |
| T06 | Stop one validator; 3/4 add three heights within 60 s | Three fresh runs: H2 to H5 | PASS |
| T07 | Restart stopped validator with preserved key/last-sign state | Fourth node caught up and matched H8 block/AppHash in all runs | PASS |
| T08 | Stop two; 2/4 add zero heights over stable 30 s | Three runs: both remaining nodes H5 to H5 | PASS |
| T09 | Restore third; 3/4 add three heights | Three runs: H5 to H8, then fourth catches up | PASS |
| T10 | Kill application PID only while Comet PID remains alive | Both pre-Commit and post-sync markers reached; app SIGKILL and Comet alive at kill | PASS |
| T11 | Replay at both Commit boundaries, exact-once effect | Matching AppHash(A) at header H=A+1; all four sender 75/1, receiver 25/0 after recovery | PASS |
| T12 | Missing and corrupt H0 | `start` refused before Comet launch with specific snapshot errors | PASS |
| T13 | Wrong validator key/address or conflicting genesis chain | `start` refused before Comet launch with identity/genesis errors | PASS |
| T14 | Fresh network repeats, CLI retry and concurrent root claim | Three fresh identity sets; one exclusive initializer succeeded and one refused; four homes intact | PASS |
| T15 | Phase 02/03D build, unit, integration, fault and race regression | `make verify`, `make test-integration`, targeted D-026, core race, M7/four-validator race and D-026 race all exit 0 | PASS |

In the final `post_fix_raw/` network-transfer logs, validator 0 recorded
three distinct P2P peer starts and the four process logs each identified
their local Comet instance as a validator. The committed-signature oracle
also observed all four distinct validator addresses by H6.

Three fresh-network quorum runs use default Comet consensus timeouts:
proposal 3 s, prevote 1 s, precommit 1 s and commit 1 s, with a 1 s
empty-block interval. The 3/4 threshold is three further heights within
60 s after stabilization. For 2/4, the harness waits 3 s after the second
stop to exclude an in-flight block and then observes 30 s with no new
committed height. These are operational bounds, not formal timing proofs.
All four homes are independently generated on one WSL2 host.

The three quorum runs began with 4/4 at H2, reached H5 with 3/4, held
both remaining nodes at H5 for the 30 s 2/4 window, and reached H8 after
restoring the third node. Fourth-node catch-up matched AppHash
`8304D49CE5F314325FEF3813AA4AB477C98F8335FFB587F822575CF80EEA7C6B`
at H8. Block hashes differ between fresh networks because identities and
genesis times differ, but match among the four nodes at H8 within each run.

The crash tests gate the existing Phase 03D M7 child application just
before Commit and after the synced Commit write. They record app SIGKILL,
Comet PID alive at the marker, Comet blockstore B, Comet consensus state C,
durable B0 application height A and AppHash(A). The harness later stops that
Comet process, restarts the same validator home and compares the block
header at A+1 with AppHash(A) across all four nodes. The account transition
must occur exactly once. A process kill is a software fault; no hardware
power-loss claim follows.

In the final run, pre-Commit interruption recorded B/C/A = 4/3/3 and
AppHash(A) `57434E9BF8B6DF4FE6C14476A65EFE33117EFE7B58291158D5E7ACF567338C5A`;
the matching recovered header at H4 had the same hash on all four nodes.
Post-sync interruption recorded B/C/A = 4/3/4 and AppHash(A)
`44295DA9566BF288667D249E95CD301217ABBF9B76B500F594110FD11F9E2FFA`;
the matching recovered header at H5 had that hash on all four nodes.
Each recovered network had sender 75/nonce 1 and receiver 25/nonce 0.

## Resource diagnostics

The diagnostic workload is 20 sequential valid amount-1 transfers through
`broadcast_tx_commit`; the wall-clock interval starts before the first RPC
request and ends after its twentieth response. Reported latency is RPC
request-to-response wall time, with nearest-rank p50/p95/p99 on 20 samples.
Per-process CPU is `ps %CPU` (lifetime average at sample time), while RSS
is sampled KiB, separately for each Pluto app and Comet child. These are
diagnostic classical B0 observations, not controlled throughput or DoS/PQC
results. The environment is WSL2 Ubuntu, Linux
6.18.33.2-microsoft-standard-WSL2, 20 visible CPUs on an Intel i7-12700H,
19 GiB visible memory. The repository and built binaries are on `/mnt/d`
via 9p, while `testing.T.TempDir` places disposable node homes under
`/tmp` on the WSL2 Linux filesystem in this run. Source/build I/O,
runtime storage and shared host scheduling therefore have different
confounders; these samples do not isolate any one storage effect.

The final source-snapshot run took 26.596 s for 20 sequential transfers,
or 0.752 tx/s. RPC wall latency was p50 1.319 s, p95 1.422 s and p99
1.424 s. All 20 sorted raw latency samples are in `raw_samples.log`.
Pluto app samples were 0.2% lifetime CPU and 29,008–29,396 KiB RSS;
Comet samples were 7.0–7.1% and 45,800–50,876 KiB RSS. Four
nodes agreed on sender 80/nonce 20 and receiver 20/nonce 0 after the
diagnostic workload. These values are one local run with only 20 samples.

## Verification, failures and evidence

The exact commands, exit codes and raw output are in the local Phase 03E
evidence directory and summarized in its publication-safe manifest.

| Command | Exit | Raw log |
|---|---:|---|
| `make test-integration-4v` with `PLUTO_4V_ARTIFACTS` after startup-order fix | 0 | `post_fix_4v.log`, `post_fix_raw/` |
| `go test -tags 'integration fourval' ./test/integration -run '^TestB0FourValidatorDiagnostics$' -count=1 -v -timeout=120s` | 0 | `raw_samples.log`, `raw_samples_raw/` |
| `make verify` after startup-order fix | 0 | `make_verify_final.log` |
| `go build ./...` after startup-order fix | 0 | `go_build_all_final.log` |
| `go test -count=1 ./...` after startup-order fix | 0 | `go_test_all_final.log` |
| `go vet ./...` after startup-order fix | 0 | `go_vet_final.log` |
| `make fmt-check` | 0 | recorded in verification summary |
| `go mod verify` | 0 | recorded in verification summary |
| `make test-integration` after bounded startup-order fix | 0 | `integration_regression_final.log` |
| `go test -tags integration ./internal/node -run '^TestD026' -count=1 -v -timeout=300s` | 0; all reached, no SKIP | `d026_regression.log` |
| `go test -race -count=1 ./internal/b0app ./internal/b0state ./internal/b0store ./internal/tx ./cmd/plutod` | 0 | `race_core.log` |
| Relevant four-validator/M7 `go test -race -tags 'integration fourval'` | 0; 5 M7 asymmetric plus 9 other M7 subtests, no SKIP | `race_integration.log`, `race_integration_raw/` |
| `go test -race -tags integration ./internal/node -run '^TestD026' -count=1 -v -timeout=300s` | 0; no SKIP | `race_d026.log` |
| `go vet -tags 'integration fourval' ./test/integration ./internal/node` | 0 | verification summary |
| `git diff --check` and `git diff --cached --check` | 0 at checkpoint; repeat before commit | recorded in verification summary |

Earlier exploratory runs remain local. `targeted_exploratory.log` first
treated Comet's RPC duplicate transaction cache HTTP 500 as application
CheckTx; the correction calls application CheckTx on a persisted identical
state. `targeted2.log` reused one Go `exec.Cmd` for the CLI retry; it was
replaced with a new process. `accepted_4v.log` overconstrained participation
to all four votes in the latest single commit; the correction verifies all
four identities signing across committed heights. None of these failed
exploratory results is marked PASS. Their logs and process logs are retained.
`targeted_rerun.log` is also retained: it used only the `integration` build
tag after the four-validator file moved to `integration && fourval`, so Go
reported "no tests to run"; it satisfies no acceptance case. All mandatory
four-validator cases in `post_fix_4v.log` actually executed without SKIP.

The first Phase 02/03D `make test-integration` rerun exited 2 in
`TestPinnedCometGenesisTransportAndB0Restart`. The new startup identity
check reported missing last-sign state before the accepted D-026 missing-B0
snapshot error when the entire data directory had been removed. The bounded
correction moved identity/config verification after read-only B0 snapshot
validation and before application open or Comet launch. The exact failed
log is `integration_regression.log`; the targeted corrected rerun
`recovery_order_target.log` exited 0 and explicitly observed the required
missing-B0 refusal. The full corrected integration rerun is recorded
separately as `integration_regression_final.log`.

Raw evidence remains local under `research/evidence/phase_03e/`; only the
sanitized manifest and report are intended for Git publication. The 291
test/fault `.log` and `.blocked` files captured before manifest validation,
including unsuccessful exploratory runs, are enumerated
in `research/evidence/phase_03e/RAW_LOG_SHA256.txt` (SHA-256
`11b3fe1bf59c6a38f9c392debda7b7a9fac27b42aa3b62c8101b64664d188627`).
`sha256sum -c` verified every listed file with exit 0; its own output is
retained locally as `raw_manifest_check.log`. The raw files are ignored by
Git and remain on the local workstation; the public manifest contains
relative paths and digests only.

Historical D-026 evidence remains untouched and untracked in
`research/evidence/phase_03d/d026/` and
`research/evidence/phase_03d/d026_review/` (353 local files together),
along with the local `pluto_d026_acceptance_evidence.tar.gz` archive
(SHA-256 `aef510dc4c6b278494c92295759bc5811877c2e43f3f1a76fb5c9690bfcdc8cc`).

## Limitations, review and Git closeout

This is a single-host, four-process crash-stop correctness baseline. It
does not test Byzantine equivocation, arbitrary network partitions,
independent host failures, physical power loss or production scalability.
PQC/hybrid authentication, EVM and DoS mechanisms are absent. Hosted CI
status is separate from local checks. The source and evidence require
independent Controller review; a Draft PR does not constitute acceptance.

The final commit SHA, final commit tree, remote branch and Draft PR URL are
recorded in the final Controller handoff and GitHub PR metadata; a commit
cannot embed its own SHA in its tracked report.
No merge or Phase 04 work is authorized. STOP for Controller/Owner review.
