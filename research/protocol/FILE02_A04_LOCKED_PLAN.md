# A04 — corrupted ML-DSA proof, locked before execution

Status: LOCKED DESIGN, 2026-11-14. Actor B modifies one byte of the ML-DSA
proof of a valid H transaction while retaining valid structure, binding and
classical signature. Each invalid H wire transaction remains 2,568 bytes
and distinct. The attack arm has six fixed legitimate H tx/s plus 14
corrupted-proof tx/s. The matched benign arm has the exact same six fixed
legitimate bytes/hashes/offsets/genesis and 14 valid H tx/s. Use 12 pairs,
24 fresh single-host four-validator loopback-RPC networks, balanced order,
20 offered tx/s and 51,360 B/s, one warm-up, 12 steady seconds, five-second
drain and two-second cooldown.

Oracle: structural, binding and secp stages pass; ML verification is reached
and fails with code 13 at `ml`; invalid bytes never commit. Measure secp and
ML wall spans, full CheckTx wall service, CheckTx socket wait, process CPU
envelope, accepted mempool and P2P relay envelope. The prespecified cost
hypothesis is A04 CheckTx service greater than A01 structural reject;
report the difference descriptively with its independent-run uncertainty,
without calling wall time CPU. The primary service endpoint is the A01
3,000-ms all-four inclusion fraction for fixed legitimate traffic, with
paired benign-minus-attack 20,000-resample CI seeded 20261113 and the same
bounded decision rule.

QC requires all 241 writes/arm, 73/241 valid exact-once commits in attack/
benign, 168 invalid code/stage/span oracles, identical fixed legitimate
stream/genesis and offered bytes by pair, all-four BlockID/AppHash agreement,
no drops, safety abort or censored primary sample, and intact raw. Retain
25 tx/s, 65,536 B/s, 16 outstanding and 60 seconds/run caps. No failed pair
is replaced, and no DoS/knee/multi-host/P2P-origin conclusion follows.
