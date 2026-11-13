# A03 — corrupted ECDSA signature audit

Date: 2026-11-14. The method and stage oracle were committed at `9d4fd07`
before A03 execution. Private session:
`research/artifacts/file02/a03-20261113T172941Z-5a42d176`.
Plan SHA-256:
`678405eb5642f8027bd129ba9dfc3e6ee7fd94fe2168bb5348601eaf90f80903`;
immutable raw manifest SHA-256:
`46855a3034478e02dc49e20d1815b4e47da18e21dd24fc95e92b0ad9f268c9a1`.

All 12 matched pairs, 24 fresh four-validator single-host networks and
per-run oracles passed QC. Every arm offered 20 H-sized tx/s (51,360 B/s)
with the same six fixed legitimate H tx/s in each pair. The attack background
was 14 distinct H transactions/s whose classical signature byte was
altered after signing. Actor B needs a signed transaction to modify but
does not need the signer's secret key. All 168 invalid transactions per
attack arm failed with code 4 at `secp`, after structural and binding stages;
the observed ML-reach fraction was exactly 0. No invalid transaction
committed; all valid transactions committed once with all-four BlockID/
AppHash agreement.

All arms observed 72/72 fixed legitimate transactions on all four validators
within the registered 3,000-ms upper-bound deadline. The paired
benign-minus-attack mean was **0.000**, 95% CI **[0.000, 0.000]** across 12
independent pairs. No positive degradation of this endpoint was detected at
20 tx/s. This endpoint is a sampled inclusion upper bound, not exact commit
time.

| Descriptive metric | Benign | A03 attack |
| --- | ---: | ---: |
| Fixed legitimate inclusion upper bound, mean over 864 tx | 1,122.21 ms | 1,174.92 ms |
| Maximum fixed legitimate inclusion upper bound | 2,011.64 ms | 2,024.71 ms |
| Shared application-process CPU / 241 offered attempts | 8,865.84 µs | 5,791.84 µs |
| Invalid CheckTx wall service, mean of 12 run means | — | 179.21 µs/tx |
| secp span wall time, mean of 12 run means | — | 166.31 µs/tx |

The secp span is wall-clock duration within the CheckTx implementation, not
isolated verifier CPU time. Class-specific CPU is `UNAVAILABLE`; the process
CPU quotient includes the integration harness and all transaction classes.
No ML invocation after secp failure was observed. The comparison cannot
establish a saturated service regime or a DoS mechanism.

`python3 research/tools/file02_attack.py run A03` exited 0 with QC PASS;
`python3 research/tools/file02_attack.py verify
research/artifacts/file02/a03-20261113T172941Z-5a42d176` exited 0 with
`ATTACK_VERIFY=OK`. Private plan, raw, per-run JSON/CSV, QC, aggregate,
figure status and decision remain preserved. **A03 PASS for bounded stage
oracle and negative service-endpoint test.** No knee, P2P-origin or
multi-host conclusion follows.
