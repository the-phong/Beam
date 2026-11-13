# COMMON-01 — actual runtime configuration audit, 2026-11-12

Scope: bounded QA on `research/file02-common-foundation`, source commit
`025b4423baa0a89c9c9ad5d697613b8c1deae493`. This audit checks that
the actual four-validator configuration is captured and matched to the pinned
plan. It is not a File 02 effect experiment or a multi-host result.

## Executed checks

| Command | Exit | Result |
| --- | ---: | --- |
| `make test-file02-common` | 0 | Eleven tests, including plan/runtime mismatch and altered actual Comet mempool config rejection. |
| `make test-s07-smoke` | 0 | Fresh C/CP/P/H single-host loopback smoke, four runs, zero excluded; S07 frozen manifest `9ecf614c8d8a5e0a336d905ef9fece8eefc2f5a21b0661663323c8f425a3e2a7`. |
| `python3 research/tools/file02_runtime.py capture-s07 research/artifacts/s07/s07-20261112T130047Z-94bd1f6d` | 0 | S07 frozen manifest reverified, four actual runtime inventories captured and frozen under ignored `research/artifacts/file02/`; inventory manifest `f9e6f914a5f74a015929678d9ffa20149f33df0d2d31fb22d37ea721f0d6e649`. |
| `python3 research/tools/file02_runtime.py verify-s07-capture research/artifacts/file02/common01-s07-20261112T130047Z-94bd1f6d` | 0 | Four profiles and 16 actual node config hashes rechecked, source clean, pinned config and S07 manifest bound to the frozen inventory. |

Each inventory is tied to run ID, source commit/tree, `go.mod`/`go.sum`,
the pinned S06 config SHA-256
`9bc640db8b8875af3f9a5e4f89faa1882e8f4bc10459997d1a891397768cf21a`,
actual `config.toml` SHA-256 for each validator, the shared genesis JSON
hash, and binary hashes. The capture code reads the four actual Comet files
and checks mempool type/size/bytes/cache/max transaction size, recheck,
broadcast, empty-block interval, commit timeout, genesis block-byte limit,
distinct listener ports and three loopback peers per validator. Public or
mismatched endpoints fail the capture. `qc-run` now requires this inventory
inside each future File 02 raw run and compares it to the locked plan before
analysis.

| Profile | Private inventory SHA-256 | Actual nodes checked |
| --- | --- | ---: |
| C | `792d99618c58eb3adb23870f1cf13839ad2d00613432061e44c724d07b6cb058` | 4 |
| CP | `77e194a4a874568c15854c6e2142376a2a76ba6b737272f68c7766df3d517303` | 4 |
| P | `ed1fd3f66f699c150e4ebf04c4496399af1c8abd493568000ddd13f50b769ade` | 4 |
| H | `26f7656cae50671f90daf406f0d683f2e18ef19a4441a79faa7374dfa24b005b` | 4 |

Captured software: Go `go1.24.1` on linux/amd64, CometBFT `v0.38.26`,
Cloudflare CIRCL `v1.6.4`, secp256k1 `v4.4.1`, Linux
`6.18.33.2-microsoft-standard-WSL2`. The CPU is an Intel i7-12700H with 20
logical CPUs and 102 common flags across the visible logical CPUs. Actual
placement is four validators, four observers and one load process on one
host. All RPC/P2P/ABCI listeners are loopback. The actual mempool is `flood`,
size 128, max 8 MiB, cache 1024, max transaction 8192 B, recheck and
broadcast enabled; max block 1 MiB, empty-block interval and commit timeout
1 s. These values match `research/config/s06_pilot.json`.

The audit supports COMMON-01 for the available single-host diagnostic
topology and makes the capture mandatory for future File 02 run QC. It does
not establish independent infrastructure, a capacity knee, clock accuracy,
precision or any RQ result. Private node homes, keys and raw logs remain in
ignored storage and are not included in this report or PR.
