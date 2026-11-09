# M01-01 — immutable source baseline

This is an inventory of the public `test-mainet` source at the start of the
new experimental program. Historical measurements remain historical; this
inventory does not certify them for the new research questions.

## Source and environment

| Field | Observed value |
|---|---|
| Source ref | `origin/test-mainet` after `git fetch origin test-mainet` (exit 0) |
| Commit | `0bdd869f2c6b3ed59b0ae825954d8667b3c7481e` |
| Git tree | `09c24280590aee51d97b53b18c5d9e193dbe6dbe` |
| Go module directive | `go 1.24.0` |
| Local toolchain | `go version go1.24.1 linux/amd64` |
| Direct crypto dependency | `github.com/cloudflare/circl v1.6.4` (ML-DSA-44) |
| Direct classical dependency | `github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1` |
| Consensus | `github.com/cometbft/cometbft v0.38.26` |
| Storage | `github.com/cockroachdb/pebble v1.1.5` |

The complete module graph is pinned by this commit's `go.mod` and `go.sum`.
The source tree and source manifest identify exact content; this report does
not copy any historical runtime home, validator key, or raw evidence.

## Reproducible inventory

`python3 research-v2/tools/freeze_baseline.py --ref
0bdd869f2c6b3ed59b0ae825954d8667b3c7481e --out
research-v2/evidence/m01/baseline_manifest.json` exited 0 and emitted 148
SHA-256 digests: 74 source/build/CI files, 14 specifications and 60 selected
already-published summary evidence files. The selection deliberately excludes
raw logs and review bundles. The manifest SHA-256 is
`2aafada7f216984e9c8cb75bdbc41f9734bbfc58aa1679d7188f08e180a644de`.

`go test -list . ./... > research-v2/evidence/m01/test_list.txt` exited 0.
It lists 80 `Test`, `Benchmark`, or `Fuzz` entries (99 output lines); its
SHA-256 is
`bc3798e88a12fcf5b2f301e90ce209fcbdf40c6f00c256ecfaa1d38fe1028923`.
The remote branch layout and exact ref tips were captured by
`git for-each-ref refs/remotes/origin --format='%(refname:short) %(objectname)'`
(exit 0) in `remote_refs.txt`, SHA-256
`e43bfbec0160f2edf1554fc87468c2efcb8297eb430a3ba0a356c000307479f3`.

The baseline CI definition is `.github/workflows/verify.yml` at the fixed
commit. It runs `make verify`, local integration, B0 and B1 four-validator
integration, Phase 05 corpus/oracle, analyzer tests, and two four-validator
smokes. The exact workflow bytes are in the source manifest.

## Local build check

`make verify` on the untouched baseline source exited 0 with Go 1.24.1:
Go build, `go test ./...`, `go vet ./...`, format check, `go mod verify`
(`all modules verified`), and `plutod version` (`Revision: 0bdd869`) passed.
This run did not execute integration, load, or multi-host experiments. Those
gates remain open.

## Audit boundary

The Git tree freezes public baseline bytes. Raw evidence already present in
its history is referenced by commit only and will not be recopied into the
new experimental artifacts. The working tree of `refactor/unified-architecture`
had extensive pre-existing uncommitted changes, so work proceeds in the
separate `research/reproducible-testbed` worktree.
