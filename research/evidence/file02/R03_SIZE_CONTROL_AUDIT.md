# R03 — C/CP size-only control, partial disposition

Date: 2026-11-13. The [R03 plan](../../protocol/FILE02_R03_LOCKED_PLAN.md)
and producer were committed at `9002dcb` before a new private `LOCKED`
plan and twelve fresh direct-ABCI process runs. Every process used a fresh
genesis, 240 valid CheckTx calls per profile after warm-up, the same
logical intents and counterbalanced profile order. C and CP both use
secp256k1 only; canonical CP padding adds exactly 2,420 wire bytes. QC
passed 12/12 processes with all 240 C and CP observations, expected stage
sets and zero recorder drops. The locked plan, immutable raw, per-run
JSON/CSV, QC, aggregate, figure-status and decision artifacts are in
`research/artifacts/file02/r03-20261113T032100Z-9002dcb` (private).
Plan SHA-256:
`b26bb56fe89561a3bce973aba8661d1cb0992c43f5856d195fafdfc084336d8e`.
Raw manifest SHA-256:
`3ceb0e73edee17e733d4f785cc98832a793f988bde12d1de290bb55e7398579b`.
Independent re-verification with `python3 research/tools/file02_r03.py
verify research/artifacts/file02/r03-20261113T032100Z-9002dcb`
exited 0.

CP minus C was exactly **+2,420 B/tx**. The paired mean structural parse
difference was **+724 ns/tx**, 95% process-level bootstrap CI
**[+635, +834] ns/tx**. The full CheckTx difference was **−1.38 µs/tx**,
CI **[−7.93, +5.23] µs/tx**; this interval crosses zero. Process CPU
difference was **−1.24 µs/tx**, CI **[−7.73, +5.31] µs/tx**. Mean allocated
bytes and allocation counts were nearly equal in this direct path; exact
per-run values and ranges are in the aggregate. The sign of the full
service/CPU comparison is not established by these data. Structural parse
cost changed with canonical size, but no full-path effect is attributed
solely to transport.

**R03 is PARTIAL, not PASS.** Exact P2P relay propagation latency remains
`UNAVAILABLE` because current RPC-origin telemetry lacks matched peer
send/receive timestamps. RPC latency and block observation cannot fill that
field. The direct application comparison is a valid bounded measurement,
but the checklist's relay output is missing. No RQ1 or DoS conclusion follows.
The R03 box remains unchecked. R04 is next.
