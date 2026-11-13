# Gate COMMON — bounded QA result and remaining preregistration

Date: 2026-11-12. The fresh four-profile S07 smoke was rerun after
COMMON-01–09 QA work: `make test-s07-smoke` exited 0 with four new
four-validator single-host networks, `runs=4`, `excluded=0`. Its private
session is `research/artifacts/s07/s07-20261112T145947Z-e71de5d2`, source
commit `cec652b2465a470033e275618026851a23999208`, source tree
`3bdea12209174a1914b39eff8002a221333fe058`, manifest SHA-256
`85c0e8290a7b91d6db5df45ebe0f5f2b3f154d289a83b5929c081a4b03b9efe0`.
`python3 research/tools/s07_orchestrate.py verify` exited 0 and rehashed
92 frozen files. The oracle accepted/committed the valid stream, rejected
the invalid stream and checked four-validator state and AppHash. The
metrics recorder yielded 36 telemetry samples for C and analogous streams
for CP/P/H. The bounded open-loop rate limiter sent exactly nine planned
events in each profile's smoke. The [COMMON-09 safety audit](COMMON09_SAFETY_STOP_AUDIT.md)
separately verified an immediate zero-send abort on the same S06 guard path;
the in-flight abort and public-target refusal unit tests also passed.

The [T06 pilot preregistration](../../protocol/FILE02_T06_PILOT_PLAN.md)
was locked before its calibration and precision executions. It fixed the
safe QA rate sequence and the p99 rank/cluster CI rule, which passed for
COMMON-06. The common contract and metric/factor registries remain a
review candidate; numerical near-knee, at-knee and just-over-knee load
levels cannot be set from the authorized 2/6/12/20 tx/s calibration.
No capacity knee was searched or observed. Confirmatory case-specific
plans, topology and detector thresholds are not locked.

**Disposition: smoke/oracle/recorder/rate limiter/abort QA PASS; Gate COMMON
remains open for complete preregistration and Controller/Owner review.**
This evidence does not open the T/R/A/Q/D case program or support an RQ
claim. The next decision is whether to authorize a separately bounded T06
knee search and, if no knee exists within unchanged safety caps, how to
record the unavailable load levels in section 2.1.

## Subsequent bounded search

The Owner authorized a separate search within the existing caps. Its
[36-network report](T06_KNEE_PILOT.md) found no defensible knee at 20,
23 or 25 tx/s: eventual goodput matched offered rate and p95/socket queue
wait did not rise persistently. The literal preregistered rule produced
timely-goodput flags for C/H, but the same threshold failed at the 20 tx/s
reference because of end-of-window censoring. Those flags and the limitation
remain published. Gate COMMON stays open pending Owner disposition of
section 2.1's unavailable near/at/above-knee levels.
