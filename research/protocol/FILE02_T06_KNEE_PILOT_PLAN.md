# T06 bounded knee search — locked before execution

Owner authorized this separate pilot on 2026-11-12. It uses only the existing
single-host four-validator loopback network and the unchanged S06 config
SHA-256 `9bc640db8b8875af3f9a5e4f89faa1882e8f4bc10459997d1a891397768cf21a`.
This is an exploratory diagnostic, not confirmatory T06 or RQ evidence.

## Fixed search and measurements

- C, CP, P, H; three fresh independent networks per profile and offered
  rate, each with distinct seed, genesis, validator homes and 301 funded
  senders. Rates are 20, 23, 25 valid tx/s in increasing rate order, with
  profile order rotated in each repetition. There are at most 36 networks.
- One warm-up second, twelve steady seconds and three drain seconds. One
  unique funded account transaction per send. No attack traffic. The H
  25 tx/s encoded rate must be below 65,536 bytes/s, otherwise that cell
  is `NOT_FEASIBLE_WITHIN_SAFE_RANGE` and the search stops. Every run retains
  the unchanged 25 tx/s, 60-second, queue, CPU/RSS and disk guards.
- QC requires all planned events actually sent; each steady one-second
  window at least 90% of target and mean within 5%; direct all-four oracle
  confirming exactly-once commits, matching state/AppHash; no safety abort;
  complete block-scan observations and four recorder streams with zero drops.
  An invalid run stops the search and is never silently replaced.
- Per run record: planned and achieved offered tx/s and bytes/s; timely
  all-four goodput as count of steady-origin sends first seen on all four
  nodes during the twelve-second steady interval divided by 12; eventual
  goodput as all steady-origin commits divided by 12; p50/p95 latency upper
  bounds from 100-ms block scanning with n; max accepted mempool count and
  bytes; mean CheckTx socket queue wait from differences of cumulative
  recorder count/sum (when available). This socket wait is only a queue
  proxy, not the full P2P or mempool wait. Missing counters are reported
  `UNAVAILABLE`, never zero.

## Decision fixed before data

The 20 tx/s cell is the within-search reference. A **knee observed** for a
profile requires at least two adjacent tested rates, 23 and 25 tx/s, each
meeting one of these criteria in at least two of its three independent runs:
timely goodput below 90% of achieved offered rate, or p95 inclusion upper
bound at least twice the profile's median 20 tx/s p95 **and** CheckTx socket
queue wait at least twice the profile's median 20 tx/s queue wait. The first
qualifying rate is the observed knee bracket; the exact knee is not resolved
between discrete rates. If queue wait is unavailable, the latency-and-queue
criterion is unavailable; the goodput criterion still applies. If neither
criterion holds, report `KNEE_NOT_OBSERVED_WITHIN_TESTED_RANGE` for that
profile. A rate miss or safety abort yields `INVALID_PILOT`, not a knee.
No safety cap is increased and no extrapolation beyond 25 tx/s is made.
