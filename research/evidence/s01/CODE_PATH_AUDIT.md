# M01-02 — transaction path audit

Source baseline: `0bdd869f2c6b3ed59b0ae825954d8667b3c7481e`;
CometBFT module: `github.com/cometbft/cometbft v0.38.26` from `go.mod`.
These code paths describe the CList mempool used by the local node. Exact
capacity limits and runtime telemetry remain separate M06 work.

| Observation boundary | Exact code path | What the code establishes | Dynamic evidence |
|---|---|---|---|
| RPC submit | Comet `rpc/core/mempool.go`: `BroadcastTxAsync`, `BroadcastTxSync`, `BroadcastTxCommit` call `Mempool.CheckTx` | RPC timing and return type depend on chosen endpoint; `broadcast_tx_sync` waits for CheckTx, `broadcast_tx_commit` also waits for inclusion. | B1 four-validator test uses commit endpoint, received bad-ML code and one valid inclusion in each of three fresh networks. |
| P2P ingress | Comet `mempool/reactor.go`: `Reactor.Receive` obtains peer ID and calls `mempool.CheckTx` | Peer-originated transactions share mempool admission but retain peer sender metadata in Comet. | New no-quorum test observes a valid tx in peer 1's accepted mempool after RPC admission only on node 0, before any block; the remaining path must be P2P mempool relay. Direct raw peer injection and timing are later M06/M07 work. |
| Pre-admission | Comet `mempool/clist_mempool.go`: `CListMempool.CheckTx` calls `isFull`, checks `MaxTxBytes`, optional `preCheck`, app connection, then cache push | A full pool, oversized tx, precheck failure or duplicate can stop before Pluto `CheckTx`; those must not be charged to crypto verification. | Source-verified only; full/oversize/cache thresholds still need controlled runtime probes. |
| Pluto admission | `internal/b0app/app.go:CheckTx`, `internal/b1app/app.go:CheckTx`; B1 calls `b1state.EvaluateTrusted` | Reads committed state; B1 stages are parse/chain/amount, both keys and binding, ECDSA, ML-DSA, then state. | `TestB1HybridFourValidator` verified bad ML rejection and valid CheckTx/commit; direct unit tests cover the code taxonomy. |
| Accepted mempool | Comet `mempool/clist_mempool.go:resCbFirstTime` calls `addTx` only when `ResponseCheckTx.Code == 0` and postcheck passes; rejected tx may be removed from cache | An invalid CheckTx response does not itself occupy accepted mempool bytes or relay list. Cache retention depends on `KeepInvalidTxsInCache`. | New no-quorum test observes exactly one valid tx in peer 1's accepted list; a distinct bad-ML tx returns code 13 and does not appear there. Capacity curves remain M06 work. |
| Relay | Comet `mempool/reactor.go:broadcastTxRoutine` iterates accepted mempool list and sends `MempoolChannel` envelopes to peers except sender | Relay of accepted tx is distinct from consensus block propagation. | New no-quorum test confirms relay to peer mempool at height zero, before consensus inclusion. |
| Proposal | `internal/b0app/app.go:PrepareProposal/ProcessProposal`, `internal/b1app/app.go:PrepareProposal/ProcessProposal`; Comet `state/execution.go:ProcessProposal` | Proposal candidate state is independently evaluated, without assuming every validator admitted through local CheckTx. | B1 test directly submits invalid proposals to all four persisted apps and observes rejection/state neutrality. |
| Finalization and commit | `internal/b1app/app.go:FinalizeBlock/Commit`; Comet `state/execution.go:ApplyBlock/Commit` | `FinalizeBlock` prepares pending state and returns AppHash/tx codes; `Commit` performs the synced Pebble write. Comet then updates mempool/rechecks. | Three fresh B1 networks compared ordered block, state, supply and AppHash after commit/restart. |

Local command: `go test -tags 'integration fourval' ./test/integration -run
'^TestB1HybridFourValidator$' -timeout 900s -count=1 -v` exited 0:
three subtests passed in 67.00 s total. Output was not copied into the public
artifact because it includes raw signed transactions and temporary paths.
`go test -tags 'integration fourval' ./test/integration -run
'^TestResearchV2M01P2PRelayBoundary$' -timeout 120s -count=1 -v` exited 0
in 1.86 s; a valid tx reached the peer's accepted mempool before any block,
and a bad ML proof returned code 13 without joining that list. This new test
uses two of four localhost validators, deliberately below quorum, and
isolated temporary homes. Its proof covers the RPC-to-peer relay path, not
distributed external-peer ingress or relay throughput under load.
The static dependency files above are the installed source for the exact
`v0.38.26` module; no CometBFT code was modified.
