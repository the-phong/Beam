# R08 — mempool count constraint under valid load

Status: **LOCKED DESIGN BEFORE EXECUTION**, 2026-11-13. This case follows
R07. Four independent blocks each reuse one newly generated 81-sender
logical fixture across C/CP/P/H and two mempool-count arms; every arm starts
a fresh four-validator single-host network. The baseline uses
`mempool.size=128`; the constraint arm uses `mempool.size=1`.
`mempool.max_txs_bytes=8,388,608` and `max_tx_bytes=8,192` remain equal in
both arms. This changes only the requested count constraint; S06 safety
caps stay byte-for-byte unchanged. Profile and arm order rotate over blocks.

Each arm sends one warm-up and 80 steady distinct, valid nonce-zero transfers
at 20 tx/s for four one-second windows via validator 0 RPC loopback, with
three seconds drain and two seconds cooldown. C/CP/P/H wire sizes are
148/2568/2504/2568 B for chain ID `pluto-file02-r08`, all within the
unchanged 65,536 B/s cap. The load generator records HTTP write times,
completion times, application codes and a bounded classification of Comet
RPC mempool-full errors. Exact earliest count-full onset is the first
`mempool_count_full` RPC completion offset from the scheduled run start;
`null` means it was not observed in that arm. Do not convert missing onset
to zero. Accepted count/bytes, rejection count/reason and sampled accepted
mempool count/bytes curves are secondary. Independent unit is one fresh
network; report each arm, a four-block paired summary and 95% bootstrap CI
where meaningful. No outcome-based arm replacement.

Per-arm COMMON plan and executable load plan are locked before sends with
actual endpoints, genesis/corpus, source and Comet config hashes. QC requires
all 81 planned socket writes, no safety/rate abort or observer drop, only
valid transactions, every RPC-accepted transaction committed exactly once,
every count-full rejected transaction absent from blocks, four validators
live and same-height BlockID/AppHash. The baseline must not show count-full;
the constrained arm must show count-full before the byte cap can bind.
Failure to reach count-full is a measured negative result, not a zero onset.
This is a controlled limit probe, not a capacity knee or DoS experiment.
