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
| T07 | open | [host/time partial audit](T07_HOST_TIME_PARTIAL_AUDIT.md): single-host clock continuity verified; independent multi-host placement and cross-host uncertainty unavailable. |

The T06 formal plan and implementation were committed before its run at
`5e8998f11441e8ce6be6a9552ac17936a536b706`. T06 raw manifest SHA-256:
`477765a706a65d2d9c5a94ef15ebe70ed2a09c24c297321eef1c088260883b75`.
The T07 single-host plan and capture code were committed before its survey
at `f0478ab6742fcecc7feb95b8ce0050447ab10d6c`; raw manifest SHA-256:
`60275183e7c4f04bc5f70642e2c5d3633003b1d4fd3d40bbc546dc449a533239`.
Both private sessions remain Git-ignored and immutable. No historical raw
evidence or accepted safety caps were changed.

Verification at this handoff: `make test-file02-t06-formal` exited 0;
`file02_t06_formal.py verify` exited 0; `file02_t07_host.py capture` and
`verify` exited 0; `make test-file02-common` passed 31 Python tests;
`make verify` passed Go build, unit tests, vet, module verification and
version check; `git diff --check` passed. The T07 tests include a synthetic
wall-clock step that correctly fails continuity.

**Gate T remains open.** T07 needs at least two independent controlled
private hosts/VMs, approved access and clock synchronization details. The
registered per-host inventory, before/after cross-host offset and uncertainty,
warm-up/drain and bounded deployment must then be executed and audited. The
single-host WSL2 survey cannot substitute for that. Do not begin R/A/Q/D
cases or infer RQ/DoS findings. T06 rates are tested rates, not a knee or
sustainable capacity claim. Thermal/throttling metrics are unavailable in
this guest; the T05 descriptive perturbations lack confidence intervals.
