# X06 isolated crypto CPU profile plan

Status: locked before new CPU sampling. Use the accepted R01 valid H
signature fixture and exactly the R01 secp256k1 and ML-DSA-44 verification
primitive benchmark code. Four process pairs alternate primitive order;
each isolated subbenchmark runs with `GOMAXPROCS=1`, one CPU, 2-s Go
benchtime and a CPU pprof file. Record raw benchmark text, pprof, `pprof
-top` text and parseable summary; freeze raw after all processes exit.

Acceptance: eight successful benchmark processes and readable pprof files,
each with at least one second of sampled CPU. Treat each process as a
sampling unit and benchmark iterations as correlated operations. Report
observed sample count in 10-ms-equivalent ticks, between-process ranges,
and the profiling overhead relative to frozen R01 nonprofiled results.
Sampling is statistical, compiler inlining can move work, and the
benchmark path omits parsing, binding, digest, state, recorder and Comet.
Do not divide app process CPU by request count or label R01 primitive CPU
as per-stage CheckTx CPU. `CPU_crypto` in an actual network run remains
`null` (`UNAVAILABLE_STAGE_CPU_ATTRIBUTION`).

Command: `python3 research/tools/file02_x06_profile.py run
research/artifacts/file02/<session>`; verifier uses `verify`.
