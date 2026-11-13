# R15 — bounded benign capacity ramp

Status: **LOCKED BEFORE R15 EXECUTION**, 2026-11-13. This is the
authorized formal R15 case, separate from T06's historical calibration.
Use 20, 23 and 25 **valid-only** tx/s in ascending rate order for C, CP,
P and H. Each rate/profile cell has three fresh four-validator single-host
networks, 36 networks total. Within a repetition rotate profile order.
Each run uses one warm-up second, twelve steady seconds, three drain
seconds and two cooldown seconds, one distinct funded nonce-zero sender
per transaction, validator-0 loopback RPC and unchanged S06 caps. The
maximum CP/H 25 tx/s encoded rate must stay under 65,536 B/s. A safety
abort or rate miss stops the sequence; no replacement or cap increase.

Per-run QC requires every planned socket write, exact planned one-second
offered counts, generator CPU below its guard, 0 observer/recorder drops,
all valid transactions committed exactly once on four validators,
same-height BlockID/AppHash, and per-run COMMON plan locked with actual
endpoints and config/genesis/source hashes before sends. Report achieved
offered tx/s and bytes/s; all-four inclusion p50/p95 **upper bounds** from
100-ms block scans; sampled accepted mempool count and bytes; CheckTx
socket queue wait (not total queue time); eventual goodput; and a
time-aligned timely-goodput fraction. Missing metrics are `null` with
`UNAVAILABLE`, never zero. This short run cannot establish sustainable
steady-state throughput or a full queue residence distribution.

## Prospective knee rule

The T06 pilot's original rule counted every send in the last seconds of
the steady window against a same-window commit deadline, falsely flagging
C/H under ordinary ~1.8-second inclusion lag. Preserve that historical
rule and raw unchanged. For R15, an **eligible** send is a steady send
planned in the first nine of twelve steady seconds (UTC after warm-up,
strictly before steady second ten). Count it timely when all four nodes
first observe its committed block by the end of steady second twelve.
Thus each eligible send has at least three seconds to be observed. The
time-aligned fraction is timely eligible / all eligible. Every run still
reports all late and eventual commits without discarding them.

The 20 tx/s reference must have time-aligned fraction ≥0.95 in at least
two of three runs per profile. A knee bracket 20–23 is supported for a
profile only if **both** 23 and 25 tx/s have ≥2/3 runs meeting either:

1. Time-aligned fraction <0.90, or
2. p95 inclusion upper bound ≥2× that profile's median 20-tx/s p95
   **and** CheckTx socket wait ≥2× its median 20-tx/s wait.

If the reference itself fails, mark the knee `NOT_IDENTIFIABLE_REFERENCE`.
If neither high rate qualifies, mark
`NOT_OBSERVED_WITHIN_SAFE_RANGE`. Do not assign 23 or 25 tx/s the exact
knee. Four profiles are evaluated separately. Report each run and
criterion flag literally, including negative findings; calculate
20,000-resample percentile bootstrap 95% intervals for rate-level means
using independent networks, seed 20261113. Three runs per cell give wide
intervals and do not establish long-run stability. The 2/6/12 tx/s T06
levels remain calibration context only, never pooled as R15 confirmatory
runs. No inference above 25 tx/s, to multi-host/P2P-origin, or to DoS.
