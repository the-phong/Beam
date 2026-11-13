# COMMON-04 — equal transaction and byte offered-rate QA

Date: 2026-11-12. Scope: bounded single-host loopback QA; no confirmatory
T/R/A/Q/D experiment, resource effect or RQ1–RQ3 inference.

Source commit: `f93fd228a5ebc160e45b0983c8c43f63ee9b950a`;
source tree: `2672710f156219fe7d8fbee7813717eff1ee6c15`.
Branch: `research/file02-common-foundation`.

`make test-file02-common04` exited 0. It built the pinned instrumented Comet
binary and ran eight fresh four-validator networks: C/CP/P/H at equal 2 tx/s,
then C/CP/P/H at a target of 2,573 encoded bytes/s. The private auditor
verified every plan/result/genesis hash, actual monotonic send offset, unique
funded account/nonce, successful RPC code, direct common-height four-validator
oracle, and the per-second offered schedule. It required a ≤5% byte-rate
difference from the preregistered target in **every** steady window.

| Regime | C achieved | CP achieved | P achieved | H achieved | Match |
| --- | ---: | ---: | ---: | ---: | --- |
| Equal tx/s, target 2 | 2 | 2 | 2 | 2 | Exact in each of four steady windows |
| Equal bytes/s, target 2,573 | 2,601 | 2,573 | 2,509 | 2,573 | Maximum deviation 2.49%, within ±5% in each window |

The equal-byte tx/s rates were respectively 17, 1, 1 and 1. Different
transaction counts are part of this byte-matched control, so these runs
cannot also be treated as an equal-tx/s comparison. Every offered valid
transaction committed exactly once and all four validators agreed on the
common-height BlockID/AppHash and account state.

Private frozen session:
`research/artifacts/file02/common04-20261112T133207Z-40eb96fc`.
Manifest SHA-256:
`257160dc3c88559c79db50e7b170abd618df4e635d36636bb1b1efbc9ab6fc02`.
`python3 research/tools/file02_common04.py verify research/artifacts/file02/common04-20261112T133207Z-40eb96fc`
exited 0 and rederived all eight runs from frozen raw. No private payload,
validator key or runtime home is tracked.

This establishes that both matching schedules are feasible for these low QA
rates within current safety caps. Higher matched rates and capacity knees
remain subject to T06 calibration and the `NOT FEASIBLE WITHIN SAFE RANGE`
rule. One run per cell supports no confidence interval or p99 estimate.
