# Q07 — no-invalid-accepted and state-safety oracle audit

Status: LOCKED SECONDARY AUDIT before derived outputs, 2026-11-14. Reverify
the frozen/QC-PASS A01–A05/A08/A13 and corrected Q02 v2 sessions: 248 fresh
networks. For each run, compare every planned transaction hash and expected
CheckTx code/stage with the actual socket result and validator-0 recorder.
The committed all-four observation set must contain every legitimate hash
once and no adversarial hash. Require the live oracle's `OK` status,
common height, BlockID/AppHash and valid/invalid counts. Any mismatch
excludes the run and its matched pair/block from performance inference.

Bind the live oracle implementation from each source commit: the same
`s07Oracle` function must check all four validators' committed transaction
multisets, common-height BlockID/AppHash, receiver balance/nonce and each
sender balance/nonce. Expected supply over the genesis funded accounts is
two units per one-use sender; valid transfers move one unit to the receiver,
invalid transfers leave their sender untouched. Report the expected total
and the oracle implementation digest. The raw `oracle.json` stores an
attestation/status rather than each account's detailed state; this audit
cannot independently reconstruct account state from raw alone. Preserve
that limitation; do not fabricate per-validator state snapshots.
