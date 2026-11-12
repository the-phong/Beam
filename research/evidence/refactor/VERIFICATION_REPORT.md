# Domain-oriented refactor verification (2026-11-11)

## Revision, scope and environment

Source: Owner-merged `test-mainet` commit `ed6bc8f474c13ed709d5ab5f57b3856edcbf602a`, tree `8c84b0b2ad85fb064d1286ff78f4d37871e49bcb`. Work branch: `refactor/domain-oriented-architecture`. Result revision is the Draft PR head containing this report; it awaits Controller and Owner review. `master` was inspected at `5af503cf22cf47a0cde9cc6cc764b338cda2cae6` for layout only. Go `go1.24.1 linux/amd64`; WSL2 Linux 6.18.33.2, Intel i7-12700H (20 logical CPUs), 19 GiB RAM. Pinned CometBFT v0.38.26, Pebble v1.1.5 and cryptographic dependencies remain as in `go.mod`. Tests used isolated loopback homes. The four validator tests share one host.

The pre-edit working tree contained unrelated untracked `research/evidence/phase_03d/d026/` and `d026_review/`, ignored `research/artifacts/` and other raw evidence, and `stash@{0}`. They were not staged, moved or removed. `build/` and new `research/artifacts/refactor/` hold disposable/private local outputs and are ignored. [Old/new mapping](../../../docs/REFACTOR_MAPPING.md) documents every active package and CLI.

## Verification commands

All commands ran from repository root. Each listed command exited **0** after the noted path fixes; raw logs remain local in ignored `research/artifacts/refactor/`. Their SHA-256 digests are in [LOG_SHA256.txt](LOG_SHA256.txt). The `make verify` baseline ran before edits and exited 0; the final `make verify` ran after `gofmt` and exited 0.

| Command | Result and purpose |
| --- | --- |
| `make verify` on source commit | 0; build, unit, vet, format, module verify, version baseline |
| `go test ./internal/legacy/... ./internal/classical/... ./internal/hybrid/...` | 0; domain group after move |
| `go test ./internal/experiment/...` | 0; profiles, corpus, load, telemetry and benchmark group after move |
| `go test -tags 'integration fourval' ./test/integration -run '^$'` | 0; all integration imports/selectors compile |
| `make build corpus-s04-verify corpus-s06-verify benchmark-corpus benchmark-contention-corpus` | 0; six built binaries and four public corpus verifiers |
| `go build ./...`, `go test ./...`, `go vet ./...`, `go mod tidy -diff` | 0 each; tidy diff empty |
| `make verify` | 0; final local gate |
| `make test-integration` | 0; real single-validator, replay and recovery suite (281.7 s) |
| `make test-integration-4v` | 0; B0 four-validator suite (275.7 s) |
| `make test-integration-b1-4v` | 0; B1 four-validator suite (98.9 s) |
| `make test-s04-4v` | 0; C/CP/P/H correctness (64.7 s) |
| `make test-s05-telemetry` | 0; C/CP/P/H telemetry (66.6 s) |
| `make test-s06-pilot` | 0; isolated bounded offered-load pilot (88.3 s) |
| `make benchmark-smoke`, `make benchmark-contention-smoke`, `make benchmark-micro` | 0 each; bounded historical tools |
| `python3 -m unittest discover research/benchmarks/b0b1 -p 'test_*.py' -v` | 0; 9 analyzer tests |
| `python3 research/benchmarks/b0b1/analyze.py` on existing 2026-11-10 smoke artifact | 0; 1 complete historical run read |
| `python3 research/benchmarks/b0b1/analyze_contention.py` on existing 2026-11-10 contention smoke artifact | 0; 1 complete historical run read |
| `go test -race ./internal/...` | 0; all 25 internal packages compiled, relevant package tests passed under race detector |
| `python3 research/tools/verify_refactor.py` | 0; 108 Go files have only expected path/import/format changes; 445 tracked corpus/evidence/testdata files byte-identical |
| `git diff --check` and local Markdown link audit | 0; no whitespace faults or broken links in active tracked docs |
| `git diff --cached --binary | gitleaks stdin --no-banner --redact=100` | 0; pinned Gitleaks v8.30.1 found no secret in staged changes |
| `make verify`, normalized source/data audit and pinned Gitleaks commit-range scan after implementation commit `3c71074f6fe6ff327e56cac0f4227ec326c790f9` | 0 each; version output now names the committed revision, 108/445 equivalence holds, and no leak found |

The first domain-unit run failed because a B1 test referred to `../b1tx/testdata`; the fixture was moved intact and the relative path was corrected to `../tx/testdata`. The first experiment-unit runs exposed old `internal/profiles/*` imports and a corpus path in moved profile tests; only those paths were corrected. The first `make verify` after movement stopped at `fmt-check` for reordered import blocks; `gofmt` resolved it. No assertion was removed or weakened.

## Behavior preservation

| Contract | Evidence |
| --- | --- |
| Transaction bytes, signed intent, hashes, verification order and reached stages | Normalized comparison of all 108 Go files to source commit found 0 unexpected changes. B0/B1 golden tests, C/CP/P/H public-vector and stage tests, four corpus verifiers pass. Historical B0/B1 corpus SHA-256 remains `242e5c848aa3c97f731a3e70f9ca0876a9ec68bffbbbab2ead8c222c005cd535`; contention corpus remains `459148e8d350cecd1509130d88de909b9ce37c2420ef2999d224db9265ec1646`. |
| Genesis, nonce, balance, state transition, AppHash and ABCI result codes | Domain and profile unit tests, direct ABCI oracle, full integration and S04 four-validator suites pass. S04/S05 H profile logged the same AppHash `2C4252A230A13A397BE9A094097BB90513E5F4ACB46BF76E1123A4D4E` in their separate local runs. |
| Cache, dedup, recheck, replay and persistence | S04 direct oracle, full recovery integration, B0/B1 four-validator suites pass. |
| Telemetry schema and ON/OFF behavior | `TestRecorderBoundedAndSafeSchema`, `TestS05CheckTxTelemetryONOFFCorpus`, S05 four-validator telemetry and race tests pass; schema source is byte-equivalent apart from moved imports. |
| Offered-load regimes, capability labels and safety | S06 plan/guard unit tests include `RATE_MISS`, abort, kill switch, endpoint refusal and corpus oracle; seven-job S06 pilot passes under the accepted single-host limits. |
| Historic evidence and analyzers | 445 tracked files byte-identical; existing local historical smoke and contention records parse with current analyzers. Ignored raw/private files remain local. |

No new RQ1–RQ3 confirmatory or full DoS run was performed. An accepted single-host pilot remains a pilot, and unavailable telemetry capture points remain `NOT MEASURED`. Historical reports and source manifests retain their original path spelling; the mapping provides current rerun paths. The distinct B0/B1 and C/CP/P/H implementations were not merged. The refactor introduces no dependency or protocol version changes.

## Handoff gate

The staged file list contains only refactor source, documentation and refactor evidence; untracked D026 files and ignored artifacts are excluded. The pinned Gitleaks v8.30.1 staged-diff scan found no leak. The Draft PR must target `test-mainet`; hosted CI and Controller/Owner acceptance are independent gates. The coding agent does not merge or open S07.
