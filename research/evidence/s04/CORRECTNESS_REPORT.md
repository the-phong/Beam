# S04 correctness report — 2026-11-09

## Authority and scope

The Owner merged S03 PR #14 into `test-mainet` at
`1c8084207adb5da787e34e230dd0dc6d9a3de05f` (tree
`9903377fba8b4babbdce170bd9f53c621b7a0667`) and authorized S04.
This implementation is on `research/s04-bft-correctness` in the original
`pluto` folder. Result source commit
`b8ccd7cf8c500fe2495143a21c76606d41c558d2`, tree
`0c3c331e2929e576a2d5bf133be3624ca107087c`, is in
[Draft PR #15](https://github.com/the-phong/DoS-in-Blockchain-PQC/pull/15)
against `test-mainet`. It changes no accepted B0/B1 protocol or dependency
version. The S04 candidate is for independent Controller review; Owner
acceptance and merge are separate. No S05 or loaded performance experiment
was run.

## Artifacts and oracle

The [public corpus](../../corpus/s04_public_corpus.json) contains one common
Genesis V4 and 34 immutable cases: eight per profile plus CP's nonzero pad
and H's late bad ML proof. Each case records ID, profile, exact signed or
mutated wire bytes, SHA-256, expected code and last stage, actor capability
under S02, prerequisite state, direct-ABCI source path, correctness-only
scenario label and oracle result. All keys in genesis are public. The
generator uses ephemeral cryptographic randomness; reproduction uses the
committed bytes and file SHA-256, not a pseudo-random seed. Corpus SHA-256:
`d0f4bfaea4c541ee6f71547e44c30aef588704051459e57e04d6478c83`.
`go run ./cmd/s04-corpus verify research-v2/corpus/s04_public_corpus.json`
rechecks every byte hash, expected code/stage and rejected-state neutrality.
The command above records the original S04 run. After the directory
consolidation, use `go run ./cmd/profile-corpus verify research/corpus/s04_public_corpus.json`;
the corpus bytes and SHA-256 are unchanged.

`TestS04CorpusABCIOracle` executes every case through `CheckTx` New and
Recheck, `ProcessProposal`, `FinalizeBlock` and `Commit`. For rejected
cases, balances, nonces and supply remain unchanged; the committed AppHash
changes only because `FinalizeBlock` advances block height. It also compares
the optional stage recorder with the normal path for every case. Structural,
profile, chain, amount and binding errors never visit secp or ML verification.
H's late bad ML proof visits secp before ML. `TestS04ProposalDuplicateAndRecheck`
checks proposal dedup, direct duplicate execution result and post-commit
nonce rejection. `TestS04ProfileGenesisBindingAndUncertainCommit` checks
profile/genesis mismatch refusal and fail-closed reads after an injected
post-sync response loss, followed by validated reopen. The store uses a
separate `PLUTOS4!` record, profile byte and canonical genesis digest.

## Real four-validator testbed

`make test-s04-4v` starts four distinct CometBFT v0.38.26 validator/P2P
identities, four separate Pebble v1.1.5 stores and four ABCI++ socket
servers on localhost for each of C, CP, P and H. Each profile uses the
same checked-in public corpus genesis and its valid signed bytes. The four
Comet processes use the same canonical genesis, four equal-power validators,
persistent localhost peers and empty blocks. The test asserts the transfer
appears exactly once at the transaction height on every validator, block
hashes agree within a profile, the next block header's AppHash matches an
independent sequential state oracle at that height, and every node reports
sender balance/nonce `93/1` and receiver `7/0`. Four distinct validator
identities are observed in commits over the next eight heights; each
individual block may validly commit with three of four votes. A second
broadcast of the same bytes is rejected by Comet's transaction cache, before
ABCI `CheckTx`, and no duplicate is included at the following height.

The publication-safe [four-validator summary](four_validator_summary.log)
has the per-profile transaction height, block hash, AppHash, voter count,
account state and cache result. All profiles committed at height 3 in the
recorded run. Their transaction block hashes differ because their wire bytes
differ; their post-transfer AppHash is identically
`2C4252A230A13A397BE9A094097BB90513E5F4ACB46BF76E1123A4D4E528E4DC`.
Raw Comet logs and generated validator/node keys remain under disposable or
Git-ignored local paths and are not published.

## Verification ledger

| Command | Exit/result |
|---|---|
| `go run ./cmd/s04-corpus verify research-v2/corpus/s04_public_corpus.json` | 0; 34 cases checked by the oracle |
| `go test ./internal/researchv2app ./internal/researchv2store -run S04 -count=1` | 0; direct corpus, persistence and duplicate tests |
| `make test-s04-4v` | 0; four profile subtests PASS, 70.67 seconds; raw log SHA-256 `172181e7d16fa18b006513a3dfcfeda81326768d5b8d6aaf9a366b16c081a002`; published summary SHA-256 `45c663fa22ced6e142a0a89c514053db228bdaf9ef27def59c0f44b9b1a6c3bd` |
| `make verify` | 0 on the final source tree; build, `go test ./...`, `go vet ./...`, `go mod verify` and version command; raw log SHA-256 `a66cf7c805f087282acc5cc20952832347e5273f016ef323ba1096f42c016ac3` |
| `make verify` after implementation commit `b8ccd7c` | 0; binary reports the result revision; raw log SHA-256 `0d64b569c197abdb5a80a0a274976649e6ab47e62e1f480b131a9ee131790a94` |
| `go test -race -count=1 ./internal/researchv2 ./internal/researchv2app ./internal/researchv2store ./internal/researchv2corpus ./internal/researchv2genesis` | 0; raw log SHA-256 `c1171729a67f7ea445afda644f527437909be1648fa758c97c743ae66d80035d` |

Directory consolidation rerun (2026-11-10): the public corpus was moved
byte-identically to `research/corpus/s04_public_corpus.json` (SHA-256
`d0f4bfaea4c541ee6f71547b6ea47e44c30aef588704051459e57e04d6478c83`).
`go run ./cmd/s04-corpus verify research/corpus/s04_public_corpus.json`,
`make verify`, and `make test-s04-4v` each exited **0**. All four profile
subtests passed in 65.93 seconds; the common post-transfer AppHash was
`2C4252A230A13A397BE9A094097BB90513E5F4ACB46BF76E1123A4D4E528E4DC`.
This rerun checks the path migration only and does not expand the S04 gate.

Code-folder consolidation rerun (2026-11-10): `go run ./cmd/profile-corpus
verify research/corpus/s04_public_corpus.json`, `go test -race -count=1
./internal/profiles/...`, `make verify` and `make test-s04-4v` each exited
**0** with the versionless source folders. All four profile subtests passed
in 67.03 seconds with the same post-transfer AppHash. The B0/B1 public
corpora retained SHA-256 `242e5c848aa3c97f731a3e70f9ca0876a9ec68bffbbbab2ead8c222c005cd535`
and `459148e8d350cecd1509130d88de909b9ce37c2420ef2999d224db9265ec1646`;
both corpus verifiers, Python analysis tests, and the two B0/B1 smoke runs
exited **0**. This verifies folder relocation, not new scientific results.
The ignored local smoke ledgers are
`research/artifacts/b0b1-runs/20261110T034758Z-smoke/execution.jsonl`
(SHA-256 `284bac58b300ecbeb46bfde63fcb6dc60d4f7f359bea1e59583cbffc278f8d07`)
and `research/artifacts/b0b1-runs/20261110T034832Z-contention-smoke/execution.jsonl`
(SHA-256 `25b34e9d91097dc9900aeb8013c29a43f892b95272d8daa0f3ee3a8d4f0abb02`).

The PR verification workflow also invokes `make test-s04-4v` so hosted CI
checks this new network path, in addition to the existing B0/B1 gates. Its
hosted outcome is tracked on Draft PR #15 and is not inferred from local tests.

The first four-validator trial incorrectly required four votes in one commit;
CometBFT validly finalized with three. The next trial passed that check but
expected a `CheckTx` nonce code on replay, while Comet rejected the bytes
earlier in its cache with HTTP 500. The final test checks the actual cache
error and aggregates voter identities across eight commits. These diagnostic
failures are not reported as successful runs. The result above is the final
fixed-corpus run, not an inferred benchmark.

The run used Go 1.24.1 on WSL2 Ubuntu linux/amd64, 20 reported vCPUs on
Intel Core i7-12700H. Comet, Pebble, CIRCL and secp256k1 versions remain
pinned in `go.mod`; `make verify` confirms all modules. The pre-commit
`plutod version` line reports base revision `1c80842`; the post-commit
verification reports `b8ccd7c`. Hosted CI is checked separately. Runtime homes and four
generated validator/node identities are disposable. This is one
physical host and failure domain, not four VMs or multi-host BFT. The
integration harness runs ABCI servers in the test process and Comet in four
subprocesses; `plutod` does not expose a research-v2 production CLI profile.
Comet-driven mempool recheck was not forced by a retained pending transaction;
the oracle invokes `CheckTxType_Recheck` explicitly before and after commit.
No latency, CPU, memory, relay-load, DoS, Byzantine or power-loss conclusion
is supported. S05 remains closed pending separate Owner authorization.

## Technical gate

S04.1–S04.4 have bounded local correctness evidence: all 34 cases reach the
expected stage/code without rejected account mutation, the four validator
profiles agree at the same heights, and recorder/callback/cache/recheck
decisions are checked as specified above. **Local technical Gate S04: OK**
for this scope. This report is an engineering claim for Controller review,
not Controller/Owner acceptance or permission to start S05.
