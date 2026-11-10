# M01-07 — public-history and new-artifact security audit

Baseline and scope: `origin/test-mainet` at
`0bdd869f2c6b3ed59b0ae825954d8667b3c7481e`; new worktree on
`research/reproducible-testbed`. No historical file was removed or rewritten.
No private raw report is committed with this audit.

## Checks and observations

| Check | Result |
|---|---|
| Official Gitleaks v8.30.1 Linux x64 archive checksum | Downloaded from the official release; SHA-256 matched the published `551f6fc83ea457d62a0d98237cbad105af8d557003051f41f3e7ca7b3f2470eb`; `gitleaks version` returned `8.30.1`. |
| `gitleaks git --redact=100 --log-opts='origin/test-mainet' .` | Exit 0; 26 commits, about 11.28 MB scanned, no findings. Redacted JSON report kept outside repo. |
| `gitleaks dir --redact=100 .` | Exit 0; about 11.09 MB current worktree scanned, no findings. Redacted JSON report kept outside repo. |
| `gitleaks git --redact=100 --log-opts='--all' .` | Exit 1; 27 commits, one `generic-api-key` match in Markdown prose at `ARCHITECTURE.md:38` on unrelated `refactor/unified-architecture` commit `97c35a1`. Manual structural triage found no credential assignment, URL or encoded token; the match begins with ordinary words “key registry”. No raw candidate value is reproduced here, no automatic allowlist was added. |
| Git ancestry filename audit | No `priv_validator_key.json`, `priv_validator_state.json`, `node_key.json`, `.env`, `.pem`, or `.key` path was found in baseline ancestry. This is a filename check, not proof that arbitrary content is harmless. |
| Existing raw logs | 280 tracked files are under historical `research/evidence/**/raw/` at the baseline. They remain at the immutable source revision and are not copied into new public artifacts. Their privacy status is not silently upgraded. |

`.gitignore` now excludes new `research-v2/artifacts/`, private evidence,
raw subdirectories and local Gitleaks output. A pinned
`.github/workflows/research-v2-secrets.yml` downloads a checksum-verified
Gitleaks executable and scans proposed commits relative to `test-mainet` on
push/PR. Its hosted execution is pending until the branch is pushed. The
newly published inventory files contain only code/spec paths, SHA-256,
classification, aggregate counts and test outcomes; no keys, homes or raw
network output are copied. Future public data require their own sanitization
and review under M10, so this audit alone does not approve publication of
future run data.
