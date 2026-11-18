# Build and reproduce Beam/Pluto

Run commands from the repository root on Linux/WSL2 with Go 1.24.x, Python 3 and GNU Make. Use only disposable localhost node homes and networks. `go.mod` pins the Go dependencies; first download requires network access. `matplotlib` is optional for exact PNG regeneration and `cryptography` is needed for the independent Python B0 vector oracles.

## Build and validate public inputs

```bash
make build
make verify
make test-file02-common
python3 experiments/analysis/file02_section2.py validate-registry
make corpus-s04-verify corpus-s06-verify benchmark-corpus benchmark-contention-corpus
```

`make build` places `plutod`, the pinned CometBFT binary, research app, generator and observers in ignored `build/`. `make verify` runs Go build/tests/vet/format/module checks. `make test-file02-common` runs the Python analyzer/schema/integrity suite. To run the bounded real-node regressions, use `make test-integration`, `make test-integration-4v`, `make test-integration-b1-4v`, `make test-s04-4v`, `make test-s05-telemetry` and `make test-s06-pilot`. These spawn local processes and may take minutes. They are correctness/QA checks, not replacements for historical research runs.

## Start a local node

```bash
./build/plutod init --home .pluto --chain-id pluto-dev-1
./build/plutod start --home .pluto
```

The compatibility mode rejects transactions. `plutod init-b0` takes an accepted canonical Genesis V2 binary; a public fixture can be written from `tests/fixtures/b0_state_vectors.json` using its `genesis_wire_hex`. `plutod init-b0-4v` and `init-b1-4v` construct controlled four-validator homes; inspect `./build/plutod help` for current flags. The node binds localhost ports by default. Keep generated keys, node homes and raw logs out of Git. The CLI does not expose the controlled C/CP/P/H profile application as a normal node mode.

## Run an isolated public smoke

```bash
make test-s07-schema
make test-s07-smoke
```

The command prints `S07_SESSION` under ignored `research/artifacts/s07/`. It creates fresh localhost networks, verifies outcomes and freezes SHA-256-bound private raw. To verify that session in the same source revision, run `python3 experiments/analysis/s07_orchestrate.py verify <absolute-session-path>`. A failed run remains immutable and excluded; create a new session rather than editing or reusing it. The smoke is not a new RQ estimate.

## Recheck published results

The final effects, confidence intervals and their source/raw identifiers are in [`results/RESOURCE_COST.md`](results/RESOURCE_COST.md), [`results/SERVICE_IMPACT.md`](results/SERVICE_IMPACT.md) and [`results/provenance/INDEX.md`](results/provenance/INDEX.md). Public aggregate/figure hashes can be checked with `sha256sum` against the cited public manifests. Case-specific plans are under `research/protocol/`, and public corpora under `experiments/scenarios/corpus/`.

Full byte-for-byte rederivation of an accepted case requires its **private immutable raw session** and the **source commit locked in that case's plan**. Structural path changes in this refactor do not rewrite old plans or raw. Use ordinary Git history in this same repository to check out that exact source revision, then run that revision's case verifier with the absolute private session path. The `research/artifacts/` directory is Git-ignored and is not shipped in a fresh clone. Without authorized raw access, a reader can verify published aggregate/checksum consistency and inspect plans and analyzers, but must report full raw reproduction as `NOT_RUN`, not PASS. Do not use smoke output to replace locked results or combine excluded attempts with passing sessions.

The historical RQ1b 3-second endpoint, later continuous observation upper bound and X16 probe-conditioned CPU results answer different measurement questions. [Methodology](docs/EXPERIMENT_METHODOLOGY.md), [metrics](docs/METRICS.md) and [limitations](results/LIMITATIONS.md) define their scope.
