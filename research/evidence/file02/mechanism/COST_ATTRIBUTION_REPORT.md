# X06 — crypto-stage attribution and CPU sampling

Status: **PASS for isolated primitive CPU-path profiling; network
`CPU_crypto` remains `null`**. The accepted RQ1 H−CP effect remains
+81.47 µs/tx for application CheckTx wall time in its tested network
stratum. No value below replaces that result or identifies a per-stage
network CPU contribution.

## Matched isolated primitive profile

The accepted R01 valid H fixture supplied identical intent and signatures
to the secp256k1 and ML-DSA-44 primitive subbenchmarks. The first X06
profiled-only diagnostic raw manifest is
`60b553f5aadf18006808a5340bee0ed967e7ae24696a40235566d257705bf45f`.
It is retained but excluded from the profiler perturbation comparison.
The prospectively amended second session ran four pairs of isolated
profiled and immediately following unprofiled processes, balanced in
primitive order. Its private immutable raw manifest is
`b2af73a3787c4be47b69f68ebaf37a678c53d95ed0b0b1924eaec31f70ad9d89`;
`file02_x06_profile.py verify` returned that digest. Both are Git-ignored.

| Primitive | Profiled wall ns/op mean | Unprofiled wall ns/op mean | Mean paired ratio | Ratio range | CPU samples across four profiles |
|---|---:|---:|---:|---:|---:|
| secp256k1 | 135,234 | 136,590 | 0.992 | 0.890–1.063 | 15.09 s (~1,509 10-ms ticks) |
| ML-DSA-44 | 23,612 | 24,714 | 0.956 | 0.905–0.988 | 12.20 s (~1,220 ticks) |

Every profiled process had at least 2.46 s of CPU samples and a readable
pprof. The first secp profile put 39.0% of samples in `FieldVal.Mul2`,
29.6% in `SquareVal`, 11.8% in `Normalize`; the first ML profile put 15.0%
in `nttAVX2`, 13.9% in `mulHatAVX2`, 11.4% in SHA3 permutation. These are
flat **sample proportions within isolated primitives**, not additive
CheckTx stage CPU percentages. At a nominal 100-Hz sampling rate, a 15%
proportion with 280 ticks has approximate binomial standard error
sqrt(0.15×0.85/280)=2.1 percentage points; autocorrelation, inlining,
stack attribution and host scheduling can widen it. Four process pairs
measure between-process spread but are too few for narrow overhead CI.

The paired profiled/unprofiled wall ratios vary around or below 1.0.
They do not establish that profiling speeds up verification: the fixed
profile-first order, runtime warmup, Go benchmark iteration choice and
host drift confound an overhead estimate. The measured perturbation is
bounded only by the observed ratio ranges, not a causal overhead bound.
The earlier frozen R01 nonprofiled means (104,762 ns/op secp; 18,294
ns/op ML) differ materially from this session; they must not be used as
an overhead denominator across days/configuration states.

## Full path and work redistribution

R02's matched direct-ABCI wall stage means for CP/H (µs/tx) were:
structural 0.96/0.58; binding 0.71/0.70; digest 0.22/0.25; secp
115.74/117.60; ML absent/44.92; state rules 16.03/15.26; state copy
22.98/21.51. R02's full CheckTx process CPU deltas were 157.78/202.26
µs/tx (CP/H). Recorder, allocations, Go runtime and state work are in
those process totals. Stage wall spans can overlap scheduler effects and
are not CPU. Network RQ1 H−CP +81.47 µs/tx includes a different execution
context and cannot be decomposed by subtracting R02 primitive numbers.

Different rejection stages redistribute work: A01 structural-invalid
CheckTx wall mean 7.25 µs/tx; A04 bad-ML 391.31 µs/tx; A05 bad-state
438.28 µs/tx. Those are different case networks and not paired CPU
comparisons. Accepted valid traffic may additionally enter Comet mempool,
relay, proposal and commit; rejected invalid traffic normally does not.
The workload replacement contrast B→I therefore measures the total
pipeline change, not a pure crypto intervention.

`CPU_crypto` per network transaction is `null` with reason
`UNAVAILABLE_STAGE_CPU_ATTRIBUTION`. X05's separate app-process CPU series
do not disaggregate concurrent CheckTx, FinalizeBlock, Go runtime, GC or
idle work. A calibrated per-stage CPU tracer would be required for an
exact CPU decomposition. The isolated profiles identify where primitive
CPU samples fall, while R02 gives stage wall durations; neither licenses
conversion of wall spans into CPU nanoseconds.
