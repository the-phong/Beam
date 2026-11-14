# A09 — matched-rate secondary analysis, locked before synthesis

Status: LOCKED ANALYSIS, 2026-11-14. This is an explicit secondary synthesis
of the already preregistered A01–A05 and A08 matched pairs, not a new
network experiment or a retrospectively preregistered positive hypothesis.
All 72 pairs (144 arms) offered 20 H-sized writes/s, with identical six
fixed legitimate H tx/s within each pair. Each original case retains its
own 12-pair primary 3,000-ms all-four inclusion CI and QC. The analysis
does not pool the six case CIs into 72 pseudo-independent repetitions.

Report by case: benign-minus-attack service endpoint and its original CI,
offered bytes/tx and tx/s, whole application-process CPU per offered
attempt, sampled accepted-mempool peak, CheckTx socket queue mean wait and
P2P relay send envelopes. Identify telemetry features whose observed
benign/attack run-level ranges overlap as **false-positive-confusable
candidates**. This does not estimate classifier FPR: no detector, alert
threshold, training or Q/D-case evaluation exists. Per-class CPU/tx is
`UNAVAILABLE` because the application process shares the integration
harness. Do not infer DoS or distributed effects from matched offered rate.

QC requires all six upstream sessions to pass independent raw rederivation,
all 144 arms to retain QC PASS, exact 20 writes and 51,360 B in each steady
one-second window, 2,568 bytes per wire transaction, original fixed-stream
and genesis matching, and intact upstream/raw analysis manifests. A failed
input withholds the synthesis rather than silently removing a case.
