# R14 — CheckTx recheck under valid state-changing commits

Date: 2026-11-13. Source and protocol were locked at `a97825b` before
execution. The private session is
`research/artifacts/file02/r14-20261113T073256Z-6a4d335c`; top-level
plan SHA-256 is
`3266da302cc6d5b00a224e1749c7583cf3f41d3b7925b8ead0f8681087ecb48f`.
The immutable raw manifest SHA-256 is
`5378c732f2e368ac9bb7dd0b30b8efdbe0bf1617a4ed2e7367e79da38cdc71e0`.
`python3 research/tools/file02_r14.py run` exited 0 (integration 0,
QC PASS); independent `python3 research/tools/file02_r14.py verify
research/artifacts/file02/r14-20261113T073256Z-6a4d335c` exited 0,
rechecking the manifest and deriving every output from raw.

Sixteen fresh four-validator single-host networks covered four rotated
blocks × C/CP/P/H. Each sent one warm-up and 80 steady distinct valid
transfers via validator-0 loopback RPC at 20 tx/s; the accepted
`mempool_recheck=true` configuration and all S06 caps were unchanged.
Every network delivered all 81 sends, committed all 81 exactly once,
kept four-validator same-height BlockID/AppHash agreement, met each
one-second rate window, and had zero observer/recorder drops. The private
bundle contains locked per-run plans, immutable raw, per-run JSON/CSV,
QC, aggregate with run-level CI, sampled mempool count/byte curves,
figure status and decision.

| Profile | NEW/run | RECHECK counts across four runs | Mean RECHECK/run | 95% bootstrap CI | RECHECK result codes |
| --- | ---: | --- | ---: | --- | --- |
| C | 81 | 26, 28, 24, 22 | 25.00 | [23.00, 27.00] | 100 code 0, 0 reject |
| CP | 81 | 26, 25, 21, 22 | 23.50 | [21.50, 25.50] | 94 code 0, 0 reject |
| P | 81 | 23, 27, 24, 25 | 24.75 | [23.50, 26.25] | 99 code 0, 0 reject |
| H | 81 | 27, 24, 19, 25 | 23.75 | [20.50, 26.25] | 95 code 0, 0 reject |

All recheck transaction hashes matched a planned valid transaction. The
Comet cumulative recheck counter matched the application recorder count
in every network. Sampled accepted mempool count and bytes were retained
per run and drained after sends; this is a sampled queue evolution, not
an exact queue residence time. The rechecks occurred while the chain
committed valid transfers and pending transactions remained in the
mempool. Because senders had independent nonce-zero accounts, their
rechecks stayed valid after other accounts changed. The exact triggering
commit for each recheck and per-event Comet CPU are `UNAVAILABLE`.

**R14 PASS in the bounded RPC-origin single-host stratum.** Recheck events
are internal validation work and must not be counted as new offered
transactions or a new attack. This case does not establish a knee, a
sustainable rate or DoS.
