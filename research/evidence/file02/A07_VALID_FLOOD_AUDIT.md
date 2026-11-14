# A07 — bounded valid-load pressure audit

Date: 2026-11-14. The [locked plan](../../protocol/FILE02_A07_LOCKED_PLAN.md)
and source were committed at `a706024` before A07 execution. Private
session: `research/artifacts/file02/a07-20261113T182203Z-ea676645`.
Plan SHA-256:
`6392b7513fd417e78ec7b364d557d32b06ae04c139c16ffb4dd227a8156c2524`;
immutable raw manifest SHA-256:
`d1ca24660207b5e80c1acf89b2468e716639048ce62f9155dc09e381ac767500`.

Four matched fixture blocks used eight fresh four-validator single-host
loopback-RPC networks. The order of 20 and 25 H tx/s was balanced. Each
arm had six byte/hash/schedule/genesis-identical fixed legitimate H tx/s;
the control added 14 distinct valid H tx/s and Actor C's valid-flood arm
added 19. All 2,568-byte transactions were correctly signed by registered
funded accounts. Every one-second rate window, the 51,360/64,200 B/s
envelope, 81/101 exact-once valid commits, four-validator BlockID/AppHash
and account-state oracles passed. No invalid-verification workload occurred.

The registered 3,000-ms all-four fixed-legitimate inclusion endpoint was
24/24 for every arm. The paired rate-25 minus rate-20 effect was **0.000**,
95% paired-block bootstrap CI **[0.000, 0.000]**. No degradation of this
endpoint was observed through the tested 25-tx/s safe cap. The inclusion
measurement is an observation upper bound, not exact commit time.

| Descriptive metric | 20 tx/s control | 25 tx/s valid flood |
| --- | ---: | ---: |
| Fixed legitimate inclusion upper bound, mean over 96 tx | 1,160.50 ms | 1,108.50 ms |
| Max fixed legitimate inclusion upper bound | 1,878.29 ms | 1,866.03 ms |
| Sampled validator-0 accepted mempool bytes, median peak | 82,176 B | 87,312 B |
| Validator-0 P2P relay send envelope bytes, mean run delta | 617,760 B | 772,200 B |
| Eventually committed steady tx / four-second offered window | 20 tx/s | 25 tx/s |

Relay send is an envelope counter, not exact application-payload relay
latency. Some relay-receive samples are unavailable; no missing value was
replaced with zero. The goodput quotient counts all steady valid tx that
eventually committed after the bounded drain and is **not** sustainable
steady-state throughput. Four short blocks do not establish p99 precision,
a knee or a capacity boundary; fixed traffic remained on time under both
arms. The 25-tx/s arm was within the unchanged 25 tx/s, 65,536 B/s,
16-outstanding and 60-second safety caps.

`python3 research/tools/file02_a07.py run` exited 0 with 8/8 QC PASS;
`python3 research/tools/file02_a07.py verify
research/artifacts/file02/a07-20261113T182203Z-ea676645` exited 0 with
`A07_VERIFY=OK`. Private plan, per-run JSON/CSV, QC, aggregate, figure
status and decision remain preserved. **A07 PASS for bounded valid-load
pressure and negative 3-second service-endpoint test.** It is not an
invalid-verification-waste or DoS result, and it does not support a
P2P-origin or multi-host conclusion.
