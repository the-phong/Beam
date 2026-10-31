# Phase 02 execution evidence — 2026-10-15

These are observed command outputs and excerpts from this local WSL2 run.
Private validator/node keys, temporary node databases and full noisy consensus
logs are intentionally excluded. The integration test source is
`test/integration/node_test.go`; it uses a dedicated temporary directory and
removes it after the test. Paths in test output are temporary and do not persist.

## Environment and dependencies

```text
$ go version
go version go1.24.1 linux/amd64
$ ./build/cometbft version
0.38.26
$ go list -m github.com/cometbft/cometbft github.com/cockroachdb/pebble
github.com/cometbft/cometbft v0.38.26
github.com/cockroachdb/pebble v1.1.5
$ go mod verify
all modules verified
```

CometBFT v0.38.26 module sum:
`h1:MIPYgOSvyyiN1jo3SZljKy4ZXqs896aQnj3pQFfUmaE=`.
Pebble v1.1.5 module sum:
`h1:5AAWCBWbat0uE0blr8qzufZP5tBjkRyy/jWe1QWLnvw=`.

## Verification commands

All commands below exited **0** after the final implementation changes.

```text
$ go build ./...
(no output)
$ go test ./...
ok  example.com/pluto/cmd/plutod (cached)
ok  example.com/pluto/internal/app (cached)
?   example.com/pluto/internal/node [no test files]
ok  example.com/pluto/internal/store (cached)
?   example.com/pluto/internal/version [no test files]
$ go vet ./...
(no output)
$ make fmt-check
(no output)
$ go test -race ./internal/app ./internal/store
ok  example.com/pluto/internal/app 1.231s
ok  example.com/pluto/internal/store 1.111s
$ make verify
mkdir -p build
go build -ldflags "-X example.com/pluto/internal/version.Revision=unavailable" -o build/plutod ./cmd/plutod
go build -o build/cometbft github.com/cometbft/cometbft/cmd/cometbft
go test ./...
ok  example.com/pluto/cmd/plutod (cached)
ok  example.com/pluto/internal/app (cached)
?   example.com/pluto/internal/node [no test files]
ok  example.com/pluto/internal/store (cached)
?   example.com/pluto/internal/version [no test files]
go vet ./...
go mod verify
all modules verified
./build/plutod version
plutod 0.2.0-foundation
Go: go1.24.1
Platform: linux/amd64
Revision: unavailable
$ make test-integration
mkdir -p build
go build -ldflags "-X example.com/pluto/internal/version.Revision=unavailable" -o build/plutod ./cmd/plutod
go build -o build/cometbft github.com/cometbft/cometbft/cmd/cometbft
go test -tags integration ./test/integration -timeout 120s -count=1
ok  example.com/pluto/test/integration 7.031s
```

The verbose unit run, `go test -count=1 -v ./...`, exited 0: `TestRun`
(five subtests), `TestLifecycleAndDeterministicReplay`,
`TestFinalizeWithoutCommitAndFailure`, `TestDurableReloadAndExclusiveOpen`,
`TestCanonicalCommitmentAndEncoding`, and `TestInvalidState` all passed.
No test was skipped. `internal/node` and `internal/version` have no direct unit
tests; node behavior is exercised by integration.

Additional `go test -cover ./internal/app ./internal/store` exited 0 and
reported application statement coverage **77.5%** and store statement
coverage **88.2%**. Coverage is diagnostic, not a correctness proof.

## Automated real-node RPC/restart observations

The final verbose command was
`go test -tags integration ./test/integration -timeout 120s -count=1 -v`, exit 0.
Its actual relevant output was:

