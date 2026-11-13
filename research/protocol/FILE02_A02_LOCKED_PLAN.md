# A02 — invalid account/key binding, locked before execution

Status: LOCKED DESIGN, 2026-11-14. Use the A01 paired design, source and
artifact contract with 12 fresh-genesis pairs, 24 fresh single-host
four-validator networks, balanced arm order, validator-0 loopback RPC,
H profile, one warm-up, 12 steady seconds, five-second drain and two-second
cooldown. Each arm offers 20 distinct 2,568-byte transactions/s (51,360
B/s); six legitimate H tx/s have identical bytes, hashes, schedule and
genesis across arms. The benign background is 14 valid H tx/s. Actor A's
attack background alters the account address in otherwise well-formed H
bytes at 14 tx/s. No signing key is required to create this invalid wire
payload. The registered oracle is code 11 at `binding`, no secp or ML
verification span, no invalid inclusion and state-neutral behavior.

Primary endpoint and decision are the A01 preregistered 3,000-ms all-four
fixed-legitimate inclusion fraction, paired benign-minus-attack mean and
two-sided 20,000-resample percentile bootstrap with seed 20261113. A
positive lower CI bound supports only bounded degradation at this endpoint;
zero-crossing is inconclusive and an upper bound at or below zero opposes it.
Secondary metrics are the invalid CheckTx binding/service wall spans,
fraction reaching secp/ML, process CPU envelope per offered attempt,
accepted mempool, CheckTx socket queue and P2P relay envelopes. Compare A02
service cost descriptively with A01 using the independent case samples;
do not attribute whole-process CPU to one class.

QC requires exact stream/genesis and rate/byte matching per pair, 241 sends
and result records per arm, 73/241 exact-once valid inclusion for attack/
benign, 168 invalid binding rejects in attack, all four validators agreeing
on BlockID/AppHash at height, intact raw, no drops or safety abort, and all
72 fixed legitimate observations per arm. The unchanged S06 limits remain
25 tx/s, 65,536 B/s, 16 outstanding, 60 seconds and loopback/private
targets. No failed pair is replaced. No DoS, knee, multi-host or P2P-origin
inference is authorized from this case.
