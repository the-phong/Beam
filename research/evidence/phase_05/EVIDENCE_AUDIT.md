# Phase 05 evidence and publication boundary

> **Historical exploratory evidence audit.** The 122-run matrix below is
> retained unchanged as a data source, but its short runs and varying valid
> offer count cannot establish matched-rate service degradation. The
> Controller correction evidence is in
> [CONTENTION_EVIDENCE_AUDIT.md](CONTENTION_EVIDENCE_AUDIT.md).

The accepted Phase 04 merge baseline is
`b861479982b0114be2ad60f1e7727362075a7f73`. The definitive workload
matrix uses staged source tree `fc629192cb046befcdae2058fac675c92d6de1db`
and public corpus SHA-256
`242e5c848aa3c97f731a3e70f9ca0876a9ec68bffbbbab2ead8c222c005cd535`.
The tested-source file hashes are in `TESTED_SOURCE_SHA256.txt`. Later analysis
or report changes get their own final tree and manifest; they are not
retroactively claimed as the executed workload source.
The first implementation commit is
`ea5b2ce802dbb315b7b4c6718d83f698a11a112b`, tree
`97dc21e9dc082ca3399a63b56a10732b5900cbc9`. The 19,489-byte
`TESTED_TO_IMPLEMENTATION_CODE.patch.gz` contains the bounded code/spec
changes from the definitive matrix tree to that commit. Its compressed
SHA-256 is `46f145454b4f8f4db72eec8d89f4e8b8b3aafd1609584ec8a685fba8b91f1dd7`;
the original patch SHA-256 after `gzip -dc` is
`0910622552c27d32c7d053114f7a6ad665b54cea169a7687fdfe35956dc51395`.
`git apply --reverse --check` against the implementation commit succeeded.
Both hosted workflow runs on that implementation commit succeeded: push
`40286051014` and Draft PR `40286103670`. They are public GitHub Actions
logs; they do not include the private full-matrix raw files. A final
evidence-only commit needs a fresh CI check at its own HEAD.

Raw material is retained locally under ignored `raw/` and is **not for Git**.
It includes per-run JSON, exact command/exit ledger, Pluto and Comet logs,
the full benchmark output, failed exploratory runs, and one interrupted
experimental home archive containing private validator keys. The archive
must never be staged or uploaded. Public signed corpus bytes and public
genesis are in Git; no private transaction signing key is serialized.

The definitive matrix root is `raw/20261026T062154Z-full/`: 122 run JSON
records, 122 command logs, four validator logs per run, one append-only
execution ledger, analysis files and verification logs. The matrix driver
and `analyze.py --require-complete` each exited zero; the analyzer found 122
complete records, 122 ledger entries, zero failed entries and one source
tree identity. `RESULT_TABLES.md` and `ADMISSION_LATENCY.svg` are sanitized
outputs derived from those records. The final `RAW_SHA256.txt` contains
relative paths and hashes; it is checked against retained local files before
handoff. Raw logs may expose local paths, PIDs and environment details;
hashes alone do not publish those contents. Controller access to private raw
data requires an Owner-controlled transfer, outside this Draft PR.

The first exploratory matrix root `raw/20261026T053931Z-full` contains five
completed but methodologically excluded B0 runs and a sixth interrupted run.
Its `INTERRUPTION.md` documents the exact terminated process groups and the
private archive checksum. The second exploratory root
`raw/20261026T054450Z-full/` reached run 042, where accepted
nonce-before-balance precedence exposed a workload-sender context bug. Run
045 also failed; all 45 ledger records are retained. Both exploratory roots
are excluded from final aggregates. The initial self-calibrated
microbenchmark was deliberately terminated with exit 143 after an
unintended large count of untimed durable commits; its raw log and correction
record remain in the definitive root. No exploratory failure was silently
relabeled as PASS.

The CPU user/system split correction was tested in one supplementary
source-tied real-network smoke root `raw/20261026T070850Z-smoke/` at staged
tree `118d2bdd4451ad363bcb42027164a146b4fbc615`. The preceding
`raw/20261026T070759Z-smoke/` executed while the relevant working file
was unstaged and therefore recorded an obsolete staged tree identity; it is
retained but excluded from source-tied metrics. No primary matrix result is
retroactively assigned the new fields.
The timeout-safe runner's final smoke root is
`raw/20261026T071922Z-smoke/`, staged tree
`4277e04d851f247dff85f72c81e804fd46248a0e`; it ran six valid and six
invalid B1 requests with matching AppHashes and zero failures. This smoke
overlapped the long B0 regression suite on the shared host and is used only
for correctness/cleanup verification, not performance comparison.
