# Research directory

All research material lives under this directory. The earlier B0/B1 engineering
phases and Phase 05 measurements are historical inputs to the current RQ1–RQ3
study; they are not new RQ1–RQ3 results.

| Path | Contents |
|---|---|
| [`checklists/`](checklists/01_CHECKLIST_CHUAN_HOA_MO_RONG_REPO_THUC_NGHIEM.md) | Active S01–S07 checklist. The operator's progress copy is beside `pluto`. |
| [`protocol/`](protocol/RESEARCH_PROTOCOL.md) | RQ1–RQ3 contract, threat model, measurement contract, C/CP/P/H equivalence and S06 testbed configuration. |
| [`corpus/`](corpus/s04_public_corpus.json) | Public S04 correctness and S06 workload corpora alongside historical Phase 05 corpora. |
| [`evidence/s01/`](evidence/s01/BASELINE_AUDIT.md) | Baseline audit and the former M01 inventory together. |
| [`evidence/s02/`](evidence/s02/CONTRACT_AUDIT.md), [`s03/`](evidence/s03/PROFILE_AUDIT.md), [`s04/`](evidence/s04/CORRECTNESS_REPORT.md), [`s05/`](evidence/s05/TELEMETRY_REPORT.md), [`s06/`](evidence/s06/TESTBED_REPORT.md) | New study evidence and bounded gate reports. |
| [`evidence/phase_02/`](evidence/phase_02), [`PHASE_REPORTS/`](PHASE_REPORTS) | Historical engineering evidence and reports; retain their manifests and distinct fault cases. |
| [`benchmarks/`](benchmarks/b0b1), [`tools/`](tools/freeze_baseline.py) | Historical B0/B1 analysis and the baseline hash tool. |
| `artifacts/` | Git-ignored local backups, raw logs and private runtime material; never publish this directory. |

The previous `research-v2/` directory was merged here. Historical reports
retain their original commands and hashes, including paths used at the time;
for a current rerun, substitute `research/` for `research-v2/`. The public
files were moved without changing their bytes unless a current link or path
needed repair. The former M01 inventory now lives with S01 evidence.
The current C/CP/P/H implementation is grouped under
[`internal/experiment/profiles/`](../internal/experiment/profiles/tx.go), with purpose-specific
subpackages. Its tools are `cmd/research/corpus/profiles/` and `cmd/research/vectors/profiles/`.
Historical report commands retain the import paths and command names used
at the time; use these current locations for reruns.
The B0/B1 corpus and analysis code is under `internal/experiment/corpus/b0b1/`,
`cmd/research/corpus/b0b1/`, `corpus/b0b1/` and `benchmarks/b0b1/`. Historical Phase 05
source manifests and log hashes keep their original path strings as evidence
of the source revision; the current `Makefile` uses the relocated paths.

The obsolete V1 checklist and a byte-identical local copy of checklist 02
are preserved under `artifacts/private-baseline/checklists/` on this machine.
The original checklist 02 remains beside `pluto`. Neither old document is an
S06 authorization. The Owner merged S05 PR #16 and explicitly authorized
S06, and subsequently merged PR #17 into `test-mainet`. The [testbed contract](protocol/TESTBED_CONFIG.md) and
[pilot report](evidence/s06/TESTBED_REPORT.md) are the bounded historical handoff.
Run `make verify` and `make test-s06-pilot` from the repository root.
The four validators run on one localhost host; independent infrastructure
and File 02 inference remain untested.

The Owner also merged [structural refactor PR #18](https://github.com/the-phong/Beam/pull/18)
at `e8f20726f5cea94b47c4dcddb756e39d21207448`; post-merge CI passed.
The [Controller review closeout](evidence/refactor/CONTROLLER_REVIEW_CLOSEOUT.md)
records conditional technical acceptance and limits. After PR #19 merged,
the Owner separately authorized S07. Its [reproduction instructions](../REPRODUCE.md),
[readiness report](evidence/s07/RUN_READINESS_REPORT.md) and
[public smoke example](evidence/s07/smoke_sample.json) cover a bounded
four-profile QA run. Private raw files and failed-run exclusion ledgers stay
under ignored `artifacts/s07/`. This smoke does not authorize File 02 or
establish RQ1–RQ3 results. The package paths above are current rerun paths.
