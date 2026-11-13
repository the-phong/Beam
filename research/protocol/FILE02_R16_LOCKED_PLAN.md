# R16 — matched legitimate burst, ramp and sustained control

Status: **LOCKED BEFORE R16 EXECUTION**, 2026-11-13. Four independent
logical fixture blocks are each replayed across C/CP/P/H and three
waveforms, one fresh four-validator single-host network per arm: 48
networks. Within each profile/block, the same 41 funded sender slots,
nonce-zero transfers, genesis, state, endpoint role, bytes/transaction,
collector, safety and network configuration are used. Arms differ only
in scheduled offsets. Profile order rotates; arm order follows a recorded
Latin rotation. All sends are valid, from validator-0 loopback RPC.

Each arm sends one warm-up transaction and 40 steady transactions across
four one-second windows, followed by three seconds drain and two seconds
cooldown. Sustained has counts `[10,10,10,10]`; ramp `[4,8,12,16]`;
burst `[0,20,0,20]`. Thus every arm averages 10 offered tx/s and has
equal total encoded bytes for its profile; the burst peak is 20 tx/s.
All windows remain below 25 tx/s and 65,536 B/s. The unchanged S06
60-second, concurrency, CPU/RSS/disk, observer and loopback guards apply.
No invalid or adversarial traffic is present.

Per-arm COMMON plans bind actual endpoints, source, config, genesis,
seed, order and signed corpus before sends. QC requires every scheduled
send in its intended window, each planned-zero burst window empty,
40/40 steady valid transactions committed once across all four
validators, same-height BlockID/AppHash and correct state, zero recorder
and observer drops, no safety abort, and immutable raw integrity. A
failed arm excludes its matched block; no outcome-driven replacement.

Report per-run sampled accepted mempool count/bytes and P2P relay
ingress bytes/s, CheckTx socket queue wait, 100-ms inclusion latency
**upper bound**, and time-to-recover **upper bound** from the last send
to the first subsequent observer sample with zero accepted mempool count.
If no such sample exists, recovery is `null`/`UNAVAILABLE`. Neither
sampled count nor socket wait represents total transaction queue depth
or residence time. `false-alert candidates` are descriptive benign
signals where burst/ramp exceeds matched sustained peak; no detector
or false-positive rate is evaluated in R16.

The primary paired comparison is burst minus sustained maximum sampled
accepted mempool bytes per block/profile. Report all four differences,
their mean and a two-sided percentile bootstrap 95% interval from
20,000 block resamples with seed 20261113. Secondary ramp comparisons
and recovery bounds are descriptive, with the same independent block
uncertainty where available. No knee, DoS, P2P-origin, multi-host or
RQ2 detector inference follows from these benign controls.
