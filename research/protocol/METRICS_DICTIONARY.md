# S05 telemetry dictionary — implementation candidate

Status: bounded S05 implementation on `research/s05-observability`. This
dictionary names the observation point, unit, first availability and quality
of every signal. `null` means **NOT MEASURED**; it is never interpreted as
zero. No S05 signal is an RQ1–RQ3 result or a DoS threshold.

## Sources and clocks

The controlled localhost RPC proxy (`cmd/profile-ingress`) receives a complete
GET `broadcast_tx_sync` or `broadcast_tx_commit` request, decodes only its
length, forwards it to the pinned CometBFT v0.38.26 RPC listener and records
the completed response. It is an optional ingress path; direct calls to Comet
bypass it. The application records only `CheckTx` events through a bounded
in-memory `internal/telemetry.Recorder`. The sampler
(`cmd/profile-observer`) reads Comet's `num_unconfirmed_txs` RPC, pinned
Prometheus metrics, and `/proc/<pid>/{stat,statm}` for the distinct Comet and
application processes. The four-validator harness enables Prometheus on
loopback only. Comet's mempool P2P channel is `0x30` in v0.38.26; its peer
byte counters count encoded channel messages, not transaction payload bytes.

All duration measurements use `time.Now`/`time.Since` in one Go process and
therefore use the monotonic component. `ingress_utc` and `sent_utc` mark
proxy arrival and client send; `observed_utc` marks when a result is
**complete and available**. Node samples also include `sample_started_utc`
and `sample_duration_ns` to bound the sequential RPC/Prometheus/process
scrape interval. A detector may only consume `observed_utc`. A
cross-process subtraction is prohibited without clock-offset calibration.
The `/proc` sampler converts cumulative process user+system ticks using
`getconf CLK_TCK`; CPU core equivalents require two samples from the same
PID and `start_ticks`, then `delta(cpu_ns)/delta(monotonic_wall_ns)`. RSS is
resident pages times OS page size. CPU is for the whole process, never PQC
alone. A reset, missing series, failed scrape or process exit yields `null`
and a quality failure, not an invented observation.

## Signals

