# A02 — invalid binding audit

Date: 2026-11-14. The [locked plan](../../protocol/FILE02_A02_LOCKED_PLAN.md)
and stage audit were committed at `9d4fd07` before any A02 send. Private
session: `research/artifacts/file02/a02-20261113T171854Z-5021cdef`.
Plan SHA-256:
`b764235392649fda4f3b506d9de59262c3bde2eca1c8c3cd215f394c62e808ea`;
immutable raw manifest SHA-256:
`9acf1b51ac4e0e9ba7132730df35de0698e06cc4c84f904dc6890727ea71646d`.

All 12 matched pairs and 24 fresh single-host four-validator networks passed
QC and independent raw rederivation. Each arm offered 20 H-sized tx/s
(51,360 B/s) with exactly the same six fixed legitimate H tx/s per pair.
Actor A changed the account address of otherwise signed H bytes without
possessing a registered key. Each attack arm submitted 168 distinct invalid
bytes: all returned code 11 at `binding`, reached neither secp nor ML
verification (observed fraction 0/168 in every attack arm), were absent from
blocks and left the valid-state oracle intact. Every arm had its expected
73/241 valid exact-once commits in attack/benign conditions and all-four
BlockID/AppHash agreement.

The registered fixed-legitimate 3,000-ms all-four observed-inclusion
endpoint was 72/72 in every arm. All 12 paired benign-minus-attack effects
were zero: mean **0.000**, 95% paired-bootstrap CI **[0.000, 0.000]**.
This is no positive service-degradation effect at the tested endpoint and
20-tx/s rate. The all-four timing is an observation upper bound.

| Descriptive metric | Benign | A02 attack |
| --- | ---: | ---: |
| Fixed legitimate inclusion upper bound, mean over 864 tx | 1,118.89 ms | 1,170.86 ms |
| Maximum fixed legitimate inclusion upper bound | 1,913.12 ms | 2,055.49 ms |
| Shared application-process CPU / 241 offered attempts | 8,454.36 µs | 5,383.82 µs |
| Invalid binding CheckTx wall service, mean of 12 run means | — | 9.27 µs/tx |
| Binding span wall time, mean of 12 run means | — | 1.91 µs/tx |

The A02 invalid CheckTx service mean is descriptively 2.02 µs greater than
the A01 structural-invalid mean (7.25 µs/tx). This cross-case comparison
uses separate fresh networks and is **not** a paired CPU effect. Process
CPU is shared with the harness, so class-specific CPU/tx is `UNAVAILABLE`;
wall spans are not labeled CPU. CheckTx socket queue and ingress/mempool
samples remain available in immutable raw; the queue represents only
Comet-to-application socket wait.

`python3 research/tools/file02_attack.py run A02` exited 0 with 24/24
QC PASS; `python3 research/tools/file02_attack.py verify
research/artifacts/file02/a02-20261113T171854Z-5021cdef` exited 0 with
`ATTACK_VERIFY=OK`. Private plan, per-run JSON/CSV, QC, aggregate, figure
status and decision are preserved. **A02 PASS for binding rejection and
bounded negative service-endpoint test.** It does not establish a knee,
DoS, P2P-origin or multi-host effect.
