# R06 — equal valid transaction rate, single-host network

Status: **LOCKED DESIGN BEFORE EXECUTION**, 2026-11-13. R05 was disposed
before this lock. This case follows the accepted RQ1 H–CP design in
`FILE02_RQ1_SINGLE_HOST_PLAN.md`; it does not change H0 or H1.

Run twelve independent blocks. Each block uses one newly generated genesis
and logical nonce-zero transfer stream, offered once under each of C, CP, P
and H. Every arm starts a fresh four-validator single-host network. A seeded
initial profile permutation is rotated across blocks; each profile occurs
three times in each order position. An arm has one warm-up and 240 steady
valid transactions at exactly 20 planned tx/s over twelve one-second windows,
five seconds of drain, two seconds of cooldown and no invalid workload.
RPC origin is validator 0 on loopback. Chain ID `pluto-file02-rq1` makes
C/CP/P/H 148/2568/2504/2568 B; CP and H are exact byte matches. The unchanged
S06 caps remain 25 tx/s, 65,536 B/s, 16 outstanding and 60 seconds/run.

Primary endpoint is the per-network mean of 240 accepted, steady-window
application CheckTx service spans at validator 0, joined by SHA-256 to
distinct successful sends. The paired H−CP effect uses one difference per
block and a two-sided percentile bootstrap with 20,000 block resamples,
seed 20261113. A positive RQ1 result requires the 95% CI lower bound above
zero and all primary QC checks to pass. No failed arm is replaced.

Secondary outcomes: actual one-second offered rates, RPC response time,
eventual included count, 100-ms block-scan inclusion upper bounds, app/Comet
process CPU deltas and RSS, accepted mempool bytes and sampled P2P envelope
bytes. Missing streams are `UNAVAILABLE`, never zero. p99 remains descriptive
unless its registered cluster precision test passes (twelve complete
networks/profile, 2,880 samples, optimistic rank coverage ≥95%, two-stage
2,000-draw cluster bootstrap with both CI sides ≤20% of the point estimate).
Exact commit time, sustainable throughput and peer propagation delay are
unavailable from these captures.

QC requires the locked source/config and per-run load plan, all 241 distinct
successful sends and exact-once valid inclusion, 240 steady CheckTx joins,
zero recorder/observer drops, all validators live, same-height BlockID and
AppHash, no safety abort or generator overload, ±5% achieved tx/s and bytes/s
and at least 90% of each planned one-second window. Raw hashes and all
derived artifacts must verify. Although 20 tx/s was delivered in pilot,
**no knee was observed in the safe 2/6/12/20/23/25 tx/s range**. Therefore
the literal R06 phrase “under knee” remains `NOT_OBSERVED_WITHIN_SAFE_RANGE`
even if all bounded-network data pass; do not tick R06 on that basis.
