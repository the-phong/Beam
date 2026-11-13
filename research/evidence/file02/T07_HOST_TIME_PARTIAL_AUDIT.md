# T07 — host and time calibration: single-host evidence only

Date: 2026-11-13. The [T07 plan](../../protocol/FILE02_T07_HOST_TIME_PLAN.md)
and capture tool were committed at
`f0478ab6742fcecc7feb95b8ce0050447ab10d6c` before the survey.
`python3 research/tools/file02_t07_host.py capture` exited 0. The private
session is `research/artifacts/file02/t07-host-20261112T185808Z-77d019ba`;
manifest SHA-256 is
`60275183e7c4f04bc5f70642e2c5d3633003b1d4fd3d40bbc546dc449a533239`.
`python3 research/tools/file02_t07_host.py verify` exited 0 and recomputed
the survey from the frozen samples.

| Check | Observation |
| --- | --- |
| Placement | One WSL2 guest, Linux `6.18.33.2-microsoft-standard-WSL2`, x86_64; previous four-validator runs share this host. |
| CPU | Intel i7-12700H, 20 logical CPUs; `constant_tsc`, `nonstop_tsc`, `tsc_reliable`, `rdtscp` and `hypervisor` flags exposed. |
| Storage/network | Repository on ext4, 81.09% free at survey; loopback only, MTU 65,536. |
| Clock service | `timedatectl`: `NTPSynchronized=yes`, `NTP=no`; this alone does not establish UTC accuracy. |
| Sampling | One-second warm-up, then 200 `CLOCK_MONOTONIC`/`CLOCK_REALTIME` samples at 50 ms spacing; elapsed monotonic span 10.003 s. |
| Continuity | Monotonic order passed; maximum adjacent wall-minus-monotonic offset change 6,166 ns, below the locked 10 ms threshold. |
| Timestamp mapping | Maximum monotonic read bracket 14,695 ns; 7,348 ns is the read-bracket-only same-host mapping bound. Nominal reported clock resolution is 1 ns. |
| Thermal/throttling | `UNAVAILABLE`: WSL2 guest exposes no readable thermal sensor or thermal throttle counters; no zero was substituted. |
| Cross-host offset/uncertainty | `UNAVAILABLE`: no independent second host or approved deployment was provided; recorded as `null`, not zero. |

This survey supports only **single-host clock continuity during its 10-second
window**. It does not bound earlier T01–T06 clocks continuously, provide
cross-host timestamp uncertainty, prove synchronized UTC, or prove an absence
of thermal throttling. The monotonic spans already captured in earlier runs
remain separate evidence. The 7,348 ns bracket is not a bound on host-to-UTC
or host-to-host error.

**T07 status: PARTIAL; checklist box remains open.** To finish T07, the
Owner must provide at least two independent controlled private hosts/VMs,
authorized access, clock synchronization information and network placement.
Then perform the registered per-host inventory, before/after offset and
uncertainty measurements, warm-up/drain check and bounded multi-host run.
Do not call four validators on this WSL2 host a multi-host test. Gate T is
not met until that evidence passes and T07 is verified.
