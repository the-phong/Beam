# R14 — CheckTx recheck on state-changing commits

Status: **LOCKED BEFORE R14 RUNS**, 2026-11-13. Four matched logical
fixtures, each with 81 funded nonce-zero senders, are replayed across C,
CP, P and H. Each arm uses a fresh four-validator single-host network,
with 16 networks total and rotated profile order. Every arm sends one
warm-up and 80 steady valid transfers at 20 tx/s over four seconds via
validator-0 loopback RPC, then three seconds drain and two seconds
cooldown. All profiles use the unchanged `s06_pilot.json`, including
`mempool_recheck=true`, 128 mempool transactions, 8 MiB mempool bytes,
8,192-B max transaction and every S06 safety cap.

The primary descriptive endpoint is the count of application CheckTx
`recheck` events at validator 0 in each independent network, stratified by
profile. Match each recheck hash to the 81 planned hashes; report result
codes and service spans separately from `new` calls. The Comet cumulative
`mempool_recheck_times` metric must agree with the recorder count. Report
sampled accepted mempool count/bytes before, during and after the run.
All 81 valid transfers must commit exactly once on all four validators,
with same-height BlockID/AppHash agreement, no observer/recorder drops and
all planned sends/rate windows met. A failed network is not replaced.

Use four independent network counts per profile, 20,000 percentile
bootstrap resamples with seed 20261113, and report 95% intervals for mean
recheck count. Recheck is an internal consequence of pending valid
transactions during state-changing commits, **not** a new offered attack.
These nonce-zero transactions come from independent senders, so a recheck
can still return code zero after other senders commit. Per-event Comet CPU,
exact queue residence time and which individual commit triggered each
recheck are `UNAVAILABLE`; do not fill zero. No DoS or knee inference.
