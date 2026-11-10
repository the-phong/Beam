# S05 telemetry engineering report — 2026-11-10

## Authority, source and boundary

The Owner merged S04 PR #15 into `test-mainet` at
`630f2d8c81398f84073796747108c30ab9dee3c8` and authorized S05 on
`research/s05-observability` in the original `pluto` folder. The four-profile
public corpus SHA-256 remains
`d0f4bfaea4c541ee6f71547b6ea47e44c30aef588704051459e57e04d6478c83`;
the S03 public vectors remain
`1a0a8e0baf1a963d35b8bc357c9abe70221d72bb7fe011cee9eaaba88581dd06`.
Go is 1.24.1 on linux/amd64 (WSL2, Intel Core i7-12700H). CometBFT remains
v0.38.26 and Pebble remains v1.1.5. No accepted B0/B1 or consensus rule
changed. The [dictionary](../../protocol/METRICS_DICTIONARY.md) is the
normative map from each metric to observation point, unit, availability and
quality.

## What is implemented

- Optional bounded `CheckTx` recorder: new/recheck, code, last stage, wire
  length, monotonic service/app-lock durations and separately completed secp,
  ML and state spans. OFF invokes the unchanged evaluator. Neither timing nor
  capacity influences code, state or proposal validity.
- Local controlled RPC proxy and client wrapper: decoded byte length,
  bounded response reason, client RPC and proxy round-trip durations. A cache
  hit is classified before ABCI and creates no application event. The proxy
  does not stand for Comet's true RPC receive timestamp.
- Independent sampler: exact Comet accepted-list count/bytes from
  `num_unconfirmed_txs`; native failed/resource-rejected/recheck counters and
  encoded P2P mempool-channel `0x30` send/receive byte counters; separate
  process CPU/RSS samples from `/proc`. Missing series are `null`.
- `pluto.telemetry.v1` JSONL schema, bounded event buffers, explicit drop
  counters and localhost-only HTTP endpoints. Events contain no raw tx,
  account, key, peer ID, case ID or scenario label.

## Verification ledger

| Command | Result and retained private log SHA-256 |
|---|---|
| `make verify` | exit 0; build, Go tests, vet, format, modules and version; `17466d740145ede6d9c4061d50975bd3b99f9f5ec69b02d94cf70971ca22d733` |
| `go test -race -count=1 ./internal/telemetry ./internal/profiles/...` | exit 0; `abc53fbc88c8c77ded509bcdb8d6d1fb905e44cc3a64c26c19b1a44b8da5d008` |
| `make test-s04-4v` | exit 0; OFF, four profiles and 4 distinct validator identities each; `df89bb4bff8ee4bb98fb5a1308b570eed9fed836d447075da4e35c0351cf9c82` |
| `make test-s05-telemetry` | exit 0; ON, four profiles, real native mempool/P2P/process readings, controlled proxy classification; `719199863e825fd28e0fe5a8a357f164c2b31d1a61387198c8ceb521f7848993` |
| `go test ./internal/profiles/app -run '^$' -bench '^BenchmarkS05CheckTx$' -benchtime=500ms -count=5 -benchmem` | exit 0; private output `f2e1133378d187777e721dd4434302782c04ace34a665b02d749ca5ac551cddb` |
| `go test ./internal/profiles/app -run '^$' -bench '^BenchmarkS05CheckTx/H/ON$' -benchtime=2s -cpuprofile ...` and `go tool pprof -top` | exit 0; private CPU profile `6ffb9510f29db5ebe966a4d4d08b16c1424e4c654820f09fdc35daf6934b9c7b`, top summary `aaf9218d550a83139ed931452a3559f5df39254f9bfbddab7a187da63ef4c778` |

The OFF and ON four-validator runs each committed the same logical transfer
at height 3, agreed within each profile, reached sender `93/1` and receiver
`7/0`, and had the identical post-transfer AppHash
`2C4252A230A13A397BE9A094097BB90513E5F4ACB46BF76E1123A4D4E528E4DC`.
The 34-case direct oracle compared ON/OFF CheckTx codes, stages and committed
state for New and Recheck. A real malformed RPC submission returned a
nonzero application code while accepted mempool count/bytes stayed zero.
The real replay was classified as `duplicate_cache`. The sampler found
nonzero mempool-channel relay bytes after the valid submission. A synthetic
Comet RPC fixture independently checked positive occupancy `2` tx /
`2575` bytes while failed and resource-rejected counters remained separate.
The real four-validator test did not force a positive occupancy sample before
the transaction committed, so it does not estimate peak occupancy.

The five-repeat local microbenchmark medians were CP OFF 127,863 ns/op,
CP ON 129,580 ns/op (+1.34%); H OFF 223,296 ns/op, H ON 225,660 ns/op
(+1.06%). ON added about 112 bytes and two allocations per H operation;
about 112 bytes and two allocations per CP operation. These are descriptive
local microbenchmarks with fixed sequential order and WSL2 scheduler noise,
not run-level overhead confidence intervals or RQ1 causal results. The H ON
CPU profile sampled secp field multiplication and squaring, SHA-256, and
ML-DSA Keccak/polynomial work; sampled process CPU cannot be called PQC CPU.

The first real-network S05 test run exited 1 because its assertion required
Comet's `failed_txs` series on all validators. Comet emits that series only
after a local failure. The assertion was corrected to require it only on the
validator that received the invalid probe; all four profile subtests then
passed. This is a test expectation correction, not a protocol change.

## Gate and remaining work

S05.2, S05.3 and S05.5 have bounded evidence. S05.1 remains open because
the pinned Comet build exposes no exact P2P transaction ingress count/bytes
and no CheckTx enqueue timestamp. S05.4 remains open because true Comet
queue wait cannot be separated from RPC/proxy and application service by
these capture points. The dictionary explicitly records `NOT MEASURED` for
those fields and for P2P propagation latency/channel-specific queue. The
S05 Gate is **NOT OK** until the missing required capture points are
implemented and independently checked. S06 and loaded RQ1–RQ3 experiments
remain closed. No DoS, multi-host timing or resource-effect claim follows.

Private raw logs and pprof artifacts are under ignored
`research/artifacts/s05/`; only their hashes and bounded summaries are
published. The local D026 directories and preexisting refactor stash are
untouched.
