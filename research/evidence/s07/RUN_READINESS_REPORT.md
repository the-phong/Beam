# S07 run readiness — bounded four-profile smoke, 2026-11-12

## Authority, source and scope

The Owner merged PR #19 into `test-mainet` at
`49facc28ca29102e3e7295546f29b74fe3503a00` (tree
`994527bc2efe4aebff97cedd3c69808b92894d76`) and authorized S07.
The S07 implementation was committed on `research/s07-reproducibility` at
`ea393d1b9408aeb1a969e4eb245530d529391d1c` (tree
`e325945cce8467adb924e78bb17ff443c42c7456`). Beam is only the GitHub
repository name; Pluto remains the application and Go module. The smoke used
Go 1.24.1 on Linux 6.18.33.2 WSL2, x86-64, one host with 20 logical CPUs.
Four real CometBFT v0.38.26 validator processes, four observer processes and
a separate load process were used for **each** profile. RPC/P2P/ABCI stayed
on loopback. This is a single-host QA smoke, not an independent-host or
confirmatory workload study.

The exact [S06 safety config](../../config/s06_pilot.json) has SHA-256
`9bc640db8b8875af3f9a5e4f89faa1882e8f4bc10459997d1a891397768cf21a`;
the unchanged [public scenario corpus](../../corpus/s06_scenarios.json) has
SHA-256 `6f43b725156f60deb0f0d2e6c451605bae479a5e890ff58f7e3be46c5a8ca0e2`.
Each profile ran the fixed-legitimate-plus-adversarial small plan with seed
711, 712, 713 or 714, respectively, one second warm-up, four seconds steady
and three seconds drain. [REPRODUCE](../../../REPRODUCE.md) gives the command
and raw-to-derived trace.

## End-to-end result and integrity

Successful private session ID: `s07-20261112T061020Z-6a222cf3`. Its
`manifest.json` SHA-256 is
`53046b1b87122e320010ac1aecb4332c98e5a0cc114a3e32b28b6ccab971e8b2`.
`python3 research/tools/s07_orchestrate.py verify <session>` exited 0 and
rehashed **92 frozen files**, checked schemas and recomputed all derived rows
from raw plan, result, four observers, commit observations and oracle. The
private `integration.log` SHA-256 is
`115e576da070ac150671800a0b763df6e7f0cc1e81e64b15233b87631a3e321b`;
private `derived.csv` SHA-256 is
`bdb85561baae30a7db6f651a44db98597ec34c3d4a4872d7c3f1517a63c3c0bc`.
The sanitized [public smoke sample](smoke_sample.json) SHA-256 is
`5aa6aa78ed63a3984d6dd9b34b2e51a5ec24636e4224d36a8f5e1300931390f3`.
Private paths, signed wire payloads, keys and validator homes are not in the
public sample.

| Profile | Planned / actual sends | Valid committed on four nodes | Invalid rejected | Four-observer samples | CheckTx queue samples | 95% run-level CI |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| C | 9 / 9 | 5 | 4 | 36 | 32 | not estimable, n=1 |
| CP | 9 / 9 | 5 | 4 | 36 | 28 | not estimable, n=1 |
| P | 9 / 9 | 5 | 4 | 36 | 28 | not estimable, n=1 |
| H | 9 / 9 | 5 | 4 | 36 | 28 | not estimable, n=1 |

The oracle checked the same common-height block ID and AppHash, exactly one
commit for each valid transaction, no invalid commit, and expected nonce and
balances for every sender and receiver on all four nodes. Each legitimate
transaction also appeared in the indexed `/tx` result on all four nodes.
The analyzer reports size, process CPU/RSS, RPC latency, inclusion-observation
latency upper bound, queue, accepted mempool, observed relay and conservative
steady-window goodput lower bound in one row per run. Three of four relay
receive streams emitted a metric in this small smoke; coverage is explicit
and the fourth stream is not treated as zero. Exact inclusion latency remains
`null`. No p99 or 95% effect CI is inferred from five transfers or one
independent run per profile.

## Exclusion ledger and failure handling

Four implementation attempts were retained as separate ignored, frozen
sessions and marked `INVALID_FOR_ANALYSIS`; none contributes to the table:

| Session suffix | Reason | Manifest SHA-256 |
| --- | --- | --- |
| `044407Z-2bfdd51b` | WSL path spelling rejected before network execution | `a55d7c890a1eddff4d50712c0250e7d9523fb0f989c987fad2116745439e9d2b` |
| `055045Z-44c421d1` | Same path check before the inode-based correction | `e97e44f4441f86ca6d24f6214cc105db0b3ae187874ddb71ddedd6a4571c0133` |
| `055255Z-4902f887` | Go nanosecond timestamps exceeded the local Python parser | `93441e3bf733e631cd0b03ec163da4b4f7ad40bc8b5623f5a675ae875bede418` |
| `060045Z-5df96884` | Analyzer initially required a zero-use relay stream to emit a counter | `a0ac4b761747a4f6911528f68ac16a3f2d3247ffbcdc914a89ee341a17ebdb24` |

All five frozen sessions passed the integrity verifier after these fixes.
The four exclusions remain invalid under their original analyzer revision;
their raw files were not edited or reclassified. Reruns received new IDs.
The unit tests also prove hash tampering and reusing an existing result path
are rejected.

## Verification ledger and remaining gate

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go build ./...`, `go test ./...`, `go vet ./...`, `go mod tidy -diff` | 0 each | No module diff or new dependency. |
| `python3 -m unittest discover research/tools -p 'test_s07_*.py' -v` | 0 | Three integrity/schema tests. |
| `python3 research/tools/s07_orchestrate.py smoke --publish-sample research/evidence/s07/smoke_sample.json` | 0 | Successful session and manifest above; `make test-s07-smoke` uses the same orchestrator without publishing. |
| `make verify` | 0 | Private log SHA-256 `a529379030e6af42402685f1302be94b399f83c3b0491415de16781941b6c741`. |
| `go test -race -tags 'integration fourval' ./test/integration -run '^TestS07ReproSmoke$' -timeout 900s -count=1 -v` | 0 | Four profile subtests; private log SHA-256 `6d7afad0b5e87186990774390d89d5ead48306bb50e85844fc9253c783bdff3c`. |
| `make test-s06-pilot` | 0 | S06 helper regression: 7/7 original jobs PASS; private log SHA-256 `0b07d51c0ba8e8f81331717d08bbc1e780bdd3668761fd17c4584ad86dccc7a8`. |
| `git diff --check` | 0 | No whitespace errors. |

The [quick CI workflow](../../../.github/workflows/s07-quick.yml) runs build,
Go units, public vectors/corpora, schema/integrity tests and this four-node
smoke on PRs to `test-mainet`; the existing separate PR workflow scans
secrets. Hosted CI must pass on the Draft PR before S07.4 and the local
technical gate can be marked OK. Independent Controller/Owner review remains
separate from that technical gate.

## Limits before File 02

This smoke does not calibrate benign thresholds, establish the ≤1% FPR
upper bound, measure multi-host clocks, find saturation, quantify DoS or test
an ML detector. RPC sync is not block inclusion. Poll-based all-four indexing
provides a bounded observation time, not an exact commit timestamp. The
limited steady-window goodput is a lower bound at 100 ms polling resolution.
No RQ1–RQ3 estimate, causal comparison or File 02 confirmatory conclusion is
accepted from this output. The Owner must separately decide File 02 scope
after independent review of S07 readiness.
