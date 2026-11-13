# R06 — bounded equal-valid-rate network audit

Date: 2026-11-13. The [locked R06 design](../../protocol/FILE02_R06_LOCKED_PLAN.md)
and accepted RQ1 H–CP hypothesis preceded execution. Source commit
`635dfd3` added a mandatory COMMON-schema `research_plan/plan.json` for
each network, validated **before any send** with `file02_common.py
validate-plan`. The bounded four-validator, single-host, RPC-origin run used
20 valid tx/s, zero invalid tx/s, twelve one-second steady windows, and
unchanged S06 safety caps. Each of 12 independent blocks reused one logical
fixture across C/CP/P/H; every arm used a fresh network. Profile order was
counterbalanced. No arm was replaced. The exact CP and H wire size was
2,568 B/tx, so both offered 51,360 B/s.

Private session: `research/artifacts/file02/r06-20261113T035435Z-b008d52c`.
Source commit `635dfd3`; plan SHA-256
`cedcf16ae2a0d19c585cd2158b7f77455382a9664d066e0d8df8ef69639f834f`;
immutable raw manifest SHA-256
`a8d08f01e8a19bc80b568966fdc988f7e50c07a1fd8002840121eb384878dcb7`.
The private `plan.json`, 48 per-run COMMON plans and executable load plans,
raw streams, `result_per_run.json/csv`, `qc_report.json`,
`result_aggregate.json`, `figures/status.json` and `decision.md` are
available for Controller audit. `python3 research/tools/file02_r06.py verify
research/artifacts/file02/r06-20261113T035435Z-b008d52c` exited 0.
Integration exited 0, all 48 arms passed bounded-network QC, 240 distinct
steady CheckTx spans/profile/arm joined to successful sends, all 241 valid
transactions/arm eventually committed exactly once, no invalid commit or
recorder/observer drop, and every measured one-second send window had 20 tx.

| Profile | Mean app CheckTx service, µs/tx | Mean RPC response, ms | Mean inclusion upper bound, ms | p99 inclusion upper bound, ms | Mean shared app CPU delta, s | Mean shared app RSS peak, MB | Mean max mempool bytes | Mean validator-0 relay send envelope bytes |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| C | 217.81 | 0.946 | 1,141.72 | 1,914.07 | 1.597 | 122.4 | 4,995 | 110,880 |
| CP | 216.36 | 0.961 | 1,125.96 | 1,861.84 | 1.660 | 125.2 | 85,600 | 1,853,280 |
| P | 143.36 | 0.887 | 1,156.79 | 1,912.64 | 1.397 | 124.7 | 85,762 | 1,807,200 |
| H | 297.83 | 1.121 | 1,154.34 | 1,911.07 | 1.883 | 124.1 | 90,736 | 1,853,280 |

The primary H−CP difference of per-network mean CheckTx service was
**+81.47 µs/tx**, 95% paired block bootstrap CI **[+75.16, +87.26] µs/tx**
with twelve blocks and 20,000 preregistered resamples. The CI lower bound
exceeds zero for this bounded stratum. All four inclusion-upper-bound p99
estimates passed the registered twelve-cluster precision gate; their
individual C/CP/P/H 95% CIs were [1,844.13, 1,970.53],
[1,826.17, 1,885.82], [1,876.20, 1,943.06], and
[1,867.70, 1,955.46] ms respectively. These are first block-scan observation upper
bounds, not exact commit times. Eventual inclusion is not sustainable
steady-state throughput. The shared application process CPU counter covers
all four validators on the host; it is not per-validator crypto CPU.
Relay envelope byte counters are observation at RPC-origin networks, not
P2P-origin injection or peer propagation delay.

**R06 remains PARTIAL and unchecked.** The phrase “under knee” cannot be
established because T06 observed no knee in the safe 2/6/12/20/23/25 tx/s
range. `under_knee=NOT_OBSERVED_WITHIN_SAFE_RANGE`. The measured positive
H−CP effect may support the bounded single-host RQ1 mechanism, but it is not
a saturation, sustainable-throughput, distributed-network or DoS result.

An earlier 48-arm session
`research/artifacts/file02/r06-20261113T033106Z-4d87a532` has frozen raw
manifest `8aa6e25b01858751816a8a7f9d355c8d0c92152ab6a136ef8b4488bac8835464`.
It passed load/span measurement QC, but the per-run COMMON plans were absent
before sends. It is retained as **diagnostic only and excluded from RQ1
confirmatory inference**. Its first derived version is frozen in
`analysis-v1/`; no historical raw bytes were changed. This protocol
deviation triggered the pre-send plan binder and the separate complete run
above; no observations were pooled across sessions.
