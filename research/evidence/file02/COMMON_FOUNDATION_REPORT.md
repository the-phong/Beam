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
| COMMON-02 | Verified for available single-host QA path | [Phase audit](COMMON02_PHASE_AUDIT.md) proved fresh run/validator homes, disjoint monotonic sends and a live-network cooldown across C/CP/P/H. Future case runs must retain this contract. |
| COMMON-03 | Verified for available single-host QA path | [Paired audit](COMMON03_FIXED_LEGITIMATE_AUDIT.md) used eight fresh networks; each benign/attack pair kept identical legitimate bytes, schedule, accounts and nonce, then added four distinct invalid sends. All valid transactions committed once and all invalid transactions were rejected. Future cases must use the same invariant. |
| COMMON-04 | Verified at bounded QA rates | [Equal-rate audit](COMMON04_EQUAL_RATE_AUDIT.md) executed eight fresh four-validator networks: exact 2 tx/s across C/CP/P/H, and 2,573 bytes/s target within ±2.49% across C/CP/P/H. Confirmatory higher-rate comparisons still require T06. |
| COMMON-05 | Verified at bounded QA rates | [Matched-pressure audit](COMMON05_MATCHED_PRESSURE_AUDIT.md) ran four new benign/attack pairs; each profile matched 2 offered tx/s and identical encoded bytes/s in every steady window while declaring accepted rate, mempool occupancy and verification path unmatched. Confirmatory case-specific comparisons remain future work. |
| COMMON-06 | Verified for bounded single-host QA | The initial [precision pilot](COMMON06_PRECISION_PILOT.md) failed at n=36/profile. After Owner-authorized bounded T06 [calibration](T06_CALIBRATION_PILOT.md), the [precision extension](T06_PRECISION_PILOT.md) ran 48 fresh networks, 12/profile and n=2,880 steady transactions/profile. All four met the preregistered rank and cluster CI gates; latency is a block-scan upper bound, not exact commit time. |
| COMMON-07 | Verified for bounded QA path | [Run identity audit](COMMON07_RUN_IDENTITY_AUDIT.md) rehashed and QC-validated all 48 T06 pilot runs before producing [per-run records](COMMON07_RUN_RECORDS.json) with source tree, profile, case, capability and planned/achieved tx/s and bytes/s. Future case producers must satisfy the same gate. |
| COMMON-08 | Partial | QC code excludes declared fault flags and rate misses; independent fault signal integration awaits case producer. |
| COMMON-09 | Open by checklist order | Loopback-only four-validator smoke and S06 safety guards passed, but the item remains unchecked until COMMON-06–08 and future case guard wiring are verified. |
| COMMON gate | Open | Precision pilot, complete preregistration and case-specific oracle integration remain. |
| 2.1 / 2.2 | Registry/dictionary implemented | P2P origin, multi-host primary and numerical knee levels are explicitly deferred. |
| 2.3 | Artifact foundation tested, per-case outputs open | No T/R/A/Q/D experiment exists here; figures, CI and decisions cannot be populated truthfully. |

All unverified boxes remain unchecked in the outside Checklist 02. The
foundation's synthetic test data are neither a fresh File 02 experiment nor
evidence for RQ1–RQ3, capacity, benign FPR, DoS or an ML detector. The
separate bounded T06 pilot supports only the stated p99 precision gate.
