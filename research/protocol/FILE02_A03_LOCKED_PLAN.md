# A03 — corrupted classical signature, locked before execution

Status: LOCKED DESIGN, 2026-11-14. Actor B starts from a correctly signed H
transaction and flips a classical signature byte, preserving structural
format, address/key binding and 2,568-byte size. The fixed legitimate H
stream is six tx/s and identical across paired arms; 14 valid H tx/s in the
benign background are replaced by 14 distinct corrupted-signature tx/s in
the attack background. Use 12 new-genesis pairs and 24 fresh single-host
four-validator loopback-RPC networks, balanced order, 20 total tx/s and
51,360 B/s per arm, one warm-up, 12 steady seconds and five-second drain.

The oracle is code 4 at `secp`: structural and binding stages pass, the
classical verifier is reached and ML verification is not called. The invalid
transaction must remain absent from blocks. Measure CheckTx secp wall span
and whole-service wall time; exact per-class CPU time is unavailable without
disturbing the verifier. The primary service endpoint, paired 3,000-ms
fixed-legitimate inclusion fraction and 20,000-resample CI/decision rule
are identical to A01 and preregistered before sending. CheckTx socket wait,
whole-process CPU envelope, mempool and relay are secondary descriptions.

QC requires exact legitimate bytes/hashes/schedule/genesis, 241 sends per
arm, 73 valid and 168 invalid results in attack, 241 valid results in benign,
exact-once valid inclusion, 168 code/stage/span oracles, all-four BlockID/
AppHash agreement, no drops, safety abort or censored primary sample, and
intact raw/provenance. Keep S06 caps 25 tx/s, 65,536 B/s, 16 outstanding,
60 seconds; do not replace failed pairs. No DoS, knee, distributed/P2P-origin
or sustainable-throughput claim follows.
