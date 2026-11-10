# Current research program (RQ1–RQ3)

This directory holds the new controlled-profile study. The earlier B0/B1
engineering phases and Phase 05 results remain under [`research/`](../research/PHASE_STATUS.md).
They are historical inputs, not measurements for this program.

| Directory | Contents |
|---|---|
| [`checklists/`](checklists/01_CHECKLIST_CHUAN_HOA_MO_RONG_REPO_THUC_NGHIEM.md) | The tracked V2 execution checklist. The operator's local progress copy lives beside the `pluto` directory. |
| [`protocol/`](protocol/RESEARCH_PROTOCOL.md) | Research questions, threat model, measurement contract and four-profile equivalence. |
| [`corpus/`](corpus/s04_public_corpus.json) | Public, hash-checked synthetic transaction bytes and genesis. No signing keys. |
| [`evidence/`](evidence/s04/CORRECTNESS_REPORT.md) | S01–S04 audits, reports and publication-safe summaries. `m01/` records the earlier baseline inventory. |
| `tools/` | Small research preparation scripts. |
| `artifacts/` | Git-ignored local backups, raw logs and private runtime material. Do not publish this directory. |

The obsolete V1 checklist and a byte-identical local copy of checklist 02
are preserved under `artifacts/private-baseline/checklists/` on this machine.
The original checklist 02 remains beside `pluto`; neither old document is an
S05 authorization. The tracked V2 checklist and
[`CORRECTNESS_REPORT.md`](evidence/s04/CORRECTNESS_REPORT.md) are the S04
handoff. Rebuild and check it with `make verify` and `make test-s04-4v` from
the repository root. Four validators in S04 run on one localhost host.
