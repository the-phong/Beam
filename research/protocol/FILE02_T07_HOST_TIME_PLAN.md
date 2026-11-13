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

## U1/U2 VMware guest calibration addendum (locked before capture)

The Owner supplied U1, U2 and U3 under `D:\VMware` for T07. Use U1 and U2
as distinct Ubuntu guest hosts on the private VMware NAT subnet; retain U3
as spare. The two guests share one physical Windows workstation and
hypervisor, so this is **multi-guest**, not independent physical-hardware
evidence. Do not extrapolate host-failure independence or production network
latency. No transaction workload is part of this clock survey; the existing
S06 caps and loopback/private endpoint rules remain unchanged.

After confirming distinct VM config/MAC and reachable private IPs, capture
the same inventory and 200-sample, 50 ms clock-continuity survey on both
guests, with a one-second warm-up, ten-second steady observation and
three-second drain. For each guest,
collect 32 bounded request/response timestamp probes from the controller
immediately before and after the steady survey. For controller wall-clock
times `c0` before request and `c3` after response, and remote wall-clock
times `r1` before reply and `r2` after reply, the causal offset interval is
`[r1-c3, r2-c0]`. Report the narrowest observed interval at each boundary,
the controller/guest continuity checks, and the conservative union spanning
before and after. No symmetric-network assumption is used. Cross-guest
uncertainty derives from the two guest-to-controller intervals; a claimed
cross-guest timing effect whose interval overlaps zero remains unresolved.
If a clock step exceeds 10 ms, an interval is malformed, or a guest becomes
unreachable, retain raw and fail this calibration; do not replace it under
the same ID. Record any absent thermal/clock-sync signal as `UNAVAILABLE`.
