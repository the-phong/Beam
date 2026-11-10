# M01-05 — documentation versus merged HEAD

`git show -s --format='%H %s' origin/test-mainet` exited 0 and returned
`0bdd869f2c6b3ed59b0ae825954d8667b3c7481e Merge pull request #10 from
the-phong/phase/05-b1-experimental-evaluation`.

The baseline's README still described only single-validator profiles and
said Phase 03E and later work were unapproved. The architecture/status/master
plan/manifest still used Phase 05 candidate wording as current status.
The current-state sections of `README.md`, `research/ARCHITECTURE.md`,
`research/PHASE_STATUS.md`, `research/MASTER_PLAN.md` and
`research/config/manifest.yaml` now identify the merged Phase 05 baseline,
preserve older statements as dated history, and distinguish the new checklist
from historical B0/B1 results. `AGENTS.md` records the later Owner request;
D-031 in `research/DECISIONS.md` records the provenance choice. One broken
historical status link to an untracked Phase 03D document was corrected to the
tracked Phase 03D report without importing the unrelated local artifact.

Checks after the edits:

- `git diff --check`: exit 0.
- Python `yaml.safe_load` of `research/config/manifest.yaml`: exit 0; current
  Phase 05 status is `merged_into_test_mainet_PR10_historical_bounded_evidence`.
- Local Markdown link check over the four edited README/architecture/status/
  plan documents: exit 0, `missing links: []`.

This is a documentation consistency check, not independent approval of the
historical Phase 05 measurements or of any new research protocol.
