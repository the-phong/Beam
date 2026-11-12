# File 02 T06 bounded generator calibration pilot — locked before execution

Authorization: Owner explicitly allowed a **bounded T06 pilot** on
2026-11-12. This document is a pilot plan for the original Pluto folder and
`research/file02-common-foundation`; it is not authorization for T/R/A/Q/D
confirmatory cases, a safety-cap increase, multi-host traffic or an RQ result.

## Fixed design

- Four real validators on one host; all RPC/P2P endpoints remain on loopback.
- The unchanged `research/config/s06_pilot.json` is used, SHA-256
  `9bc640db8b8875af3f9a5e4f89faa1882e8f4bc10459997d1a891397768cf21a`.
- C, CP, P and H each use fresh network/keys/genesis and 90 pre-funded
  senders. Every signed transaction is unique at nonce zero. One warm-up
  second, four steady seconds and three drain seconds per run.
- Target rates, in increasing safety order: 2, 6, 12, 20 valid tx/s for
  each profile. The highest H wire rate is about 51,460 encoded bytes/s,
  below the pinned 65,536-byte/s hard cap. All rates are below 25 tx/s.
  No invalid workload, attack case or saturation claim is planned.
- The open-loop generator's actual socket writes determine achieved tx/s
  and bytes/s. `broadcast_tx_sync` acceptance is a separate result; direct
  block/state oracle checks every valid transaction on four validators.
- A rate is achieved only when total steady tx/s is within ±5%, every
  nonempty one-second window reaches at least 90% of planned count, every
  request is actually sent, no generator error or safety abort occurs, all
  valid transfers commit exactly once, and same-height BlockID/AppHash and
  account state agree. Failed runs retain private diagnostics and are never
  silently retried under one run ID.
- Generator user/system CPU time, wall duration and peak RSS are recorded.
  The four-validator observer and S06 host guard remain active. Raw is
  frozen under ignored `research/artifacts/file02/` with hashes.

This is a **generator calibration**, not a capacity-knee search. Its
one-run-per-cell observations cannot define a safe knee, scientific
comparison or p99. The preexisting COMMON-06 rank-band rule (sample p99
population rank within 98.5–99.5% at ≥95% coverage) stays fixed. Any
later precision extension must separately count transaction samples and
independent runs, account for clustering/censoring, and withhold p99 if its
registered precision criterion is not met.

## Precision extension locked after calibration, before execution

The separate calibration session at
`research/artifacts/file02/t06-calibration-20261112T141031Z-b8ad1a5f`
(manifest SHA-256
`5c591674c86b69e081c61e1f360f6d538c3e21120461453cd294c8098ff90f81`)
achieved every preregistered 2/6/12/20 tx/s window on C/CP/P/H, with zero
unsent requests and a passing four-validator oracle. At 20 tx/s, H offered
51,360 encoded bytes/s, within the unchanged S06 cap. This observation is
why the precision extension selects **20 tx/s**, without asserting a
capacity knee or scientific effect.

The precision extension fixes 12 independent fresh networks per profile
(48 total), with a random initial profile order and 12 cyclic blocks, so
every profile occupies every order position three times. Each run uses a
fresh genesis/validator set, unique seed, 241 distinct funded account
transactions at nonce zero: one warm-up send plus 20/s for 12 steady
seconds, followed by three drain seconds. All run durations, tx/s, bytes/s,
queue, CPU/RSS and disk guards remain the accepted S06 caps. A failed run
stops the sequence; it is not silently retried. Exact planned/achieved
send windows, complete all-four block/state oracle, no censoring and all
241 block-scan inclusion observations are required for QC.

The latency observation is the first time each transaction is found in
the same block on all four validators by 100 ms block scanning. It is an
**upper bound** on inclusion time, not the exact commit timestamp.
For each profile, the 12 run clusters provide 2,892 transaction samples.
The already fixed empirical p99 population-rank band is 98.5–99.5% with
≥95% coverage under the optimistic i.i.d. order-statistic model. To account
for dependence, a deterministic two-stage bootstrap resamples 12 complete
networks with replacement and then transactions within each selected
network, for 2,000 replicates. The 95% percentile CI must have both sides
within **20% of the pooled p99**. A profile passes only if the rank floor,
cluster CI width, all run QC and sample counts pass. Otherwise p99 is
reported `INSUFFICIENT_PRECISION`; no effect comparison is made. The
numerical p99 remains pilot-only evidence even if precision passes.
