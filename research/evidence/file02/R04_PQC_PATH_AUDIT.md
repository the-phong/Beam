# R04 — CP/P/H path contrasts and canonical byte limitation

Date: 2026-11-13. The [R04 plan](../../protocol/FILE02_R04_LOCKED_PLAN.md)
was committed at `07c2d64` before twelve new direct-ABCI process runs.
Each run had a fresh genesis, common logical intents across C/CP/P/H,
counterbalanced profile order and 240 accepted valid CheckTx calls/profile
after warm-up. All 12 process runs passed QC with zero recorder drops.
Private session:
`research/artifacts/file02/r04-20261113T033800Z-07c2d64`.
Plan SHA-256:
`b1cb51ec67110e4ac7da02c8a692f2bbac3ea5d23916ed46e8e797b41e8aa620`.
Raw manifest SHA-256:
`fd3e9712067d9af117ed783b9419255376442351a1d4693451f50299e790326c`.
The locked plan, immutable raw, per-run JSON/CSV, QC, aggregate,
figure-status and decision artifacts are private. Independent
`python3 research/tools/file02_r04.py verify
research/artifacts/file02/r04-20261113T033800Z-07c2d64` exited 0.

| Direct CheckTx contrast | Wire delta | Mean service delta | 95% paired process CI | Interpretation |
| --- | ---: | ---: | ---: | --- |
| P−CP | −64 B/tx | −82.25 µs/tx | [−84.88, −79.64] µs/tx | Changes verifier type and bytes; confounded. |
| H−P | +64 B/tx | +126.77 µs/tx | [+123.52, +130.51] µs/tx | Adds secp and bytes; confounded. |
| H−CP | 0 B/tx | +44.53 µs/tx | [+40.92, +48.53] µs/tx | Exact-size secondary H/CP direct control. |

Mean stage wall times were CP secp 118.78 µs, P ML 44.00 µs, H secp
119.60 µs and H ML 45.86 µs per accepted CheckTx. Per-process CPU,
allocation and stage rows are in the private per-run results. T03's
[rejection oracle](T03_REJECTION_PIPELINE_AUDIT.md) independently showed
late bad-ML H reaches secp before ML and is rejected without commit.
Stage wall time does not equal stage CPU time; the direct path does not
measure live network relay, goodput or consensus latency.

The canonical P wire is 2,504 B; CP/H are 2,568 B. The current accepted
codec has no 64-byte P padding. `P_byte_match_status` is
`UNSUPPORTED_CANONICAL_CODEC`, so P−CP and H−P cannot isolate a pure
cryptographic causal effect. Changing the codec is outside this task.
**R04 is PARTIAL, not a fully causal PQC-only PASS**, despite successful
measurement QC. Its box remains unchecked. R05 verify-order control is next.
