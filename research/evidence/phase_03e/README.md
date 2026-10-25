# Phase 03E evidence handling

The public Phase 03E implementation report records the exact source tree,
commands, exit codes, test observations and SHA-256 manifest. Raw process
logs are retained in the local `research/evidence/phase_03e/` workspace and
are excluded from Git publication. The `PLUTO_4V_ARTIFACTS` environment
variable directs the disposable-process integration harness to save its
application and CometBFT logs under a private local directory. The raw logs
may contain runtime paths, node IDs and network details. Validator signing
keys, P2P private keys, last-sign state and node homes are never copied into
this evidence directory or the public report.

The sanitized manifest lists SHA-256 digests of retained logs so an
independent Controller can request and verify a private handoff.
`SOURCE_SHA256.txt` contains the 15 staged source/document blob hashes at
the tested pre-report checkpoint identified in the Phase 03E report.
`SOURCE_REVIEW.patch` contains those actual changes and tests verbatim,
excluding the later report and manifest. The Draft PR contains the final
full diff. Neither patch contains generated validator homes.

These tests use one WSL2 host and software process faults. Their results do
not establish independent fault-domain, Byzantine, physical power-loss,
PQC, hybrid authentication or DoS claims.
