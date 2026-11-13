# R01 — cryptographic primitive verification lock

Status: **LOCKED BEFORE R01 MEASUREMENT**, 2026-11-13. Owner authority:
Checklist 02 section 4 R01–R16, sequentially. This is a mechanism
microbenchmark, not the accepted RQ1 primary network endpoint. Its valid H
transaction supplies both signatures over the same canonical H intent.
No RQ1 hypothesis or safety cap is changed.

## Fixed design

- Unit: one fresh operating-system benchmark process containing a paired
  secp256k1 and ML-DSA-44 verification measurement. Twelve process runs;
  six `secp_first` and six `ml_first` orders, shuffled by master seed
  `20261113`. All runs use the same checked-out source, Go toolchain,
  single host and `GOMAXPROCS=1`.
- Timing excludes key generation, transaction signing, wire parsing,
  digest construction and key decoding. The benchmark verifies both valid
  proofs before timing. Each sub-benchmark has Go `-benchtime=200ms`,
  `-benchmem`, `-cpu=1`, and a 30-second subprocess timeout. Runtime code
  and cryptographic library versions come from the pinned `go.mod`.
- Primary descriptive outputs: `ns/op`, `B/op`, `allocs/op` for each
  primitive, one value per process run. Pair-level ML minus secp `ns/op`
  and ratio are aggregated across twelve independent processes. A
  two-sided 95% paired percentile bootstrap uses 20,000 resamples,
  deterministic seed `20261113`, and 2.5th/97.5th percentiles. No
  transaction iteration is a CI replicate. Report raw pair values and
  distribution; never interpret primitive time as total CheckTx or CPU
  attribution.
- `cycles/op` is `UNAVAILABLE` unless a verified cycle counter with
  comparable fixed-frequency semantics and permission exists; do not
  substitute wall nanoseconds or fill zero. No `perf` access is assumed.
- QC: both benchmark cases present exactly once per process, positive
  iterations/ns/op, nonnegative bytes/allocations, valid-proof checks and
  subprocess exit zero. Record actual host CPU model/features, OS, Go,
  source commit/tree, `go.mod` digest, command, order and elapsed time.
  Any failed process remains in immutable raw and is excluded; n below 12
  yields `INSUFFICIENT_RUNS` and no inferential contrast. No post-outcome
  replacement or extra repetitions.
- Safety: no network traffic. The process timeout is below the unchanged
  S06 60-second per-run maximum; raw lives only under ignored
  `research/artifacts/file02/`. No keys, signed transaction bytes or
  runtime homes enter tracked evidence.

The R01 producer must write a private locked `plan.json` with the exact
source commit/tree, command and host inventory before the first benchmark
process. It then writes immutable raw, per-run JSON/CSV, QC, aggregate and
decision artifacts and verifies their hashes. Only a verified result may
tick R01. Later R02–R16 plans and runs remain separate, in checklist order.
