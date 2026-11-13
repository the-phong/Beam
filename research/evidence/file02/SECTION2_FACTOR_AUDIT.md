# Section 2.1 factor and level audit

The versioned [factor registry](../../config/file02_factors.json) was checked
against Checklist 02 section 2.1, `RESEARCH_PROTOCOL.md`, the S06 safety
config and the T06 pilot evidence. C/CP/P/H and nine workload classes retain
their documented meanings. `MLDSA_INVALID` is invalid for C/CP;
`ECDSA_INVALID` is invalid for P. Capability D and multi-source workloads
remain unavailable. The four comparison regimes remain separate; matching
offered bytes is not matching CPU or accepted mempool pressure.

The safe, tested **pilot** offered rates are 2, 6, 12, 20, 23 and 25 tx/s.
The 2/6/12/20 cells come from the [calibration](T06_CALIBRATION_PILOT.md);
20/23/25 come from the [bounded knee search](T06_KNEE_PILOT.md). These are
diagnostic rates on four validators placed on one host. `PILOT_LOW` is
limited to 2 and 6 tx/s; the 12/20/23/25 levels are tested without a
causal load-region label. `SUBCRITICAL` is
`NOT_ESTABLISHED`; `NEAR_KNEE`, `AT_KNEE` and `SLIGHTLY_OVER_KNEE` are each
`NOT_OBSERVED_WITHIN_SAFE_RANGE`. Neither 23 nor 25 tx/s is a knee level.
The literal preregistered C/H knee flags remain untouched in the pilot raw;
the report explains the end-of-window censoring that prevents a credible
knee conclusion. The 25 tx/s and 60-second S06 caps are unchanged.

| Factor | Available now | Unsupported/deferred |
| --- | --- | --- |
| Origin | RPC single source in QA | RPC multi-source, P2P-origin injection; observed P2P relay from RPC is not injection |
| Topology | Four validators, one host, loopback diagnostic | Four-host primary topology and cross-host clock control |
| Policy | Baseline verification order | Reverse order and early rejection |
| Load | Six tested pilot rates | Numerical near/at/above-knee labels and confirmatory safe capacity |

The registry describes testability and scientific status, not authority to
start official T/R/A/Q/D cases. Every future case still needs its own locked
rate, workload/profile compatibility, independent-run target and oracle.
