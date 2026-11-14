# A14 — one-validator process fault, locked before execution

Status: LOCKED DESIGN, 2026-11-14. This is a same-host **process-fault
diagnostic**, not a multi-host fault-domain experiment. Four independent
fixture blocks use balanced control/fault order on eight fresh local
four-validator networks. Each arm submits the same 25 distinct valid H
transactions (one warm-up, 6/s for four steady seconds) to validator-0
loopback RPC; all bytes, offsets, genesis and seed match within a pair.
No invalid transaction or attacker-origin traffic is involved.

The fault arm suspends only validator-3's CometBFT process using SIGSTOP
about 2,000 ms after load-process launch and resumes it with SIGCONT about
1,500 ms later. The corresponding control arm has no signal. The harness
records actual signal times and per-node RPC height snapshots before,
during and after; each transaction's first block-index observation on every
validator is polled at 100 ms. A finally-style SIGCONT prevents leaving
the test process suspended on measurement failure. The application
instances remain in one integration process on the same physical host;
CPU and network are shared, so physical fault isolation is not claimed.

Primary metric: for each arm, median of validator-3 first-observation
upper-bound latency minus median of validators 0–2, then fault-minus-control
across four paired blocks. Use a two-sided 20,000-resample percentile
bootstrap with seed 20261114. Secondary: healthy-node progress during the
pause, local versus all-four observation upper bounds, final common height,
BlockID/AppHash/state agreement and the complete signal/height trace.
The observation timestamp is **not** the exact consensus commit time.

QC requires 25/25 accepted and exact-once commits, no invalid acceptance,
no safety abort, all four eventual observations, source/config/corpus and
matched-pair hashes, successful SIGSTOP/SIGCONT only in the fault arm,
unchanged 25 tx/s, 65,536 B/s and 60-second safety caps, intact immutable
raw, no recorder drops and no replacement of a failed pair. A process-fault
effect is reported even if negative, but no distributed resilience or DoS
claim follows. The 4-run CI is exploratory and may be wide.
