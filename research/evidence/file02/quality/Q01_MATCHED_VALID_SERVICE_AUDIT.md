# Q01 — matched legitimate-service audit

**Disposition: PASS for the available H invalid-background contrasts.**
The [locked secondary analysis](../../../protocol/FILE02_Q01_LOCKED_ANALYSIS.md)
and tool were committed before producing the keyless
[`Q01_PAIRED_COMPARISON.csv`](Q01_PAIRED_COMPARISON.csv) and
[`Q01_SUMMARY.json`](Q01_SUMMARY.json). `python3 research/tools/file02_q01.py
run` and an independent `verify` each exited 0. Seven source sessions
(A01–A05, A08, A13) passed frozen-raw integrity and original QC again.

The table has 76 independent pair rows, covering 152 fresh four-validator
networks: 12 pairs for each A01–A05/A08 and four for A13. Within each pair,
the analysis rechecked H profile, one physical host, RPC-origin topology,
same seed/genesis/config/software, identical 73 planned fixed legitimate
hashes and send offsets (including warm-up), six fixed legitimate H tx/s,
14 background tx/s in the on phase, and 20 tx/s/51,360 B/s in both on arms.
All 76 pair QC statuses are PASS; there were no missing, replaced or
excluded arms. The source oracles independently verified exact-once valid
inclusion, expected invalid rejection and four-validator agreement.

Every A01–A05/A08 arm had 72/72 eligible fixed transactions observed on all
four validators within the locked 3,000-ms deadline. A13 had 36/36 in its
on phase per arm. Each case's original paired-block effect was 0.000
proportion points, with its original 95% bootstrap CI [0.000, 0.000].
The 72 or 36 transactions within one network are **not** independent CI
replicates. No pooled cross-family CI was computed. A06 is a replay
mechanism case, A07 valid-only load and A14 a process fault, so none is
misrepresented as an invalid-background RQ1b matched-service pair.

This confirms control fairness and the negative finding at the registered
deadline for these exact strata. A degenerate CI at a saturated binary
endpoint is **not** an equivalence test and cannot rule out smaller timing
changes, another deadline, a higher safe rate or a different topology.
Actual socket-send times may jitter even though planned offsets and
achieved one-second rates matched; the first-all-four observation is an
upper bound on inclusion, not an exact commit timestamp.
