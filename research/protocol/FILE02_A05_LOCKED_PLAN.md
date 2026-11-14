# A05 — state-invalid after valid signatures, locked before execution

Status: LOCKED DESIGN, 2026-11-14. Actor C is a registered signing-key
holder and generates correctly signed H transactions with nonce 1 when
its account is still at nonce 0. This is a stronger adversarial capability
than Actor A arbitrary bytes. The oracle is code 6 at `state` after valid
structural, binding, secp and ML stages; invalid transactions never commit
and cannot change account state. The 14 attack-background tx/s are distinct
2,568-byte signed state-invalid bytes. Six fixed legitimate H tx/s remain
byte/hash/schedule/genesis-identical to the matched benign arm, which adds
14 valid H tx/s. Both arms offer 20 tx/s or 51,360 B/s.

Use 12 new-genesis pairs, 24 fresh four-validator single-host networks,
balanced order, validator-0 loopback RPC, one warm-up, 12 steady seconds,
five-second drain and two-second cooldown. Measure all verification and
state wall spans, full CheckTx service, process CPU envelope, CheckTx socket
wait, mempool and relay. The primary endpoint is the preregistered 3,000-ms
all-four observed-inclusion fraction of 72 steady fixed legitimate tx per
arm. The paired benign-minus-attack mean uses a 20,000-resample percentile
bootstrap seeded 20261113; infer bounded degradation only when the lower
CI bound is positive and QC passes.

QC requires 241 writes/arm, 73/241 exact-once valid commits in attack/
benign, 168 late invalid code/stage/span oracles, no invalid inclusion,
unchanged fixed legitimate stream/genesis and offered bytes, all-four
BlockID/AppHash agreement, intact provenance/raw, no drops, safety abort or
missing primary stamp. Keep S06 25 tx/s, 65,536 B/s, 16 outstanding and
60 seconds/run limits. Do not replace a failed pair or infer DoS, knee,
multi-host or P2P-origin effects.
