# R05 — verify-order control disposition

Date: 2026-11-13. R05 requires H baseline secp→ML verification compared
with a reverse or earlier-reject policy under the same workload, plus
stage reach counts, rejected-transaction CPU and correctness. The
[factor registry](../../config/file02_factors.json) explicitly marks only
`BASELINE_VERIFY_ORDER` implemented; `REVERSE_VERIFY_ORDER` and
`EARLY_REJECT` are `UNSUPPORTED`. The accepted
[`EvaluateTrustedObserved`](../../../internal/experiment/profiles/state.go)
source executes `UsesSecp()` before `UsesML()` (currently lines 360–375).
The [T03 oracle](T03_REJECTION_PIPELINE_AUDIT.md) verifies that an H bad-ML
transaction reaches secp then ML under this baseline order. The focused
`go test ./internal/experiment/profiles/app -run '^TestFile02T03' -count=1`
exited 0 on this branch.

No reverse/early-reject policy or control workload exists. Changing
consensus-relevant validation order or adding an unaccepted policy to
manufacture a comparison would alter the accepted protocol and exceed
this task's boundary. Existing T03 stage evidence cannot supply the
missing alternative arm or a CPU/transaction delta. Therefore R05 is
**UNSUPPORTED**, with no case run or fabricated 0-valued result; its
checklist box remains open. A later separately accepted policy variant,
correctness oracle, pre-run lock and matched run would be required.
