# T05 recorder perturbation plan

Status: locked before execution, 2026-11-13. For each C/CP/P/H profile,
run a paired ON and OFF application CheckTx-recorder condition on two fresh
four-validator single-host loopback networks. Both conditions reuse the
same pre-signed nine-valid-transfer plan, genesis, seed, 2 tx/s steady
schedule, collector, instrumented Comet build and unchanged S06 caps. Only
`App.SetRecorder(recorder)` versus `App.SetRecorder(nil)` changes. Alternate
order by profile: C/P ON→OFF, CP/H OFF→ON.

Primary correctness gate: each condition sends all nine planned events,
commits all nine exactly once, reaches the same AppHash at fixed height 8
within its pair, and has no observer or generator safety failure.
ON must record CheckTx events with zero recorder drops; OFF must record
none. Report descriptive ON-minus-OFF differences in median client RPC
completion latency, observed Comet CPU delta, application-process CPU
delta and peak RSS. The application process includes the test harness and
all four app servers, so its CPU/RSS is a coarse process measure, not crypto
CPU. Four pairs do not establish a causal overhead CI or universal threshold.
Do not treat any descriptive sign as an RQ result.
