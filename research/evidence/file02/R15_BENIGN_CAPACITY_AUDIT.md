# R15 — bounded benign capacity ramp

Date: 2026-11-13. The R15 design and time-aligned knee rule were locked
before accepted execution. The final source for the run is `a4b38cd`;
the later `2163640` analysis revision adds 95% CIs for every measured
rate-level mean. Revision `c065d2a` additionally derives four-Comet-process
CPU deltas from the already frozen observer samples. Neither changes raw
or the knee rule. The private
session is `research/artifacts/file02/r15-20261113T074602Z-751a6e1b`.
Top-level plan SHA-256:
`60f4e1e1a374229303ac2625f83f89ce2fcecfa0641443e61b4e6f8d0e0e0314`.
Immutable raw manifest SHA-256:
`e94f4e413a57a3689016f507b54a5a722b95e03f6b14fbea690ac35bf0d54800`.
`python3 research/tools/file02_r15.py run` exited 0 (integration 0,
36/36 QC PASS); `python3 research/tools/file02_r15.py rederive ...`
and independent `verify` exited 0. The later `rederive-cpu` and independent
`verify` also exited 0. `analysis_revision_000_aggregate.json` and
`analysis_revision_001/002.json` preserve both derived-output corrections
and their source/hash provenance. The historical T06 pilot and its original
boundary-biased flags were not modified.

The 36 fresh four-validator single-host networks cover C/CP/P/H ×
20/23/25 valid tx/s × three independent repetitions, with ascending
rate and rotated profile order. Each run used 12 steady seconds,
one distinct funded sender per transaction, validator-0 loopback RPC,
the unchanged 25 tx/s, 65,536 B/s and 60-second S06 safety caps, and
no invalid load. All planned sends and one-second rate windows passed;
all valid transfers committed once on all four validators, with
same-height BlockID/AppHash, no observer/recorder drops and no safety
abort. Per-run plans were locked with actual source, endpoints, config
and genesis before sends. The private artifact bundle contains immutable
raw, per-run JSON/CSV, QC, aggregate with run-level bootstrap CIs,
figure status and decision.

The table shows means across three independent networks per cell.
`aligned` is the timely fraction for sends in the first nine steady
seconds, each given at least three seconds before the observation
deadline. `eventual` counts commits by drain end divided by 12 seconds;
it is **not** sustainable steady-state throughput. p95 is an all-four
inclusion upper bound from 100-ms block scanning. Queue is only CheckTx
socket queue wait; mempool bytes are sampled maxima.

| Profile | tx/s | Aligned fraction | Eventual tx/s | p95 upper ms | Socket queue ms | Max sampled mempool B |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| C | 20 | 1.00 | 20 | 1,862 | 0.0604 | 5,377 |
| C | 23 | 1.00 | 23 | 1,835 | 0.0626 | 6,265 |
| C | 25 | 1.00 | 25 | 1,804 | 0.0660 | 6,611 |
| CP | 20 | 1.00 | 20 | 1,758 | 0.0750 | 89,880 |
| CP | 23 | 1.00 | 23 | 1,814 | 0.0744 | 107,000 |
| CP | 25 | 1.00 | 25 | 1,788 | 0.0786 | 117,272 |
| P | 20 | 1.00 | 20 | 1,775 | 0.0771 | 91,813 |
| P | 23 | 1.00 | 23 | 1,778 | 0.0715 | 99,325 |
| P | 25 | 1.00 | 25 | 1,783 | 0.0764 | 108,507 |
| H | 20 | 1.00 | 20 | 1,840 | 0.0762 | 89,024 |
| H | 23 | 1.00 | 23 | 1,838 | 0.0835 | 108,712 |
| H | 25 | 1.00 | 25 | 1,786 | 0.0743 | 107,856 |

All four profiles pass the 20-tx/s reference check. At both 23 and
25 tx/s, **0/3 runs per profile** trigger the time-aligned-goodput
branch and **0/3** trigger the joint p95/socket-queue branch. Thus all
four results are **`NOT_OBSERVED_WITHIN_SAFE_RANGE`** under the
prospective R15 rule. No numerical near/at/over-knee level is assigned.
The per-cell 95% intervals are in `result_aggregate.json`; three short
runs per cell provide limited precision and no long-run stability proof.
Four-Comet-process CPU deltas over each sampled run are also in per-run
results and aggregate, with 95% run-level intervals. Mean Comet CPU
seconds for 20/23/25 tx/s respectively are C 7.80/8.27/8.48,
CP 8.48/8.15/8.18, P 8.74/8.64/9.20 and H 8.50/8.51/8.17.
These process totals combine consensus, P2P and mempool work; they do not
isolate CheckTx CPU. The application shares a process with the integration
harness, so application CPU is `null` with
`UNAVAILABLE_SHARED_WITH_INTEGRATION_HARNESS`, not a fabricated zero.
No R15 p99 precision gate was preregistered, so p99 is
`UNAVAILABLE_INSUFFICIENT_PRECISION`; the p95 bounds are reported instead.

Two earlier R15 sessions are excluded from inference and retained as
private diagnostic provenance. `r15-20261113T074214Z-c620246c`
(manifest `acff85c8a45bedcee7a2d1f42ce817fd3af178b4be74e70b311d79c18478cc15`)
stopped before sends because `order` was missing from the pre-send COMMON
plan binder. `r15-20261113T074320Z-62398090`
(manifest `23a08c732694980dcd562f2b60e5dd6bc7ddd1091f81eaa8eab5f75476ec8457`)
was stopped when scheduled run IDs did not match harness IDs. Neither
enters the accepted aggregate. Both errors were corrected and committed
before the final run.

**R15 PASS as a bounded negative knee observation.** It does not show
that the system has no knee above 25 tx/s or on another topology, nor
does it prove sustainable capacity, P2P-origin behavior or DoS.
