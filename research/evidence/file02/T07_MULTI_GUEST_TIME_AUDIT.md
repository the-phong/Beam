# T07 — host and time calibration on two VMware guests

Date: 2026-11-13. The [locked T07 plan](../../protocol/FILE02_T07_HOST_TIME_PLAN.md)
and two-guest probe were committed before the final capture at
`dcbe8bd9d50bc47622605bcf5821940d587e5ae6`. The Owner supplied U1/U2
under `D:\VMware`; both were reached over the private VMware NAT network
using temporary SSH control sessions. The VMX folder names and display
names are swapped: `U2/U2.vmx` is display U1, MAC ending `a3:e3:8d`,
guest `u1-ubuntu1`; `U1/U1.vmx` is display U2, MAC ending `86:25:e4`,
guest `u2-ubuntu2`. The VMX UUIDs, MACs and guest machine-ID hashes differ.
Both guests run on **one physical Windows workstation/hypervisor**. This is
a two-guest host/time calibration, not independent physical infrastructure.

The final private frozen session is
`research/artifacts/file02/t07-multiguest-20261112T195237Z-5f747b41`.
Manifest SHA-256:
`352f1bbbfcde2174b177b713ff00aa60eb256c8d063d739927e02924f6a47743`.
`python3 research/tools/file02_t07_multiguest.py capture` exited 0;
`verify` exited 0, rechecking every raw file hash and recomputing the
summary. Each guest had 32 causal round-trip offset probes before and after
its survey. WSL, U1 and U2 each had a one-second warm-up, 200 monotonic/wall
samples at 50 ms spacing over about 10.03 seconds, and a three-second drain.

| Host | Clocksource / sync | CPU / fs / network | Max adjacent wall–monotonic step | Read-bracket mapping bound | Thermal/throttle |
| --- | --- | --- | ---: | ---: | --- |
| WSL controller | `tsc`; `NTPSynchronized=yes`, `NTP=no` | i7-12700H, 20 logical CPUs; ext4, 81.09% free; WSL eth0 MTU 1280 | 0.022 ms (including offset probes) | 0.024 ms | `UNAVAILABLE` |
| U1 guest | `tsc`; `systemd-timesyncd=active`, NTP/synchronized=yes | 2 vCPU; ext4, 46.63% free; private NAT ens33 MTU 1500 | 0.036 ms | 0.038 ms | `UNAVAILABLE` |
| U2 guest | `tsc`; `systemd-timesyncd=active`, NTP/synchronized=yes | 2 vCPU; ext4, 46.63% free; private NAT ens33 MTU 1500 | 0.130 ms | 0.132 ms | `UNAVAILABLE` |

All three report the Intel i7-12700H model and expose `constant_tsc`,
`nonstop_tsc`, `tsc_reliable`, `rdtscp` and `hypervisor` flags. All monotonic
samples were ordered; no observed wall-minus-monotonic step exceeded the
locked 10 ms discontinuity threshold. The guest kernel is
`6.14.0-37-generic`; the controller is WSL2 `6.18.33.2`.
Thermal sensors and throttle counters were not exposed to these guests,
so their values are `null`/`UNAVAILABLE`, never zero.

The causal offset interval uses local send/receive wall timestamps `c0/c3`
and remote receive/reply timestamps `r1/r2`: `[r1−c3, r2−c0]`. It makes no
symmetric-path assumption. The narrowest interval from 32 probes at each
boundary gave U2-minus-U1 offsets:

| Boundary | Offset interval (ms) |
| --- | ---: |
| Before survey | [6.007, 9.048] |
| After survey | [6.988, 9.553] |
| Conservative before/after envelope | **[6.007, 9.553]** |

The envelope width is 3.547 ms (midpoint 7.780 ms, half-width 1.773 ms).
This is an **observed clock-mapping bound for this short survey**, not an
absolute UTC-accuracy guarantee. Any later cross-guest latency or alert lead
time needs new before/after bounds around its own run; an effect that overlaps
the uncertainty interval is unresolved. The survey sends no transactions
and provides no multi-host four-validator throughput/correctness result.
Both guests' wall clocks were also about **3.45 seconds behind the WSL
controller** during this survey despite their `NTPSynchronized=yes` flags.
Thus uncorrected UTC timestamps across WSL and either guest would be wrong;
the causal per-run offset mapping is mandatory. Subtracting the two
guest-to-controller intervals yields the U2-minus-U1 bounds above.

The first capture at `t07-multiguest-20261112T194950Z-05206760` was
`INVALID_FOR_ANALYSIS`: its original audit required a guest DMI product UUID
that unprivileged accounts could not read. It remains frozen privately with
manifest `fa23722f24da70e9a6b67820d3991ecaaa7b94ac2e863190f5da462c719c30aa`.
The next passing capture at `t07-multiguest-20261112T195111Z-06e6d17d`
remains frozen with manifest
`90fcc646c56d26fe4e2f8d25d154faa9b557ec5050f20508cf5ba9c847f93eca`;
the final run supersedes it by also recording clocksource and sync daemon.
No historical raw bytes were modified or silently reused.

**T07 host/time preflight: PASS for the bounded two-guest virtual topology.**
It verifies distinct guest hosts, measured clock continuity and published
timestamp uncertainty. It does not make the deferred
`FOUR_VALIDATOR_MULTI_HOST_PRIMARY` factor available, establish independent
physical-host behavior, or support an RQ/DoS inference. Gate T is ready for
Controller/Owner review within this pre-case scope.
