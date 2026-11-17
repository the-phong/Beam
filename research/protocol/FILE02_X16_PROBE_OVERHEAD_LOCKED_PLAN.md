# X16 stage-clock overhead QA — locked before execution

Status: LOCKED FOR ENGINEERING QA, 2026-11-16. This is an instrument
equivalence check, not a new RQ1b service-effect contrast. It supplements
`FILE02_X07_X16_REWORK_LOCKED_PLAN.md` without changing historical endpoints,
RQ1/RQ1b hypotheses, S06 caps or the 48-run X16 confirmatory source lock.

Eight fresh matched blocks each run the same signed I fixture twice: six
fixed valid H/s and fourteen late-bad-ML H/s at 20 total H-sized tx/s,
51,360 B/s, 12-second steady interval. Both arms keep Comet hash tracing
and the app recorder ON. Only the dedicated app's Linux thread-clock stage
probe differs. Blocks 0,2,4,6 use OFF→ON; blocks 1,3,5,7 use ON→OFF.
The run/network is the unit, paired within block; no run is replaced after
failure. Each run retains a fresh four-validator single-physical-host
loopback network and identical genesis, signed bytes, send offsets and
config within its block.

Compare paired ON−OFF application, Comet, observer and generator process CPU,
RPC producer latency and fixed-valid observation upper bound. Report stage
wall differences and instrumented thread CPU with their proper denominators.
Use paired percentile bootstrap 20,000 resamples, seed 20261116, and 95% CI.
This QA estimates whole-network perturbation under this workload, not a
per-clock-call correction or exact unperturbed stage CPU. If the probe has
material or imprecise effects, leave native stage CPU `null` with the reason.
All code/results, AppHash, zero drops, PID identity, trace coverage and raw
provenance must pass independent QC; any failed pair withholds inferential
overhead conclusions. No capacity, knee, DoS or multi-host inference follows.

Safety remains max 25 tx/s, 65,536 B/s, 16 outstanding and 60 s/run. Raw
and keys remain private; freeze a manifest after process shutdown. The
analysis tool may publish only derived statistics and hashes.
