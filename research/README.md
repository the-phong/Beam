# Research directory

All research material lives under this directory. The earlier B0/B1 engineering
phases and Phase 05 measurements are historical inputs to the current RQ1–RQ3
study; they are not new RQ1–RQ3 results.

| Path | Contents |
|---|---|
| [`checklists/`](checklists/01_CHECKLIST_CHUAN_HOA_MO_RONG_REPO_THUC_NGHIEM.md) | Active S01–S07 checklist. The operator's progress copy is beside `pluto`. |
| [`protocol/`](protocol/RESEARCH_PROTOCOL.md) | RQ1–RQ3 contract, threat model, measurement contract and C/CP/P/H equivalence. |
| [`corpus/`](corpus/s04_public_corpus.json) | Public S04 synthetic corpus alongside historical Phase 05 corpora. |
| [`evidence/s01/`](evidence/s01/BASELINE_AUDIT.md) | Baseline audit and the former M01 inventory together. |
| [`evidence/s02/`](evidence/s02/CONTRACT_AUDIT.md), [`s03/`](evidence/s03/PROFILE_AUDIT.md), [`s04/`](evidence/s04/CORRECTNESS_REPORT.md), [`s05/`](evidence/s05/TELEMETRY_REPORT.md) | New study evidence and bounded gate reports. |
| [`evidence/phase_02/`](evidence/phase_02), [`PHASE_REPORTS/`](PHASE_REPORTS) | Historical engineering evidence and reports; retain their manifests and distinct fault cases. |
| [`benchmarks/`](benchmarks/b0b1), [`tools/`](tools/freeze_baseline.py) | Historical B0/B1 analysis and the baseline hash tool. |
| `artifacts/` | Git-ignored local backups, raw logs and private runtime material; never publish this directory. |

The previous `research-v2/` directory was merged here. Historical reports
retain their original commands and hashes, including paths used at the time;
for a current rerun, substitute `research/` for `research-v2/`. The public
files were moved without changing their bytes unless a current link or path
needed repair. The former M01 inventory now lives with S01 evidence.
The current C/CP/P/H implementation is grouped under
[`internal/profiles/`](../internal/profiles/tx.go), with purpose-specific
subpackages. Its tools are `cmd/profile-corpus/` and `cmd/profile-vectors/`.
Historical report commands retain the import paths and command names used
at the time; use these current locations for reruns.
The B0/B1 corpus and analysis code is under `internal/benchmarks/`,
`cmd/b0b1-corpus/`, `corpus/b0b1/` and `benchmarks/b0b1/`. Historical Phase 05
source manifests and log hashes keep their original path strings as evidence
of the source revision; the current `Makefile` uses the relocated paths.

The obsolete V1 checklist and a byte-identical local copy of checklist 02
are preserved under `artifacts/private-baseline/checklists/` on this machine.
The original checklist 02 remains beside `pluto`. Neither old document is an
S05 authorization. The Owner merged S04 PR #15 and explicitly authorized
S05. The [S05 telemetry report](evidence/s05/TELEMETRY_REPORT.md) and
[metric dictionary](protocol/METRICS_DICTIONARY.md) are the current handoff.
Run `make verify` and `make test-s05-telemetry` from the repository root.
S05's four validators run on one localhost host; its full gate is still open.
