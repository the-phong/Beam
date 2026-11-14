# A11 — matched-resource feasibility disposition

Date: 2026-11-14. A11 asks for benign and attack arms with equivalent
observed CPU, CheckTx socket queue and bytes, followed by a test of whether
those resource-equivalent instances remain distinguishable. The accepted
A01–A05/A08 runs and the independently verified [A09 matched-rate
synthesis](A09_MATCHED_RATE_AUDIT.md) provide 72 benign/attack pairs at
exactly 20 H-sized writes/s and 51,360 offered B/s, with identical six
fixed legitimate H tx/s in each pair. A09's private immutable analysis
input manifest is
`6d967f73705bd24915db9dcd61769ded117757231e67774335842189fe77e90f`;
its raw-bound verifier exited 0 with QC PASS over all 144 arms.

The measured whole-application-process CPU envelope per offered attempt
was **7,883.82–10,663.90 µs** in benign arms and
**4,979.25–7,427.39 µs** in attack arms. Even the closest global endpoints
are separated by **456.43 µs/attempt**. Thus there is no pair in these
already collected 20-tx/s, equal-byte runs with overlapping measured CPU
envelopes; the joint CPU/queue/byte match set is empty before considering
queue. The CheckTx socket-queue ranges themselves overlap, so queue alone
is not the blocking condition. These CPU quotients include the test harness
and do not establish class-specific CPU equivalence.

No equivalent matched-resource instance exists **in the available stratum**,
and no detector exists to estimate distinguishability or an
impossible-to-separate rate. Selecting a loose tolerance after seeing the
separation would create an outcome-dependent match. Other workloads or
rates within the safe range were not searched as an A11 study; this report
does **not** claim that resource matching is impossible everywhere up to
25 tx/s. Closing A11 requires a prospectively locked match variable,
tolerance justified by measurement precision, candidate workload/rate
grid, new QC-passing runs and a defined distinguishability test, all
within unchanged safety caps and with fixed legitimate service.

**A11 is NO_RESOURCE_MATCH_IN_AVAILABLE_20_TX_S_STRATUM, PARTIAL and
unchecked.** No A11 effect size, CI, classifier FPR, DoS or knee is
fabricated. A12 is next.
