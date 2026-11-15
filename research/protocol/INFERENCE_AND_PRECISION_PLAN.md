# X15 — inference, precision and multiplicity record

This document consolidates the **pre-send** decisions in
`LOCKED_EXTENSION_PROTOCOL.md`, `FILE02_X10_THREE_ARM_LOCKED_PLAN.md` and
`FILE02_X13_SAFE_RANGE_LOCKED_PLAN.md` with the completed QC. It does not
retroactively relabel an exploratory contrast as confirmatory. The unit
of independent replication is a fresh four-validator **network**; a matched
block contains separate networks with the same fixture. Transactions within
a run, four validators on one host and 100-ms scanner polls are correlated
observations, not independent samples.

| Study | Locked units and order | Estimand / CI | Inferential status |
| --- | --- | --- | --- |
| Historical A01/A03/A04/A05/A08 | 12 matched pairs per class, balanced benign/attack order | Original fixed-H observed-by-3s fraction, 20,000 paired bootstrap draws | Preregistered historical binary endpoint; all five 0 [0,0]. New continuous reanalysis and its class-specific p50/p95 CIs are exploratory. Never pool classes after seeing effects. |
| X10 L/B/I | 24 blocks, each six-order permutation four times; same fixed hashes/genesis/offsets | Primary paired I−B **mean** first-all-four observation upper bound, 20,000 block-bootstrap draws seed 20261115 | New prospective primary. I−L, p50/p95 and 3-s fraction are secondary. The first failed full attempt is excluded wholesale; the successful second attempt has 24/24 complete blocks. |
| X13 invalid-background rate | 12 blocks, six 20/23/25 permutations twice each | Per-rate mean upper bound and paired 23−20/25−20 95% block-bootstrap intervals | Prospectively designed **descriptive sensitivity**, no knee or equivalence test. QA and interrupted/corrupted attempt are excluded; successful whole-session restart has 12/12 blocks. |

The historical three-case continuous-latency pilot suggested paired SD
about 60 ms. For X10, `ceil((1.96×60/40)^2)=9` was an optimistic minimum;
24 blocks were locked to allow measurement variance and QC loss. The
registered target is primary CI half-width ≤40 ms. The successful X10
estimate is **+60.03 ms [39.51,81.54]**, half-width **21.01 ms**. X13's
12 blocks were locked as sensitivity, not powered for a no-effect claim.
No sample size was enlarged after observing a treatment effect.

The original **3,000-ms binary endpoint** and its complete follow-up are
reported beside continuous latency; zero misses cannot establish latency
equivalence. For the new continuous measure, each fixed valid H has actual
monotonic socket-write time and the first all-four block-index observation
stamp. It is an **upper bound** with nominal 100-ms polling uncertainty and
host scheduling effects, not exact commit or finality time. Per-run p50/p95
are computed across that run's 72 steady fixed transactions, then compared
as network-level statistics. `X15_SECONDARY_CI.json` recomputes paired X10
secondary p50/p95 intervals and X13 per-rate p50/p95 intervals from frozen
raw. They remain secondary or exploratory; no p99 precision is claimed.

X04 registered a **±100-ms research margin for X10 I−B mean only** before
new confirmation data. It was chosen after looking at X03 exploratory
data, and is not an externally validated service SLO. The conservative
two-sided 95% CI containment rule constitutes the predeclared practical
equivalence check: if the entire interval lies strictly inside
`[-100,+100]` ms, QC passes and the 3-s endpoint has no material discordance,
report no research-relevant **mean** effect in this testbed. X10 satisfies
that rule despite a positive direction. The result does not establish
equality of latency distributions, p95 equivalence, immunity or DoS absence.
X13 does not inherit the ±100-ms decision. For five X11 class comparisons,
the predeclared Holm familywise adjustment would be required for a joint
positive claim; none is made. Unadjusted class CIs are labeled exploratory.

Every absent CPU-stage, exact commit, full-queue, P2P-origin or independent
host metric is `null` plus a reason in the relevant aggregate/report. Missing
runs, source/config mismatch, wrong achieved one-second rate, oracle error,
dropped observation or safety abort fail QC. Excluded sessions remain
visible. The event criterion for future early degradation detection is two
consecutive nonoverlapping 3-s windows with fixed-H timely fraction ≤0.90
after complete follow-up; no such event is established here. No lead time
is filled with zero, and D/ML work is not opened by this plan.
