# Q03 — valid-included legitimate p99 quality analysis

Status: LOCKED SECONDARY ANALYSIS before derived outputs, 2026-11-14.
Source sessions are the seven frozen/QC-PASS A01–A05/A08/A13 sessions in
`file02_q01.py` and the QC-PASS Q02 session. No new load is sent. Analyze
each attack family, condition and Q02 profile separately; do not pool unlike
attack classes or treat transfers as independent experimental runs.

Only the 72 steady fixed-legitimate hashes per A01–A05/A08/Q02 arm, or the
36 A13 on-phase fixed-legitimate hashes per arm, enter. Each source analyzer
must reverify the raw manifest, exact-once inclusion and all-four observation
before its latency array is accepted. Exclude warm-up, invalid background,
valid background and any uncommitted transaction. The quantity is the
send-to-first-all-four-observation **upper bound** in milliseconds, never
the exact commit timestamp. Report nearest-rank descriptive p99, count of
independent networks, count of valid included samples and run IDs.

Apply the accepted preregistered precision gate: optimistic iid coverage
of the 98.5–99.5% rank band must be at least 0.95; if it passes, compute a
two-stage (network then within-network transaction) 2,000-draw percentile
CI with seed 20261114 and require relative CI radius at most 0.20. If a gate
fails, inferential p99 and CI remain `null`, status
`INSUFFICIENT_SAMPLES`/`INSUFFICIENT_PRECISION` with the failed gate named.
No post-data increase in runs or case pooling. The primary RQ1b deadline
fraction and its existing CI remain unchanged.
