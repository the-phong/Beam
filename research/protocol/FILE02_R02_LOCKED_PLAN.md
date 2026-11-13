# R02 — direct full CheckTx validation lock

Status: **LOCKED BEFORE R02 MEASUREMENT**, 2026-11-13. R01 was verified
first. R02 isolates the accepted C/CP/P/H application CheckTx path on the
same physical host, genesis and logical transfer stream. It is direct ABCI
measurement; the separate R06/R07 cases address live network behavior.
The accepted RQ1 network endpoint and H0/H1 are unchanged.

## Fixed design and measurements

- Twelve independent operating-system process runs, each with one fresh
  genesis of 242 accounts: 241 distinct funded nonce-zero senders and one
  receiver. Each process measures all four profiles on this same genesis,
  one warm-up and 240 distinct valid steady CheckTx calls per profile.
  Profile-specific signatures bind the same logical intents. The chain ID
  is the fixed 16-byte `pluto-file02-r02`, yielding C/CP/P/H wire lengths
  148/2,568/2,504/2,568 bytes. No state commit occurs during CheckTx.
- Profile order follows a seeded starting permutation and cyclic rotation
  across 12 process runs, giving each profile each order position three
  times. Seed `20261113`; `GOMAXPROCS=1`; one host/toolchain/source for all
  runs. Each subprocess has a 55-second timeout. No network traffic is
  generated.
- Recorder ON for all arms. Primary per-run service measure is the mean
  application CheckTx entry-to-return duration for 240 accepted valid
  transactions, excluding app lock wait. Report per-run service p50/p95/p99
  and pooled service p99 only if the preregistered two-stage cluster
  bootstrap precision gate passes. The gate requires 12 complete clusters,
  2,880 samples/profile, optimistic 98.5–99.5% rank coverage at least 95%,
  and both CI sides within 20% of pooled p99, with 2,000 draws and seed
  `20261113`. Otherwise p99 is descriptive per run only and pooled
  inference is `INSUFFICIENT_PRECISION`.
- Optional measurement spans separate canonical structural parse, account
  binding/registry lookup, intent digest, secp verification, ML-DSA
  verification, state rules and state-copy. Each span is wall duration,
  not a CPU-time attribution. The residual from total service is reported
  as instrumented callback/other time, not silently assigned to crypto.
  Direct in-memory CheckTx performs no disk commit; storage write latency
  is `NOT_ON_CHECKTX_PATH`, not a measured zero.
- Process user+system CPU delta and Go `TotalAlloc`/`Mallocs` delta are
  measured around each 240-call batch and divided by 240. They include
  recorder and runtime overhead; they do not identify crypto-only CPU.
  Include the service and stage distributions, sample counts, host and
  source identity, and actual recorder drops.
- Across process runs, report 95% paired percentile bootstrap CI for the
  H-minus-CP difference of per-run mean CheckTx service duration, 20,000
  resamples, seed `20261113`. This direct-ABCI contrast is supporting
  mechanism evidence; it is not the RQ1 network result. No outcome-based
  sample addition or method change.
- QC: source/tree and host match the private pre-run `LOCKED` plan; all
  profiles and 240 accepted records present with no recorder drop; one
  genesis hash and exact wire lengths per process; stage sets match each
  profile; all durations and memory/CPU deltas valid; order schedule
  matches the lock. A failed process is preserved and excluded. Fewer than
  12 valid process runs yields `INSUFFICIENT_RUNS` and no CI. Raw remains
  immutable, private and SHA-256 verified. Full per-run, QC, aggregate and
  decision artifacts are required before R02 may be ticked.

Measurement hooks add only observational durations when enabled. Ordinary
unobserved validation and all consensus outcomes must remain identical,
covered by the existing and new regression tests. This case cannot support
network throughput, relay, knee, P2P-origin, multi-host or DoS claims.
