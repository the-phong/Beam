# A08 — preregistered mixed invalid workload, locked before execution

Status: LOCKED DESIGN, 2026-11-14. Use 12 independent matched pairs on
24 fresh four-validator single-host loopback-RPC networks, with six
benign-first and six attack-first orders. Each pair has one new 241-sender
genesis. The warm-up and six fixed legitimate H transfers/s are byte/hash/
schedule/genesis-identical across arms. Each steady second offers 20
2,568-byte H-sized transactions (51,360 B/s): six fixed legitimate plus
14 valid benign-background H transfers, or six fixed legitimate plus
**4 structural-invalid, 5 ML-DSA-invalid and 5 fully signed nonce-invalid**
background transactions. The ratio is fixed per second, not selected after
seeing outcomes. Structural bytes require Actor A only; malformed ML proof
requires Actor B access to a signed transaction; nonce-invalid signed bytes
require Actor C registered keys. Per-event capability is recorded. No
arbitrary sender is credited with Actor C capability.

The primary endpoint and decision are the A01 preregistered fraction of
72 steady fixed legitimate transactions observed included on all four
validators by 3,000 ms after socket write. Use paired benign-minus-attack
mean and a two-sided 20,000-resample percentile bootstrap, seed 20261113.
Only a positive lower CI bound with full QC supports bounded service
degradation at this endpoint. Secondary metrics: per-class stage and
CheckTx wall-service spans, sampled mempool/CPU/CheckTx socket-queue curves,
relay-envelope counters, valid goodput and fixed inclusion upper bounds.
The 72-per-arm p99 is `INSUFFICIENT_PRECISION` without the registered
cluster precision gate; report null rather than a false precise estimate.

Oracles: structural code 1 before crypto, bad ML code 13 after secp, and
nonce-invalid code 6 after both signatures; none commits. Each attack arm
must have 48/60/60 invalid events by class, 73 exact-once valid commits;
each benign arm 241 exact-once valid commits. QC requires exact rate/bytes
and fixed stream/genesis matching, all planned writes, valid stage spans,
all-four BlockID/AppHash and account-state agreement, intact raw and
recorder/observer streams without drops, no safety abort or censored primary
sample. No failed pair is replaced. Keep S06 25 tx/s, 65,536 B/s,
16 outstanding, 60 seconds and loopback/private limits. No DoS, knee,
multi-host or P2P-origin claim follows.