```text
=== RUN   TestSingleValidatorRestart
    node_test.go:201: first RPC: height=2 header_app_hash=2FEEDFE06A0D35171D8FEC8B7EF7CBDB8899E37A2B7FEB6A1FCDB213E3A4C8A4 application_height=2 application_app_hash_base64=7rQjwkAiR497QXqTRGSoM/yUq/Lh/YWYEE9qNIdvgjI=
    node_test.go:202: block RPC: height=2 app_hash=2FEEDFE06A0D35171D8FEC8B7EF7CBDB8899E37A2B7FEB6A1FCDB213E3A4C8A4 tx_count=0 validator_count=1
    node_test.go:223: first node log: I[2026-10-15|14:25:33.374] Completed ABCI Handshake - CometBFT and App are synced module=consensus appHeight=0 appHash=F16021C6994972E7C3F5C401C81C94937DB9CD84D3D8861A87406516C67FFC3A
    node_test.go:223: first node log: I[2026-10-15|14:25:33.407] Started node                                 module=main nodeInfo="{ProtocolVersion:{P2P:8 Block:11 App:1} DefaultNodeID:fdfd1cbec852942c9e2528ca5912138e1d98bef2 ListenAddr:tcp://127.0.0.1:35605 Network:pluto-integration Version:0.38.26 Channels:40202122233038606100 Moniker:PhongJunior Other:{TxIndex:on RPCAddress:tcp://127.0.0.1:43901}}"
    node_test.go:223: first node log: I[2026-10-15|14:25:34.427] finalized block                              module=state height=1 num_txs_res=0 num_val_updates=0 block_app_hash=2FEEDFE06A0D35171D8FEC8B7EF7CBDB8899E37A2B7FEB6A1FCDB213E3A4C8A4
    node_test.go:230: restart RPC: height=4 header_app_hash=D45F6AD501507ABF86CB14CE4154DB5CB4E7336E733979FEBCD51DAADEF7BBCE recovered_height=2 recovered_app_hash=eeb423c24022478f7b417a934464a833fc94abf2e1fd8598104f6a34876f8232
    node_test.go:232: restart node log: I[2026-10-15|14:25:36.461] Completed ABCI Handshake - CometBFT and App are synced module=consensus appHeight=2 appHash=EEB423C24022478F7B417A934464A833FC94ABF2E1FD8598104F6A34876F8232
    node_test.go:232: restart node log: I[2026-10-15|14:25:36.500] Started node                                 module=main nodeInfo="{ProtocolVersion:{P2P:8 Block:11 App:1} DefaultNodeID:fdfd1cbec852942c9e2528ca5912138e1d98bef2 ListenAddr:tcp://127.0.0.1:35605 Network:pluto-integration Version:0.38.26 Channels:40202122233038606100 Moniker:PhongJunior Other:{TxIndex:on RPCAddress:tcp://127.0.0.1:43901}}"
    node_test.go:232: restart node log: I[2026-10-15|14:25:37.517] finalized block                              module=state height=3 num_txs_res=0 num_val_updates=0 block_app_hash=D45F6AD501507ABF86CB14CE4154DB5CB4E7336E733979FEBCD51DAADEF7BBCE
    node_test.go:237: durable after restart: {ChainID:pluto-integration Version:1 Height:4 AppHash:a8c1f506f3d3e8bc310d38b596040c2b1b954295e80684fff73b9536a3ca6f2a}
    node_test.go:253: missing application database with existing CometBFT state was rejected
--- PASS: TestSingleValidatorRestart (6.99s)
PASS
ok  example.com/pluto/test/integration 6.996s
```

At first shutdown, the independently reopened Pebble state had chain ID
`pluto-integration`, format version 1, committed height 2 and AppHash
`eeb423c24022478f7b417a934464a833fc94abf2e1fd8598104f6a34876f8232`.
After restart and two further blocks, independently reopened state was height
4 and AppHash `a8c1f506f3d3e8bc310d38b596040c2b1b954295e80684fff73b9536a3ca6f2a`.
The header at height 2 records state height 1, hence it differs from the
application's height-2 AppHash.

## Manual local node observation

An additional `plutod init/start` run used chain ID `pluto-phase02-test` in
`/tmp/pluto-phase02-og1KAt/node`. The first process stopped gracefully with
independent `plutod state` output:

```json
{"chain_id":"pluto-phase02-test","state_version":1,"height":54,"app_hash":"ba3182ad9ec57ca2e31806edc03ba4aef981421cb23e610d3c1bd2be67f7cfc9"}
```

The restarted CometBFT process logged these actual handshake lines:

```text
ABCI Handshake App Info ... height=54 hash=BA3182AD9EC57CA2E31806EDC03BA4AEF981421CB23E610D3C1BD2BE67F7CFC9
ABCI Replay Blocks appHeight=54 storeHeight=54 stateHeight=54
Completed ABCI Handshake ... appHeight=54 appHash=BA3182AD9EC57CA2E31806EDC03BA4AEF981421CB23E610D3C1BD2BE67F7CFC9
Started node ... Network:pluto-phase02-test Version:0.38.26 ... ListenAddr:tcp://127.0.0.1:26656 RPCAddress:tcp://127.0.0.1:26657
```

Post-restart RPC `/status` observed height **127** with latest header AppHash
`D72DC790DD659D173C742E8FBF37B0CE88E928A669B9E6FE2DE850FEF4232499`.
After graceful stop the independent state read returned:

```json
{"chain_id":"pluto-phase02-test","state_version":1,"height":130,"app_hash":"fba049d5ae0d86521e437041fe008b77f4c753f0649ca8898fca3777cf18d285"}
```

This manual run is additional observation, not the automated gate. It does
not establish arbitrary power-loss crash safety or multi-validator behavior.
After confirming both manual-run processes stopped, the temporary node home
was moved to the system trash to avoid leaving generated test signing keys in
`/tmp`; the reported outputs remain in this evidence file.

## Local Phase 01 baseline preservation

`BASELINE_SHA256.txt` was generated and `sha256sum -c` returned `OK` for all
16 files **before** Phase 02 source modifications. The Phase 01 report and
command evidence still match their original baseline hashes:

```text
e3a2bb4b78da69cc932108607f72ad0654f8aed4665e14178710eae0989df077  research/PHASE_REPORTS/PHASE_01_REPORT.md
095dccef67101e74ba88f7f7c99177d6dec3c6bd07df4cc90e85c3  research/evidence/phase_01_commands.txt
```

The rest of the baseline intentionally differs after Phase 02 edits; rerunning
the old manifest on the changed tree is not expected to pass.
`sha256sum -c research/evidence/phase_02/RESULT_SHA256.txt` exited 0 after
completion with `OK` for all 23 listed source/config/developer-document files.
