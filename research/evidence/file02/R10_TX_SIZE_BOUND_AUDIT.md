# R10 — canonical and Comet transaction size boundary

Date: 2026-11-13. Source `c8f13cc` locked the bounded diagnostic before
the accepted run. The H codec permits at most a 50-byte chain ID, yielding
a 2,602-B canonical transaction; Comet's unchanged `max_tx_bytes` is
8,192 B. A valid H transaction at 2,602 B, malformed 8,192-B extension
and 8,193-B extension were sent one second apart to a fresh four-validator
single-host network through validator-0 loopback RPC. All three fit the
unchanged S06 rate, byte, duration and endpoint caps. A per-run plan with
actual endpoint and config hashes was written before sends. The harness
checked a manual STOP and fresh zero-drop observers before each send; it
did not claim S06's separate-process `HostGuard` was active.

Private session: `research/artifacts/file02/r10-20261113T072332Z-06d16341`.
Plan SHA-256 `c56ae23608b4532f07260e8cc3eede61fc5457fa254bd914cc519efe38e0d0a6`;
immutable raw manifest SHA-256
`b810aa4529d26586376d00f8fc1261f2ae73ce6d299727a038ff6f12648736e0`.
Integration exited 0 and independent
`python3 research/tools/file02_r10.py verify
research/artifacts/file02/r10-20261113T072332Z-06d16341` exited 0.
The private session has locked plan, immutable raw, per-run JSON/CSV,
QC, aggregate, figure status and decision. All four validators agreed
on BlockID/AppHash; only the valid transaction committed once.

| Input | RPC result | App CheckTx | Commit | One observed app service span |
| --- | --- | --- | --- | ---: |
| Valid canonical H, 2,602 B | code 0 | accepted | once | 437.4 µs |
| Malformed extension, 8,192 B | code 1 | structural rejection | never | 9.9 µs |
| Extension, 8,193 B | `mempool_tx_too_large` | not reached | never | `UNAVAILABLE` |

The single observed spans are diagnostic, not means or CPU estimates.
Per-request Comet precheck CPU is `UNAVAILABLE`; with one network there
is no confidence interval. A previous attempt
`research/artifacts/file02/r10-20261113T072204Z-642fa2e2` stopped
before any send because S06 `HostGuard` rejects an in-process integration
application. Its raw manifest
`f31988c801e4010d0bad8f13954d82bbb41d91fd05f8ae4efab0a04c519ea515`
is retained as failed diagnostic provenance, not pooled with the accepted
run.

**R10 is PARTIAL and unchecked:** the valid-near-8,192-B arm is
`NOT_FEASIBLE_WITHIN_ACCEPTED_CODEC`; the actual oversize rejection stage
is established. No claim of DoS or capacity follows.
