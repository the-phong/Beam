# R07 — matched encoded byte rate with different transaction rates

Status: **LOCKED DESIGN BEFORE R07 EXECUTION**, 2026-11-13. R06 was
disposed as bounded evidence with the under-knee condition unobserved.
COMMON-04 established the feasible schedule and a preregistered ±5%
per-second byte-rate acceptance band. This case uses fresh networks and is
not a relabeling of COMMON-04 QA.

Twelve independent blocks, each with one newly generated same-intent fixture
shared across C/CP/P/H, run four fresh four-validator single-host networks
in a seeded, rotated profile order. RPC origin is validator 0 loopback.
Chain ID `pluto-file02-rq1` gives 148/2568/2504/2568 B per C/CP/P/H valid
transaction. To target 2,568 encoded bytes/s, schedule 17/1/1/1 valid
tx/s respectively; achieved byte schedules must be 2,516/2,568/2,504/2,568
B/s in each of four one-second steady windows, all within ±5% of target.
The maximum relative deviation is 2.49%; this is tolerance-matched, while
CP/H are exactly matched. One warm-up transaction, four steady seconds,
three-second drain and two-second cooldown per arm remain below S06 caps.
No invalid or attack workload is sent.

Per-run COMMON plan, exact endpoint ports, actual node config hashes, source,
genesis, signed corpus and safety config are locked and validated before
load. One observation per network enters a twelve-block paired bootstrap
(20,000 resamples, seed 20261113) for H−CP mean valid steady CheckTx service.
Report the corresponding C/P contrasts as descriptive and confounded by
different transaction rates and canonical sizes. Rate QC requires all
planned sends, exact per-second counts/bytes and the ±5% byte target,
correct inclusion and four-validator oracle, complete joined CheckTx spans,
no recorder/observer drop, safety abort or generator saturation. Failed arms
are excluded without replacement. p99 must be `INSUFFICIENT_PRECISION`
unless the registered cluster gate independently passes; with only four
steady transactions per CP/P/H arm, no p99 inference is expected.

Secondary outcomes are RPC response, block-scan inclusion upper bounds,
sampled app/Comet CPU and RSS, mempool accepted bytes and observed relay
envelope bytes. Eventual inclusion is not steady-state throughput. Exact
commit and peer propagation times are unavailable. No knee, P2P-origin,
multi-host or DoS inference is permitted. A matched-byte result cannot be
read as an equal-transaction-rate comparison.
