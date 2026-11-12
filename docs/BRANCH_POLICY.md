# Branch status and Owner decision needed — 2026-11-12

| Branch | Observed role | Verified HEAD on 2026-11-12 |
| --- | --- | --- |
| `test-mainet` | Current research integration and base for reviewed PRs | `e8f20726f5cea94b47c4dcddb756e39d21207448` |
| `master` | GitHub default; older code and documentation | `5af503cf22cf47a0cde9cc6cc764b338cda2cae6` |

The [structural refactor PR #18](https://github.com/the-phong/Beam/pull/18)
was merged into `test-mainet`. [PR #13](https://github.com/the-phong/Beam/pull/13)
is still open from `test-mainet` to `master`; GitHub reports `CONFLICTING`
and `DIRTY`. It must not be merged or resolved as part of the refactor
governance closeout. The GitHub default branch has not been changed.

**Current operational rule:** new research work starts from the verified
`test-mainet` HEAD and goes through a reviewed PR back to `test-mainet`.
`master` remains an older default branch, not the current implementation
baseline. No agent may infer a `master` merge from a passing CI run.

**Owner decision pending:** choose whether `master` remains an archival stable
branch while `test-mainet` becomes the GitHub default for current code, or
whether a separately reviewed integration should update `master`. The first
option is recommended for discoverability because it makes GitHub's default
tree match the current research source without merging PR #13. Before any
default-branch change, review branch protection, CI triggers, links and
collaborator expectations. The second option requires a scoped review and
conflict resolution for PR #13. Neither action is authorized by this record.

**Naming:** GitHub repository `the-phong/Beam` is the renamed repository;
the application and CLI remain Pluto and the provisional Go module remains
`example.com/pluto`. The Owner has not decided on an application or module
rename. A repository rename alone is no reason to change imports, binaries,
protocol identifiers or evidence paths.
