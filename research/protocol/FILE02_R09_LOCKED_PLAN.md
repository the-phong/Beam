# R09 — mempool byte constraint under valid load

Status: **LOCKED DESIGN BEFORE EXECUTION**, 2026-11-13. R08 was completed
first. Four independent blocks each reuse one newly generated 81-sender
logical fixture across C/CP/P/H and two byte-cap arms. Every arm starts a
fresh four-validator single-host network. The baseline has
`mempool.max_txs_bytes=8,388,608`; the constrained arm has
`mempool.max_txs_bytes=2,700`. Both keep `mempool.size=128`,
`max_tx_bytes=8,192` and all S06 safety caps unchanged. Profile order
rotates; arm order alternates across blocks.

Each arm sends one warm-up and 80 steady distinct valid nonce-zero transfers
at 20 tx/s for four seconds by validator-0 RPC loopback, followed by three
seconds drain and two seconds cooldown. No invalid workload is sent.
For chain ID `pluto-file02-r09`, valid C/CP/P/H transactions are
148/2568/2504/2568 B. The 2,700-B pool can hold at most 18 C
transactions or one CP/P/H transaction at an instant, if no drain occurs.
The actual limit is observed by first bounded RPC
`mempool_bytes_full` error; `null` means the byte limit was not observed.
The classifier checks current bytes **plus the new transaction size**
against the byte cap. Do not mistake current bytes below the cap for
evidence that the byte cap was not binding. The earliest classified RPC
completion offset is a bound on onset, not Comet's internal crossing instant.

Per-arm COMMON plans and executable load plans bind actual endpoints,
source, config, genesis and signed corpus before sends. QC requires all
81 planned socket writes, no safety or rate abort, only valid transactions,
every RPC-accepted transaction committed exactly once, every byte-full
rejected transaction absent from blocks, four validators live with
same-height BlockID/AppHash, and no observer or recorder drops. Baseline
must show no byte-full; constrained arms should show byte-full before the
128-transaction count limit. Report per-run accepted count/bytes, sampled
mempool byte curves and drains, first byte-full offset and rejection reason.
Use four matched blocks for paired accepted-count differences and 20,000
percentile bootstrap draws with seed 20261113. A negative or incomplete
boundary remains recorded as such, without replacement. This is a
controlled limit probe, not a knee or DoS experiment.
