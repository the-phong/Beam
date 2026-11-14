# X08 — valid service observation contract

Status: locked for new X runs. The historical RQ1b timely fraction is
`observed_by_3000_ms` from a 100-ms four-validator block scanner. Its
original raw, QC and negative paired effect 0 [0,0] stay unchanged.

For every fixed legitimate valid H transaction in a new X10–X13 run,
record the generator's **actual** socket-write UTC and Linux monotonic
timestamp, planned offset, RPC result, transaction SHA-256, and each
validator's first block-index observation UTC and Linux monotonic stamp.
The four per-validator records must agree on committed height, block ID
and AppHash through the correctness oracle. The all-four observation is
the latest per-validator scan; its monotonic stamp minus actual send is an
**upper bound** on the send-to-inclusion observation interval on this
single host. Its lower bound on exact commit latency is unavailable; an
observed upper bound can be zero or positive due to scheduling. Do not
subtract wall-clock UTC across processes to claim submillisecond latency.

The scanner's nominal poll period is 100 ms, but RPC round trips, block
catch-up and scheduler delays can extend observation lag beyond 100 ms.
Record empirical poll gaps, RPC errors and missing observations. If a
transaction is not observed by the 3-s deadline, its binary timely value
is false; continuous latency is censored/`null` with reason
`NOT_OBSERVED_WITHIN_WINDOW`, never zero. Continue post-window follow-up
to check eventual exact-once correctness separately. A successful RPC
response is not a commit timestamp. `broadcast_tx_commit` in instrumentation
QA is an oracle convenience, not the X workload's send time.

For each independent network, report the fixed-stream mean, p50 and p95
of upper-bound observations only if all planned legitimate transactions
have complete observations or preregistered censoring handling applies.
Use the paired network as the statistical unit and retain the historical
3-s binary result alongside continuous outcomes. The candidate sustained
service event, if ever measured, follows X04's two-window definition;
without it, RQ2 lead time remains `null`. Exact consensus finality time,
cross-host network latency and P2P-origin propagation are unavailable in
this architecture.

QA requirement before X10: a fresh four-validator valid H test must produce
four ordered per-validator monotonic/UTC stamps and one shared block
height/hash, with source committed before execution. This QA does not
estimate a service effect.
