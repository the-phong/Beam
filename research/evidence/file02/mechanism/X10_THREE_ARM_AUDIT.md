# X10 — prospective three-arm L/B/I service audit

Status: **PASS for the registered single-host RPC-origin upper-bound
observation contrast**. Source `c3ce45b4adae9d6477ac2749469762468e38d938`
and `FILE02_X10_THREE_ARM_LOCKED_PLAN.md` were committed before the
72-network confirmatory run. Its private plan records source tree,
host/Go/module/config SHA, all 24 balanced arm orders, run IDs, safety
and CI. The source tree remained clean during execution. `make verify`,
the triplet builder test and integration compile passed before load.

`python3 research/tools/file02_x10.py run` completed all 72 fresh
four-validator networks (24 L/B/I blocks) without rate miss, safety
abort or excluded arm. Each run had four dedicated app PIDs distinct
from the harness. The independent verifier recomputed every run's
source/config/plan/genesis identity, exact fixed H bytes and offsets,
B/I 2,568-byte offered match, 12 one-second achieved windows,
CheckTx code/stage, distinct exact-once valid commits, 168 bad-ML
rejections in each I arm, four per-validator observation stamps,
app/observer PIDs, zero drops and AppHash/block oracle. QC was **PASS
72/72, 24/24 blocks**. Raw manifest SHA-256:
`a97e1665deca67e747f30a1c6b03268140b716886e42fb24d9c1e1a6512233c8`.
`file02_x10.py verify` returned `X10_VERIFY=OK` on frozen raw and all
derived JSON/CSV/decision artifacts. Private runtime and signed plan
contents remain ignored.

| Fixed valid H outcome | L: 6/s only | B: 6/s + 14 valid/s | I: 6/s + 14 late-bad-ML/s |
|---|---:|---:|---:|
| Mean of 24 run means, first-all-four observation upper bound (ms) | 1161.95 | 1112.87 | 1172.91 |
| Observed by 3 s, steady transactions | 1728/1728 | 1728/1728 | 1728/1728 |
| Mean four-app process CPU interval sum (s/run), descriptive | 1.598 | 2.415 | 1.721 |

The preregistered paired **I−B** effect on the fixed-stream *mean
observation upper bound* is **+60.03 ms**, 20,000-resample block-bootstrap
95% CI **[+39.51, +81.54] ms**, n=24. Achieved half-width 21.01 ms
met the ≤40-ms target. The secondary **I−L** effect is +10.95 ms
[−9.67, +30.89] ms. The unchanged binary 3-s endpoint remains 100%
timely in every arm, with paired I−B effect 0 [0,0]. The upper-bound
latency CI is entirely above zero **and** inside the prospectively
registered ±100-ms research margin. Thus the direction of the observed
metric differs, but a ≥100-ms research-relevant mean effect is not
supported for this testbed; this is not a universal equivalence claim.

The first full attempt was retained and excluded for the documented
`RATE_MISS` at block 10; see `X10_FIRST_ATTEMPT_QC.md`. It supplied no
effect or CI. A later analyzer correction reordered its branch logic to
apply X04's registered ±100-ms rule before assigning the “positive
direction, size inconclusive” label. The original derived files from
the complete second session were preserved byte-for-byte in frozen
`analysis-v1/` (manifest
`b389caac0b63e07fdc8ab9c4f379a54da38ff355c8f55e267d525e85d72c2046`).
The corrected analyzer at `df943a7` changed **only the decision label**
to `NO_RESEARCH_RELEVANT_MEAN_EFFECT_WITHIN_REGISTERED_MARGIN`; effect,
CI, per-run data and raw manifest stayed identical. Five decision
tests cover inside/crossing/above-margin and precision/binary cases.

These are inclusion **observation upper bounds** from nominal 100-ms
block scanning, not exact commit times. Same-host scheduling and block
timing can contribute to the difference, and valid B background enters
admission/relay/commit while invalid I background is rejected. The
contrast therefore measures total workload replacement, not pure ML
verification CPU. The all-four 3-s endpoint remains a ceiling-limited
negative finding; no DoS, saturation, knee, P2P-origin, cross-host or
sustainable-throughput conclusion follows.
