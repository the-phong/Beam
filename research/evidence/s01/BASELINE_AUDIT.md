# S01 baseline audit — 2026-11-09

Directory note (2026-11-10): the M01 inventory files now sit beside this
report in `research/evidence/s01/`. Commands and raw-log paths below retain
their original execution spelling; current files are under `research/`.

This audit checks the existing source before any new RQ1–RQ3 experiment. Historical Phase 05 measurements are not reused as results for the new protocol. See the detailed `BASELINE.md`, `LEGACY_INVENTORY.md`, `SECURITY_AUDIT.md`, and `PIPELINE_MAP.md`.

## Immutable source and backup

| Item | Exact value |
|---|---|
| Source branch | `origin/test-mainet` |
| Source commit / tree | `0bdd869f2c6b3ed59b0ae825954d8667b3c7481e` / `09c24280590aee51d97b53b18c5d9e193dbe6dbe` |
| Working branch | `research/reproducible-testbed`, in a separate worktree; the original `refactor/unified-architecture` dirty worktree is untouched |
| Go | `go.mod`: 1.24.0; local: `go1.24.1 linux/amd64` |
| Pinned modules | CometBFT v0.38.26; CIRCL v1.6.4 (ML-DSA-44); secp256k1/v4 v4.4.1; Pebble v1.1.5 |
| Base `go.mod` SHA-256 | `6a70b6804672b3c1d4865bb9da32872c430a1c13f396ca7f9c9894cdc07bb8f8` (`git show 0bdd869:go.mod`) |
| Base `research/config/manifest.yaml` SHA-256 | `46a75bc5669944e877f1819b4567d1bddc974da76cdb3f9fb0b5a3320ba83e4a` (`git show 0bdd869:research/config/manifest.yaml`) |
| Source hash manifest | `baseline_manifest.json`, 148 selected source/spec/public-summary digests, SHA-256 `2aafada7f216984e9c8cb75bdbc41f9734bbfc58aa1679d7188f08e180a644de` |
| Full-history backup | `research-v2/artifacts/private-baseline/test-mainet-0bdd869.bundle`, 4,084,006 bytes; SHA-256 `0310c9738e001c11e010bb69c7d85fcb5a2a1f3933ab684a4680d746c2f6fd42` |

`git bundle create research-v2/artifacts/private-baseline/test-mainet-0bdd869.bundle origin/test-mainet` exited 0. `git bundle verify` exited 0, reported a complete history and the expected remote ref at the source commit. The bundle is local and ignored: historical ancestry contains raw evidence, so it must not be published with the review branch. The public hash manifest selects only source, specifications, and sanitized historical summaries; it does not pretend to be a privacy approval for all baseline history.

The manifest was regenerated from the same commit with `freeze_baseline.py` into the ignored private artifact directory. It again emitted 148 entries (exit 0); `cmp -s` against the public manifest exited 0, and both SHA-256 values were `2aafada7f216984e9c8cb75bdbc41f9734bbfc58aa1679d7188f08e180a644de`. A second `git bundle verify` exited 0.

## Configuration and environment

`internal/node/b0_network.go:126-139` and `internal/node/b1_network.go:123-136` start from Comet `config.DefaultConfig`, use localhost ABCI/RPC/P2P addresses, allow duplicate IPs and non-strict address books for four local validators, set persistent peer lists, and produce empty blocks at a one-second interval. Comet `config/config.go:795` defaults to the flood mempool with `Recheck=true`, `Broadcast=true`, 5,000 accepted transactions, 1 GiB accepted bytes, 10,000 cache entries and a 1 MiB per-transaction cap. The local verification host is WSL2 Linux `6.18.33.2-microsoft-standard-WSL2`, 20 exposed CPUs, Intel Core i7-12700H, `v9fs`; four localhost validators on it are a single-host functional test, not independent infrastructure or a network-performance result.

## Re-execution and classification

