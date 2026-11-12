# File 02 T06 bounded generator calibration pilot — locked before execution

Authorization: Owner explicitly allowed a **bounded T06 pilot** on
2026-11-12. This document is a pilot plan for the original Pluto folder and
`research/file02-common-foundation`; it is not authorization for T/R/A/Q/D
confirmatory cases, a safety-cap increase, multi-host traffic or an RQ result.

## Fixed design

- Four real validators on one host; all RPC/P2P endpoints remain on loopback.
- The unchanged `research/config/s06_pilot.json` is used, SHA-256
  `9bc640db8b8875af3f9a5e4f89faa1882e8f4bc10459997d1a891397768cf21a`.
- C, CP, P and H each use fresh network/keys/genesis and 90 pre-funded
  senders. Every signed transaction is unique at nonce zero. One warm-up
  second, four steady seconds and three drain seconds per run.
- Target rates, in increasing safety order: 2, 6, 12, 20 valid tx/s for
  each profile. The highest H wire rate is about 51,460 encoded bytes/s,
  below the pinned 65,536-byte/s hard cap. All rates are below 25 tx/s.
  No invalid workload, attack case or saturation claim is planned.
- The open-loop generator's actual socket writes determine achieved tx/s
  and bytes/s. `broadcast_tx_sync` acceptance is a separate result; direct
  block/state oracle checks every valid transaction on four validators.
- A rate is achieved only when total steady tx/s is within ±5%, every
  nonempty one-second window reaches at least 90% of planned count, every
  request is actually sent, no generator error or safety abort occurs, all
  valid transfers commit exactly once, and same-height BlockID/AppHash and
  account state agree. Failed runs retain private diagnostics and are never
  silently retried under one run ID.
- Generator user/system CPU time, wall duration and peak RSS are recorded.
  The four-validator observer and S06 host guard remain active. Raw is
  frozen under ignored `research/artifacts/file02/` with hashes.

This is a **generator calibration**, not a capacity-knee search. Its
one-run-per-cell observations cannot define a safe knee, scientific
comparison or p99. The preexisting COMMON-06 rank-band rule (sample p99
population rank within 98.5–99.5% at ≥95% coverage) stays fixed. Any
later precision extension must separately count transaction samples and
independent runs, account for clustering/censoring, and withhold p99 if its
registered precision criterion is not met.
