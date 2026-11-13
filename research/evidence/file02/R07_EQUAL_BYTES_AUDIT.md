# R07 — matched encoded-byte-rate network audit

Date: 2026-11-13. The [R07 design](../../protocol/FILE02_R07_LOCKED_PLAN.md)
was committed at `ba5e5e2` before execution. Twelve independent blocks
each used a newly generated common logical fixture and four fresh
four-validator single-host networks. The profile order was counterbalanced.
All traffic was valid and sent by RPC to validator 0 on loopback; no attack
load or P2P-origin injection occurred. The target was 2,568 encoded B/s
for each of four steady one-second windows. Planned and achieved valid rates
were C/CP/P/H = 17/1/1/1 tx/s and bytes/s =
2,516/2,568/2,504/2,568. Maximum target deviation was 2.49%, below the
preregistered ±5% tolerance. CP and H matched exactly in both bytes/s and
tx/s; C and P did not match their transaction rates to CP/H.

Private session:
`research/artifacts/file02/r07-20261113T041958Z-15a73906`.
Top plan SHA-256:
`5fadc189f1b77baf624d26afe9591f56693d712e688f03800f4e948c18938bdd`.
Immutable raw manifest SHA-256:
`ebc6bd3216429395a0c67e742e5b32f8aefeeb137a2f193fe3348bebb5f4138f`.
Private artifacts include the top locked plan, per-network COMMON-schema
`research_plan/plan.json` validated before sends, executable load plans,
raw telemetry and block oracles, `result_per_run.json/csv`, `qc_report.json`,
`result_aggregate.json`, figure status and decision. Integration exited 0;
all 48 arms passed QC with no replacements. Every planned send succeeded,
every valid transaction was included exactly once and four validators agreed
on BlockID/AppHash. Every required CheckTx span joined by transaction hash;
recorder and observer drops were zero. Independent
`python3 research/tools/file02_r07.py verify
research/artifacts/file02/r07-20261113T041958Z-15a73906` exited 0.

| Profile | Achieved B/s | Mean CheckTx service, µs/tx | Mean RPC response, ms | Mean inclusion upper bound, ms | Mean shared app CPU delta, s | Mean shared app RSS peak, MB | Mean max accepted mempool B | Mean validator-0 relay-send envelope B |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| C | 2,516 | 184.98 | 0.812 | 1,136.5 | 0.404 | 91.2 | 4,156 | 31,416 |
| CP | 2,568 | 181.83 | 0.784 | 1,097.5 | 0.213 | 91.3 | 2,782 | 30,888 |
| P | 2,504 | 116.33 | 0.669 | 1,107.8 | 0.217 | 90.8 | 3,130 | 30,120 |
| H | 2,568 | 283.96 | 0.868 | 1,110.5 | 0.228 | 91.0 | 2,782 | 30,888 |

The twelve paired block differences in mean CheckTx service gave H−CP
**+102.13 µs/tx**, 95% paired bootstrap CI **[+66.67, +148.84] µs/tx**,
20,000 preregistered resamples. This is a low-rate, exact-size H/CP contrast
within the matched-byte study, not a capacity result. CP/P/H have only four
steady transactions per arm (48 total/profile), and C has 68 per arm (816
total). Their optimistic rank coverage for p99 is only 0.302 and 0.838,
respectively, below the registered 0.95 gate; p99 is
`INSUFFICIENT_PRECISION`, not a made-up value. Inclusion times are
block-scan observation upper bounds. Eventual inclusion does not establish
sustainable steady-state throughput. Shared app CPU covers all four apps
on one host; relay envelope counters do not measure P2P-origin propagation.

**R07 technical conditions pass within the preregistered ±5% byte
tolerance.** This controls offered encoded bytes, while C/P comparisons
still change transaction count and verification path. It does not establish
a knee, distributed-network effect, P2P attack or DoS.
