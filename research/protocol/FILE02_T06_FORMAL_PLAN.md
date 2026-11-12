# T06 — formal bounded generator calibration plan

Status: locked before execution, 2026-11-13. This is the T06 preflight
case, separate from the historical T06 pilots. Run 16 fresh four-validator
single-host loopback networks: C/CP/P/H at 2, 6, 12 and 20 valid tx/s,
in increasing rate order. Each uses 90 freshly funded senders, one warm-up
second, four steady seconds, three drain seconds and a distinct run ID,
seed and genesis. The existing `s06_pilot.json` remains unchanged:
25 tx/s, 65,536 bytes/s and 60 seconds per run are the upper caps. All
transactions are distinct, valid and pre-signed before the run.

Success requires every scheduled request written to the loopback RPC
socket, zero dropped/unsent requests, no loadgen or safety abort, achieved
steady tx/s and encoded bytes/s within ±5% of each target, at least 90%
of planned sends in every one-second window, actual rate below both caps,
and the independent four-validator oracle confirming exact-once commit,
account state and same-height block/AppHash. Generator process CPU time
divided by elapsed wall time must stay below **0.25 core** in every cell;
this conservative diagnostic threshold is fixed from the previous bounded
pilot and only checks that the generator itself is not CPU-saturated in
this tested region. Generator peak RSS, CPU fraction, target/achieved rates
and dropped count are reported for every cell. A failed cell remains
invalid and raw is retained; no post-hoc rate adjustment is allowed.

No knee search, capacity claim, p99 or RQ comparison follows. The 20 tx/s
level is a tested bounded preflight level, not a proven subcritical or
near-knee level. This plan does not raise any safety cap. Raw lives under
ignored `research/artifacts/file02/t06-formal-*`, frozen with a SHA-256
manifest; public evidence contains only keyless summaries.
