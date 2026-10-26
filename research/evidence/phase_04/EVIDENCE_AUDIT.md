# Phase 04 B1 evidence audit

This publication-safe index describes the local raw evidence. The raw files remain under the ignored `research/evidence/phase_04/raw/` directory. `RAW_SHA256.txt` identifies 329 retained raw files by relative path and SHA-256; `sha256sum -c research/evidence/phase_04/RAW_SHA256.txt` returned 0 from the repository root. The manifest itself has SHA-256 `7f89c1b8bb34f8501ea2845210c2c9eedea2aa92d87ee5aac7cd62ece916c53d`. `SOURCE_SHA256.txt` records the changed source and public test material of commit `e0b33751e5cca0337ee8e623979f09ac52ccc872` (tree `fe298b4eaafc207790b22aa85b842c6a568d62fc`). `SOURCE_REVIEW.patch.b64` is a text bundle of the exact tested diff. Decode with `base64 -d research/evidence/phase_04/SOURCE_REVIEW.patch.b64 > /tmp/pluto-phase04-source.patch`; the restored patch has SHA-256 `ac983e705cb74d0e978fe9190b1e23cf321c1b75eb4e1934a9187662fcc95df5`. No validator home, signing key, last-sign state or unsanitized process log is included in this index or the Phase 04 commit.

## Final successful execution

| Command | Raw output | Observed exit | Required execution |
|---|---|---:|---|
| `make verify` | `raw/make_verify_closure.log` | 0 | Build, unit tests, vet, format, module integrity |
| `make test-integration` | `raw/make_test_integration_final.log` | 0 | 40 named Phase 02/B0 runs, zero SKIP |
| `make test-integration-4v` | `raw/make_test_integration_4v_final.log` | 0 | 16 named Phase 03E runs, zero SKIP |
| `PLUTO_4V_ARTIFACTS=…/raw/b1_4v_final make test-integration-b1-4v` | `raw/make_test_integration_b1_4v_final_source.log` | 0 | 15 named B1 runs, zero SKIP, three fresh networks |
| `go build ./...` | `raw/go_build_closure.log` | 0 | All packages |
| `go test -count=1 ./...` | `raw/go_test_closure.log` | 0 | All default-tag packages |
| `go vet ./...` | `raw/go_vet_closure.log` | 0 | All packages |
| `go mod verify` | `raw/go_mod_verify_closure.log` | 0 | Pinned module archive hashes |
| Core `go test -race -count=1 …` | `raw/race_core_closure.log` | 0 | B0/B1 application, state, store, transaction and CLI packages |
| `go test -race -tags 'integration fourval' ./test/integration -run '^TestB1Hybrid' -count=1 -v -timeout=900s` | `raw/race_b1_4v_closure.log` | 0 | 15 named B1 runs, zero SKIP; real network and crash tests |
| D-026 `go test -race -tags integration ./internal/node -run '^TestD026' -count=1 -v -timeout=300s` | `raw/race_d026.log` | 0 | 28 named D-026 runs, zero SKIP |
| B0 four-validator/M7 race | `raw/race_b0_4v_m7.log` | 0 | 27 named runs, zero SKIP |
| CIRCL ML-DSA-44 ACVP package tests | `raw/circl_acvp_closure.log` | 0 | keyGen, sigGen and sigVer vectors |
| B1 decoder/evaluator fuzzing, 15 seconds each | `raw/fuzz_decode_closure.log`, `raw/fuzz_evaluate_closure.log` | 0 | Parser and state mutation |
| GitHub Actions push run [`40267930032`](https://github.com/the-phong/DoS-in-Blockchain-PQC/actions/runs/40267930032) | `raw/hosted_source_push.log`, `raw/hosted_source_push_status.json`, `raw/hosted_source_push_job.json` | SUCCESS | Commit `e0b3375…`, all four workflow verification steps, 71 named runs, zero SKIP |
| GitHub Actions PR run [`40267954738`](https://github.com/the-phong/DoS-in-Blockchain-PQC/actions/runs/40267954738) | `raw/hosted_source_pr.log`, `raw/hosted_source_pr_status.json`, `raw/hosted_source_pr_job.json` | SUCCESS | Same source commit, all four workflow verification steps, 71 named runs, zero SKIP |

The final B1 log includes exact signed transaction bytes, callback markers and three matching-height block/AppHash pairs. The public report records the pair for each fresh network; the local raw log contains complete transaction and consensus output. The crash test records application PID, Comet PID, Comet liveness at the marker and after the outage, B/C/A, matching-height AppHash, balances/nonce, and a production restart with one transaction effect. `raw/b1_4v_final/` retains child process logs and marker files. The pre-write and post-sync cases use application-only SIGKILL; other store tests use injected software errors. These observations do not establish physical power-loss durability. Both hosted logs reached the two application-only crash markers; neither contains an unintended SKIP, FAIL or data race.

## Exploratory failures retained

- `raw/failed_acvp_public_api_exploratory.log`: an ACVP internal-message vector was initially passed directly to the framed public CIRCL API; this was an API representation mismatch, not a failed package ACVP verifier.
- `raw/failed_initialization_exploratory.log`: a different valid genesis member's validator key initially passed membership validation; per-home public identity binding now rejects it.
- `raw/b1_malformed_key_exploratory.log`: all-`ff` fixed-width ML key bytes were expected to fail CIRCL parsing, but are syntactically accepted and correctly fail the authoritative registry binding.
- Other intermediate runs remain in the local raw directory and are hashed in `RAW_SHA256.txt`; only logs explicitly identified above are used for final PASS claims.

## Reproducibility boundary

The environment is WSL2 on a single i7-12700H host with shared `v9fs` storage and Go 1.24.1. `raw/environment.txt`, `raw/dependencies.json`, the exact source commit, the public vector and the test commands allow independent software reruns. The three validator-network trials do not represent independent fault domains, Byzantine faults, a latency benchmark, DoS resistance or a certified FIPS module.
