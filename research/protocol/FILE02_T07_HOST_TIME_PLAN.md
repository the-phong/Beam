# T07 — host and clock calibration plan

Status: locked before the T07 host survey, 2026-11-13. T07 has two required
placements: the current single host and at least two independent hosts. A
four-validator deployment inside one WSL2 VM counts only as single-host.
Neither placement may be labeled complete until its checks and raw provenance
are verified. Multi-host placement must be on Owner-approved private hosts;
no public endpoint or safety cap expansion is implied by this plan.

## Single-host survey

Record source commit/tree, kernel/virtualization, CPU model and selected
timing features, logical CPU count, filesystem type/free fraction, loopback
MTU, clock source and sync status. Record thermal/throttling counters if the
guest exposes them; otherwise use `null` with a reason. After a one-second
warm-up, sample `CLOCK_MONOTONIC` around `CLOCK_REALTIME` 200 times at 50 ms
intervals. Require monotonic readings strictly increasing and no adjacent
change in the realtime-minus-monotonic offset above 10 ms. The maximum
bracketing interval is an *intra-host sampling uncertainty bound*, not proof
of UTC accuracy. Retain raw samples privately and publish only aggregate
statistics. A failed check stays failed; no replacement under the same ID.

## Multi-host requirement

Before any multi-host network run, record each independent host's inventory,
CPU features, fs/network profile, clock-sync mechanism and thermal/throttle
availability. Measure inter-host offset and its uncertainty immediately
before and after each run using a bounded round-trip method or an audited
clock service. Reject cross-host latency/lead-time interpretations when
uncertainty spans the claimed effect or a clock step occurs. Warm-up and
drain durations must be explicit, and all controlled endpoints must remain
private and within the existing S06 caps. A deployment and execution plan
needs the Owner's actual host addresses and access method; absent those,
record `UNAVAILABLE` rather than substitute local validator processes.
