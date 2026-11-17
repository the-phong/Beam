# X18 — legitimate service impact in the measured scope

**Scope:** four Comet validators on one physical host, four dedicated ABCI
app processes for the new X runs, validator-0 loopback RPC origin, H-sized
transactions and the unchanged S06 caps. The fixed legitimate stream is
six valid H/s. This report keeps the accepted historical RQ1b 3,000-ms
endpoint and adds separately registered continuous observation estimates.
The continuous metric is send-to-first-all-four block-index **observation
upper bound**, sampled nominally every 100 ms, not exact commit time.

## Historical endpoint and new X10 matched contrast

The accepted A01–A05/A08 benign-versus-invalid pairs each offered six
fixed valid H/s plus 14 background H/s, 20 tx/s and 51,360 B/s. Every
fixed valid H in both arms was observed by 3 s; the original paired timely
fraction effect was **0 [0,0]** for each case. This is a ceiling-limited
negative result for that endpoint and topology. X11's reanalysis found
exploratory continuous attack−benign observation shifts by class, but it
does not replace the historical endpoint or make a joint class claim.

X10 used **24 new matched L/B/I blocks**, 72 fresh networks and independent
raw/QC verification. L had six fixed H/s only. B added 14 valid H/s; I
added 14 late-bad-ML H/s. B and I matched total offered 20 H-sized tx/s
and 51,360 B/s. All 1,728/1,728 steady fixed valid H transactions per arm
were observed within 3 s, so the original-style binary I−B effect remains
**0 [0,0]**. Mean first-all-four observation upper bounds were L
**1161.95 ms**, B **1112.87 ms**, I **1172.91 ms**. The prospectively
primary paired **I−B = +60.03 ms**, 95% block-bootstrap CI
**[+39.51,+81.54] ms**; secondary **I−L = +10.95 ms**
**[−9.67,+30.89] ms**. Secondary I−B per-run p50 effect was +72.44 ms
[53.06,91.89]; p95 effect +57.24 ms [26.25,89.89]. The direction in the
upper-bound metric is positive, but the **primary** CI is wholly within
the prospectively registered ±100-ms research margin. It therefore does
not support a ≥100-ms research-relevant **mean** effect on this metric.
The margin was set after exploratory X03 and is not an external SLO.
This is neither proof of equal latency distributions nor a universal
no-effect statement.

## Safe-range response

X13 added separate new, balanced **12-block/36-network** sensitivity data
with six fixed valid H/s and 14/17/19 distinct late-bad-ML H/s at total
20/23/25 tx/s. At each rate, 864/864 fixed H were observed within 3 s.
The mean upper bounds were **1177.47 / 1171.39 / 1182.16 ms**.
Paired 23−20 was **−6.08 ms [−28.27,+15.44]** and 25−20 was
**+4.69 ms [−21.15,+35.44]**. This is descriptive rate sensitivity,
not a powered equivalence or sustainable-throughput study. The prior R15
benign 20/23/25 pilot and new X13 did **not observe a knee** within the
safe range. **25 tx/s is a safety cap, not estimated capacity.** The
interrupted X13 attempt is quarantined for integrity failure and supplied
no estimate; its runs are not mixed with the successful session.

## Event and decision boundary

X04 prospectively defined a candidate service event as fixed-H timely
fraction ≤0.90 in two consecutive nonoverlapping 3-s send windows after
complete 3-s follow-up and passing oracle. No such event occurs in the
QC-passing new X10/X13 networks; all their fixed transactions meet 3 s.
There is **no measured sustained legitimate-service degradation** by this
event criterion and no established early-warning target. RQ2 early
service-degradation lead time is `null/NO_MEASURED_EVENT`, not zero. RQ3
ML benefit is `NOT_RUN`, and no D case is opened. A future adversarial-
workload detector would be a different research question requiring a
prospectively approved addendum and natural-error benign controls.

The measured upper-bound latency difference can include shared-host
scheduling and scanner phase. Accepted valid B traffic also enters
mempool, relay, proposal and commit while invalid I traffic is rejected
at CheckTx, so I−B is a workload-replacement effect, not pure ML CPU.
Full queue residence, exact commit timestamp, observer self CPU,
crypto-stage CPU, independent-host replication, P2P-origin impact and
distributed behavior remain unavailable. **No DoS claim follows.**

Primary sources: [X10 audit](X10_THREE_ARM_AUDIT.md),
[X13 audit](X13_SAFE_RANGE_SENSITIVITY_AUDIT.md),
[X15 secondary intervals](X15_SECONDARY_CI.json),
[X16 mechanism](MECHANISM_ATTRIBUTION_REPORT.md),
[reproducibility package](EXTENSION_REPRODUCIBILITY_PACKAGE.md) and the
unchanged historical A/Q/R audits. Private raw manifests and exact
recompute commands are indexed in the reproducibility package.
