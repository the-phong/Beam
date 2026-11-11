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
- S05-only Comet instrumentation: `research/telemetry/comet_overlay.py`
  checks SHA-256 of three files in pinned v0.38.26, makes a disposable copy
  under ignored `build/`, and builds `build/cometbft-s05` with a temporary
  modfile. `Reactor.Receive` counts each decoded P2P transaction and its
  payload bytes before `CheckTx`; the ABCI socket client observes monotonic
  enqueue-to-writer-dequeue wait for New and Recheck requests. The normal
  S04 Comet binary stays separate. This patch changes only metric calls,
  not mempool acceptance, consensus or ABCI payloads.
- `pluto.telemetry.v1` JSONL schema, bounded event buffers, explicit drop
  counters and localhost-only HTTP endpoints. Events contain no raw tx,
  account, key, peer ID, case ID or scenario label.

## Verification ledger

| Command | Result and retained private log SHA-256 |
|---|---|
| `make verify` | exit 0; build, Go tests, vet, format, modules and version; `17466d740145ede6d9c4061d50975bd3b99f9f5ec69b02d94cf70971ca22d733` |
| `make verify` (final S05 source) | exit 0; ignored log SHA-256 `8d34461584783792fff8b009452e96ae029cb44d7c94020257291a27420d4912` |
| `make verify` (scanner identifier follow-up) | exit 0; ignored log SHA-256 `ff43a2c111fd4ed8e89ac47d7a7fa18a9375b53d8849b4616064d471230b35cc` |
| `go test -race -count=1 ./internal/telemetry ./internal/profiles/...` | exit 0; `abc53fbc88c8c77ded509bcdb8d6d1fb905e44cc3a64c26c19b1a44b8da5d008` |
| `go test -race -count=1 ./internal/telemetry ./internal/profiles/...` (final S05 source) | exit 0; ignored log SHA-256 `b5bf5ba14cc9b51fe341fd8fa4b945764123bcb2df76e1c81e72c9d13f52de51` |
| `make test-s04-4v` | exit 0; OFF, four profiles and 4 distinct validator identities each; `df89bb4bff8ee4bb98fb5a1308b570eed9fed836d447075da4e35c0351cf9c82` |
| `make test-s04-4v` (final S05 source) | exit 0; OFF binary, all four profiles; ignored log SHA-256 `7a710afc9ada226d66c6b0a042bf4b9b078a5e97ef50368d17825d46df78a50f` |
| `make test-s05-telemetry` (initial native Comet build) | exit 0; ON, four profiles, real native mempool/P2P/process readings, controlled proxy classification; `719199863e825fd28e0fe5a8a357f164c2b31d1a61387198c8ceb521f7848993` |
| `make build-comet-s05` | exit 0; SHA-checked pinned source and disposable instrumented Comet build; confirmed again by the S05 four-validator test. |
| `make test-s05-telemetry` with instrumented Comet | exit 0; four profiles observed positive decoded P2P ingress and CheckTx socket queue; ignored log SHA-256 `0cb18adfff4b0abd1e2847b3f675088f47e3ecbfa9017194dd7ca8239b375c77` |
| `make test-s05-telemetry` (final S05 source) | exit 0; all four profiles, source-checked Comet build and derived-rate collector; ignored log SHA-256 `dca97061ae25d0cfac256b46f82774eabb9ceb456f005f92a4f33d8d6fa95c36` |
| `go test ./internal/telemetry -run '^$' -bench '^BenchmarkS05CometHooks$' -benchtime=500ms -count=5 -benchmem` | exit 0; isolated Prometheus call cost, ignored log SHA-256 `3d57960a2974d564381448eb811e317fa68f1448e8a10716a42a37b629f36f1b` |
| `go test ./internal/profiles/app -run '^$' -bench '^BenchmarkS05CheckTx$' -benchtime=500ms -count=5 -benchmem` | exit 0; private output `f2e1133378d187777e721dd4434302782c04ace34a665b02d749ca5ac551cddb` |
| `go test ./internal/profiles/app -run '^$' -bench '^BenchmarkS05CheckTx/H/ON$' -benchtime=2s -cpuprofile ...` and `go tool pprof -top` | exit 0; private CPU profile `6ffb9510f29db5ebe966a4d4d08b16c1424e4c654820f09fdc35daf6934b9c7b`, top summary `aaf9218d550a83139ed931452a3559f5df39254f9bfbddab7a187da63ef4c778` |
| `./build/gitleaks git --no-banner --redact=100 --log-opts='origin/test-mainet..HEAD' .` | exit 0 after an exact-fingerprint false-positive correction; ignored log SHA-256 `9f12a886d21e482d0bc51e124c137aef18aa10a673f3c4566b4eef4bfbd76695` |

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

