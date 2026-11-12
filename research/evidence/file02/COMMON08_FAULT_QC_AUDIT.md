# COMMON-08 — fault exclusion and raw preservation

Date: 2026-11-12. Scope: bounded single-host loopback QA for the common
quality gate. Producer source commit
`0b6e88f918b63ee58ce410dd0af78d9d94b42586`, tree
`a7ebfe4911d87c3e1b4a628a0412aefe4d742f13`. A fresh H-profile
four-validator network sent nine valid transactions at 2 tx/s, with four
steady windows. The direct oracle checked committed state, transaction
sets and same-height BlockID/AppHash. The runner copied each independent
observer JSONL stream into the private raw run folder before QC. The new
auditor checks monotonic send offsets against UTC timestamps, actual rate,
observer dropped/unavailable samples, oracle status, and generator CPU
against the pinned 90% safety ceiling. All six fault flags were false and
the run was `USABLE_FOR_QA_ONLY`; observer drop count was zero and generator
CPU fraction was 0.0035 of one core.

`python3 research/tools/file02_common08.py smoke` exited 0. Frozen private
session:
`research/artifacts/file02/common08-20261112T145039Z-81d2f4f1`.
Manifest SHA-256:
`4b9606c1ff26235d25c6f0c1167dd44fb8fd35380b935a7a05374c86f9fb6fa7`.
`python3 research/tools/file02_common08.py verify` on that session exited
0 and rederived the QC decision from hashed raw.

`python3 research/tools/file02_common08.py inject` on the frozen session
exited 0. It made disposable private copies, one per fault: validator loss
signal, AppHash divergence signal, wall-clock discontinuity, missed target,
observer drop and generator saturation. Each copy returned
`INVALID_FOR_ANALYSIS` with the matching reason; a seventh copy lacking the
oracle returned `INVALID_FOR_ANALYSIS` with `INCOMPLETE_EVIDENCE`. The
original frozen manifest was reverified after injections. These are fault
injections into copied evidence, not claims of seven naturally occurring
faulty networks. The common aggregate path excludes `INVALID_FOR_ANALYSIS`
rows and keeps the raw manifest binding.

An actual run that aborts before all artifacts are produced must be frozen
as incomplete and excluded. The QA runner now writes identity metadata
before starting the network and preserves observer streams even when load
aborts; the COMMON-08 smoke freezes the run before returning failure. A
failure before network startup can still leave only partial private raw and
requires operator inspection. The future T/R/A/Q/D case producers must
derive these signals from their own raw data and cannot inherit this pilot's
QC status.
