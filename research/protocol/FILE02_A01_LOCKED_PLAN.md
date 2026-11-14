# A01 — structural-invalid load with fixed legitimate H service

Status: **LOCKED DESIGN BEFORE EXECUTION**, 2026-11-13. This case is
authorized by the Owner's Checklist 02 section 5 instruction, after bounded
Gate RQ1 acceptance. It does not change RQ1 or silently turn RQ1b into RQ1.

Each of 12 preregistered matched pairs uses one new 241-sender genesis. The
two arms run on separate fresh four-validator single-host networks, sending
through validator-0 loopback RPC. The warm-up transfer and 72 steady
legitimate H transfers use identical bytes, hashes, accounts, nonce 0 and
offsets within a pair. Both arms offer 20 H-size transactions per one-second
steady window (51,360 B/s). The benign arm adds 14 valid H transfers per
window from independent accounts. The attack arm adds 14 distinct malformed
2,568-byte payloads per window, varying wrong magic, inconsistent declared
chain-ID length and forbidden chain-ID encoding. Actor A can construct these
arbitrary bytes without a registered signing key. Each malformed hash must
return structural code 1, reach no crypto stage and never commit.

The registered primary per-network endpoint is the fraction of the 72
steady fixed legitimate transactions first observed included on all four
validators no later than **3,000 ms** after the measured socket write.
The generator and block observer record Linux `CLOCK_MONOTONIC` nanoseconds
on the same host. The scanner runs every 100 ms; first observation is an
inclusion upper bound, not an exact commit instant. A missing monotonic
stamp, censored deadline or missing valid observation fails primary QC.
The paired effect is mean `Y_benign - Y_attack` across 12 independent
pairs. Use a two-sided percentile bootstrap with 20,000 pair resamples and
seed 20261113. CI lower bound above zero is evidence for a bounded
degradation of this endpoint; crossing zero is inconclusive; upper bound
at or below zero opposes the proposed positive effect in this stratum.
No practical SLO or DoS threshold is inferred from this test.

The 12 pairs have six benign-first and six attack-first orders, fixed in a
seeded `schedule.json` before any send. Each run has one-second warm-up,
12-second steady period, five-second drain and two-second separate cooldown.
No run exceeds the unchanged S06 caps: 25 tx/s, 65,536 B/s, 16 outstanding,
60 seconds and loopback-only ingress. The case producer binds source/tree,
host inventory, exact endpoints and Comet config hashes in a write-once
COMMON plan before sending. All failed/excluded arms remain in the raw
record; there is no outcome-based replacement.

Secondary measurements: achieved offered tx/s and bytes/s by second,
application CheckTx service spans by hash/stage at validator 0, recorder
drops, socket queue wait, sampled accepted-mempool bytes, Comet CPU/RSS and
P2P relay envelope bytes. Application process CPU is shared with the test
harness and cannot be attributed to one attack class; report it as such.
The 3,000-ms deadline is a prospective research endpoint from the accepted
RQ1b planning candidate, not a measured production SLO. p99 is descriptive
or `INSUFFICIENT_PRECISION` unless its preregistered cluster gate passes.

QC requires exact fixed-legitimate stream/genesis and byte-rate matching,
all planned socket writes, 6 fixed plus 14 background sends per steady
window, response-stage oracles, 73/241 exact-once valid inclusion in
attack/benign arms, absence of all malformed hashes from blocks,
same-height BlockID/AppHash across validators, intact raw manifest, no
recorder/observer drops, no safety abort and no uncensored primary sample.
An invalid QC arm excludes its pair and withholds inference. The case may
be accepted as a verified negative or inconclusive finding; it need not
produce a positive degradation effect. It cannot establish P2P-origin,
multi-host, sustainable throughput or DoS behavior.
