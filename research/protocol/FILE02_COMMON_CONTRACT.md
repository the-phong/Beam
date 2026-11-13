# File 02 common contract v1 — sections 0–2

Status: Owner accepted for bounded pre-case readiness on 2026-11-13 under
D-038; PR #21 merged into `test-mainet` at
`6de41446272520809fbdcb84d3b1155831dc24d5`. The Owner authorized
sections 0–2 of the outside Checklist 02. This freezes
definitions and exclusion rules for preparation; case-specific plans and
confirmatory RQ1–RQ3 runs still need their own locked parameters and gates.

## Questions and comparisons

RQ1 asks whether the controlled authentication profiles change resource cost
and service after accounting for wire size, offered rate, state and topology.
C versus CP isolates bytes with the same classical verification; CP versus H
controls wire size while adding ML-DSA verification. P versus H changes the
AND policy. A B0/B1 comparison is not a crypto-specific effect. The primary
endpoint is CheckTx service time for valid accepted transfers; full-stack
goodput, inclusion latency and resource measures are secondary. An effect is
an estimate with run-level uncertainty, not an assumed direction.

RQ2 asks whether attack traces are distinguishable from matched benign
pressure using features available before the service bound is breached.
Ground truth comes from scenario and independent correctness oracle, never
from a rejection code alone. RQ3 asks whether a detector improves upon fixed
threshold, EWMA/CUSUM and cost-aware rules on the same held-out runs at a
fixed FPR and acceptable overhead. No detector or RQ2/RQ3 result is in scope
for this foundation.

Profiles C, CP, P and H, capabilities A–D, T0–T4 stages, and labels
BENIGN/ATTACK_KNOWN/MIXED/NATURAL_REJECTION/OUT_OF_SCOPE retain the exact
meaning in the outside Checklist 02 and
[research protocol](RESEARCH_PROTOCOL.md). RPC is the current controlled
origin. P2P origin is available only after an actual P2P injector and oracle
are implemented; observing relay does not make an RPC-origin run P2P-origin.
The factor levels and exclusions are encoded in
[`file02_factors.json`](../config/file02_factors.json). Numerical load levels
tested by the bounded T06 pilots are 2/6/12/20/23/25 tx/s on the single-host
diagnostic topology. These are tested pilot rates, not confirmatory case rates
or proven load regions. The [knee pilot](../evidence/file02/T06_KNEE_PILOT.md)
did not establish a defensible knee within the unchanged safe range:
`NEAR_KNEE`, `AT_KNEE` and `SLIGHTLY_OVER_KNEE` are
`NOT_OBSERVED_WITHIN_SAFE_RANGE`. A future case may preregister a supported
bounded rate and report this limitation; this contract does not require or
authorize another knee search.

## Run unit, matching and time

One independent run is one fresh four-validator network, genesis, seed and
plan; transaction samples within that run are clustered. Every run records
source commit/tree, versions, host topology and CPU features, full Comet
mempool/block configuration, corpus/config/genesis hashes, profile, case,
capability, origin, seed, order, planned and actual tx/s and bytes/s.
Warm-up, steady, drain and cooldown use monotonic process durations; UTC is
for correlation only. Cross-host timing needs measured uncertainty before
latency or early-warning claims. Pairing fixes business intents, genesis,
state, limits, collector mode and placement. `equal_tx_s`, `equal_bytes_s`,
`fixed_legitimate_plus_adversarial` and `matched_pressure` are distinct
regimes. Under attack the legitimate stream stays fixed while the adversarial
stream is added. An attacker/benign comparison declares exactly what is
matched and what remains unmatched; byte matching outside safe capacity is
`NOT_FEASIBLE_WITHIN_SAFE_RANGE`, never an approximate success.

Order is randomized or counterbalanced with a recorded seed. Independent
repetition count, p99 precision, benign calibration and split design are
fixed from pilots before confirmatory analysis. No current S07 one-run-per-
profile smoke can supply a confidence interval or p99 precision. The
[measurement contract](MEASUREMENT_CONTRACT.md) controls rate fidelity:
steady total within ±5% and at least 90% of each planned decision window;
burst/zero windows follow their frozen schedule. Achieved offered load means
actual socket sends and encoded bytes, not RPC acceptance or commit throughput.

## Quality and artifacts

The metric names, units, capture points and unavailable values are in
[`FILE02_METRICS.md`](FILE02_METRICS.md). A missing stream is JSON `null`
with an availability reason, never zero. Raw per-run files remain in ignored
private storage, with a SHA-256 manifest and write-once run ID. A failed run
retains raw evidence and receives `INVALID_FOR_ANALYSIS` for validator loss,
AppHash divergence, clock discontinuity, rate miss, excess observer drops,
generator saturation, safety abort, oracle failure or integrity failure.
Neither an invalid run nor an unmeasured endpoint may enter an effect
aggregate. An independent oracle must check expected code/stage, inclusion,
state and same-height agreement across all four validators.

Each future case directory must contain `plan.json`, an immutable private
`raw/<run_id>/` with digest, `result_per_run.json` and CSV, `qc_report.json`,
`result_aggregate.json`, reproducible `figures/` plus code, and `decision.md`.
An aggregate uses one estimate per independent run, reports n and CI or
`INSUFFICIENT_RUNS`, and does not mix profile, case, topology, regime or
attack family. The executable foundation in `research/tools/file02_common.py`
validates locked plans and QC envelopes and refuses invalid aggregation;
case-specific producers and figures remain for the later case work.

## Safety and current gate

Only controlled loopback/private targets, disposable test keys and the S06
caps in [`s06_pilot.json`](../config/s06_pilot.json) are allowed. The generator
and observer abort guards remain mandatory. S07 demonstrates a bounded
four-validator oracle/recorder/load/abort chain; it is QA evidence, not
confirmatory data. The Owner-authorized bounded T06 calibration and
precision extension satisfy COMMON-06 with 12 independent networks and
2,880 steady samples per profile; [the gate audit](../evidence/file02/COMMON_GATE_AUDIT.md)
records the result. The later bounded knee pilot did not identify a
defensible near/at/above-knee region; its preregistered flags and the
observation-window limitation remain in the historical evidence. Gate COMMON
was accepted only for bounded pre-case readiness; unavailable knee-relative
levels remain unavailable. The distinct RQ1b question is recorded in the
[Owner-directed addendum](FILE02_RQ1B_ADDENDUM.md), without changing accepted
RQ1 or its H–CP endpoint.
Case-specific rates, sample sizes, endpoints and exclusions still require
their own locked plans after authorization. No new confirmatory T/R/A/Q/D
case has been run by this document.
