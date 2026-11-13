# A05 — signed state-invalid transaction audit

Date: 2026-11-14. The [locked plan](../../protocol/FILE02_A05_LOCKED_PLAN.md)
and stage oracle were committed at `9d4fd07` before A05 execution. Private
session: `research/artifacts/file02/a05-20261113T175129Z-edc80e07`.
Plan SHA-256:
`7ce7550e35ad3c11c9188409cfe0e4f869cea4bfe904bb9d9e90995f5856d4ee`;
immutable raw manifest SHA-256:
`b90bc2b95ac43388d9d7db8e85cfa3a2dc052258c421623a56cbe2d1ccc1026b`.

All 12 matched pairs and 24 new four-validator single-host networks passed
QC and independent raw rederivation. Each arm offered 20 H-sized tx/s or
51,360 B/s; the same six fixed legitimate H tx/s were byte/hash/schedule/
genesis-identical within each pair. Actor C controlled registered signing
keys and created 14 fully signed nonce-1 H transactions/s while its accounts
remained at nonce 0. This capability cannot be assigned to arbitrary-byte
Actor A. Every attack arm's 168 invalid transactions had structural, binding,
secp, ML and state spans, failed with code 6 at `state`, and never committed.
The 73 valid attack-arm and 241 benign-arm transactions committed once,
with all-four BlockID/AppHash and account-state oracles passing.

All arms observed 72/72 fixed legitimate transactions included on all four
validators by the registered 3,000-ms observation upper-bound deadline.
The paired benign-minus-attack mean was **0.000**, 95% bootstrap CI
**[0.000, 0.000]** over 12 independent pairs. There is no positive
service-endpoint effect at this tested 20-tx/s stratum.

| Descriptive metric | Benign | A05 attack |
| --- | ---: | ---: |
| Fixed legitimate inclusion upper bound, mean over 864 tx | 1,122.25 ms | 1,178.08 ms |
| Maximum fixed legitimate inclusion upper bound | 1,939.04 ms | 1,969.65 ms |
| Shared application-process CPU / 241 offered attempts | 8,872.75 µs | 5,943.98 µs |
| Invalid CheckTx wall service, mean of 12 run means | — | 438.28 µs/tx |
| secp wall span, mean of 12 run means | — | 167.32 µs/tx |
| ML wall span, mean of 12 run means | — | 83.67 µs/tx |
| state wall span, mean of 12 run means | — | 1.43 µs/tx |

The invalid request passed both signatures and spent CheckTx wall time
before its late state rejection. The process CPU quotient includes the
integration harness and all transaction classes; per-class CPU/tx remains
`UNAVAILABLE`. CheckTx socket queue is sampled separately in raw and is
not the entire transaction queue. The measured cost does not imply a DoS
effect or a service knee.

`python3 research/tools/file02_attack.py run A05` exited 0 with QC PASS;
`python3 research/tools/file02_attack.py verify
research/artifacts/file02/a05-20261113T175129Z-edc80e07` exited 0 with
`ATTACK_VERIFY=OK`. Private plan, raw, per-run JSON/CSV, QC, aggregate,
figure status and decision remain preserved. **A05 PASS for the signed
late-state failure mechanism and bounded negative service-endpoint test.**
No P2P-origin or multi-host conclusion follows.
