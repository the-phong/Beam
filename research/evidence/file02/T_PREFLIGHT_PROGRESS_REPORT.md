# File 02 T preflight — progress and handoff

Date: 2026-11-13. Branch `research/file02-t-preflight` is based on accepted
COMMON commit `0d753bf6e3247deec12ba2f83c055bf7dd16071d` (Draft PR #21,
still unmerged). Checklist 02 is the Owner's outside file at
`/home/ntp/research/02_CHECKLIST_MA_TRAN_THUC_NGHIEM_VA_KET_QUA.md`.

| Case | Checklist | Evidence and result |
| --- | --- | --- |
| T01 | checked | [baseline audit](T01_BASELINE_AUDIT.md): four fresh networks, valid transfer/state oracle. |
| T02 | checked | [size/codec audit](T02_SIZE_CODEC_AUDIT.md): C/CP/P/H byte sizes and CP–H exact match. |
| T03 | checked | [rejection audit](T03_REJECTION_PIPELINE_AUDIT.md): 34 direct ABCI cases and stage oracle. |
| T04 | checked | [mixed consensus audit](T04_MIXED_CONSENSUS_AUDIT.md): four fresh networks, valid/invalid mix and state oracle. |
| T05 | checked | [recorder audit](T05_RECORDER_PERTURBATION_AUDIT.md): four matched ON/OFF pairs, correctness and descriptive perturbation. |
| T06 | checked | [formal calibration audit](T06_FORMAL_CALIBRATION_AUDIT.md): 16 fresh networks; all registered rates achieved, zero dropped/unsent. |
| T07 | checked | [two-guest host/time audit](T07_MULTI_GUEST_TIME_AUDIT.md): WSL and two distinct VMware guests verified; before/after cross-guest offset interval published. Physical hardware is shared. |

The T06 formal plan and implementation were committed before its run at
`5e8998f11441e8ce6be6a9552ac17936a536b706`. T06 raw manifest SHA-256:
`477765a706a65d2d9c5a94ef15ebe70ed2a09c24c297321eef1c088260883b75`.
The T07 single-host plan and capture code were committed before its survey
at `f0478ab6742fcecc7feb95b8ce0050447ab10d6c`; raw manifest SHA-256:
`60275183e7c4f04bc5f70642e2c5d3633003b1d4fd3d40bbc546dc449a533239`.
The final multi-guest probe source was committed before capture at
`dcbe8bd9d50bc47622605bcf5821940d587e5ae6`; raw manifest SHA-256:
`352f1bbbfcde2174b177b713ff00aa60eb256c8d063d739927e02924f6a47743`.
The superseded and invalid T07 raw sessions remain frozen with reasons in
the [audit](T07_MULTI_GUEST_TIME_AUDIT.md). All private sessions remain
Git-ignored and immutable. No historical raw evidence or accepted safety
caps were changed.

Verification at this handoff: `make test-file02-t06-formal` exited 0;
`file02_t06_formal.py verify` exited 0; T07 single-host and multi-guest
capture/verify exited 0; `make test-file02-common` passed 33 Python tests;
`make verify` passed Go build, unit tests, vet, module verification and
version check; `git diff --check` passed. The T07 tests include a synthetic
wall-clock step and malformed offset probe that correctly fail validation.

**Gate T has a bounded technical pre-case candidate and awaits independent
Controller/Owner acceptance.** T07 verified two distinct guest operating
systems and clock bounds, but they share one physical hypervisor. The
`FOUR_VALIDATOR_MULTI_HOST_PRIMARY` factor remains unsupported; no
multi-host blockchain network or independent physical-host run was made.
Do not begin R/A/Q/D cases or infer RQ/DoS findings. T06 rates are tested
rates, not a knee or sustainable capacity claim. Thermal/throttling metrics
are unavailable in these guests; the T05 descriptive perturbations lack
confidence intervals.
