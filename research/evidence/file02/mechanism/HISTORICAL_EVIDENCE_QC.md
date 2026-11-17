# X02 — frozen R/A/Q evidence integrity and historical-tool audit

Date: 2026-11-15. Scope: 31 local frozen R/A/Q sessions under the ignored
private `research/artifacts/file02/` tree, including aborts and superseded
sessions. No raw file was changed. The public, keyless machine-readable
outputs are [source/QC audit](HISTORICAL_EVIDENCE_QC.json) and
[historical-tool raw audit](HISTORICAL_TOOL_RAW_AUDIT.json). Their rows name
sessions, source commits and manifest digests so a reviewer can locate
evidence without publishing private runtime homes, keys or unsanitized logs.

| Check | Result | Meaning |
|---|---:|---|
| Locked `source_commit` exists and `source_tree` equals its Git tree | 31/31 MATCH | Identifies the exact committed source tree recorded before each session. |
| `module_sha256` equals `go.mod` at locked source commit | 27/27 available MATCH; 4 `UNAVAILABLE_IN_PLAN` | Missing historical fields are not filled with today's module hash. |
| Top-level raw config equals plan `config_sha256` | 18/18 available MATCH; 11 absent in plan; 2 only per-run config | R08/R09 have case-specific per-run config files and no top-level `raw/config.json`; their accepted per-case QC remains the relevant check. |
| QC plan and raw-manifest hashes | 30/30 reports MATCH; one aborted R10 has no QC report | A matching manifest digest alone does not prove every file in the manifest still matches. |
| All raw file bytes using current independent common verifier | 30/31 PASS | One excluded R15 diagnostic session has a file-level hash mismatch. |
| All raw file bytes using `file02_common.verify_raw` **from each session's locked source commit** | 30/31 PASS | A temporary detached checkout was used for each historical commit and removed after checking. The same excluded R15 diagnostic fails. |

The one raw-integrity failure is
`r15-20261113T074320Z-62398090`, already
`INVALID_FOR_INFERENCE` in the R15 case audit. Its manifest records the
SHA-256 of an empty `runs/r15-r20-n1-CP/result.json`
(`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`),
but the current 75,079-byte file hashes to
`b805d52b2a8a82ab75649597651bf319b9603887d4d58f7d5bf1d202e2548193`.
The file timestamp is about 0.6 s after its manifest timestamp, consistent
with a late write after freeze; the exact cause is **not established**.
This session remains excluded, is not repaired in place and must never be
used for a performance aggregate. The distinct first R15 abort is also
`INVALID_FOR_INFERENCE` but its raw bytes do verify. The first Q02 attempt
is an excluded safety abort; Q02 v1 has technically valid raw/QC yet is
superseded for cross-profile inference by arm-order confounding. Only
Q02 v2 is the current cross-profile result.

## R15 tool-version drift versus raw corruption

The final R15 session is `r15-20261113T074602Z-751a6e1b`, raw manifest
`e94f4e413a57a3689016f507b54a5a722b95e03f6b14fbea690ac35bf0d54800`.
Its plan locks source `a4b38cdc9907944b729e028e3d5d9440883f3e1f` and
module hash `fa145accb83e8ac85519449625995831213c990b97a3330f74d6dbe37869f091`;
both match the historical Git objects. Running today's
`file02_r15.py verify` exits **2** with `R15 plan drift` because it compares
the locked historical `go.mod` with current module bytes. This is a
**verifier compatibility limit**, not final-session raw corruption.

The exact historical analysis revision `c065d2a` was run from a temporary
detached checkout against the unchanged private final R15 session:

```text
python3 research/tools/file02_r15.py verify \
  /home/ntp/research/Beam/research/artifacts/file02/r15-20261113T074602Z-751a6e1b
exit 0: R15_VERIFY=OK qc=PASS n=36 raw_manifest_sha256=e94f4e413a57a3689016f507b54a5a722b95e03f6b14fbea690ac35bf0d54800
```

The earlier locked source revision `a4b38cd` reached `R15 derived artifact
mismatch` (exit 2) on the current derived files because later documented
`2163640`/`c065d2a` analysis revisions added CIs and Comet CPU from the
same frozen raw. It did **not** report plan drift at that historical source.
The accepted R15 negative knee result uses the final QC-passing session and
the documented revised analysis; no result is inferred from either aborted
session. [R15's original audit](../R15_BENIGN_CAPACITY_AUDIT.md) remains
the case-level evidence.

## Reproduction commands and limits

```text
python3 research/tools/file02_x02_audit.py write
python3 research/tools/file02_x02_audit.py check
python3 research/tools/file02_x02_historical.py write
python3 research/tools/file02_x02_historical.py check
```

The first audit compares exact historical Git tree/module objects, frozen
plan/QC hashes and every listed raw file. The second runs the historical
common raw verifier at each plan's source commit in a temporary checkout.
Both explicitly expect and surface the one excluded R15 hash failure;
they raise on any additional failure. This does **not** re-run every
case-specific aggregate derivation under every later analysis revision;
current A/Q verifiers and original R/A case audits remain separate checks.
The 31 historical sessions include one R10 abort without QC, so the 30-row
QC inventory must not be misread as complete coverage of all attempts.
