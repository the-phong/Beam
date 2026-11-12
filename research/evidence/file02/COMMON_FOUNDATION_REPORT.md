# File 02 sections 0–2 — implementation and bounded QA report

Date: 2026-11-12. Branch: `research/file02-common-foundation` from merged
`test-mainet` commit `e0bcd9e0f903a2e807a3d28adc5554dffa48e1f1`
(tree `71d03c1d083724b1bba73e62b024766cfaaedc5a`). The Owner explicitly
authorized sections 0–2 of the outside Checklist 02. This is an engineering
candidate for independent Controller/Owner review, not a scientific result.

## Delivered

- Section 0: [locked definitions candidate](../../protocol/FILE02_COMMON_CONTRACT.md)
  for RQ1–RQ3, C/CP/P/H, origin, capability, stage, labels, matching,
  independent-run unit and prohibited conclusions. RQ1–RQ3 inferential
  thresholds retain the S02 research/measurement contracts; no effect is
  asserted.
- Section 1: executable plan validation, write-once raw freezing/hash
  verification and QC
  exclusion in [`file02_common.py`](../../tools/file02_common.py). The tool
  rejects incompatible workload/profile/capability, unavailable P2P origin or
  multi-host topology, exceeded safety caps, unlocked plans, plan/raw hash
  mismatch, missing metric availability and output overwrite. It checks
  achieved steady rates and windows, all four-validator/clock/observer/guard
  flags, and prevents invalid runs from entering an aggregate. A flag must be
  supplied by an independent producer; the tool does not invent oracle data.
- Section 2: versioned [factor registry](../../config/file02_factors.json),
  [metric dictionary](../../protocol/FILE02_METRICS.md),
  [plan template](../../config/file02_plan.template.json) and
  [artifact workflow](../../protocol/FILE02_ARTIFACTS.md). Per-run JSON/CSV,
  QC JSON and an aggregate with independent n are executable. Missing metrics
  stay `null` with reasons; CI stays null until a preregistered method exists.

## Reproduction and QA evidence

| Command | Exit | Result |
| --- | ---: | --- |
| `make test-file02-common` | 0 | Fourteen tests including actual runtime mismatch, altered Comet config, disjoint monotonic phases, phase-boundary crossing and cooldown duration. |
| `make verify` | 0 | Build, Go unit tests, `go vet`, format and module verification passed; Go 1.24.1, linux/amd64. |
| `make test-s07-smoke` | 0 | Four fresh single-host loopback validator networks, one per C/CP/P/H; `runs=4`, `excluded=0`. This is QA only. |
| `python3 research/tools/s07_orchestrate.py verify research/artifacts/s07/s07-20261112T115501Z-81905384` | 0 | Rehashed 92 frozen files and rederived results; source commit `e0bcd9e0f903a2e807a3d28adc5554dffa48e1f1`. |
| `git diff --check` | 0 | No whitespace errors. |

The new private S07 QA session ID is
`s07-20261112T115501Z-81905384`; its manifest SHA-256 is
`a61f0e7e9e60af6acc192eb7e390a51892d83973d643d618f40df48ab26960d3`.
It is under ignored `research/artifacts/s07/`, with no private raw, signed
payload, runtime home or key published. The previously existing S07 session
was separately verified (92 files) and left untouched.

## Checklist disposition

| Item | Status | Evidence / unmet part |
| --- | --- | --- |
| 0 | Contract implemented for review | Scientific acceptance and case-specific numerical preregistration are separate. |
| COMMON-01 | Verified for available single-host QA topology | [Runtime audit](COMMON01_RUNTIME_AUDIT.md) captured actual software, CPU features, topology and 16 Comet configs across four profiles; future File 02 QC requires a matching inventory. Multi-host remains untested. |
| COMMON-02 | Partial | S07 fresh networks and phase separation passed; future File 02 cooldown/reset and monotonic records need case runs. |
| COMMON-03 | Partial | S06/S07 fixed-legitimate plans and distinct transactions exist; case-specific attack pair checks need integration. |
| COMMON-04 | Open | Equal tx/s and bytes/s confirmatory comparisons require T06 safe-rate pilot. |
| COMMON-05 | Open | Matched attacker/benign pairs and unmatched-variable declarations need real case plans. |
| COMMON-06 | Open | S07 has n=1/profile; order, repeated independent runs and p99 precision are unestablished. |
| COMMON-07 | Partial | Plan/result schemas require IDs and planned/achieved rates; future case producer must feed and validate every run. |
| COMMON-08 | Partial | QC code excludes declared fault flags and rate misses; independent fault signal integration awaits case producer. |
| COMMON-09 | Verified in bounded QA | Loopback-only four-validator smoke and S06 safety guards passed; future case runs must use the same guard. |
| COMMON gate | Open | Precision pilot, complete preregistration and case-specific oracle integration remain. |
| 2.1 / 2.2 | Registry/dictionary implemented | P2P origin, multi-host primary and numerical knee levels are explicitly deferred. |
| 2.3 | Artifact foundation tested, per-case outputs open | No T/R/A/Q/D experiment exists here; figures, CI and decisions cannot be populated truthfully. |

All unverified boxes remain unchecked in the outside Checklist 02. The
foundation's synthetic test data are neither a fresh File 02 experiment nor
evidence for RQ1–RQ3, capacity, p99, benign FPR, DoS or an ML detector.
