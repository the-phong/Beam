# R08 — valid-only mempool count limit

Date: 2026-11-13. The [locked R08 plan](../../protocol/FILE02_R08_LOCKED_PLAN.md)
was committed at `6aec8b2` before execution. Four independent blocks
shared one newly generated logical fixture across each profile and arm;
every arm ran a fresh four-validator single-host network. Only the Comet
mempool `size` field changed between `baseline=128` and `count1=1`.
`max_txs_bytes=8,388,608`, `max_tx_bytes=8,192` and all S06 safety caps
were identical. Each network offered 81 distinct valid nonce-zero transfers
(one warm-up, 20/s for four steady seconds) by validator-0 loopback RPC.
No invalid workload was sent.

Private session: `research/artifacts/file02/r08-20261113T044055Z-59569ce0`.
Locked top plan SHA-256:
`778d7c0c3fdc097e5272e52572d889f3e3e42257930b39b5da06c7471b00a222`.
Immutable raw manifest SHA-256:
`ef888b142affaa2a6b75439707d96313768967025411a35c423f886bf20413da`.
Per-arm COMMON plans, executable load plans, actual private network configs,
RPC results, recorder/observer streams and block oracles are in the frozen
raw. `result_per_run.json/csv` contains every accepted count/byte count,
count-full onset, reason and sampled mempool count/byte curve;
`qc_report.json`, `result_aggregate.json`, figure status and `decision.md`
are separate derived artifacts. Integration exited 0; all 32 arms passed
QC with no replacement. Independent
`python3 research/tools/file02_r08.py verify
research/artifacts/file02/r08-20261113T044055Z-59569ce0` exited 0.
All planned socket writes occurred at 20/s; no safety or generator abort,
observer drop or unsupported RPC error occurred. Every RPC-accepted valid
transaction committed exactly once, every count-full rejection was absent
from blocks, and all four validators agreed on the common-height
BlockID/AppHash. The bounded RPC classifier recorded
`mempool_count_full`, rather than treating a missing CheckTx code as an
application signature failure.

| Profile | Baseline accepted, each of 4 runs | `size=1` accepted | `size=1` valid rejected | First count-full onset, ms from run start | Mean `size=1` sampled mempool bytes |
| --- | ---: | --- | --- | --- | ---: |
| C | 81 | 4, 5, 5, 4 | 77, 76, 76, 77 | 1,048.8, 1,095.6, 1,095.6, 1,048.5 | 148 |
| CP | 81 | 5, 5, 4, 4 | 76, 76, 77, 77 | 1,096.4, 1,096.0, 1,047.8, 1,048.9 | 2,568 |
| P | 81 | 5, 5, 5, 4 | 76, 76, 76, 77 | 1,096.1, 1,095.9, 1,095.7, 1,048.6 | 2,504 |
| H | 81 | 4, 4, 5, 5 | 77, 77, 76, 76 | 1,048.7, 1,047.9, 1,095.8, 1,095.5 | 2,568 |

The paired baseline-minus-`size=1` accepted-count difference was 76.5
transactions for C/CP/H, 95% four-block percentile-bootstrap CI [76, 77];
for P it was 76.25, CI [76, 76.75]. The repeated boundary is largely
determined by the deliberately tiny count cap; these CIs describe this
controlled setting, not general capacity. At count-full onset the sampled
mempool bytes were far below 8 MiB, so the **count constraint bound first**.
The 1-second observer cadence may miss the exact moment the pool became
full; onset here is the first classified RPC rejection, not the internal
instant of threshold crossing. Total accepted count across the run exceeds
one because blocks drain the pool between sends. This case does not locate
a benign capacity knee or establish DoS.

**R08 technical conditions pass.**
