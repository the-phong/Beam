# X18 — MECHANISM/SERVICE evidence gate, engineering disposition

**Technical disposition for Controller/Owner review:**
**CONDITIONAL GO for reporting bounded mechanism and service measurements;
NO-GO/NOT ESTABLISHED for opening the original early service-degradation
detector RQ2/RQ3.** This engineering decision is not independent Gate
acceptance and does not authorize D01–D13 or ML training. The historical
RQ1 and RQ1b contracts and 3-s result are unchanged.

| Gate question | Evidence-based answer | Boundary |
| --- | --- | --- |
| (a) Stage/resource asymmetry observed? | **YES, bounded.** Historical invalid CheckTx application wall means span 7.25 µs early structural to 391.31 µs late bad ML and 438.28 µs state-invalid on different networks. In new matched X10, B valid background versus I late-invalid had +0.694 s/run four-app CPU, +0.404 s/run four-Comet CPU, +1,297,296 B/run validator-0 relay-send envelope delta and +62,060 B peak accepted mempool, each with 24-block paired CIs in X16. | Wall is not per-stage CPU. Valid B reaches accepted relay/commit, I fails CheckTx; exact causal CPU shares are unavailable. |
| (b) Service latency effect? | **A direction is observed in the new upper-bound metric.** X10 I−B mean +60.03 ms [39.51,81.54] at 20 tx/s, n=24, QC PASS. Secondary p50/p95 CIs are reported. | Primary CI lies inside preregistered ±100-ms research margin; no ≥100-ms relevant mean effect supported. The metric is a 100-ms-scanner upper bound, not exact commit or universal equivalence. |
| (c) Sustained degradation or bounded negative? | **Bounded negative at the locked 3-s endpoint.** Historical matched A cases remain 0 [0,0]; X10 and X13 have every fixed H observed by 3 s. The prospective two-window ≤0.90 event is absent. | No saturation, sustained degradation, DoS or sustainable-throughput claim. X13 20/23/25 does not establish a knee; `NOT_OBSERVED_WITHIN_SAFE_RANGE`. |
| (d) Bottleneck located? | **UNAVAILABLE.** X07 has app entry/lock/result and the socket subqueue; X16 has process CPU and sampled mempool/relay. | No linked full queue residence, observer self CPU, exact per-stage CPU or randomized mediator isolation. The accepted path appears costlier, but a bottleneck is not localized. |
| (e) Claims excluded? | **Explicitly excluded:** exact commit/finality latency, per-stage crypto CPU shares, P2P-origin or many-peer effects, independent physical-host replication, distributed DoS, universal no-effect/equivalence, capacity at 25 tx/s, detector lead time or ML benefit. | R12/A12/Q06 and X07/X16 remain open/limited. |

Decision on research targets: the bounded RQ1 resource-cost and RQ1b
service-impact reporting can proceed with their limitations. The new
RQ-Mechanism can report stage wall and process/downstream asymmetry, but
its bottleneck and stage CPU components remain **PARTIAL**. The original
early-degradation RQ2 has no measured event in the tested scope and is
**NOT ESTABLISHED/UNTESTABLE**; RQ3 comparison is **NOT_RUN**. An alternative
adversarial-workload-detection question would require a distinct Owner-
approved prospective contract; this X18 decision itself opens none.

The independent review request is to accept or return this bounded
technical disposition while leaving X07 and X16 unchecked. Rework can
target full queue linkage, stage/observer CPU, event timestamp uncertainty
or independent-host/P2P study only under a separately approved plan and
unchanged safety rules. No historical raw is revised. Supporting reports:
[service](SERVICE_IMPACT_REPORT.md),
[mechanism](MECHANISM_ATTRIBUTION_REPORT.md),
[scope](SCOPE_AND_UNAVAILABLE_MATRIX.md) and
[reproducibility](EXTENSION_REPRODUCIBILITY_PACKAGE.md).
