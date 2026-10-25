# Phase 03D recovery closure execution log

Date: 2026-10-25, Asia/Bangkok. Branch: `phase/03d-b0-abci-integration`; source `HEAD`: `b56c81ccce7ea4a0e7f52053c7459c4601c15b38`; reviewed implementation tree: `87b504853cb0ff98790976879ea8832db92b83e0`. Go: `go1.24.1 linux/amd64`. Host: WSL2 Linux `6.18.33.2-microsoft-standard-WSL2`, Intel i7-12700H, 20 logical CPUs. Source mount: `9p` at `/mnt/d`. No remote CI ran. Validator homes were created by `t.TempDir()`; no private validator keys were copied into this evidence. Public test transaction scalar: 1.

Exact commands and observed exit codes, run from the repository root unless noted:

| Command | Exit | Raw output |
|---|---:|---|
| `git write-tree` before changes | 0 | `494341fca6a952587c16decdf4ebf05d98bf0b36` |
| `git cat-file -t fa28e6f7b02b70020899c8294b74af02c7ab51d0` and `git cat-file -t 494341fca6a952587c16decdf4ebf05d98bf0b36` | 0 each | Both `tree`. |
| `git diff --binary fa28e6f7b02b70020899c8294b74af02c7ab51d0 494341fca6a952587c16decdf4ebf05d98bf0b36` | 0 | `tested_to_final.diff` (129,294 bytes). |
| `sha256sum -c research/evidence/phase_03d/remaining/source_manifest.sha256` before changes | 0 | All 41 entries `OK`. |
| `go test ./internal/b0store -run '^TestM4ReturnedWALWriteAndSyncErrors$' -count=1 -v` | 0 | `m4_store_faults.log`. |
| `go test ./internal/b0store -run '^TestM11GenesisCreationFaultBoundaries$' -count=1 -v` | 0 | `m11_store_faults.log`. |
| `PLUTO_RECOVERY_ARTIFACTS=<absolute closure/raw path> go test -tags integration ./test/integration -run 'TestB0M4RealCrossStoreWALFault\|TestB0M6CometPostCommitAckFailpoint\|TestB0M7AsymmetricApplicationCrash\|TestB0PinnedHeightAndHashMismatchFailClosed\|TestB0M11RealInitChainResponseLoss' -count=1 -v -timeout 300s` | 0 | `real_node_faults.log`, plus raw child and Comet files in `raw/`. This initial targeted run preceded the added WAL SIGKILL and pruned-history subcases. |
| `PLUTO_RECOVERY_ARTIFACTS=<absolute closure/raw path> go test -tags integration ./test/integration -run '^TestB0M4RealCrossStoreWALFault/(wal-write-kill\|wal-sync-kill)$' -count=1 -v -timeout 120s` | 0 | `m4_process_kills.log` and raw `m4-*-kill-*.log`. |
| `PLUTO_RECOVERY_ARTIFACTS=<absolute closure/raw path> go test -tags integration ./test/integration -run '^TestB0PinnedHeightAndHashMismatchFailClosed/(app-behind-pruned\|replay-hash-conflict)$' -count=1 -v -timeout 120s` | 0 | `mismatch_extra.log` and corresponding raw mismatch logs. |
| `make verify` | 0 final | `make_verify.log`; includes build, `go test ./...`, `go vet ./...`, formatting, module verification and `plutod version`. |
| `make test-integration` | 0 final | `make_test_integration_final.log`; 122.389 seconds, timeout 180 seconds. |
| `go test -race -count=1 ./internal/b0app ./internal/b0state ./internal/b0store ./internal/b0genesis` | 0 final | `race_final.log`. |
| `go test -cover -count=1 ./internal/b0app ./internal/b0state ./internal/b0store` | 0 | `coverage.log`: 84.8%, 88.0%, 82.9% statements. |
| `sha256sum -c research/evidence/phase_03d/closure/source_manifest.sha256` | 0 | All listed current source files `OK`. |
| `git diff --check` and `git diff --cached --check` | 0 each | No whitespace errors. |

The first `make verify` run exited 2 because a before-mkdir test called `OpenExisting` on an absent path. The test revealed Pebble created the path despite returning an error; `OpenExistingWithOptions` now checks the directory before Pebble opens it. The exact failed output is in `exploratory_make_verify.log`. A later M11 test build exited 1 due a missing test import; its exact compiler output is in `exploratory_m11_build_failure.log`. Both failures were corrected and rerun. Earlier exploration found that Pebble v1.1.5's `errorfs` does not inject `SyncData`/`SyncTo`; the closure harness wraps those file methods explicitly. Exploratory direct WAL tests also showed Pebble's fatal child exit for returned write/sync errors and that an injected sync failure can reopen a complete new record. These outcomes were retained in the final fault cases rather than reclassified as power loss.

`FAIL_TEST_INDEX=7` is CometBFT v0.38.26's own `fail.Fail()` call immediately after `proxyApp.Commit()` returns and before `stateStore.Save(state)` in `state/execution.go`. The M6 test changed only the disposable home's `create_empty_blocks` configuration to make the first block contain the signed transfer. The log's `committed state` line, `*** fail-test 7 ***`, B=1/C=0/A=1 and identical H1 hash after restart identify the reached point. It is an intentional process exit, not a physical power failure.

All benchmark workload samples, profiles, seeds and methods from the earlier Phase 03D run remain under `../remaining/`; no new resource-cost estimate is inferred from recovery timing. The 9p mount, WSL2 caching, Pebble WAL replay and host scheduler confound latency and durability extrapolation.