| Signal | Point, unit, first availability | Quality and interpretation |
|---|---|---|
| Controlled RPC ingress tx/s, bytes/s | Proxy handler receipt of complete HTTP request (`ingress_utc`); count `rpc_proxy` events and sum `wire_bytes` in a nonoverlapping collector window. Available only at `observed_utc`. | Direct for traffic routed through the proxy only; decoded canonical wire bytes, not HTTP bytes. A malformed hex request has unknown wire bytes and `proxy_encoding`. |
| P2P ingress tx/s and transaction bytes/s | Intended Comet mempool reactor receive. | **NOT MEASURED**: pinned Comet metrics expose channel envelope bytes but no per-transaction receive count or exact payload byte total. Do not infer this stream from CheckTx or RPC. |
| Precheck, cache, application rejection | Proxy response: `duplicate_cache`, `mempool_full`, `app_reject`, `proxy_encoding`, `transport_unavailable`, or `precheck_or_rpc_unknown`; native `failed_txs` and `rejected_txs` are cumulative. | Known proxy reasons are direct for controlled RPC only. Native `rejected_txs` combines resource and already-present mempool cases; it is not a cache-specific count. Unclassified errors stay unknown. No raw Comet error text is exported. |
| CheckTx new/recheck count, code, last stage, wire bytes | Application `CheckTx` return, per event. | Direct for callbacks that actually reached Pluto; `source=unknown` because ABCI does not encode RPC versus P2P. A proxy cache hit never creates a CheckTx event. |
| CheckTx service and application lock wait | Application callback entry to recorded result, ns; lock wait from entry to `App.mu.RLock` acquisition. | Direct local monotonic durations, separate fields. Service excludes the application lock wait. Neither duration is Comet enqueue or client RPC latency. |
| Comet CheckTx queue wait | Comet mempool enqueue to application callback entry, ns. | **NOT MEASURED**: pinned Comet exposes no per-request enqueue timestamp or correlation ID at the ABCI boundary. It cannot be obtained by subtracting proxy and service durations. |
| Secp, ML and state spans | Application stage completion, ns, only if the stage was reached. | Direct monotonic durations on the observed `CheckTx` path. The state span covers `ApplyAuthenticatedTrusted`; it does not include proposal, commit or transport. Missing stage is absent, never zero. |
| Accepted mempool count and bytes | Comet `num_unconfirmed_txs` `total` and `total_bytes`, snapshot when RPC returns. | Direct accepted-list snapshot. Rejected and cache-hit transactions are excluded by Comet. This is not peak occupancy between polls or a count of all valid CheckTx responses. |
| Recheck, invalid and resource rejects | Comet Prometheus `mempool_recheck_times`, `mempool_failed_txs`, `mempool_rejected_txs`, cumulative counts after scrape. | Direct native counters when emitted. A missing series remains `null`. Deltas across process restarts are invalid. |
| Relay receive/send bytes | Comet `p2p_peer_{receive,send}_bytes_total` summed only for `chID="0x30"`, cumulative encoded bytes after scrape. | Direct mempool-channel traffic, separated from RPC and consensus gossip. Per-peer labels are deliberately omitted from output. Includes channel framing, not just transaction bytes. |
| P2P propagation latency and mempool-channel queue | Peer send to peer receipt or channel-specific pending queue. | **NOT MEASURED**: Comet supplies no correlated tx timestamps; `peer_pending_send_bytes` has no channel label. |
| Comet and application CPU/RSS | `/proc` cumulative `cpu_ns` and resident `rss_bytes` per PID at sample completion. | Direct Linux process sampling, with `start_ticks` to detect PID reuse. CPU is user+system process cost; a separate Go CPU profile only identifies sampled hotspots. |
| Client RPC round trip | `Submitter.BroadcastSync` request start to full response read, ns at completion. | Direct for the controlled client, separate from proxy round trip and application service. It is not inclusion latency. |
| Proxy round trip | Proxy receipt of complete request to complete Comet RPC response, ns. | Direct for proxied RPC. It contains Comet precheck, queue, ABCI service and network/RPC overhead; none is isolated by subtraction. |
| Drop count | Recorder capacity overflow, proxy queue overflow, or sampler source failure. | Direct counters; `Recorder` and proxy report drops independently. A sampler source failure increments `dropped_samples` and leaves its fields `null`. |

## Schema and privacy

`pluto.telemetry.v1` JSONL records have `kind=check_tx`, `rpc_proxy`,
`rpc_proxy_drops`, `client_rpc`, or `node_sample`. CheckTx contains only
profile, new/recheck type, encoded length, code, stage and durations. Proxy
contains source, encoded length, bounded reason, status and duration. Node
samples contain aggregate counters and process data. No raw transaction,
signature, public/private key, account, transaction hash, peer ID, case ID or
scenario label is written. Exports go to ignored `research/artifacts/` until
independent publication review. Both HTTP endpoints and proxy listener are
loopback restricted. All event buffers are bounded and never block validation
or RPC forwarding; lost samples are explicit.

## Reproduction and limits

Run `make test-s05-telemetry` for real isolated C/CP/P/H four-validator
checks; `go test ./internal/profiles/app -run TestS05CheckTxTelemetryONOFFCorpus
-count=1` for the 34-case ON/OFF oracle; and `go test ./internal/telemetry
-count=1` for schema, loss, parser and localhost tests. Build the two tools
with `make build`. For a manually started isolated network, enable Comet
Prometheus on `127.0.0.1` in each node's configuration and run
`build/profile-observer -rpc http://127.0.0.1:<rpc> -metrics
http://127.0.0.1:<metrics> -comet-pid <pid> -app-pid <pid> -count 0` with
output redirected to an ignored private file. `build/profile-ingress
-listen 127.0.0.1:<proxy> -backend http://127.0.0.1:<rpc>` serves the
controlled RPC path and emits JSONL to stdout. The generic `plutod` B0/B1
runtime does not become a C/CP/P/H node; this S05 path is the existing
isolated four-profile harness. Production setup and load generation belong
to separately authorized stages.

The S05 implementation covers directly observable application, accepted
mempool, relay-envelope and process signals. It does **not** establish true
Comet queue wait or P2P transaction ingress count/bytes. Until those capture
points exist, F3 cannot include P2P transaction rate or propagation latency,
and no queue-service decomposition or loaded RQ1 estimate is permitted.
