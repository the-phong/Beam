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
invalid transfers leave their sender untouched. Independently replay all
included valid bytes from the frozen genesis through the observed block
heights, check full account count, total supply and AppHash. Comet block
header at height H carries application state hash from H-1; require all
observed valid commits before H and compare replay at H-1. This binds the
full expected state, including A13's 84 unused funded senders, to the
same-height four-validator AppHash attested by the live oracle. Report the
oracle implementation digest. Raw `oracle.json` still omits individual
per-validator account snapshots; do not fabricate them.