The instrumented four-validator test additionally requires a positive decoded
P2P ingress transaction count and payload byte total, and a positive CheckTx
socket queue count and duration sum on every profile run. These aggregate
Comet metrics are distinct from controlled RPC ingress. The queue span ends
when the socket writer dequeues the request; it excludes write/flush,
transport, server dispatch and Pluto's app lock/service. The histogram
cannot be paired with a particular application event, so no per-request
end-to-end decomposition is claimed.
The final four profile runs respectively observed aggregate P2P ingress of
`9/1332`, `9/23112`, `8/20032` and `9/23112` transactions/payload bytes
across the four validators. Each observed five socket-queued CheckTx
requests, with total queue seconds `0.000262663`, `0.000293270`,
`0.000349246` and `0.000194179`. These tiny correctness workloads do not
estimate loaded rates or queue distributions.
The collector fixture also checked positive P2P tx/s and bytes/s after a
second same-PID sample, and `null` rates on the first sample or a counter
reset. Each runtime sample has start/completion UTC and monotonic sample
duration; a rate uses the observer's monotonic elapsed time between complete
samples and an unchanged Comet PID start tick.

The five-repeat local microbenchmark medians were CP OFF 127,863 ns/op,
CP ON 129,580 ns/op (+1.34%); H OFF 223,296 ns/op, H ON 225,660 ns/op
(+1.06%). ON added about 112 bytes and two allocations per H operation;
about 112 bytes and two allocations per CP operation. These are descriptive
local microbenchmarks with fixed sequential order and WSL2 scheduler noise,
not run-level overhead confidence intervals or RQ1 causal results. The H ON
CPU profile sampled secp field multiplication and squaring, SHA-256, and
ML-DSA Keccak/polynomial work; sampled process CPU cannot be called PQC CPU.
The isolated Prometheus hook benchmark medians were 11.51 ns/op for the two
P2P counter calls and 26.88 ns/op for a queue histogram observation, with
0 allocations. It excludes the timer call, reactor loop, socket transport
and node scheduler, so it is not whole-node overhead.

The first real-network S05 test run exited 1 because its assertion required
Comet's `failed_txs` series on all validators. Comet emits that series only
after a local failure. The assertion was corrected to require it only on the
validator that received the invalid probe; all four profile subtests then
passed. This is a test expectation correction, not a protocol change.

The first CI scan on commit `7988130` exited 1 with five Gitleaks
`vault-service-token` findings. Local inspection of the redacted finding
metadata and source showed the scanner matched only a long Go P2P rate field
access, including two matches on one test line; no
credential or value was present. The identifier was shortened without
changing the JSON schema. `.gitleaksignore` contains only the four unique
fingerprints from that immutable historical commit; the full proposed-commit
scan then exited 0 locally. The exception does not suppress other rules,
files, lines or future commits.

## Gate and remaining work

S05.1–S05.5 have bounded local technical evidence. The metric dictionary
defines each signal's observation point, unit, first availability and quality.
The S05 technical gate is **locally OK, pending independent review and Owner
acceptance**. Total Comet enqueue-to-app wait, P2P propagation latency and
channel-specific relay queue remain `NOT MEASURED`; the scoped socket queue
component and decoded P2P ingress are measured directly. S06 and loaded
RQ1–RQ3 experiments remain closed until a separate Owner transition. No
DoS, multi-host timing or resource-effect claim follows.

Private raw logs and pprof artifacts are under ignored
`research/artifacts/s05/`; only their hashes and bounded summaries are
published. The local D026 directories and preexisting refactor stash are
untouched.
