# S06 isolated testbed and offered-load pilot — 2026-11-11

## Provenance and scope

Owner merged S05 Draft PR #16 into `test-mainet` at `ac91250ddff31d07b6dcb19f81878e149256e84f` and authorized S06. This report covers branch `research/s06-safe-testbed` in the original `pluto` directory. The test used Go 1.24.1 on Linux 6.18.33.2 WSL2, x86-64, Intel i7-12700H (20 logical CPUs), 19 GiB reported memory. Four validator processes shared this host. CometBFT is pinned to v0.38.26. The exact executable config is [`../../config/s06_pilot.json`](../../config/s06_pilot.json), SHA-256 `9bc640db8b8875af3f9a5e4f89faa1882e8f4bc10459997d1a891397768cf21a`. The public scenario corpus is [`../../corpus/s06_scenarios.json`](../../corpus/s06_scenarios.json), SHA-256 `6f43b725156f60deb0f0d2e6c451605bae479a5e890ff58f7e3be46c5a8ca0e2`.

## Checks and observed result

`make test-s06-pilot` exited 0. It built the normal and source-checked S05 Comet binaries, verified the S06 public corpus, then ran `TestS06IsolatedOpenLoopPilot` for 91.10 s. Seed 611 randomized seven jobs. Each job started a fresh isolated four-validator network with separate collectors and a separate `profile-load` process. All expected CheckTx codes matched, all planned requests reached the socket-write callback, every legitimate transfer committed with matching sender/receiver state on all four validators, and observer logs showed separate P2P relay ingress. The filtered, publication-safe [`pilot_summary.log`](pilot_summary.log) has SHA-256 `2a830375bfe629605f0eacf72bd2c19fd5477ad2a0b2d6f26d2d47f905b5b964`. The complete raw command log stays ignored at `research/artifacts/s06/pilot_private_final.log`, SHA-256 `ac9b1cb79ceaa36b5767f4d16f8a3551d92163aface13cbfbdd6d155be1472b7`.

| Order | Profile and condition | Planned / actual socket writes | Legitimate committed |
|---:|---|---:|---:|
| 0 | P, equal tx/s | 9 / 9 | 9 |
| 1 | H burst, equal tx/s | 9 / 9 | 9 |
| 2 | CP, equal bytes/s | 5 / 5 | 5 |
| 3 | C, equal bytes/s | 69 / 69 | 69 |
| 4 | H, equal tx/s | 9 / 9 | 9 |
| 5 | H valid higher rate, 6 steady tx/s | 25 / 25 | 25 |
| 6 | H, fixed legitimate plus invalid | 9 / 9 | 5 |

The C/CP equal-byte pair deliberately sends different transaction counts because wire sizes differ; `ComparablePlans` verifies the same serialized-byte target. The P/H equal-tx pair has the same target. Unit tests construct and validate all three regimes for all four profiles, test CP/H exact wire size, reject duplicate transaction hashes and a falsely declared P2P source, and prove a full queue yields `RATE_MISS`. The public corpus verifier evaluates all scenario bytes through the direct oracle, including malformed early, bad secp256k1, bad ML-DSA-44 after valid secp256k1, insufficient balance and valid/benign cases. The runtime guard test injects an observer failure and verifies `SAFETY_ABORT` cancels load; the kill-switch test verifies preflight refusal. Public/DNS targets and HTTP redirects are rejected. These are safety behavior tests, not a claim that resource caps were physically reached on this host.

One intermediate rerun was stopped after a harness preflight failure: it created the guard inside the app's test process, which the tightened process-separation rule correctly rejected. The harness now waits only for collector files; the standalone load process performs the authoritative guard preflight. The seven-job final run above passed after this correction. The intermediate run is excluded from all achieved-rate claims.

## Boundaries and next gate

This pilot demonstrates the checklist S06 controlled-network send-rate, workload-oracle and safety requirements within the configured small loads. It does not estimate maximum throughput, resource exhaustion, DoS resistance, cross-host timing, independent-failure behavior, warning lead time or any RQ1–RQ3 effect. Direct P2P injection is unimplemented; P2P ingress here is relay measured after RPC submission. Actual send time is a client socket-write callback, not validator reception or block inclusion. The corpus contains scenario examples; it is not a statistical sample. The single-host pilot cannot provide independent-run false-positive bounds, benign calibration quantiles or File 02 confirmatory inference. Those remain explicit prerequisites in the measurement contract and cannot be inferred from seven runs. S07 remains closed until separate Owner authorization.

## Local verification ledger

All commands below ran from the original `pluto` directory on branch `research/s06-safe-testbed` with source HEAD `ac91250ddff31d07b6dcb19f81878e149256e84f` and the staged S06 changes. The full logs remain under ignored `research/artifacts/s06/`; only the filtered pilot summary is published.

| Command | Exit | Private log SHA-256 |
|---|---:|---|
| `make test-s06-pilot` | 0 | `ac9b1cb79ceaa36b5767f4d16f8a3551d92163aface13cbfbdd6d155be1472b7` |
| `make verify` | 0 | `b145cd5159976809d13f2ce3fbbab01b85ae4bf010f1e95e3d9515b3ae39ed65` |
| `go test -race ./internal/profiles/load -count=1` | 0 | `43ba173d4500c032294044b0712c997ebf0b1fe56db9956b49b595308664f4d8` |
| `make test-s04-4v` | 0 | `f18787eadfab196c5998c79afea9ba45bcabea52e7d19aee3eb3325a1c237cca` |
| `make test-s05-telemetry` | 0 | `c35af0bce8ea789ee9e929cdaca4e5bd9d5582a64fb813b8fa003a857cfbbcf1` |

`go mod tidy -diff` and `git diff --cached --check` produced no changes or errors. The staged file list excludes the unrelated D026 directories and ignored private artifacts. Hosted CI, independent review and Owner acceptance remain separate from this local technical report.
