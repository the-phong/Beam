# Checklist 02 section 2 — pre-case readiness handoff

Date: 2026-11-12. Branch `research/file02-common-foundation`, Draft PR #21.
This is an engineering readiness report for Controller/Owner review; it
contains no confirmatory T/R/A/Q/D case or RQ result. The accepted S02
research contract, prior T06 preregistration and historical raw evidence
were not changed.

## 2.1 Factors

The [factor audit](SECTION2_FACTOR_AUDIT.md) maps every Checklist 02 factor
and level to the versioned registry. The tested pilot rates are
2/6/12/20/23/25 tx/s. `NEAR_KNEE`, `AT_KNEE` and `SLIGHTLY_OVER_KNEE` are
`NOT_OBSERVED_WITHIN_SAFE_RANGE`; 23/25 tx/s are not reclassified as knees.
The original T06 knee flags are preserved. RPC multi-source, P2P-origin
injection, four-host topology and reverse/early-reject policies are marked
unsupported. Case-specific load plans and independent-run targets remain
future preregistration work.

## 2.2 Metrics

The [dictionary](../../protocol/FILE02_METRICS.md) and its
[63-entry machine registry](../../config/file02_metrics.json) give each
field a definition, unit, capture point, availability time, calculation,
limitation and status. The common QC tool rejects non-null values for
`UNAVAILABLE` fields and requires a reason for every null. Exact inclusion
latency, full ingress queue wait, crypto CPU attribution and relay
propagation remain unavailable. The pilot's CheckTx socket queue wait and
100-ms block-scan inclusion upper bound have distinct definitions. Eventual
goodput is not treated as sustainable steady throughput. Detection fields
remain pending official case execution.

## 2.3 Artifacts

The [artifact workflow](../../protocol/FILE02_ARTIFACTS.md),
[template registry](../../config/file02_artifact_templates.json),
[plan template](../../config/file02_plan.template.json) and
[`file02_section2.py`](../../tools/file02_section2.py) now specify and
validate all pre-case contracts. A locked plan must declare H0/H1, endpoint,
independent unit, CI method, exclusion rule, expected oracle, safety caps,
source/config identities and five raw streams. Raw freezing checks SHA-256,
symlinks, Git ignore and write-once behavior. The run-bundle validator
recomputes QC/results from raw, checks CSV identity and aggregate provenance,
and rejects a fabricated CI or scientific figure/decision. A pending case
template contains only a draft plan, status manifest, empty figure manifest
and `decision.md` marked `PENDING_CASE_EXECUTION`.

The original Checklist 02 boxes for `plan.json`, `raw/run_id/`, per-run
result, QC report, aggregate, figures and decision remain **unchecked**:
they require an output for **each future authorized case**, which does not
exist yet. Their schemas, templates and validators are ready. A real
multi-run CI and scientific figure are not produced by this foundation.

## Research consistency and gate

The [RQ1 comparison report](SECTION2_RQ1_CONSISTENCY.md) finds the accepted
H–CP valid-transaction CheckTx cost contrast aligned across S02 protocol,
common contract and Checklist 02's resource wording. The Owner's current
question about legitimate-service degradation under hybrid-invalid load is
a different causal contrast. No accepted hypothesis or endpoint was changed.

Gate COMMON remains open for Controller/Owner disposition of the missing
near/at/above-knee levels, the prospective correction to the boundary-biased
pilot knee rule, and the RQ1 question wording. Official case plans still
require their own independent-run, measurement, safety and oracle review.

## Verification and preserved boundaries

- `python3 research/tools/file02_section2.py validate-registry`: exit 0,
  63 fields and all artifact templates validated.
- `make test-file02-common`: exit 0, 27 focused Python tests. The section 2
  fixtures reject a missing output, wrong QC/schema/provenance, altered CSV
  column, non-null unavailable metric and fabricated CI. Existing raw hash,
  safety, factor compatibility and phase tests still pass.
- `make verify`: exit 0; Go build/unit tests, vet, format and module checks
  passed on Go 1.24.1, linux/amd64.
- `git diff --check`: exit 0.

Only factor/metric/artifact registry, their validators/tests and governance
documentation changed. No historical raw evidence, ignored private
artifact, accepted application/protocol source, safety cap, official case
or protected branch was altered. The local source branch remains the sole
target of the Draft PR.
