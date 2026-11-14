# A07 — valid-load pressure, locked before execution

Status: LOCKED DESIGN, 2026-11-14. Four independent fixture blocks each
run two H-profile arms on new four-validator single-host loopback-RPC
networks, alternating 20-first and 25-first order. Each arm sends one warm-up
transfer, four steady seconds, then drains for three seconds with a two-second
cooldown. Six fixed legitimate H transfers per steady second have identical
bytes, hashes, genesis and offsets within a block. The 20-tx/s control adds
14 other distinct valid H transfers/s; the 25-tx/s valid-flood arm adds 19.
Actor C must control registered funded signing keys. All traffic is valid,
2,568 bytes/tx, and exactly once per account. The maximum offered load is
25 tx/s or 64,200 B/s, below unchanged 25 tx/s and 65,536 B/s S06 caps.
Outstanding and duration caps remain 16 and 60 seconds.

The primary paired contrast is rate-25 minus rate-20 fraction of the 24
fixed legitimate transactions observed included on all four validators no
later than 3,000 ms after socket write. One block gives one paired
difference; use a two-sided 20,000-resample percentile bootstrap with seed
20261113. A negative CI upper bound would support bounded valid-load
pressure at this endpoint; crossing zero is inconclusive. The observation
is an inclusion upper bound, not exact commit latency. Secondary metrics:
sampled accepted mempool bytes, P2P relay envelope bytes, fixed inclusion
latency upper bounds, whole-process CPU and CheckTx socket wait. `Eventual
committed steady transactions / four-second offered window` is a bounded
goodput accounting measure; it is not sustainable steady-state throughput.

QC requires every planned 20/25 write in each one-second window, identical
fixed legitimate stream/genesis in a pair, 81/101 valid exact-once commits,
all-four BlockID/AppHash and account-state agreement, 24 uncensored fixed
observations, recorder/observer completeness without drops, immutable raw
and no safety abort. Failed arms invalidate the pair; no replacement.
This case is **valid-load pressure**, not invalid-verification waste. A
25-tx/s result does not establish a knee, DoS, multi-host or P2P-origin
effect.
