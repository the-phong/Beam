# R02 — four-profile direct CheckTx validation audit

Date: 2026-11-13. The [R02 plan](../../protocol/FILE02_R02_LOCKED_PLAN.md),
measurement-only stage hooks, producer and QC tests were committed at
`b955c4f98ad7a7a246b4331d1c0da269435bff4f` before the private
`LOCKED` plan and measurements. `make verify`, 37 File 02 Python tests,
and the real S05 four-validator telemetry regression passed before R02.
The direct ABCI run used twelve independent Go processes, each with a
fresh 242-account genesis, C/CP/P/H in counterbalanced order and 240
valid distinct nonce-zero CheckTx calls per profile after warm-up. Each
profile in a process used the same genesis and logical transfers; profile
signatures differed by their accepted domains. Recorder ON had zero drops.

Private session:
`research/artifacts/file02/r02-20261113T030748Z-b955c4f`.
Locked plan SHA-256:
`64d407aad54f7e481bba128759c80a1241be6caa548397d3918d5529d8525a9d`.
Frozen raw manifest SHA-256:
`d0560b9b311d198d64fbcc1e9ac91453975ed83ec2b0f93b155507e8e61b172c`.
The private session contains per-run JSON/CSV, QC, aggregate, analysis
provenance, figure-status and decision artifacts. The analyzer recalculates
all outputs from immutable raw. `python3 research/tools/file02_r02.py verify
research/artifacts/file02/r02-20261113T030748Z-b955c4f` exited 0 twice,
including one fresh invocation after final reanalysis.

| Profile | Wire B | Mean CheckTx service µs/tx | Mean process CPU µs/tx | Mean allocated B/tx | Mean allocs/tx | Pooled service p99 µs |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| C | 148 | 155.05 | 155.77 | 87,248 | 31 | 240.96 |
| CP | 2,568 | 157.09 | 157.78 | 87,248 | 31 | 250.95 |
| P | 2,504 | 76.80 | 77.37 | 109,514 | 21 | 141.07 |
| H | 2,568 | 201.36 | 202.26 | 110,288 | 38 | 296.30 |

Each profile had 12 independent process clusters and 2,880 CheckTx samples.
The preregistered optimistic p99 rank coverage and two-stage cluster
bootstrap radius gate passed for all four profiles; the exact p99 CI values
are in the private aggregate. Per-run p50/p95/p99, CPU, allocations and
all stage spans are in per-run results. The paired direct-ABCI H-minus-CP
mean service difference was **+44.28 µs/tx**, 95% paired-bootstrap CI
**[+40.15, +48.51] µs/tx**. This is supporting mechanism evidence, not
the accepted RQ1 *network* endpoint or a DoS inference.

Mean stage wall durations (µs/tx) separate structural parse, binding,
digest, secp, ML verification, state rules and state-copy. For CP they were
0.96, 0.71, 0.22, 115.74, absent, 16.03 and 22.98. For H they were
0.58, 0.70, 0.25, 117.60, 44.92, 15.26 and 21.51. Stage spans do not
measure stage CPU; process CPU and allocations include recorder/runtime
work. CheckTx performs no disk commit, so storage write latency is
`NOT_ON_CHECKTX_PATH`, not a measured zero. The run cannot measure mempool
or relay time.

Two analyzer issues were found and preserved transparently. The first QC
treated Linux `x86_64` and Go `amd64` as a host mismatch; parser fix
`f18d6d6` archived that failed analysis in frozen `analysis-v1/` (manifest
`fdf69aa96f32344c0f456e096bca4c8c111affde6af045b9ac185a3db3207de3`).
The second analyzer used unordered Python set iteration for a floating
point residual, causing a tiny cross-process reproducibility difference;
fix `c946179` sorted stages and archived the prior derived files in
`analysis-v2/` (manifest
`9b11242ef62e7a5015237d116befaa4902a35a2cf792b6bbeb48fc194e93ab57`).
The raw and locked plan were never edited or rerun. Final QC is `PASS`,
12/12 processes; aggregate status is `ANALYZED`.

R02 is **PASS for direct full validation and stage attribution within this
host**. Its measurements are wall durations and process deltas on a
single-host direct ABCI path. R03 size-only causal control is next.