`make verify` exited 0 on this working branch at the fixed source commit. It built Pluto and Comet, ran all default Go tests, `go vet`, formatting check and `go mod verify`, then reported `plutod 0.2.0-foundation`, Go 1.24.1 and revision `0bdd869`. The private log is `research-v2/artifacts/private-baseline/make_verify.log`, SHA-256 `0fac6e44d6fd4841c05377c8a37bf4bf060677a16afbc0ddcd6b771f60688cc0`. The working tree includes a new integration test and documentation, so this run establishes reproducibility of the fixed source plus those additions; the earlier untouched-baseline `make verify` is separately recorded in `BASELINE.md`.

After the S01 test-harness rework, `make verify` again exited 0; private log SHA-256 `a16fe85d47ef4a914a1a8b680e487fc107aeb7aa5c59f5f04ec7656b509b18bb`. `TestResearchV2M01P2PRelayBoundary` was also rerun with the `integration fourval` build tags and exited 0 in 4.141 seconds; private log SHA-256 `cbbbd1070fe69a986ed2cbad5b1b6ccba56817999ca1129058a0e16ff1a1c825`.

The prior S01 code audit reran `TestB1HybridFourValidator` in three fresh four-validator localhost networks (exit 0) and `TestResearchV2M01P2PRelayBoundary` (exit 0), as described in `CODE_PATH_AUDIT.md`. The first checks B1 valid commit, bad-ML rejection and state agreement; the second checks accepted relay into a peer mempool before block inclusion. They are functional smoke evidence, not resource or attack results.

The first `make test-integration` run exited 2. Its private log SHA-256 is `6103d85d5e2ef0965b314525116e6c2f163768ae785fa38c18fbc329f85b1dc6`: M6 returned `signal: terminated` from `restart.Wait()`, and the suite then hit its 240-second deadline in a B0 restart/hash case. An immediate one-off M6 rerun passed (exit 0; SHA-256 `50346f9fb1f513901e0b0e36b37809e364b971c2380ee0f60eb782a638175a1a`). A full direct run with `go test -tags integration ./test/integration -timeout 900s -count=1 -v` passed all 13 tests in 221.355 seconds (exit 0; SHA-256 `90c3d35ca4f10175c94aa59f143c4a4490efa8698d944f01acdc166e7e96a323`).

M6 remained genuinely flaky: its original test passed 9/10 repetitions and failed once with the same SIGTERM exit (exit 1; SHA-256 `14be244629bae5b9816c9f7ff6db94597aecb1039314986a9242447b0dec3864`). Source review of pinned CometBFT `cmd/cometbft/commands/run_node.go:118-125` and `libs/os/os.go:20-34` shows `TrapSignal` is installed only after `n.Start()` and the `Started node` log; the RPC listener can answer before that point. The test previously assumed every explicit SIGTERM caused a clean exit. `test/integration/b0_m6_test.go` now waits for the expected RPC chain and block height; after explicitly sending SIGTERM, it accepts either clean exit or an exit signaled by **SIGTERM only**. It still requires matching post-restart Comet block/consensus/application heights and unchanged sender/receiver balances and nonce. No B0/B1 runtime or persistence behavior was changed. The corrected M6 test passed 10/10 repetitions (exit 0, 30.836 seconds; SHA-256 `274d5249e4d6ebf499f309f4982c17c1e7448c85939317998f855ae736f096aa`).

One intermediate diagnostic revision incorrectly compared RPC `latest_app_hash` (the block header hash returned by Comet `rpc/core/status.go:43`) to the newly committed application AppHash. That revision was rejected after five failed repetitions and a 180-second test timeout (exit 1; SHA-256 `9e7fdf8a6efd22e0b5732f8346ee83526ab4949356fffdb816b7b5845c08a286`); the incorrect assertion is absent from the final source. The final test compares only chain and block height at RPC readiness and checks replayed application state afterward.

The original `make test-integration` target had a fixed 240-second timeout. A post-fix run passed 11 tests before that limit cut off the final two (exit 2; SHA-256 `a4f2df71d1dd5b02be042ebd617427d0bc63816a127ef491d0424368cabd852a`), showing the deadline was too tight for this WSL2/v9fs host. `Makefile` now defaults `INTEGRATION_TIMEOUT` to 900 seconds and allows an override; it does not change any test assertions. The final `make test-integration` exited 0: all 13 tests passed in 246.189 seconds; private log SHA-256 `5850ab59b9725e46ad0ece56e492d372304f0dfe10466438d04d308aff412e97`.

