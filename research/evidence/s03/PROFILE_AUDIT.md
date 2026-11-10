# S03 profile implementation audit — 2026-11-09

Directory note (2026-11-10): the original package and command paths in this
report are preserved as the executed S03 ledger. Current code is under
`internal/profiles/`; current vector tooling is `cmd/profile-vectors/`.

## Scope and provenance

- Base: `test-mainet` merge commit `2a3dcbceb52b996425d911dde87b691317f56efd` (PR #12); S02 Gate OK by explicit Owner confirmation in the work session. Branch: `research/s03-controlled-profiles` in the original `pluto` folder.
- New implementation: `internal/researchv2` (canonical codec, genesis/registry, common transfer state, stage result), `internal/researchv2app` (one direct ABCI++ callback path), `internal/researchv2test` (ephemeral signers), `cmd/rv2-vectors` and the public checked-in vector file.
- Accepted B0/B1 packages, the normal `plutod` node and `go.mod` were not changed. The research-v2 application is direct-ABCI and in-memory in S03. Durable storage, real CometBFT node wiring, four-validator correctness and loaded P2P measurements belong to S04 and later gates.

## Checklist evidence

| Item | Artifact and observed assertion |
|---|---|
| S03.1 | [`PROFILE_EQUIVALENCE.md`](../../protocol/PROFILE_EQUIVALENCE.md) and `TestOneABCIPathForFourProfiles`: one genesis, registry, B0 transfer primitive, AppHash and ABCI++ callback implementation; same post-transfer AppHash for C/CP/P/H. No B0/B1 delta is treated as an ML-only cost. |
| S03.2 | Codec `Profile.WireSize`, canonical `Parse`/`Encode`, `TestFourProfilesSameTransferAndExactSize`, `TestCPAndHExactBytesAcrossChainIDBounds`, public vectors: CP is accepted and has H's exact wire length for the same chain ID. Wrong zero-padding is rejected structurally. |
| S03.3 | Signing digest includes profile, chain, sender, receiver, amount and nonce; excludes proof and CP padding. Genesis binds both public keys to one fixed profile-independent account ID. `EvaluateTrusted` orders structural → profile → chain → amount → binding → secp → ML → state; rejected input leaves state unchanged. High-S secp is rejected. |
| S03.4 | `TestProfileNegativeBoundariesAndReachedStages` covers valid, malformed, wrong chain, zero amount, unknown account/binding, unregistered signer, bad secp/ML proof, wrong nonce, insufficient balance, CP padding and profile mismatch. Duplicate tests use the same signed bytes after a successful transition. Direct ABCI tests verify CheckTx/proposal/finalize/commit behavior; public golden vectors are reverified byte-for-byte. |

## Commands and exact results

- `go test -count=1 ./internal/researchv2 ./internal/researchv2app`: exit **0**, both packages `ok`.
- `go test -race -count=1 ./internal/researchv2 ./internal/researchv2app`: exit **0**; the app test makes concurrent CheckTx/Info calls across a Commit. Full output is [`race.log`](race.log), SHA-256 `8e789b76cbb019fed97c94213fbf0ee35278a7edf60259dc22433ea22d3df5e7`.
- `make verify`: exit **0** on Go 1.24.1/linux-amd64. It built Pluto and CometBFT, ran `go test ./...`, `go vet ./...`, `go mod verify` (`all modules verified`) and `plutod version`. Full output is [`make_verify.log`](make_verify.log), SHA-256 `d622b2ff8d3299da6c7fdaeee95c8ecf6a7c7d67ccbeb5984f28388c31b28037`. The local binary reports revision `831880b` because that was the local WIP commit at build time; the final pushed source SHA and hosted CI must be checked separately.
- Public vector file SHA-256: `1a0a8e0baf1a963d35b8bc357c9abe70221d72bb7fe011cee9eaaba88581dd06`. Its four observed sizes at chain ID `rv2-golden-v1` are C 145, CP 2,565, P 2,501, H 2,565 bytes, with identical post-transfer AppHash `30cd6c098a19ba3b57d965d0585c0d01d84669f470f8e9c73d9bce4a18ac516c`.
- `git diff --check`: exit **0**. Local Markdown link check on the S03 report and both checklist copies found **0** broken local paths. The manifest parsed as YAML.

All generated signing keys were ephemeral and discarded. Only synthetic public genesis and signed transaction bytes appear in the vector file. Historical private raw evidence was copied byte-identically into the ignored `research-v2/artifacts/` under the original project folder before old worktrees were removed; it is not staged for publication. The preexisting refactor is preserved in local stash commit `7ec52bd`.

## Limits and next gate

S03's technical gate is about source equivalence and direct callback correctness. These tests do **not** establish actual four-validator agreement, durable recovery, accepted mempool occupancy, loaded relay traffic, stage timing, resource impact, detection quality or DoS. S04 must provide those correctness and real-network checks before any performance interpretation. No S04 item is checked here.
