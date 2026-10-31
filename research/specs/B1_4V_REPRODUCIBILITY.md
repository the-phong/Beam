# Phase 04 B1 four-validator reproducibility

Status: implementation candidate. Run from the repository root with Go 1.24 and the pinned module graph. `make test-integration-b1-4v` builds the repository's Pluto and CometBFT v0.38.26 binaries, then runs all B1 four-validator integration cases with a 900-second Go test timeout. GitHub Actions runs the same target with an explicit 20-minute step timeout. The test harness creates fresh, isolated temporary roots; allocates unique localhost RPC, P2P and ABCI ports; generates four separate Comet signing and P2P keys; constructs one Genesis V3; initializes and verifies four independent B1 H0 snapshots; and starts four real P2P-connected Comet processes. It owns and terminates only the processes it created.

```bash
make verify
make test-integration
make test-integration-4v
PLUTO_4V_ARTIFACTS="$PWD/research/evidence/phase_04/raw/4v" make test-integration-b1-4v
```

`PLUTO_4V_ARTIFACTS` is optional. It retains application, Comet and callback-marker logs under an ignored local `raw/` directory; do not commit that directory. The test output itself contains public signed transaction bytes and exact committed heights/hashes. The harness signs with ephemeral in-memory test identities and does not write transaction private keys. The generated validator homes exist only in Go's temporary test directory and are removed by the test framework; do not copy them to a public artifact. The test compares block hashes, ordered committed transaction bytes and AppHash at matching heights, as well as sender/receiver balances, nonces and total supply. It stops one validator, finalizes a hybrid transfer with the remaining three, restarts the fourth, and repeats on three independently generated networks. The separate process-crash test blocks the B1 application at pre-write and post-sync Commit callbacks, sends SIGKILL only to that application process, inspects B/C/A and accounts, and restarts it to check one-time effects. Missing/corrupt H0, foreign B0 state, conflicting genesis and inconsistent validator key artifacts must refuse start.

For an isolated manual network, first generate a *fresh* B1 Genesis V3 binary with explicitly provisioned public B1 account bindings and retain all transaction signing private keys outside the repository. Then run:

```bash
make build
./build/plutod init-b1-4v --root /absolute/path/to/new-disposable-root --chain-id pluto-b1-v1 --genesis-file /absolute/path/to/genesis-v3.bin --rpc-base-port 26657 --p2p-base-port 26656 --abci-base-port 26658
./build/plutod start --home /absolute/path/to/new-disposable-root/validator-0
```

Start validators 1–3 in separate terminals with their own `--home`. The `init-b1-4v` root must not already exist; it refuses implicit reinitialization and retains an interrupted root for inspection. Each home keeps its own validator key, P2P key, last-sign state, Comet data and B1 Pebble store. The CLI accepts only localhost ports and canonical B1 Genesis V3. It does not expose a private-key import, rotation, migration or destructive recovery command. Do not copy a validator signing key or last-sign state between homes or reuse it for a conflicting chain.

For deterministic protocol verification without any private key, inspect `internal/hybrid/tx/testdata/hybrid_public_vector.json` and run `go test -count=1 ./internal/hybrid/state ./internal/hybrid/tx`. That vector is public verification evidence, not a reusable signing identity. Raw process logs and temporary homes must be handled as local research artifacts. Four local processes share one host, storage stack and power supply; the suite is a correctness and crash-stop experiment, not a multi-host or Byzantine test.
