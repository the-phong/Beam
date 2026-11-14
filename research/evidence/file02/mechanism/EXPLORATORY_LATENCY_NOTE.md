# X03 — exploratory continuous observation latency in A04/A05/A08

Date: 2026-11-15. **Exploratory reanalysis of already frozen data, not a
replacement for the preregistered RQ1b 3-second result.** Source sessions:
A04 `a04-20261113T174032Z-edd720e5`, A05
`a05-20261113T175129Z-edc80e07`, A08
`a08-20261113T182741Z-0dff8ab6`. Their case-specific verifiers and
immutable-raw/QC checks pass. [Derived JSON](EXPLORATORY_LATENCY_A04_A05_A08.json)
contains all 36 matched pair IDs and source manifest hashes. Reproduce with
`python3 research/tools/file02_x03_latency.py write` and independent
`python3 research/tools/file02_x03_latency.py check` from the repository
root; both exit 0 on the frozen sessions.

Each case has 12 fresh matched benign/attack network pairs, 72 fixed valid
H transaction observations per arm. The same fixed hashes, genesis and seed
match within every pair. For each run, compute the mean, p50 and p95 of
`first-all-four inclusion observation time − monotonic socket write` in
milliseconds; for each pair subtract benign from attack. The table reports
the **mean of 12 pair differences** and an unadjusted two-sided 95%
percentile-bootstrap interval over pairs (20,000 resamples, fixed seed
20261115). Transactions within one run are not counted as independent CI
replicates. The run-level p95 has only 72 observations and is an exploratory
descriptive tail statistic, not a precision-qualified p99.

| Frozen case | Mean upper-bound difference, attack−benign (ms), 95% exploratory CI | p50 upper-bound difference (ms), 95% exploratory CI | p95 upper-bound difference (ms), 95% exploratory CI |
|---|---:|---:|---:|
| A04 late bad ML | +56.68 [37.43, 77.69] | +61.00 [34.91, 90.63] | +67.86 [34.30, 103.06] |
| A05 signed bad state | +55.83 [38.00, 72.83] | +71.30 [48.25, 93.54] | +69.13 [54.98, 82.76] |
| A08 mixed invalid | +80.08 [47.59, 112.22] | +66.67 [36.59, 97.82] | +80.11 [36.54, 126.04] |

These are signals worth prospectively testing, **not confirmed new RQ1b
effects**. The analysis was chosen after the original binary result and
examines three cases × three metrics without multiplicity adjustment or a
preregistered practical margin. Arm-order balance reduces but cannot
eliminate shared-host time trends, block-phase/scan effects or other
unmeasured confounding. Four validators share one physical host, and the
100-ms block scanner provides a delayed **upper bound on first observed
inclusion**, not an exact commit timestamp. Scheduling may add uncertainty
beyond its nominal polling interval. Differences around tens of milliseconds
must not be called precise commit-latency changes or service degradation
without a prospective observation contract and new matched runs.

The original endpoint remains unchanged: every fixed legitimate transaction
in both arms of these cases was first observed on all four validators by
3,000 ms, so the preregistered timely-fraction effect is **0 [0,0]** in
each case. That ceiling-limited negative finding is compatible with a
smaller continuous observation-time shift. It does not establish latency
equivalence, a DoS effect, a knee or behavior on another topology. X04/X08
must define a new estimand, precision and timestamp semantics before any
confirmatory latency run; this note must never be pooled into the original
locked 3-second CI.
