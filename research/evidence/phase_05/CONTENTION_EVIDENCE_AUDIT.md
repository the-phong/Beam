# Phase 05 matched-rate contention evidence audit

This audit is for the Controller correction on Draft PR #10. It does not
confer acceptance or authorize a DoS mechanism.

## Source and public fixtures

- Reviewed pre-correction HEAD: `3758fb7fb7df973d1003a5a49ff9071351a19dcf`.
- Correction implementation commit: `547190a9f6b945cf327f8b6855704b887ad78bda`, tree `56adda92a29c845cf221d401e02d5b869ba5305e`.
- Original 78-run measurement tree: `29c16ad5f65c08a8922fcdba8448c6e84c0e4ea7`.
- Monotonic-clock repair tree: `dfb1d32a821736ad029a7e77790577003186502f`.
- Public sequenced corpus SHA-256: `459148e8d350cecd1509130d88de909b9ce37c2420ef2999d224db9265ec1646`.
- [`CONTENTION_SOURCE_SHA256.txt`](CONTENTION_SOURCE_SHA256.txt) contains 15 executable source, workflow and fixture hashes; its own SHA-256 is `df31bca6603781c705b49c1277b20cf80e84fcff1c46506fc1ca0537fa8ceab9`.

All 15 source digests were also recomputed directly from the correction
commit's Git blobs, with zero mismatches. The two earlier staged trees remain
the precise identities of executed measurements before that commit.

The original v2 corpus, 122-run exploratory dataset, old source identities
and prior SHA-256 manifests remain unchanged. No accepted B0/B1 transaction,
state, AppHash, genesis, persistence or consensus source changed.

## Raw records and exclusion

`make benchmark-contention-full` exited 0 with 78/78 test commands and raw
result files under ignored local root
`research/evidence/phase_05/raw/20261026T083247Z-contention-full/`.
The original run `034-secondary-b0-p0-r2` passed correctness but suffered a
host wall-clock discontinuity: outer duration 2463.80 s versus Go monotonic
test duration 83.85 s. It is retained with its exact log and result hash.
Analysis without replacement exits nonzero at 77/78 usable. The same
condition was rerun in one fresh network under ignored root
`research/evidence/phase_05/raw/20261026T131246Z-contention-repair34/`,
exit 0, 87.5 s outer duration, with monotonic inclusion and duration fields.
It replaces only that measurement row. `excluded_runs.json` names the reason
and original hashes. The repair wrapper's old console summary printed the
matrix index `runs=34`; the ledger proves one repair run and the wrapper's
summary count was corrected afterward.

[`CONTENTION_RAW_SHA256.txt`](CONTENTION_RAW_SHA256.txt) lists 481 local raw
matrix, repair and trial files (including command/node logs), SHA-256
`ad21e3d4c9fc6cb1d58062922c85df4ba6062fc3148fd6ff35740a11350da6d7`.
`sha256sum -c` returned 0 for every listed file. The previous exploratory
`RAW_SHA256.txt` also returned 0. Raw logs are local and Git-ignored; the
hash list can be audited independently when the corresponding files are made
available through an Owner-approved channel.

The 22 local regression/race/fuzz/benchmark command logs and their JSONL
execution ledger are identified by
[`CONTENTION_VERIFICATION_SHA256.txt`](CONTENTION_VERIFICATION_SHA256.txt)
(23 files, manifest SHA-256
`724f452be7e6373adc8b7cacef3ca10515927ed58f6e53e73a589acb745ae579`).
Every command exited 0 and the manifest check returned 0. The benchmark
matrix command and repair ledgers are in the raw manifest above.

Hosted push run `40332212464` and PR run `40332222233` both succeeded on
correction implementation commit `547190a9f6b945cf327f8b6855704b887ad78bda`.
Each had 14 successful job steps, including a real matched-rate smoke run.
Downloaded hosted logs are retained locally under ignored
`raw/hosted-ci-547190/`; their two hashes are in
[`CONTENTION_HOSTED_CI_SHA256.txt`](CONTENTION_HOSTED_CI_SHA256.txt), manifest
SHA-256 `0e26dc0a913870210a6c9eaa568521c580565cff809a2a2c14067146a6efe2b8`.
No hosted step or test-level SKIP/FAIL was observed. Full 78-run timing data
was collected locally, not in CI.

## Sanitized results

- [`CONTENTION_PER_RUN.csv`](CONTENTION_PER_RUN.csv): 78 data rows plus header, sample counts, achieved rates, valid admission/inclusion p50/p95/p99, goodput, per-validator CPU/RSS, matching-height AppHash and exact staged source tree.
- [`CONTENTION_PER_RUN.json`](CONTENTION_PER_RUN.json): the same per-run information plus rejection class costs, distributions and process summary.
- [`CONTENTION_NORMALIZED.json`](CONTENTION_NORMALIZED.json): paired within-profile ratios and CPU/RSS deltas against each repetition's own zero-invalid baseline.
- [`CONTENTION_AGGREGATE.json`](CONTENTION_AGGREGATE.json): 18 condition groups with independent-run counts and run-bootstrap intervals.
- [`CONTENTION_EXCLUDED_RUNS.json`](CONTENTION_EXCLUDED_RUNS.json): the one clock-disrupted original.
- [`CONTENTION_RESULT_TABLES.md`](CONTENTION_RESULT_TABLES.md): readable results generated from the verified JSON, not hand-entered measurements.

The corrected analyzer verified both ledgers and all referenced command/result
digests, produced 78 usable rows, and found zero failed commands. Across the
68 valid-bearing runs, 20,400 valid transfers were confirmed; 77,200 invalid
submissions were rejected across all 78. Each real-node test asserted
matching-height block and AppHash agreement, ordered transactions, sender
balances/nonces, receiver balance and supply on four validators. The
100%-invalid condition reports no valid inclusion latency.

The published tables contain no validator/node private keys, last-sign
state, runtime homes, raw PIDs, credentials, host paths or unsanitized
process logs. Public signed transactions and public genesis are in the
versioned corpus; generation keys were ephemeral and never serialized.

## Interpretation limits

Five primary and three secondary independent repetitions support descriptive
run-level comparisons. A per-run p99 from 300 valid observations is still
tail-unstable. The 16-tx/s ceiling did not locate network saturation;
combined Pluto/Comet CPU stayed low. The host is one WSL2 machine, not
independent distributed fault domains. B0/B1 whole-network differences
include bytes, registry, state and persistence in addition to authentication.
No DoS resistance, production capacity or multi-host claim follows.
