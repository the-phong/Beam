# Pluto

Pluto is a local CometBFT ABCI++ research application. It has three explicit
profiles: a transaction-free legacy chain, classical B0 transfers, and B1
transfers that require secp256k1 and ML-DSA-44 signatures. B0 and B1 are
protocol identifiers and remain part of the canonical transaction and state
formats. The project is experimental; it has no EVM or admission optimization.

## Architecture

```text
cmd/plutod                 CLI and node lifecycle
    ↓
internal/node              CometBFT process, localhost ABCI and profile selection
    ↓
internal/legacy            empty-block application and V1 Pebble state
internal/classical         B0 transaction → state → store → ABCI application
internal/hybrid            B1 transaction → state → store → ABCI application
    ↑
internal/experiment        public corpus, correctness oracle and measurements
```

Each active transaction profile has `tx`, `genesis`, `state`, `store` and `app`
packages. The hybrid profile also has `testutil` fixtures. `internal/version`
contains build metadata. `cmd/corpus` verifies or generates public test data;
`research/benchmarks` contains the experiment runners and analyzers;
`test/integration` exercises real localhost nodes. See the
[current architecture](ARCHITECTURE.md) for ownership and data flow.

CometBFT owns consensus, P2P and its mempool. Pluto owns transaction
validation, account state, persistence and AppHash. `CheckTx` filters local
admission; proposal validation independently checks transactions. `Commit`
performs the synced state write. Every profile has separate on-disk records;
node startup checks that a home matches its selected profile.

## Build and verify

Use Linux or WSL2 with Go 1.24.x or newer, Python 3 and GNU Make. The first
build downloads pinned Go modules.

```bash
make build
make verify
make test-integration
make test-integration-4v
make test-integration-b1-4v
python3 -m unittest discover research/benchmarks -p 'test_*.py' -v
```

`make build` creates adjacent `build/plutod` and pinned `build/cometbft`
executables. `make verify` runs Go tests, vet, formatting and module checks.
The integration targets create disposable localhost homes. The four-validator
suites run four processes on one host, so they do not test independent
physical fault domains or Byzantine behavior.

## Run a local node

The legacy empty-block profile needs no transaction genesis:

```bash
./build/plutod init --home .pluto --chain-id pluto-dev-1
./build/plutod start --home .pluto
```

For a disposable classical chain, derive the public canonical Genesis V2
fixture and initialize a fresh home:

```bash
python3 -c 'import json,pathlib; v=json.loads(pathlib.Path("research/evidence/phase_03c/state_vectors.json").read_text()); pathlib.Path("/tmp/pluto-b0-genesis.bin").write_bytes(bytes.fromhex(v["genesis_wire_hex"]))'
./build/plutod init-b0 --home .pluto-b0 --chain-id pluto-b0-v1 --genesis-file /tmp/pluto-b0-genesis.bin
./build/plutod start --home .pluto-b0
```

`init-b0-4v` and `init-b1-4v` initialize fresh four-validator roots from their
respective canonical binary genesis files. `start` runs one initialized home;
operators start each validator separately. RPC, P2P and ABCI bind to localhost
by default. Stop with Ctrl-C before inspecting or restarting a home.
`plutod state` inspects only the stopped legacy profile. Never publish node
homes, validator keys or last-sign state. The default `.pluto` homes, build
outputs, process logs and private raw experiments are ignored by Git.

## Experiments and historical records

The public corpus is in `research/corpus/phase_05/`; the folder name is a
historical dataset identifier. `make benchmark-corpus` verifies it against
the correctness oracle. `make benchmark-smoke` runs a bounded local workload;
`make benchmark-contention-smoke` runs the matched-rate smoke workload.
Reproduction details are in the [experiment guide](research/specs/PHASE_05_REPRODUCIBILITY.md).
Raw runs stay in the ignored `research/evidence/phase_05/raw/` directory.

`research/specs`, `research/PHASE_REPORTS`, `research/evidence` and
`research/decisions` retain accepted specifications and dated engineering
records. Their historical names and evidence paths are preserved. The
[decision record](research/DECISIONS.md), [status](research/PHASE_STATUS.md),
[manifest](research/config/manifest.yaml) and [execution contract](AGENTS.md)
document governance and experimental limits.
