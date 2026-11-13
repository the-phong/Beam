# R09 — valid-only mempool byte limit

Date: 2026-11-13. The [locked R09 plan](../../protocol/FILE02_R09_LOCKED_PLAN.md)
was committed at `cce97e1` before execution. Four independent blocks, each
with one logical fixture across C/CP/P/H and two arms, ran 32 fresh
four-validator single-host networks. Only `mempool.max_txs_bytes` changed:
8,388,608 B baseline versus 2,700 B constrained. `mempool.size=128`,
`max_tx_bytes=8,192` and all S06 safety caps stayed equal. Each arm sent
81 distinct valid transfers (one warm-up plus four seconds at 20/s) by
validator-0 loopback RPC. No invalid workload was sent.

Private session: `research/artifacts/file02/r09-20261113T070941Z-a6908879`.
Top locked plan SHA-256:
`97aa5af0d04ce40a720dc74b9b6ffcd7cbfd31c21f5e3d73bbca1bf79325d754`.
Immutable raw manifest SHA-256:
`aafca1a52bdfaf7f544cd3215f0af5659a372ea4c7e9b21efadcd40ad6e592af`.
The frozen raw has the pre-send COMMON plans, executable plans, actual
configs, RPC results, observer/recorder streams and four-validator oracles.
Per-run JSON/CSV, QC, aggregate, figure status and decision are private
derived artifacts. Integration exited 0; 32/32 arms passed QC without
replacement. Independent `python3 research/tools/file02_r09.py verify
research/artifacts/file02/r09-20261113T070941Z-a6908879` exited 0.
Every planned socket write occurred; every accepted valid transaction
committed once; byte-full rejections did not commit; and all four validators
agreed on common-height BlockID/AppHash. There was no safety abort,
observer drop, count-full rejection or unsupported RPC error. The bounded
RPC classifier checked current pool bytes plus the new transaction size.

| Profile | Baseline accepted, each of 4 runs | 2,700-B arm accepted | Valid byte-full rejected | First byte-full RPC onset, ms | Mean peak sampled pool B |
| --- | ---: | --- | --- | --- | ---: |
| C | 81 | 54, 55, 54, 57 | 27, 26, 27, 24 | 1,905.3, 1,905.2, 2,095.7, 1,905.2 | 2,664 |
| CP | 81 | 4, 4, 5, 5 | 77, 77, 76, 76 | 1,049.0, 1,048.4, 1,096.4, 1,096.3 | 2,568 |
| P | 81 | 5, 4, 5, 5 | 76, 77, 76, 76 | 1,096.2, 1,096.4, 1,095.8, 1,096.5 | 2,504 |
| H | 81 | 5, 4, 5, 5 | 76, 77, 76, 76 | 1,096.5, 1,048.7, 1,096.0, 1,096.2 | 2,568 |

The baseline-minus-constrained accepted-count means and paired 95%
four-block bootstrap CIs were C 26 [24.75, 27], CP 76.5 [76, 77],
P 76.25 [76, 76.75], H 76.25 [76, 76.75]. C's smaller canonical wire
size lets more valid transfers fit the same 2,700-B pool; CP/P/H fit only
one at a time before drain. Counts across a run exceed simultaneous pool
capacity because committed blocks drain the pool. Observer samples every
second, so the first classified RPC rejection bounds onset rather than
giving the exact internal threshold crossing. The byte constraint bound
before the count cap of 128. This does not establish a capacity knee or DoS.

**R09 technical conditions pass.**
