# File 02 artifact workflow v1

The outside Checklist 02 remains the operator checklist. Its sections 0–2
are implemented as a review candidate by the [common contract](FILE02_COMMON_CONTRACT.md),
[factor registry](../config/file02_factors.json),
[metric dictionary](FILE02_METRICS.md) and
[`file02_common.py`](../tools/file02_common.py). The template is
[`file02_plan.template.json`](../config/file02_plan.template.json). Copy it
into a private case directory and fill every field from the actual run
configuration. `DRAFT` can be inspected with `--allow-draft`; a QC run requires
`LOCKED`, a pilot reference, independent-run target and p99 precision rule.
The plan hash binds the observation. This local lock is a data-integrity
mechanism, not Controller/Owner scientific acceptance.

The case-specific producer must create one fresh private `raw/<run_id>/`
directory in ignored storage, then a `manifest.json` with schema
`pluto.file02.raw_manifest.v1` and a map of every relative file name to its
SHA-256. It must save an `observation.json` with schema
`pluto.file02.observation.v1`, `run_id`, canonical `plan_sha256`,
`raw_manifest_sha256`, all ten QC checks, achieved rates, steady windows,
metric groups and an availability reason for every null metric. The raw
directory is frozen before analysis and never reused. The QC booleans must
come from independent oracle, generator, collector and guard records; the
common tool checks their presence and excludes failures but does not create
those observations itself.

COMMON-01 requires a `runtime_inventory.json` inside each raw run before
freezing. Its `pluto.file02.runtime_inventory.v1` schema records actual
source/cleanliness, Go/Comet/crypto/OS versions, CPU model/features,
four-validator topology, mempool/block settings and each Comet `config.toml`
hash. `qc-run` compares this actual inventory with the locked plan and fails
closed on a missing or mismatched field. The bounded S07 QA inventory can be
captured with `python3 research/tools/file02_runtime.py capture-s07 PRIVATE/S07_SESSION`;
this verifies the S07 frozen manifest first and writes
a separate ignored, read-only File 02 COMMON-01 audit. S07 raw remains
untouched. This QA command is evidence for the capture path, not a
confirmatory experiment.
`python3 research/tools/file02_runtime.py verify-s07-capture PRIVATE/COMMON01_AUDIT`
rehashes the frozen audit and its source session, including all 16 actual
Comet config files. The [COMMON-01 report](../evidence/file02/COMMON01_RUNTIME_AUDIT.md)
records the bounded successful check.

COMMON-02 has a separate bounded QA command: `make test-file02-common02`.
It creates four fresh S07-style private networks with a one-second cooldown
after the generator drains and the oracle completes, then audits monotonic
phase markers, actual send offsets, the absence of sends during drain and
cooldown, and four distinct validator homes. The standard S07 command keeps
its original phase schedule. The COMMON-02 audit remains QA evidence only.

After writing every raw file, run
`python3 research/tools/file02_common.py freeze-raw PRIVATE/raw/RUN_ID`.
This creates the manifest once, hashes every file, rejects symlinks and a
non-ignored in-repository path, verifies the manifest and makes the raw tree
read-only. A rerun needs a new run ID. Keep the printed manifest SHA-256 in
the observation. The S07 orchestrator uses its own existing freeze format;
do not run this command on S07 sessions.

Run from the repository root:

```sh
python3 research/tools/file02_common.py validate-plan PRIVATE/plan.json
python3 research/tools/file02_common.py freeze-raw PRIVATE/raw/RUN_ID
python3 research/tools/file02_common.py qc-run PRIVATE/plan.json PRIVATE/observation.json PRIVATE/raw/RUN_ID PRIVATE/derived/RUN_ID
python3 research/tools/file02_common.py aggregate PRIVATE/result_aggregate.json valid_included_count PRIVATE/derived/*/result_per_run.json
```

`qc-run` emits `qc_report.json`, `result_per_run.json` and `.csv` once into a
new directory. `aggregate` requires a matching QC report beside each result,
refuses mixed strata or duplicate run IDs, excludes invalid runs and reports
independent n. It deliberately leaves the 95% CI null until a precision pilot
preregisters a procedure; one run is `INSUFFICIENT_INDEPENDENT_RUNS`. Neither
command computes an RQ effect, figure or decision. Those outputs belong to
the future case-specific analysis, after all common and case gates pass.

For foundation checks, use `make test-file02-common`, `make verify` and the
bounded `make test-s07-smoke`. The S07 command creates a separate private QA
session; its output must not be fed to the File 02 aggregate as a
confirmatory run. Existing S07 raw remains untouched. The new CLI uses only
Python standard library and adds no Go dependency.

## Section 2 contract after the bounded T06 pilots

The [factor registry](../config/file02_factors.json) records six tested
pilot rates and labels all three knee-relative levels
`NOT_OBSERVED_WITHIN_SAFE_RANGE`. It does not turn those pilot rates into
confirmatory case plans. The [metric registry](../config/file02_metrics.json)
defines every canonical field, unit, capture point, availability time,
calculation, limitation and status. The [artifact templates](../config/file02_artifact_templates.json)
declare the schemas and `PENDING_CASE_EXECUTION` placeholders for all eight
required outputs and the observation envelope.

`plan.json` now explicitly records H0/H1, primary endpoint, independent-run
unit, CI method, exclusion rule, expected oracle, safety caps, source/config
hashes and the five required raw streams. A future case producer must lock
those values before sending any transactions. `raw/<run_id>/` remains
write-once, ignored, manifest-hashed and read-only; the original raw files
must remain available even when QC excludes the run. A missing artifact or
invalid provenance is a validation failure, not a measured zero.

Use `python3 research/tools/file02_section2.py validate-registry` to audit
factor, metric and artifact schemas. `init-pending PRIVATE/CASE_DIR` creates
a draft plan, status manifest, empty figure manifest and pending decision;
`validate-pending PRIVATE/CASE_DIR` rejects any unreviewed result or figure.
`validate-run-bundle PRIVATE/CASE_DIR` checks a future locked plan,
manifested raw, observation/plan binding, recomputed QC/result, CSV identity,
run-level aggregate and pending figure/decision state. It does not execute
the case. The validator refuses a fabricated CI: scientific figures and RQ
decisions need a later, separately reviewed case-specific analysis contract.

The current handoff state for `result_per_run`, aggregate, figures and
decision is `PENDING_CASE_EXECUTION`. Their schema/validator readiness is
complete; the outside Checklist 02's per-experiment output boxes remain
unchecked until actual authorized cases create those artifacts. Gate COMMON
still requires Controller/Owner review of the knee-level limitation and the
[RQ1 wording discrepancy](../evidence/file02/SECTION2_RQ1_CONSISTENCY.md).
