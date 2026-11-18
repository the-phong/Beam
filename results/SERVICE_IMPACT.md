# Legitimate service impact

The historical matched RQ1b comparison kept six valid H/s fixed and added 14 background H/s (benign valid or invalid), for 20 H-sized tx/s and 51,360 B/s. Across A01–A05/A08, every fixed valid H in both arms was observed by the locked **3-second** deadline: paired timely-fraction effect **0 [0,0]**. This is a ceiling-limited negative result, not latency equivalence or DoS immunity.

## Prospective L/B/I contrast

X10 used 24 matched blocks/72 fresh networks. L sent six fixed valid H/s only; B added 14 valid H/s; I added 14 late-bad-ML H/s. B and I matched 20 offered tx/s and 51,360 B/s, with fixed legitimate identities, signed bytes, offsets and genesis paired. Mean first-all-four block-observation **upper bounds** were L **1161.95 ms**, B **1112.87 ms**, I **1172.91 ms**.

| Contrast | Effect on mean observation upper bound | 95% paired CI | Estimand |
| --- | ---: | --- | --- |
| I−B | +60.03 ms | [+39.51,+81.54] ms | Replacing benign valid background with invalid background at matched offered tx/s and B/s |
| I−L | +10.95 ms | [−9.67,+30.89] ms | Adding invalid background to the fixed stream |

All fixed valid H met the 3-second endpoint, and the I−B binary timely-fraction effect stayed 0 [0,0]. The positive I−B upper-bound shift is within the prospectively registered ±100-ms research-relevance margin; it does not establish a ≥100-ms mean effect or equivalence of the full latency distributions. The scanner's nominal 100-ms interval means these are not exact commit timestamps. X10's p50/p95 continuous contrasts are secondary and recorded in [the case audit](../research/evidence/file02/mechanism/X10_THREE_ARM_AUDIT.md).

## Safe-range sensitivity and mechanism

X13 used 12 matched 20/23/25-tx/s blocks/36 fresh networks with six fixed valid H/s and adjusted late-bad-ML background. All 864/864 fixed H at each tested rate were observed within 3 s. Mean upper bounds were **1177.47 / 1171.39 / 1182.16 ms**; paired 23−20 was −6.08 ms [−28.27,+15.44], and 25−20 was +4.69 ms [−21.15,+35.44]. This is descriptive sensitivity, not a capacity search. Neither X13 nor the earlier benign R15 run found a defensible knee through 25 tx/s. A stopped X13 session failed integrity and contributes no estimate.

X16 measured more app CPU and downstream accepted work in B than I, consistent with valid background entering relay/proposal/commit. It did not isolate each downstream CPU contribution or locate an exact bottleneck. X07 supplied hash-linked queue-path intervals but no additive full-transaction queue scalar; measurement probes perturb some timings. The prospective candidate sustained event (fixed H timely fraction ≤0.90 in two consecutive nonoverlapping 3-s windows) did **not** occur in QC-passing X10/X13. Original early-degradation RQ2 is **NOT ESTABLISHED**; RQ3 ML benefit is **NOT_RUN**. No sustained service degradation or DoS is demonstrated in the measured scope. See [limitations](LIMITATIONS.md), [metrics](../docs/METRICS.md) and [provenance](provenance/INDEX.md).
