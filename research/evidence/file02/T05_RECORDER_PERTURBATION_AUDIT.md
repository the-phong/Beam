# T05 — recorder ON/OFF perturbation preflight

Date: 2026-11-13. The [locked plan](../../protocol/FILE02_T05_PLAN.md)
was committed at `14b34d08522ea2e287697d01e67defa2ddb025b7` before
execution. `make test-file02-t05` exited 0. Eight fresh, four-validator
single-host loopback networks formed four profile-matched ON/OFF pairs at
2 valid tx/s, using the same signed plan, genesis, seed and offered schedule
within each pair. C/P ran ON then OFF; CP/H ran OFF then ON. Each condition
sent and committed all nine valid transactions exactly once. Each pair had
the same AppHash at height 8. The ON recorder emitted 44–51 events, the OFF
recorder emitted none, and all recorder/observer drop counts were zero.

All differences below are ON minus OFF. CPU columns are process CPU-time
deltas over the observed run; RSS columns are differences in observed peaks.

| Profile | RPC completion p50 Δ (ms) | Comet CPU Δ (s) | App process CPU Δ (s) | Comet sum peak RSS Δ (MiB) | App peak RSS Δ (MiB) | ON events |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| C | +0.053 | +0.13 | +0.02 | −1.57 | −1.87 | 49 |
| CP | +0.046 | +1.95 | +0.41 | +1.74 | +1.53 | 51 |
| P | −0.192 | −0.79 | −0.22 | −7.40 | −1.25 | 44 |
| H | +0.077 | +0.37 | −0.04 | −0.10 | +3.46 | 45 |

Private frozen session:
`research/artifacts/file02/t05-20261112T180953Z-8eff1611`.
Manifest SHA-256:
`60aa6e38bde61b6ac619c44f4ecaa1b681fce3e0e1271edb6916ef5502f467bc`.
`python3 research/tools/file02_t05.py verify` exited 0 and recomputed the
summary from frozen raw. The machine-readable `summary.json` retains the
unrounded values, run IDs, provenance, both arms and observed metrics.

**T05 correctness and measurement-capture gate: PASS.** The descriptive
CPU/RSS signs vary across profiles. The app process includes the test harness
and all four application servers, while the Comet RSS sum adds separate
process peaks. With one pair per profile, no overhead CI or claim that
monitoring overhead is universally small is supported. This test does not
estimate an RQ effect or justify a safety-cap change. T06 is next.
