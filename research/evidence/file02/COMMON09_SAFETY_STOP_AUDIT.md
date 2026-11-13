# COMMON-09 — isolated endpoint and safety stop QA

Date: 2026-11-12. Scope: one intentionally stopped four-validator H-profile
QA run. The producer used source commit
`d482a48` on `research/file02-common-foundation`. It created new validator
homes and keys in a private temporary network and pinned the unchanged S06
limits. The auditor checked four distinct RPC addresses and found all four
hosts equal to `127.0.0.1`; the workload target was one of those watched
nodes. No public endpoint was in the allowlist.

The private STOP file was installed immediately before invoking the bounded
generator. Its `HostGuard.Check` refused preflight with `SAFETY_ABORT` and
`manual kill switch`; `actual_send_count=0`, no event result and generator
wall time 6.4 ms. The missing post-run oracle was marked
`INVALID_FOR_ANALYSIS` with `INCOMPLETE_EVIDENCE`, rather than entering an
aggregate. Four recorder streams and the refusal result remain in the
frozen private raw. `python3 research/tools/file02_common09.py smoke`
returned 2 because its first audit parser expected a list for the Go
preflight result's null `events` field; the producer had already stopped
correctly. A parser-only fix then re-audited that same run with `finalize`,
exit 0, without a rerun or a change to raw. `verify` exited 0.

Frozen session:
`research/artifacts/file02/common09-20261112T145717Z-7e4881f2`.
Manifest SHA-256:
`ae4e28adcd1e990d17c55d1f33410ae39fb63458d80a9180349d6314222b9258`.
The standalone `go test ./internal/experiment/load -run
'TestS06RuntimeGuardAbortsOutstandingLoad|TestS06KillSwitchAndPublicTargetRefused'
-count=1 -v` exited 0, covering an in-flight observer guard failure and
rejection of a public RPC endpoint. The real QA run demonstrates the same
guard's preflight stop and private raw retention. It does not claim an
observed CPU/RSS cap breach or an attack result.
