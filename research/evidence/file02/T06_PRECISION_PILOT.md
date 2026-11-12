# T06 bounded precision pilot for COMMON-06

Date: 2026-11-12. Scope: Owner-authorized single-host, loopback-only QA pilot;
not a confirmatory T06 case or RQ1–RQ3 result. The design and precision rule
were locked in `research/protocol/FILE02_T06_PILOT_PLAN.md` before execution.

Producer source commit `3edf2999076e3a0972739cda6e79e2e87ec6aa8b`, tree
`a73c46b9be364c001ca170931e1c7dcac0e2f161`; analyzer timestamp-parser
fix commit `2e854604e620b83db33412d9e2aaa15fc2aca39e`. The parser fix was
made after data collection, without changing raw data, QC or the registered
statistical thresholds. `make test-file02-t06-precision` executed all 48
four-validator networks; its Python audit stage initially exited 2 because
Python 3.10 rejected a valid five-digit Go fractional timestamp. The fixed
analyzer reprocessed the same 48 runs via `finalize`, exited 0, then froze
and hashed raw evidence. A preceding attempt was stopped using the run's
safety stop file after discovering that its analyzer included warm-up in the
p99 sample; only four runs completed in that attempt. Its private raw remains
unfrozen and is excluded from this result.

The 48 fresh networks comprised 12 independent runs per C/CP/P/H, with
distinct seeds, genesis and validator homes. A randomized starting order
followed by cyclic blocks placed each profile in each run-order position
three times. Every run sent one warm-up transaction, then exactly 20 valid
tx/s in each of twelve steady one-second windows, and allowed three drain
seconds. The direct oracle found all 241 valid transactions committed once
on all four validators in every run. Every run supplied 240 uncensored
steady-state inclusion observations. The generator used at most 0.0139 CPU
core fraction; no run missed its target or triggered a safety abort.

| Profile | Independent runs | Steady samples | p99 upper bound (ms) | Two-stage cluster bootstrap 95% CI (ms) | Max relative CI side | QC |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| C | 12 | 2,880 | 1,889.933 | 1,858.265–1,922.218 | 1.71% | PRECISION_PASS |
| CP | 12 | 2,880 | 1,851.493 | 1,821.175–1,870.218 | 1.64% | PRECISION_PASS |
| P | 12 | 2,880 | 1,872.533 | 1,841.074–1,915.039 | 2.27% | PRECISION_PASS |
| H | 12 | 2,880 | 1,871.594 | 1,841.854–1,900.274 | 1.59% | PRECISION_PASS |

The preregistered optimistic i.i.d. rank-band coverage is 99.08% at
`n=2,880`, above the 95% floor for the 98.5–99.5% band. The two-stage
bootstrap resampled networks and transactions 2,000 times; each profile's
larger CI side is below the preregistered 20% maximum. The measurements are
block-scan **upper bounds** on inclusion latency, sampled at 100 ms; they
are not exact commit times. The bootstrap quantifies sampling variation for
this single-host pilot only. The close p99 values do not establish profile
equivalence or a causal effect, and the pilot does not locate a capacity knee.

Frozen private session:
`research/artifacts/file02/t06-precision-20261112T142307Z-1f61b257`.
Manifest SHA-256:
`7a324562411512e3acfe8084352fba1972c7791e9eacba4d66b0aeb41935253a`.
`python3 research/tools/file02_t06_precision.py verify research/artifacts/file02/t06-precision-20261112T142307Z-1f61b257`
exited 0, rehashed the frozen raw and rederived all 48 runs. No private
keys, validator homes, signed payloads or raw logs are tracked.
