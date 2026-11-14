# A13 — attacker start/stop and bounded service dynamics

**Disposition: PARTIAL; checklist box remains open.** The preregistered
four-block, balanced-order comparison ran on eight fresh four-validator
single-host loopback-RPC networks. Source/load plan commit `5afbc53`;
analysis parser correction commit `0b58683`. Private session:
`research/artifacts/file02/a13-20261113T190017Z-d2e1e0a6`.
Immutable raw manifest SHA-256:
`6892315cdc842ad0539b78d4dfe5a1c73b1863c49155319bdebf62cddb2dbb61`.
`python3 research/tools/file02_a13.py verify <session>` passed after
reanalysis; integration test exited 0 and all eight arms passed QC.

Each arm sent six identical fixed legitimate H tx/s throughout 12 steady
seconds. Three seconds of background-off, six seconds of 14 extra tx/s,
then three seconds off gave 6/20/6 tx/s; on-window load was 51,360 B/s,
within the accepted caps. The attack arm's extra 84 transactions had
preregistered 4 structural, 5 bad-ML, 5 signed state-invalid per on second.
The benign arm's extras were distinct valid H. Per-arm oracle: benign
157/157 valid committed; attack 73/73 valid committed and 84/84 invalid
rejected at their expected stages. All four validators agreed on committed
BlockID/AppHash/state; no recorder drops, failed pair or safety abort.

| Phase | Fixed legitimate observations per arm | Benign timely | Attack timely | Paired benign-minus-attack fraction, 95% block bootstrap CI |
|---|---:|---:|---:|---|
| Baseline | 72 | 72/72 | 72/72 | 0.000 [0.000, 0.000] |
| On | 144 | 144/144 | 144/144 | 0.000 [0.000, 0.000] |
| Recovery | 72 | 72/72 | 72/72 | 0.000 [0.000, 0.000] |

Timely means first observed inclusion on all four validators no later than
3,000 ms after socket write. This is an observation **upper bound**, not a
commit timestamp or a production SLO. Median upper bounds (benign/attack)
were 1,133/1,092 ms in baseline, 1,142/1,179 ms in on phase, and
1,225/1,289 ms in recovery. Raw-bound per-run phase latencies and sampled
mempool/CPU/CheckTx socket-queue curves are retained privately. The scheduled
attacker start/stop were 4/10 s from load origin. A service violation was
`NOT_OBSERVED` at the locked endpoint, so recovery time is `null` with
`NOT_APPLICABLE_NO_OBSERVED_DEGRADATION`; it is not zero. Alert lead time
and false-alert transients are `null`,
`UNAVAILABLE_NO_ACCEPTED_DETECTOR`. No detector or alert rule was fitted.

The first analysis attempt was rejected by QC: Python 3.10 could not parse
Go's nanosecond RFC3339 timestamps, even though the integration oracle had
passed. Its derived files remain in the private session's
`analysis_attempt_1/`. The accepted parser was reused from the existing
T06 precision tool and the **same frozen raw** was reanalyzed; no run was
replaced or repeated. The original rejected QC is preserved, not silently
rewritten. This correction changes analysis only, not experimental data.

The result supplies a negative finding for the tested 20-tx/s on windows.
It cannot answer the checklist's alert-before/after-SLO question without an
accepted detector and an observed SLO violation. It establishes neither a
knee nor DoS, multi-host behavior, or P2P-origin performance.
