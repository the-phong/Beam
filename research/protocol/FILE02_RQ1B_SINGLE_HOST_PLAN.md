# File 02 RQ1b — legitimate service under invalid H load plan candidate

Status: **DRAFT FOR CONTROLLER/OWNER REVIEW**, 2026-11-13. This is a
pre-case design, not a `LOCKED` `plan.json`, run authorization or result.
It applies the separate [Owner-directed RQ1b addendum](FILE02_RQ1B_ADDENDUM.md)
on one physical host and does not alter the accepted RQ1 H–CP cost test.

## Case and estimand

Proposed case ID: `RQ1B-H-BADML-20-SH`. Both arms send the **identical fixed
legitimate H stream** at 6 tx/s. A separate 14 tx/s H background shares the
same planned offsets in each pair. In the benign arm, background transfers
are valid and use independent accounts. In the treatment arm, each
background transaction has a valid structural envelope, binding and secp
signature but a deliberately invalid ML-DSA proof; it must reach the late
ML verification stage, be rejected and never commit. Each H wire payload is
2,568 bytes with the proposed 16-byte chain ID `pluto-rq1b-2026x`, so each
arm offers 20 tx/s and 51,360 encoded bytes/s. The
legitimate byte strings, hashes, accounts, nonces and send schedule are
identical within a pair. Accepted background count, state evolution,
mempool occupancy and verification work may differ as consequences of the
treatment; they are measured, not claimed to be matched.

For each network, the primary endpoint `Y` is the fraction of distinct
steady-window legitimate transfers first **observed in the same block on
all four validators within 3,000 ms of that transfer's monotonic send**.
The block scanner runs at 100-ms cadence. This is a deadline-observation
endpoint: its timestamp is an upper bound on inclusion, not exact commit.
The full deadline must be observable for every eligible send. If a deadline
is censored, the arm fails primary QC; it is not scored as a failure or a
success. One `Y` per fresh network enters inference. The paired effect is
`mean(Y_benign - Y_invalid)` in proportion points. H0 is `effect <= 0`;
H1 is `effect > 0`. Only a paired-run 95% CI lower bound above zero with
all QC passing supports degradation for this tested stratum. An upper bound
at or below zero opposes positive degradation; an interval crossing zero
is inconclusive. This endpoint and rule do not by themselves establish a
practical service failure or DoS.

## Proposed parameters to lock before a formal run

| Field | Review candidate |
| --- | --- |
| Workload | Benign arm factor `BENIGN_HIGH`, capability `NONE`; treatment arm factor `MIXED`, capability `B`, with `MLDSA_INVALID` background. One shared legitimate H warm-up transfer, then 6 distinct legitimate H transfers and 14 distinct background H transfers per one-second steady window. Twelve steady windows yield 72 legitimate and 168 background sends per arm. Use one-use funded accounts at nonce 0; benign and invalid background use account slots independent of the legitimate stream. |
| Rate and phase | 6 legitimate + 14 background = 20 offered tx/s; 51,360 bytes/s for either arm. One-second warm-up, 12-second steady, five-second drain, two-second cooldown = 20 seconds. The five-second drain plus cooldown must cover every 3,000-ms deadline and scanner delay. |
| Independent replication | 12 preregistered benign/invalid pairs, 24 fresh four-validator networks; pair seed/genesis/legitimate bytes and schedule fixed within pair, distinct across pairs. Six pairs start benign and six invalid, assigned by recorded master seed. No within-run transfer is an independent CI replicate. |
| Origin and topology | Validator 0 loopback RPC (`127.0.0.1`, exact port recorded before lock); four validators and collector on one physical host, `FOUR_VALIDATOR_SINGLE_HOST_DIAGNOSTIC`. Same software, genesis, Comet limits, collector mode and process placement within a pair. |
| Primary capture | Monotonic actual socket-send timestamp and first 100-ms all-four block-scan observation per legitimate hash. Record send-to-observation uncertainty, missing observations and deadline completeness. The case producer must prove the scan-to-generator clock relation on this host. |
| CI and decision | Two-sided paired percentile bootstrap over 12 pair-level `Y_benign - Y_invalid` differences, 20,000 resamples with recorded seed `20261113`, 2.5th/97.5th percentile. The analyzer and CI test must exist before lock. A wide interval remains inconclusive; no extra pairs or rate changes chosen after outcomes. |
| Secondary | Achieved legitimate/background tx/s and bytes/s; valid and invalid RPC codes/stages; eventual exact-once legitimate inclusion; deadline miss count; observation-limited latency; CheckTx service and socket queue wait separately; accepted mempool bytes; node CPU/RSS. p99 is `INSUFFICIENT_PRECISION` unless this case independently passes its preregistered sample and cluster CI rule. |
| Safety | Keep all `research/config/s06_pilot.json` caps and guards: 25 tx/s, 65,536 bytes/s, 16 outstanding, 60 seconds, loopback, CPU/RSS/disk and observer stop rules. No automatic retries on the same run ID. |
| QC | Both arms must deliver all scheduled socket writes, total and component steady rates within ±5%, each one-second window at least 90% of plan, exact same legitimate stream and total encoded bytes, no generator saturation or safety abort. The direct oracle must show every invalid hash rejected at the intended ML stage and absent from blocks, every legitimate hash committed exactly once, same-height BlockID/AppHash and correct per-arm account state. Require complete 3,000-ms observation for every legitimate send, clock continuity, zero missing primary observations, verified raw provenance and four live validators. A failed arm excludes its pair; no outcome-based replacement. |

The choice of 20 total tx/s stays within T06's **delivered-load** region.
T06 formal used a 23-byte chain ID and delivered 51,500 bytes/s for H at
20 tx/s; T02's 2,568-byte H size used a 16-byte chain ID. The locked case
must recompute canonical byte lengths and reject any schedule mismatch.
The exact 6+14 mixed schedule has not yet been formally calibrated. COMMON-03
and COMMON-05 verified fixed legitimate and matched-byte construction at
only 2 tx/s. A pre-run fixture and safety validation must demonstrate that
the 6+14 schedule is feasible without altering caps. The 12-pair count is
a bounded proposal, not a demonstrated power calculation. The 3,000-ms
deadline is prospective, informed by T06's roughly 1.9-second single-host
pilot p99 **observation upper bound**; it is not a measured SLO or exact
commit deadline. No load rate here is labeled subcritical or a knee.

## Pre-run lock and scientific boundary

The Owner must separately authorize the official RQ1b case after reviewing
this candidate. Individual `LOCKED` private plans must bind actual source
commit/tree, host inventory, software versions, exact RPC port, payload
hashes, genesis/config/corpus hashes, seeds/order, run IDs, analysis method,
deadline and QC before sending traffic. The 6+14 fixture, oracle, deadline
scanner, CI analyzer and failure tests must be implemented and verified;
`file02_common.py validate-plan` and safety preflight must pass. This draft
does not tick A/Q/R items or create per-case result artifacts.

Any later positive RQ1b estimate would be a bounded workload effect on
single-host RPC-origin H traffic. It would not isolate ML-DSA algorithm cost,
prove a service-failure SLO, show P2P-origin effects, show independent-host
performance or establish distributed DoS. A practical minimum important
effect or service SLO needs a separate prospective decision before such a
claim. Unavailable measurements are `null` with reasons, never zero.
