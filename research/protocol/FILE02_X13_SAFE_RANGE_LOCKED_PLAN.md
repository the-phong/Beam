# X13 — prospectively locked safe-range sensitivity

Status: **LOCKED BEFORE X13 SENDS**. X10 measurement QA and the 24-block
20-tx/s L/B/I result passed before this plan. X13 is a sensitivity test,
not a capacity or knee search. It does not revise the accepted RQ1/RQ1b
or X10 decision. A QA-only one-block session may validate the pipeline;
it is excluded from X13 inference.

Three **invalid-background** arms use 6 identical fixed valid H/s plus
14/17/19 distinct late-bad-ML H/s at **20/23/25 tx/s** respectively.
The offered wire rates are 51,360/59,064/64,200 B/s, all below the
unchanged 65,536 B/s cap. The same fixed signed bytes, send offsets,
funded identities and genesis are used within each matched fixture block;
each arm runs on a fresh four-validator network. Source is loopback RPC
validator 0 on one physical host. There are **12 blocks × 3 arms = 36
networks**, with six rate-order permutations each repeated twice and
deterministically shuffled using seed 20261115. No arm may be selectively
replaced. Each run has 1 s warm-up, 12 s steady, 5 s drain and 2 s cooldown,
≤16 outstanding and ≤60 s, with S06 safety aborts unchanged.

The independent unit is the fresh network within a matched block. The
primary descriptive response is the rate-specific mean of per-run fixed-H
first-all-four **observation upper bound** in ms; report 20,000-resample
95% block bootstrap intervals per rate. Paired 23−20 and 25−20 differences
use the same matched blocks and bootstrap. Secondary: per-run p50/p95,
fraction observed by 3,000 ms, CheckTx late-ML wall spans, app/Comet CPU
and accepted/rejected counts. No new equivalence or positive decision
threshold is claimed from this exploratory sensitivity study. The 3-s
endpoint remains distinct from continuous latency; exact commit time
is unavailable.

QC requires every planned run/arm, same source/module/config/genesis and
fixed stream within a block, exact one-second 6+14/17/19 achieved windows,
2,568-byte H wire, zero dropped observations, all fixed valid committed
exactly once, all 12×background bad-ML transactions per run rejected at
ML and absent from blocks, four-validator BlockID/AppHash agreement and
unchanged safety guard. Any failure is retained in immutable raw and the
whole session is invalid for sensitivity inference; no replacement after
observing outcomes. Runner freezes raw and writes plan, per-run JSON/CSV,
QC, aggregate, decision and independent verifier outputs. The report must
say **NOT_OBSERVED_WITHIN_SAFE_RANGE** for a knee unless the predeclared
capacity criterion of R15 is actually met; X13 does not establish
sustainable throughput, saturation or DoS.
