# Phase 03E four-validator B0 integration specification

Status: implementation candidate, 2026-10-25. Controller review and Owner
acceptance remain pending. D-020–D-026 stay authoritative for transaction,
state, persistence and single-home initialization behavior.

## Topology and initialization

`plutod init-b0-4v` exclusively claims a fresh network root and creates
`validator-0` through `validator-3`. The pinned CometBFT binary generates one
signing identity, last-sign state and P2P identity in each home. Pluto creates
one canonical Comet genesis with all four public validator keys at power 10,
one genesis time, one chain ID and one D-025 JSON/Base64 application genesis.
The genesis bytes and H0 B0 state are verified separately in every home.
Ports must be distinct local endpoints. Each validator has its own Comet
state, Pebble store, ABCI server, RPC and P2P listener. Persistent peers
connect the four P2P instances. Localhost duplicate IP is permitted solely
for the shared-host test network.

The D-026 fresh-home ownership, synced D-024 snapshot and fail-closed start
policy apply to each home. `start` checks that the local signing key matches
exactly one genesis validator and that the four powers are equal. No signing
key or last-sign state may be copied between homes or chains. Interrupted
network initialization leaves a diagnosable root and requires manual
operator disposition; there is no automatic repair or destructive command.

## Correctness oracle

At a selected committed height, compare the block ID hash, ordered encoded
transactions and Comet header AppHash across all available validators.
Queries for sender and receiver balances and nonces must match after nodes
have caught up. The reference transfer moves 25 of 100 units, yielding
sender 75/nonce 1 and receiver 25/nonce 0, with total supply 100.
CometBFT admission may cache a duplicate transaction before application
CheckTx; direct application CheckTx and ProcessProposal tests exercise
consensus validation at identical committed state. Invalid proposals must
return REJECT without modifying committed state. A recovered validator may
temporarily lag; comparison waits until it reaches the selected height.

With equal powers, 3/4 permits progress and 2/4 cannot form a new commit.
The automated test uses three independent fresh networks. The 3/4 window
requires three new heights within 60 seconds after stabilization. The 2/4
window requires no new height over 30 seconds after a three-second settling
period to exclude an in-flight commit. These are operational thresholds,
not BFT timing proofs.

## Fault and trust model

The harness owns all child PIDs and disposable homes. It kills only its
processes. Application-only SIGKILL cases gate real callbacks before Commit
and after a synced Commit; the Comet process is checked alive at the kill
point and its blockstore, consensus-state and B0 heights are recorded before
recovery. The remaining validators keep normal consensus running. Restart
preserves the validator key and last-sign file. Missing/corrupt H0 and
identity/genesis mismatch must fail before Comet starts.

All four validators share a WSL2 host, filesystem and network stack. The
tests cover crash-stop and software replay only. They do not establish
independent fault domains, Byzantine behavior, arbitrary partitions,
hardware power-loss durability or DoS resistance.

## Reproduction and operator handling

Build with `make build`. Supply the same canonical binary Genesis V2 file
used by `init-b0`; its byte format is unchanged. For a private disposable
network:

```sh
./build/plutod init-b0-4v --root /path/to/new/private/network --genesis-file /path/to/genesis-v2.bin --chain-id pluto-b0-v1 --rpc-base-port 26657 --p2p-base-port 26656 --abci-base-port 26658
./build/plutod start --home /path/to/new/private/network/validator-0
./build/plutod start --home /path/to/new/private/network/validator-1
./build/plutod start --home /path/to/new/private/network/validator-2
./build/plutod start --home /path/to/new/private/network/validator-3
```

Each process needs its own terminal or supervisor. The three base ports
expand by increments of ten for validators 1–3. Keep every home private;
its `config/priv_validator_key.json`, `config/node_key.json` and
`data/priv_validator_state.json` are secret or safety-critical. Do not
reinitialize or copy an existing validator home. On an incomplete home,
preserve its files and logs, inspect genesis and identity offline, then seek
operator review. On an isolated experimental network, create a wholly new
root only after preserving the old root for evidence. Never reuse a signing
identity across conflicting chains.

Run `make test-integration-4v` for the complete disposable-process suite.
Set `PLUTO_4V_ARTIFACTS` to a private local directory to retain raw process
logs. The Go test log contains test setup, height/hash observations, fault
markers, and diagnostic samples. Keep raw logs local; publish only a
reviewed, sanitized manifest with SHA-256 digests.
