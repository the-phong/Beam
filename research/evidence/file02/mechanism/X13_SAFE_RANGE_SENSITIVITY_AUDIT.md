# X13 — safe-range invalid-background sensitivity

Status: **PASS for descriptive 20/23/25-tx/s single-host RPC-origin
sensitivity**. The plan, source and whole-session restart rule were locked
at `2c2608d` before the successful confirmatory send. The successful
private session is `research/artifacts/file02/x13-20261115T135447Z-a960b885`.
All **36/36** fresh four-validator networks and **12/12** matched fixture
blocks passed QC; there were no exclusions. Immutable raw manifest SHA-256:
`9eaa4d990f43e5c9d1c38d2ab4f968db3dc7a3c4e94a6a0a2a8d60ed4e0ab0e3`.
`python3 research/tools/file02_x13.py verify` returned
`X13_VERIFY=OK qc=PASS`. Every run had 12 exact one-second 6 + 14/17/19
achieved windows, all fixed valid H committed exactly once, every
late-bad-ML background rejected before commit, four-validator agreement,
zero observer/recorder drops and no safety abort.

| Invalid background + six fixed valid H/s | Offered bytes/s | Fixed-H mean first-all-four observation upper bound, ms; 95% block CI | Mean of run p50 / p95 upper bounds, ms | Fixed H observed by 3s | Four-app CPU interval sum, mean s/run |
| --- | ---: | ---: | ---: | ---: | ---: |
| 14/s; total 20 tx/s | 51,360 | 1177.47 [1166.35, 1189.70] | 1189.28 / 1787.59 | 864/864 | 2.042 |
| 17/s; total 23 tx/s | 59,064 | 1171.39 [1152.06, 1188.46] | 1165.62 / 1775.95 | 864/864 | 2.112 |
| 19/s; total 25 tx/s | 64,200 | 1182.16 [1158.41, 1210.16] | 1171.25 / 1804.56 | 864/864 | 2.110 |

Within matched blocks, **23−20 = −6.08 ms**, 95% CI
**[−28.27, +15.44] ms**; **25−20 = +4.69 ms**, 95% CI
**[−21.15, +35.44] ms**. These are descriptive response-by-rate estimates
for an invalid-background intervention. The 100-ms X10 practical margin
is not transferred into an X13 equivalence claim. The p50/p95 and app CPU
columns are descriptive means across runs, without a confirmatory interval.
The observation is an upper bound from a nominal 100-ms block scan, not
exact commit time. The app CPU is four process envelopes, not per-stage
crypto CPU. No sustained bottleneck or knee was observed in the tested
safe range; 25 tx/s remains the safety cap, not chain capacity.

The failed QA-only schedule-representation attempt and passing QA-only
3-network attempt are documented in [X13_PREFLIGHT_QA.md](X13_PREFLIGHT_QA.md)
and excluded. The first full 36-network attempt was interrupted and is
quarantined for raw-integrity failure in
[X13_INTERRUPTED_ATTEMPT.md](X13_INTERRUPTED_ATTEMPT.md); none of its
24 complete runs enters this table or the CIs. No case was selectively
replaced. There is no DoS, distributed-network, P2P-origin or sustainable
throughput conclusion.
