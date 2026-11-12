# File 02 RQ1b — legitimate service under invalid hybrid load

Status: Owner-authorized separate research question for pre-case planning,
2026-11-13. This addendum supplements the accepted S02 protocol; it does not
replace or renumber RQ1's H-minus-CP valid CheckTx service-time hypothesis.
It grants no R/A/Q/D case execution. The case-specific rate, deadline,
sample count, attack family and CI method must be locked before such a run.

## Contrast and endpoint

RQ1b asks whether adding deliberately invalid hybrid (H) traffic degrades
service for a fixed legitimate H transfer stream relative to a matched
benign H background. The treatment and control use the same legitimate
bytes, account/nonces, send schedule, genesis, software, topology and safety
configuration. Added background traffic is matched on total offered tx/s
and encoded bytes/s where feasible; every unmatched dimension is disclosed.
Benign background uses independent accounts. Attack families are analyzed
separately. This contrast estimates a bounded workload effect, not an
isolated cryptographic mechanism.

The primary per-run endpoint is the fraction of distinct steady-window
legitimate transfers **observed included within a preregistered deadline**
`tau_ms` after monotonic send time. The block scanner's first observation
is an upper bound on inclusion time, not the exact commit instant. The plan
must fix `tau_ms`, scan interval and uncertainty before execution and keep
the full deadline observable for every eligible send. A censored deadline
invalidates that run for this endpoint; it is not counted as success or
failure. Eventual goodput cannot substitute for this endpoint or for
sustainable throughput.

With one estimate per independent fresh-network run and preregistered pairs,
define `D = mean(Y_benign - Y_invalid)`. H0 is `D <= 0`; H1 is `D > 0`.
Reject H0 only when the paired-run 95% CI lower bound is greater than zero
and rate, oracle, provenance, safety, clock and complete-window QC all pass.
An upper bound at or below zero falsifies the proposed positive effect in
the tested stratum; a CI crossing zero is inconclusive. A practical SLO or
minimum important effect must be set prospectively if a service-failure
claim is desired. A positive RQ1b result alone is not a DoS finding.

Secondary outcomes: achieved legitimate/invalid tx/s and bytes/s, eventual
included count, observation-limited inclusion latency, accepted mempool
occupancy, CheckTx service and socket queue wait, CPU and RSS. Unavailable
measurements remain `null` with reasons. RPC single-host evidence does not
support P2P-origin or multi-host conclusions.

The [design comparison](../evidence/file02/SECTION2_RQ1_CONSISTENCY.md)
explains why RQ1 and RQ1b are separate. The [proposal history](../evidence/file02/SECTION2_SERVICE_DEGRADATION_PROPOSAL.md)
records the review candidate that preceded this Owner instruction.
