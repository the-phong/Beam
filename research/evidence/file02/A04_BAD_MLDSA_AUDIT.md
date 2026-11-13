# A04 — corrupted ML-DSA proof audit

Date: 2026-11-14. The [locked plan](../../protocol/FILE02_A04_LOCKED_PLAN.md)
and stage oracle were committed at `9d4fd07` before execution. Private
session: `research/artifacts/file02/a04-20261113T174032Z-edd720e5`.
Plan SHA-256:
`a63f5a33ed86cfb84c425b13045c97b2d6d8295260b7d943d18f4a608c21182a`;
immutable raw manifest SHA-256:
`41d2bf5bef4aef6cc3c01720710a80b84704952da2c6458e3c0f68194c29c37d`.

All 12 matched pairs and 24 new four-validator single-host loopback-RPC
networks passed QC and independent raw verification. Each arm offered 20
H-sized tx/s (51,360 B/s) with identical six fixed legitimate H tx/s per
pair. Actor B modified an ML-DSA proof byte of an otherwise valid signed H
transaction for the 14 attack-background tx/s. All 168 invalid sends per
attack arm reached both secp and ML spans (fraction 1.0), failed with code
13 at `ml`, never committed and left the valid-state oracle intact. Valid
transactions committed exactly once; BlockID/AppHash agreed on all four
validators.

The registered 3,000-ms all-four observed-inclusion endpoint was 72/72
for fixed legitimate traffic in every arm. All 12 paired benign-minus-attack
effects were zero: mean **0.000**, paired-bootstrap 95% CI
**[0.000, 0.000]**. No positive endpoint degradation was detected at the
tested 20 tx/s rate. The observed inclusion time is an upper bound, not
exact commit latency.

| Descriptive metric | Benign | A04 attack |
| --- | ---: | ---: |
| Fixed legitimate inclusion upper bound, mean over 864 tx | 1,126.21 ms | 1,182.89 ms |
| Maximum fixed legitimate inclusion upper bound | 1,961.71 ms | 2,139.44 ms |
| Shared application-process CPU / 241 offered attempts | 8,772.48 µs | 5,847.16 µs |
| Invalid CheckTx wall service, mean of 12 run means | — | 391.31 µs/tx |
| secp wall span, mean of 12 run means | — | 171.14 µs/tx |
| ML wall span, mean of 12 run means | — | 49.48 µs/tx |

The prespecified directional cost hypothesis is supported **descriptively**:
A04 invalid CheckTx service run means ranged 370.36–410.06 µs/tx; A01
structural-invalid run means ranged 5.63–10.34 µs/tx, with an A01 mean of
7.25 µs/tx. These are independent case networks and wall-service durations,
not paired CPU costs or proof of overload. Class-specific CPU/tx is
`UNAVAILABLE` because process CPU is shared with the harness. CheckTx socket
wait is separately measured in raw and does not represent the entire
transaction queue.

`python3 research/tools/file02_attack.py run A04` exited 0 with QC PASS;
`python3 research/tools/file02_attack.py verify
research/artifacts/file02/a04-20261113T174032Z-edd720e5` exited 0 with
`ATTACK_VERIFY=OK`. Plan, raw, per-run JSON/CSV, QC, aggregate, figure
status and decision are preserved privately. **A04 PASS for the bounded
late-ML rejection mechanism and negative service-endpoint test.** No knee,
DoS, P2P-origin or multi-host conclusion follows.