`make test-integration-4v` exited 0 on the current worktree. All seven B0 four-validator tests passed, including three fresh quorum/catch-up networks, crash/restart and initialization cases; private log SHA-256 `7ac7fba88e6fd64464945364b778f05f9094a3469cb23da08aa10c3bd0bf5b15`. The integration package took 295.401 seconds. This is single-host functional evidence.

`make test-integration-b1-4v` exited 0 on the current worktree. All four B1 hybrid test groups passed, including three fresh four-validator networks, crash boundary, initialization refusals and CLI setup; private log SHA-256 `cdc9a500b209116dde8436b9bcfa97fc6cb309761b935d4cb35331c9039917ad`. The package took 172.694 seconds. Again, this is single-host functional evidence only.

Historical reports and Phase 05 result tables are **report-only** for the new RQ1–RQ3 contract; prior exploratory and excluded runs stay classified as such. No four-profile C/CP/P/H equivalence, byte-matched CP, multi-host relay/resource result or new detector result exists at this baseline. `LEGACY_INVENTORY.md` records the classification and B0/B1 differences in wire size, signed intent, genesis keys and registry, state policy, persistence, verification order, AppHash and mempool cache. These differences make a direct B0/B1 cost delta unsuitable as an isolated ML-DSA cost.

`SECURITY_AUDIT.md` records pinned Gitleaks v8.30.1 scans and a filename audit. Baseline history and current worktree scans had no findings; an all-ref scan found one manually triaged prose false positive in an unrelated branch. The scan is bounded evidence and does not certify unreviewed raw historical logs. New private logs and the bundle remain under the ignored `research-v2/artifacts/` path.

After S01 rework, the same pinned Gitleaks v8.30.1 binary scanned the current worktree with `gitleaks dir --redact=100 --no-banner --report-format json --report-path research-v2/artifacts/private-baseline/gitleaks_final.json .` (exit 0, about 15.93 MB scanned, no findings). The redacted output and empty JSON report stay private; log SHA-256 `e32178a63ba66f33df698e7e8d4147d36063dd5bea364ba3dd38fab881cae78c`. `git check-ignore -v` confirmed both the full-history bundle and final raw integration log match `/research-v2/artifacts/`.

## S01 gate assessment

| Item | Evidence | Result |
|---|---|---|
| S01.1 freeze and trace | Fixed source commit/tree, pinned dependencies/config, full-history bundle verified twice, 148-entry manifest regenerated byte-identically | PASS |
| S01.2 re-execution and classification | Final `make verify` 0; final `make test-integration` 0 (13/13); B0 four-validator 0 (7/7); B1 four-validator 0 (4/4); P2P relay smoke 0; historical results classified separately in `LEGACY_INVENTORY.md` | PASS |
| S01.3 code path | `PIPELINE_MAP.md` and `CODE_PATH_AUDIT.md` trace RPC/P2P, precheck, CheckTx, verification, accepted mempool, relay, proposal/commit, reject/cache/recheck with pinned source and bounded runtime tests | PASS |
| S01.4 B0/B1 and privacy | `LEGACY_INVENTORY.md` enumerates wire, keys, genesis, registry, state, persistence, verify order and AppHash; `SECURITY_AUDIT.md` records the bounded secret/history audit; new raw logs and bundle are Git-ignored | PASS |

**Gate S01: OK for the bounded source and localhost functional audit.** S02 research-contract work may begin separately. This gate does not establish a four-profile experiment, attack effect, multi-host result or detector quality.

## Limitations

- The current S01 tests use isolated localhost homes, not four independent VM hosts.
- The public manifest is a selected content inventory; the complete Git bundle is local only.
- Loaded P2P relay and mempool capacity are outside S01 functional verification and remain unmeasured.
