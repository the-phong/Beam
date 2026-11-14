# A08 — preregistered mixed-invalid workload audit

Date: 2026-11-14. The [locked plan](../../protocol/FILE02_A08_LOCKED_PLAN.md),
producer and stage audit were committed at `1786e94` before A08 sending.
Private session: `research/artifacts/file02/a08-20261113T182741Z-0dff8ab6`.
Plan SHA-256:
`01c7cb7ec6fbe51b2653d8027a3a0fd233c9834404c001c1ef21265c767568a1`;
immutable raw manifest SHA-256:
`18eed7952c91b5cff08a418055a0a742ffaeb3f70e06f94f115f4e99e5ad8b4d`.

All 12 matched pairs and 24 new four-validator single-host loopback-RPC
networks passed QC and independent raw verification. Each arm offered
20 H-sized writes/s or 51,360 B/s and the same six fixed legitimate H tx/s
per pair. Each attack second contained exactly four structural-invalid,
five bad-ML and five fully signed state-invalid transactions, giving
48/60/60 class events per attack arm. Actor capability is recorded per
event: arbitrary bytes for structural, signed-transaction modification for
bad ML and registered signing keys for nonce-invalid. All invalid events
failed at their registered code/stage and never committed; valid transactions
committed once with four-validator BlockID/AppHash and state agreement.

Every arm observed all 72 fixed legitimate transactions included on all
four validators by the registered 3,000-ms upper-bound deadline. The
12 paired benign-minus-attack effects were zero: mean **0.000**, 95% CI
**[0.000, 0.000]**. No positive degradation of the registered endpoint was
observed at 20 tx/s. The fixed inclusion upper-bound mean was
1,130.32 ms in benign arms and 1,210.40 ms in mixed attack arms; these
descriptive transaction means do not substitute for the preregistered
independent-pair endpoint. Maximum upper bounds were 1,960.70 and
2,139.10 ms, respectively.

| Attack class | Count per attack arm | Mean CheckTx wall service over 12 run means |
| --- | ---: | ---: |
| Structural invalid | 48 | 6.00 µs/tx |
| Bad ML proof after valid secp | 60 | 381.99 µs/tx |
| Fully signed nonce-invalid after both proofs | 60 | 417.60 µs/tx |

Per-run JSON contains the three class means and the validator-0 sampled
resource curves (at least 19 samples/arm) for accepted mempool bytes,
application-process CPU and CheckTx socket-queue cumulative wait. Median
sampled accepted-mempool peak was 91,164 B in benign arms and 25,680 B in
attack arms; the valid benign background accounts for much of this
difference. Eventual fixed legitimate goodput was 6 tx/s in both arms;
eventual total valid goodput was 20/6 tx/s in benign/attack arms. These
four-validator short-run counts do not establish sustainable throughput.
The p99 endpoint is `null` with status `INSUFFICIENT_PRECISION`; no precise
p99 or alert lead time is claimed. Whole-process CPU is shared with the
harness and cannot be attributed as per-class CPU.

`python3 research/tools/file02_attack.py run A08` exited 0 with 24/24
QC PASS; `python3 research/tools/file02_attack.py verify
research/artifacts/file02/a08-20261113T182741Z-0dff8ab6` exited 0 with
`ATTACK_VERIFY=OK`. Private plan, immutable raw, per-run JSON/CSV, QC,
aggregate, figure status and decision remain preserved. **A08 PASS for
bounded mixed-class mechanism and negative 3-second service-endpoint test.**
No knee, DoS, P2P-origin or multi-host conclusion follows.
