# Gate T — Owner conditional acceptance

Decision date: 2026-11-13. The Owner accepted T01–T07 technically within
the bounded preflight scope on `research/file02-t-preflight`, source
`942b9d7403cf68d12849343b972716002a350003`. Draft PR #22 had two
passing `verify` checks; Draft PR #21, its accepted COMMON dependency, had
two passing `verify`, one `scan` and one `smoke` check. This is an Owner
decision, not a new experimental observation.

| Case | Verified evidence |
| --- | --- |
| T01–T04 | [Progress report](T_PREFLIGHT_PROGRESS_REPORT.md) links baseline, codec, rejection and mixed-consensus audits. |
| T05 | [Recorder perturbation audit](T05_RECORDER_PERTURBATION_AUDIT.md): matched ON/OFF preflight; no overhead CI. |
| T06 | [Formal calibration audit](T06_FORMAL_CALIBRATION_AUDIT.md): 16 fresh networks at bounded 2/6/12/20 tx/s; no knee claim. |
| T07 | [Host/time audit](T07_MULTI_GUEST_TIME_AUDIT.md): single-host and two VMware guest surveys; two guests share one physical hypervisor. |

The accepted scope permits preparation of RQ1 and RQ1b plans for the
four-validator single-host topology, with scientific limits stated. It does
not authorize a formal R/A/Q/D run. Multi-host performance, distributed
DoS, exact cross-host latency and P2P-origin effects lack measurements and
must not be concluded. `FOUR_VALIDATOR_MULTI_HOST_PRIMARY` remains
unsupported. Before each official case, separately lock workload, offered
rate, independent repetitions, endpoint, CI procedure, safety caps,
topology and QC; the future case artifacts remain pending.

The Owner directed dependent PR completion in order: merge #21 into
`test-mainet`, then retarget/review/merge #22. A subsequent planning-only
branch begins from the verified merged head. No protected branch is pushed
directly; no merge or case execution is inferred from a passing CI alone.
