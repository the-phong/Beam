# Measurement definitions

A numeric zero means a measured zero. Missing or unsupported quantities are `null` with a reason; detector lead time without a measured event is not zero. The full machine-readable metric registry is [`configs/experiments/file02_metrics.json`](../configs/experiments/file02_metrics.json); source-locked case definitions remain in [`research/protocol/FILE02_METRICS.md`](../research/protocol/FILE02_METRICS.md).

| Measure | Unit and calculation | Capture point | Boundary |
| --- | --- | --- | --- |
| Application CheckTx service wall time | µs/tx; app method entry to return, summarized per run | ABCI application | Includes scheduler/wait effects; not CPU or full client RPC time. |
| Instrumented CheckTx stage wall span | µs/stage invocation; monotonic boundary difference | Application stage markers | Stage populations can differ by rejection path; sum is not necessarily total service. |
| Stage thread CPU | µs/stage invocation; Linux thread CPU clock between markers | Optional X16 probe | Probe increased app CPU; unperturbed crypto-only CPU remains `null`. Not process CPU/N requests. |
| App, Comet, observer and generator CPU | s/run from separate PID counter deltas; utilization in cores = CPU seconds / elapsed seconds | Process sampler | Includes GC/runtime/idle process work; app versus Comet separation is measured, exact relay/commit shares are not. |
| RSS peak | bytes; maximum observed resident set per PID | Process sampler | Sampled peak can miss a brief maximum; summing peaks need not represent a simultaneous total. |
| CheckTx socket subqueue | µs or cumulative s; socket/ABCI boundary timestamps | Comet overlay | Only the observed socket interval; never label it whole transaction queue wait. |
| X07 client/server/app/P2P queue-path intervals | µs or ms; joined transaction/envelope hash markers | Source-checked overlay and app | Attempt→dequeue can include blocking before acceptance. Client dequeue→server read and response return→callback are composites. Parallel relay/consensus paths prevent one additive full-queue scalar (`null`). |
| Accepted mempool occupancy | transactions or encoded bytes at sample | Comet mempool gauge | Rejected transactions are excluded. Accepted residence includes proposal/commit scheduling, not pure queue wait. |
| P2P relay | bytes of observed envelopes, queue attempts/dequeues per peer | Comet overlay/counters | Envelope bytes include protocol overhead; not transaction-only bytes or cross-host propagation latency. |
| Valid-service observation latency | ms; send to first observed inclusion in block index at all four validators | Generator and 100-ms block scanner | Upper bound on inclusion observation; not exact commit/finality timestamp. |
| Timely fraction | fraction of fixed valid H observed by the locked 3,000-ms deadline | Per-run oracle | Ceiling-limited when all are timely; cannot prove latency equivalence. |
| Eventual goodput | committed valid transactions / run's observed duration, tx/s | Block oracle | Not sustainable steady-state capacity. |
| Sustained degradation event | fixed H timely fraction ≤0.90 for two consecutive nonoverlapping 3-s windows with complete follow-up | Prospective X protocol | No such QC-passing event observed; original RQ2 early lead time `null/NO_MEASURED_EVENT`. |
| Paired 95% CI | two-sided run/block bootstrap interval for a preregistered estimator | Analyzer after QC | A transaction within a run is not an independent replicate. Keep absent or underpowered estimates null. |

The accepted historical RQ1b 3-second endpoint and the later continuous observation metric coexist; the later metric does not rewrite the earlier result. For instrument details, see [architecture](ARCHITECTURE.md), [methodology](EXPERIMENT_METHODOLOGY.md) and the [X07 queue-path report](../research/evidence/file02/mechanism/X07_QUEUE_V2_AUDIT_20261117.md).
