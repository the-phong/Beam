# Pluto

Pluto is a local CometBFT v0.38.26 research application with three selectable
profiles. The historical Phase 02 profile rejects transactions; B0 executes
classical secp256k1 transfers; B1 executes genesis-bound secp256k1 and
ML-DSA-44 hybrid transfers. B0 and B1 have four-validator localhost test
networks. The Phase 05 evaluation was merged into `test-mainet` at
`0bdd869f2c6b3ed59b0ae825954d8667b3c7481e`. Its bounded results do not
establish DoS, an EVM, multi-host independence, or a useful ML detector.

The [current experimental checklist](research/checklists/01_CHECKLIST_CHUAN_HOA_MO_RONG_REPO_THUC_NGHIEM.md)
tracks a separate RQ1–RQ3 program. S01–S04 are closed after the Owner merged
[PR #15](https://github.com/the-phong/DoS-in-Blockchain-PQC/pull/15).
S05 PR #16 and S06 PR #17 were merged by the Owner. S06 has a bounded
[testbed contract](research/protocol/TESTBED_CONFIG.md) and
[single-host four-validator pilot](research/evidence/s06/TESTBED_REPORT.md)
on the merged `test-mainet` baseline. The C/CP/P/H application has a four-validator localhost
integration harness; it is not a selectable profile in the normal `plutod`
CLI. Checked items link to new evidence. Historical Phase 05 results are not
new RQ1–RQ3 results.

## Directory map

| Path | Purpose |
|---|---|
| [`cmd/`](cmd), [`internal/`](internal), [`test/integration/`](test/integration) | Node and research executables, domain packages and real-node tests. See the [source guide](docs/SOURCE_GUIDE.md). |
| [`docs/`](docs/ARCHITECTURE.md) | Architecture, source guide, transaction and experiment flows, and [refactor mapping](docs/REFACTOR_MAPPING.md). |
| [`research/`](research/README.md) | One home for historical Phase 01–05 records and the current RQ1–RQ3 protocol, checklist, corpus and S01–S06 evidence. Repeated historical raw logs can represent distinct fault cases and are retained for their manifests. |
| `research/artifacts/`, `build/` | Ignored local backups, raw logs, generated keys/homes and build outputs. Rebuild binaries with `make build`. |

Read [architecture](docs/ARCHITECTURE.md), [source guide](docs/SOURCE_GUIDE.md),
[transaction flow](docs/TRANSACTION_FLOW.md) and
[experiment flow](docs/EXPERIMENT_FLOW.md) for the current implementation.

The operator's checklist 01 and checklist 02 also live one directory above
`pluto`. Checklist 01 there is the local progress copy; the tracked copy
under `research/checklists/` travels with the PR. The V1 history and an
exact duplicate of checklist 02 were moved into the ignored local archive
`research/artifacts/private-baseline/checklists/`; the parent checklist 02
remains available. Do not treat that older V1 plan as the current gate.

## Prerequisites and build

On Linux/WSL2, install Go 1.24.x or newer and GNU Make. Network access is
needed on first build to download exact pinned Go modules. Docker is not
required. From this directory:

```bash
make build
./build/plutod version
./build/cometbft version
```

`make build` creates `plutod`, pinned `cometbft`, `profile-observer` and
`profile-ingress` together under ignored `build/`.
`plutod` requires its adjacent pinned `cometbft` executable for `init` and
`start`; do not replace it with an arbitrary system-installed binary.
`make clean` removes only generated binaries, root-level Go test binaries and
Python bytecode caches under `research/benchmarks/`. It leaves node homes,
research evidence and the ignored local archive intact.

## Initialize, run, inspect and restart

Choose a private node home; the default `.pluto` is ignored by Git. Run from
one terminal:

```bash
./build/plutod init --home .pluto --chain-id pluto-dev-1
./build/plutod start --home .pluto
```

`init` creates CometBFT genesis, validator/node keys, local ports and Pluto
metadata. It refuses an existing home and never resets one. The default RPC,
P2P and ABCI ports are 26657, 26656 and 26658, bound to `127.0.0.1`; override
all three at init with `--rpc-port`, `--p2p-port`, `--abci-port` for another
local test. Empty blocks are configured every second. Protect files under
`.pluto/config/`, especially private validator and node keys; never publish
them or expose the development RPC/validator publicly.
Protect `config.toml` as well: startup does not recheck its listener and proxy
addresses against the localhost metadata written by `init`.

From another terminal, inspect real CometBFT RPC responses:

```bash
curl -fsS http://127.0.0.1:26657/status
curl -fsS 'http://127.0.0.1:26657/block?height=2'
curl -fsS http://127.0.0.1:26657/abci_info
```

Wait for at least height 2. `/status` gives the consensus height; `/abci_info`
gives the latest committed Pluto state height and base64 AppHash. A block
header's `app_hash` corresponds to the **previous** committed state height,
not the same-numbered `/abci_info` height. Stop with Ctrl-C; then inspect the
durable Pluto record and restart from the **same home**:

```bash
./build/plutod state --home .pluto
./build/plutod start --home .pluto
```

Do not run `state` while the node is active: Pebble enforces exclusive writable
access. On restart, CometBFT logs its ABCI handshake/replay and continues block
production. If `init` fails because a home exists, use the existing home with
`start` or select a *new* directory; never delete a node directory just to
make the command pass. If startup fails, check local port collisions, the
adjacent CometBFT binary, genesis/Pluto metadata mismatch, database lock and
key-file permissions. Partial initialization is not automatically cleaned up.

## Verification and research records

```bash
go build ./...
go test ./...
go vet ./...
make verify
make test-integration
make test-s04-4v
go test -race ./internal/legacy/app ./internal/legacy/store
```

`make verify` builds both executables, runs unit/ABCI tests, vet, a read-only
format check, module checksum verification and CLI version check.
`make test-integration` runs a bounded real-node test in a dedicated temporary
home: init, RPC-observed blocks, graceful stop, independent state reload,
restart and further blocks. It only cleans up its own processes/directories.
Actual Phase 02 results and limitations are in
[the Phase 02 report](research/PHASE_REPORTS/PHASE_02_REPORT.md), with local
Phase 01 baseline checksums in
[BASELINE_SHA256.txt](research/evidence/phase_02/BASELINE_SHA256.txt).
[Architecture](research/ARCHITECTURE.md), [decisions](research/DECISIONS.md),
[phase status](research/PHASE_STATUS.md) and the
[manifest](research/config/manifest.yaml) are the current governance records.
The [Phase 02R independent review](research/PHASE_REPORTS/PHASE_02R_REPORT.md)
records its historical checks and limitations. The historical 14-phase
[master plan](research/MASTER_PLAN.md) placed minimal B0-Core before the
Phase 04 EVM decision; D-028 recorded EVM NO-GO for the B1 experiment.
Phase 03B is completed, Controller-accepted and merged through PR #4 at
`8be07c87abed73763d40252e99bbf0573da487b4`; the Owner accepted D-023.
Phase 03C was merged into `test-mainet` at
`b56c81ccce7ea4a0e7f52053c7459c4601c15b38`. D-024 accepts its
single-record V2 Pebble layout only as an experimental, correctness-oriented
baseline. The Owner accepted D-025 for Phase 03D's Genesis V2 transport.
The Owner closed the D-025 checkpoint and subsequently formally accepted
Phase 03D after Controller technical acceptance of D-026 and its source
review corrections. The explicit B0 profile executes signed transfers and
supports latest committed-state account queries. Local real-node replay and
software-fault tests cover selected failure boundaries; physical power loss
and DoS resistance remain unverified. Subsequent phases added localhost
four-validator B0/B1 tests and hybrid authentication. The
[Phase 03D report](research/PHASE_REPORTS/PHASE_03D_REPORT.md) records the
earlier implementation evidence. See the earlier
[risk addendum](research/evidence/phase_03c/RISK_AND_ACCEPTANCE_ADDENDUM.md)
for the original M1–M12 obligations.
Phase 03E, 04 and 05 have since merged into `test-mainet`. The
[Phase 05 decision](research/PHASE_REPORTS/PHASE_05_RESEARCH_DECISION.md)
recommends no new DoS mechanism on its bounded evidence and explicitly leaves
broader susceptibility unresolved. D-018's branch and Owner-controlled merge
rules remain in force. The Phase 02 local-only Git exception is historical.

## Phase 03A accepted protocol specification

The owner authorized and accepted the specification-only Phase 03A B0 V1
protocol and decisions D-020–D-022 after Controller rework. The accepted
[B0 transaction format](research/specs/B0_CLASSICAL_TRANSACTION_PROTOCOL_V1.md),
[account transitions and AppHash V2](research/specs/B0_STATE_TRANSITION_SPEC_V1.md),
[ABCI validity policy](research/specs/B0_ABCI_VALIDITY_SPEC_V1.md) and
[test-vector plan](research/specs/B0_TEST_VECTOR_PLAN_V1.md) are protocol and
test requirements. Phase 03A PR #3 was merged into `test-mainet` at
`54bf720ac4625c17eaa6101a8f398ef7c0663432`. Phase 03B now implements a
standalone `internal/classical/tx` library; the historical Phase 02 profile still
rejects all transactions. Phase 03B PR #4 is merged. Phase 03C adds a
`internal/classical/state` transition engine and `internal/classical/store` V2 persistence
adapter; neither is connected to the historical Phase 02 ABCI application.
The explicit Phase 03D B0 profile connects them for genesis and classical
transfer execution. See the
[Phase 03C report](research/PHASE_REPORTS/PHASE_03C_REPORT.md) for verification
and limitations.

## Explicit B0 V2 profile (local research only)

`plutod init` selects the historical Phase 02 profile. `plutod init-b0`
explicitly selects the B0 V2 profile and requires an existing
canonical binary Genesis V2 file. For a disposable test chain, the public
nonproduction fixture can be written with:

```bash
python3 -c 'import json,pathlib; v=json.loads(pathlib.Path("research/evidence/phase_03c/state_vectors.json").read_text()); pathlib.Path("/tmp/pluto-b0-genesis.bin").write_bytes(bytes.fromhex(v["genesis_wire_hex"]))'
make build
./build/plutod init-b0 --home .pluto-b0 --chain-id pluto-b0-v1 --genesis-file /tmp/pluto-b0-genesis.bin
./build/plutod start --home .pluto-b0
```

Use a fresh home and protect the generated validator/node keys. The CLI
validates the binary, stores only its canonical JSON/Base64 representation in
CometBFT `genesis.json`, and records a digest in Pluto metadata. On startup,
the B0 application decodes the transport, validates the original binary and
uses a separate `data/pluto-b0-v2` database. Restart never reconstructs a
missing database from genesis. The historical `state` command remains V1-only.
The B0 profile accepts only the Owner-approved signed transaction wire
format. `CheckTx` uses committed state, proposals are checked independently,
and `FinalizeBlock` advances pending state before a synchronized `Commit`.
The account query path is `b0/account/v1` with a 20-byte address, latest
height only and no proof. The public test-only transaction in
`research/evidence/phase_03b/golden_vectors.json` can be submitted on a
fresh fixture chain; its sender begins with 100 units in the example
genesis. The [Phase 03D tests](test/integration/b0_transaction_test.go)
show exact RPC calls, result codes, hash alignment and restart checks.
