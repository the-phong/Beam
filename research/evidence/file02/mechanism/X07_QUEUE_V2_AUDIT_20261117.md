# X07 queue-path rework audit — ABCI and P2P trajectory

Engineering disposition: **PASS for X07's bounded queue timeline contract**,
subject to independent Gate review. The accepted topology remains four
validators on one physical host with RPC origin at validator 0. The
source-hash-checked disposable Comet v0.38.26 overlay added private
transaction/envelope-hash markers at ABCI server read/mutex/result and the
per-peer MConnection send queue. The application logic, codec, state,
consensus rules and S06 caps were unchanged. Source/plan commit `07dc11b`;
private immutable raw manifest SHA-256
`8a2082525610b5615bed6753ef38add2a79ab41d368b459cbf55b8dd68099fc2`.

Eight matched ON/OFF blocks (16 fresh networks; four OFF→ON, four ON→OFF)
used the same signed 241 H transactions and genesis within each block at
20 tx/s, 51,360 B/s. The independent tool returned
`X07_V2_VERIFY=OK QC=PASS runs=16 pairs=8`. Every run committed 73 valid
H exactly once and rejected 168 late-bad-ML H. Four-validator AppHash,
source/tree, config, actual wire bytes, recorder/PID identity, zero drops
and raw hash passed. In each ON run the verifier joined 241/241 new RPC
ingress paths through client queue, ABCI server, app result and CheckTx
callback. It independently re-encoded the pinned protobuf envelope from
raw transaction bytes, matched 219/219 successful validator-0 per-peer
queue trajectories, and confirmed accepted valid hashes at validators 1–3.
Across eight ON networks this is 1,752 verified P2P queue sends; no
invalid transaction entered the accepted/relay path. Raw transaction bytes
and keys remain private; `X07_QUEUE_V2_AGGREGATE_20261117.json` is the
publication-safe derived aggregate (SHA-256
`d7e500a82d42c4b8d255fca50678985d150560480708c21a20d08e13e1ff171c`).
The final independent verifier added protobuf re-encoding and run-level
segment CIs after the raw was frozen; it changed no source binary, plan or
raw byte, and `verify` reproduces the stored derived result.

| Trace ON segment | Per-run mean across 8 networks | 95% run bootstrap CI | Meaning |
| --- | ---: | ---: | --- |
| ABCI client queue attempt→writer dequeue | 40.74 µs | [38.84,42.59] µs | Includes possible blocking before channel acceptance |
| Client dequeue→ABCI server read | 112.81 µs | [108.08,120.31] µs | Serialization, socket transfer and dispatch; not pure queue |
| ABCI server mutex attempt→acquisition | 6.76 µs | [5.36,8.60] µs | Server-side serialized application wait |
| App method entry→app lock | 0.803 µs | [0.769,0.838] µs | Separate application lock wait |
| App service | 364.43 µs | [355.97,372.54] µs | CheckTx wall work, not CPU |
| Server app return→Comet callback | 104.60 µs | [100.47,109.59] µs | Response channel/socket/client work; not pure queue |
| Accepted mempool insertion→removal | 1,093.72 ms | [1,081.92,1,108.20] ms | Valid transactions only; includes proposal/commit scheduling |
| P2P queue attempt→dequeue | 49.14 µs | [45.62,53.25] µs | Per peer; includes possible send blocking |

The P2P accepted marker was not observed after dequeue in these 1,752
sends, but the implementation allows that scheduling race. The table
therefore reports attempt→dequeue as a conservative interval, not a pure
queue residence. All cross-process intervals use the same Linux host
`CLOCK_MONOTONIC` domain and are only valid on this single host; timestamp
placement and probe work contribute uncertainty. Accepted-mempool residence
overlaps peer relay and block scheduling. Their intervals cannot be added
into a transaction-wide queue time. `full_transaction_queue_ns=null` with
reason `PARALLEL_RELAY_AND_CONSENSUS_NO_SINGLE_ADDITIVE_QUEUE_SCALAR` is the
correct contract output. Response-buffer and proposal/consensus internal
queue-only portions remain unavailable; their composite boundaries are
reported above rather than mislabeled as queue wait.

The paired **instrumentation package** ON−OFF median producer RPC effect
was +0.0961 ms, 95% CI [+0.0856,+0.1085] ms. Four-app CPU was +0.0069
s/run [−0.0561,+0.0665]; four-Comet CPU −0.1263 s/run
[−0.4150,+0.1188]. Fixed-valid all-four observation upper-bound effect
was −13.64 ms [−42.39,+12.45]. The RPC overhead is measurable, so the
trace ON micro-intervals are **probe-conditioned** and are not native
production latency estimates. This QA validates a measurement topology and
correctness, not a new service effect, knee, DoS or multi-host timing claim.

The earlier X07 session with unmatched ON/OFF fixtures and the original
single-pair QA remain preserved and excluded from this eight-pair estimate.
No historical raw or accepted RQ1/RQ1b decision was rewritten. X07 is
technically closed only for the checklist's observable queue-path contract;
the unavailable composite queue-only portions stay `null` for future work.
