# A09 — matched offered-rate secondary audit

Date: 2026-11-14. The [analysis plan](../../protocol/FILE02_A09_LOCKED_ANALYSIS.md)
and raw-bound analyzer were committed at `f7fdced` before this synthesis.
No new network was run. Private analysis session:
`research/artifacts/file02/a09-20261113T184147Z-2d4716eb`.
Plan SHA-256:
`7b0527220a6d291f585f6ec86740dd90bccb508b229eb81e2317e4cbe4a78309`;
immutable input-manifest SHA-256:
`6d967f73705bd24915db9dcd61769ded117757231e67774335842189fe77e90f`.

The analyzer independently reverified A01–A05 and A08 upstream immutable
raw and derived artifacts, then checked all **72 original pairs / 144 arms**.
Every arm offered exactly 20 H-sized writes/s, 2,568 bytes/write and
51,360 B/s in each steady window. Each pair retained its six identical
legitimate H tx/s, genesis and original 12-pair CI. No six-case pooled CI
was computed, because the attack mechanisms differ. All six original
benign-minus-attack timely-fraction effects were **0.000**, each with its
own 95% CI **[0.000, 0.000]** at the registered 3,000-ms deadline.

| Case | Benign shared app-process CPU / attempt | Attack shared app-process CPU / attempt | Benign / attack CheckTx socket wait mean |
| --- | ---: | ---: | ---: |
| A01 | 8,734 µs | 5,685 µs | 47.26 / 45.27 µs |
| A02 | 8,454 µs | 5,384 µs | 44.30 / 41.66 µs |
| A03 | 8,866 µs | 5,792 µs | 44.70 / 41.84 µs |
| A04 | 8,772 µs | 5,847 µs | 45.21 / 40.92 µs |
| A05 | 8,873 µs | 5,944 µs | 44.78 / 40.72 µs |
| A08 | 8,188 µs | 5,401 µs | 44.42 / 41.67 µs |

All CPU values are whole-process envelopes divided by 241
offered attempts, **not class-specific CPU/tx**. Class-specific CPU is
`null`/`UNAVAILABLE_SHARED_INTEGRATION_PROCESS`. The larger benign CPU
envelope reflects its 14 additional valid, committing transactions/s;
it cannot be interpreted as the invalid workload being harmless in every
regime. The 144 per-run rows retain mempool peaks, relay envelopes and
queue waits. Across all arms, the run-level CheckTx socket-wait ranges
overlap on **40.20–57.71 µs**. Queue wait alone is therefore a
false-positive-confusable **feature candidate** in this dataset, not a
measured classifier false-positive rate. FPR is `null` with reason
`UNAVAILABLE_NO_DETECTOR`; no alert threshold or ML model was evaluated.

`python3 research/tools/file02_a09_a10.py run A09` and independent
`verify research/artifacts/file02/a09-20261113T184147Z-2d4716eb` exited
0 with QC PASS, six input cases, 72 pairs and 144 arms. Private plan,
immutable input manifest, per-run JSON/CSV, QC, aggregate, figure status
and decision remain preserved. **A09 PASS for matched-rate secondary
comparison and bounded feature-overlap disposition.** It does not add a
new service-degradation, FPR, DoS, knee, P2P-origin or multi-host result.
