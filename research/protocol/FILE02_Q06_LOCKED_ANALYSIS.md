# Q06 — independent network/seed/order replication audit

Status: LOCKED SECONDARY ANALYSIS before derived outputs, 2026-11-14. Use
the frozen/QC-PASS A01–A05/A08/A13 sessions and the new Q02 structural-
invalid session. Reverify raw/QC and compare only identical registered
3,000-ms first-all-four fixed-legitimate endpoint strata. For each A case,
check distinct pair seeds/genesis, balanced arm-first order, the original
paired 95% CI, leave-one-pair-out means and arm-order-stratified means.
For Q02 H, check 12 distinct block seeds/genesis, six benign-first and six
attack-first matched H pairs, then compare its structural-invalid effect
against A01 H as a separate cross-session replication. Do not pool CI
across different sessions or attack families. A13's four pairs receive the
same descriptive stability checks with two/two order balance.

Explicitly log the Q02 pre-send stale-observer safety-abort session and its
frozen invalid QC, the first technically complete but profile-order-confounded
`schedule.v1` session, and the corrected `schedule.v2` session; do not call
the aborted arm a valid replicate. Existing A source QC exclusions are
reported verbatim. Host replication requires a genuinely separate physical
host with an accepted deployment. Distinct fresh networks or VM guests on
the same physical machine do not satisfy it. If absent, Q06 stays PARTIAL
even when seed/order stability passes. A zero-valued ceiling-limited
endpoint CI is not an equivalence or distribution-shift robustness proof.
