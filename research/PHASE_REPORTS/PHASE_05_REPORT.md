# PLUTO Phase 05 — adversarial workload and resource-cost evaluation

**Status:** implementation and measurement candidate. This report is an
engineering handoff, not Independent Controller acceptance or Owner closure.

## Authority, source and accepted baseline

The Owner authorized one complete Phase 05 experiment on 2026-10-26 after
technically accepting and merging Phase 04 PR #9 into `test-mainet` at
`b861479982b0114be2ad60f1e7727362075a7f73`. The implementation branch
is `phase/05-b1-experimental-evaluation`. The accepted classical B0,
genesis-bound strict-AND B1, Phase 02 legacy mode, CometBFT v0.38.26,
Pebble v1.1.5, secp256k1/v4 v4.4.1 and CIRCL v1.6.4 are preserved. No
production transaction, AppHash, persistence or consensus source was changed.
D-030 records the research-only public corpus and run-level analysis design.
EVM remains NO-GO; no DoS mechanism or admission optimization is present.

The frozen definitive workload source tree before the principal matrix was
`fc629192cb046befcdae2058fac675c92d6de1db`. The final commit and tree
are recorded in the Git handoff after verification. Raw run records carry
their exact staged source tree, base HEAD, corpus digest and run ID.
All 25 entries in `TESTED_SOURCE_SHA256.txt` were independently recomputed
from that Git tree with zero mismatches. Later documentation and the bounded
microbenchmark command correction change the final worktree, so checking the
historical manifest against current files would conflate source snapshots.
A direct `sha256sum -c` against the subsequently edited worktree reported
three expected mismatches (Makefile and two Phase 05 specs); this was resolved
by checking the manifest's intended Git tree, not by changing its hashes.

## Threat model and correctness oracle

Capability A can submit arbitrary bytes, B can replay or modify public signed
bytes, and C controls its own genesis-registered signing keys. A cannot
construct a fully signed B1 state-invalid transaction for another registered
account. B1 has no post-genesis registration or key rotation. All traffic is
sent only to harness-owned localhost networks. The details and falsifiable
hypotheses were recorded before the final matrix in
[`PHASE_05_EXPERIMENT.md`](../specs/PHASE_05_EXPERIMENT.md).

The immutable public corpus is
[`public_corpus.json`](../corpus/phase_05/public_corpus.json), version
`pluto-phase05-corpus-v2`, SHA-256
`242e5c848aa3c97f731a3e70f9ca0876a9ec68bffbbbab2ead8c222c005cd535`.
It has 64 signed valid transfers per profile, 25 B0 negative/control cases,
34 B1 cases, five ordered invalid-proposal cases per profile, and two B1
genesis corruption cases. Each case records a stable ID, exact bytes or a
bounded reconstruction recipe, SHA-256, expected code, stage, state effect,
capability and context. Generation used ephemeral keys; no private signing
material is serialized. The oracle compares explicit expected codes with
accepted state evaluators and ABCI CheckTx, verifies unchanged rejected state,
and sends invalid candidate proposals directly to ProcessProposal.
Historical Phase 04 four-validator tests independently exercise direct
invalid proposals on four persisted application instances.

The CIRCL v1.6.4 ML-DSA-44 public-key unpacker accepts all fixed-width
1312-byte packings tested here. Zeroed key bytes consequently fail the
genesis registry binding (code 11) rather than an assumed parser code 12.
The first generator attempt expected code 12; its failure was retained and
the explicit corpus oracle corrected. The public corpus also contains two
distinct valid ML-DSA signatures on the same intent; after the first commit,
the other fails on nonce and cannot cause a second state effect.

## Measurement methods

`make benchmark-micro` runs five fixed 200-iteration Go benchmark repetitions with ns/op, B/op
and allocs/op for secp256k1, ML-DSA-44, parsing, key/address derivation,
state rules, full validation, invalid classes and application callbacks.
`FinalizeBlock` and `Commit` are timed separately on sequential empty blocks;
their costs are not transaction throughput. The research-only recorder wraps
S7–S10 callback boundaries, uses Go monotonic clocks and drops only samples
when full. Its ON/OFF test checks identical transaction codes, proposal
status, committed state and AppHash. S1–S6 component benchmarks are not an
additive decomposition of one CheckTx call.

`make benchmark-full` schedules 122 independent, sequential fresh networks:
80 primary (two profiles × four ratios × two loads × five repetitions), 12
secondary (25/75% × two profiles × three repetitions), and 30 class-focused
runs (four B0 and six B1 classes × three repetitions). Each run has four
distinct validator identities and stores, 65 genesis accounts, 12 paced
requests, no more than 8 tx/s in the prescribed matrix, and hard caps of 16
tx/s, 63 requests and 180 seconds per Go test. It starts after all validators
reach height 2 and records the actual window. Requested 90% invalid means
11/12 requests (91.7%) under integer rounding; raw counts take precedence.

Admission wall time spans the client `broadcast_tx_sync` call. A second
validator is polled every 100 ms throughout submissions for valid inclusion;
the first successful `/tx` observation is an upper bound on commit time.
Rejected transactions have no inclusion latency. Four nodes' blocks, ordered
transaction bytes and AppHashes are compared at matching height. AppHash for
block H is read from header H+1. All 65 accounts' balances/nonces and total
supply are checked after inclusion. Per-PID `/proc` CPU ticks and sampled RSS
measure each `plutod` PID, which embeds CometBFT in this deployment; the
load generator is separate. The PID samples cannot apportion CPU or RSS
between application and consensus work.
The Python runner/Go client generator and host-wide load were not sampled
as separate CPU/RSS series. Their effects remain confounders; no measured
process cost is attributed exclusively to ML-DSA.
All validators share one WSL2 host; data are diagnostic, not independent-host
fault-domain measurements. The Windows-mounted source/evidence path and
temporary-home filesystem are recorded in each raw result.
The observed machine was an Intel Core i7-12700H (20 logical CPUs), with
20,806,533,120 bytes of reported RAM, Go 1.24.1, Linux/amd64 kernel
6.18.33.2-microsoft-standard-WSL2, CometBFT v0.38.26 and CIRCL v1.6.4.
The source path reported `v9fs`; disposable homes reported `ext2/ext3`.
These filesystem labels and concurrent WSL2 host activity are measurement
confounders. No remote endpoint or independent host was used.

Equal offered transaction-rate comparisons are prescribed. Equal offered
byte-rate comparisons are bounded by the 16 tx/s safety cap: B1's near-4 KiB
wire exceeds B0's roughly 155-byte wire by about 25-fold, so matching even
2 B1 tx/s would require approximately 50 B0 tx/s. This condition is
UNAVAILABLE under the approved safe rate ceiling and is not implied by the
equal tx/s comparisons. The B0/B1 network delta also includes different
genesis registries, state formats, transaction sizes and persistence cost;
it is not attributed solely to ML-DSA.

## Exploratory failure and correction

The first five B0 valid-only exploratory network runs at staged tree
`8dfa430275cd05efe80295ea08c2cf3f110daaa6` completed, but the first
observer polled `/tx` only after all requests and measured CPU/throughput
through post-workload account checks. They are excluded from final analysis.
The sixth run was interrupted while repairing the measurement boundary;
the harness-created orphaned process groups were terminated by their exact
PIDs. Its log and private home archive remain locally under ignored
`research/evidence/phase_05/raw/20261026T053931Z-full/`, with an
`INTERRUPTION.md` audit. No other process or home was targeted. On corrected
B1 smoke, the six observed inclusion upper bounds were 1002, 1401, 1700,
1101, 601 and 801 ms; the earlier observer had reported 5503, 4504, 3503,
1504, 1809 and 810 ms for the same traffic pattern. The corrected smoke
passed under source tree `fec752cf1f861ee6d578fbad781d29fe7fed6397`.

The subsequent exploratory matrix reached a mixed B0 workload whose A03
transaction followed a committed V000 transfer. B0 correctly returned nonce
code 6 under its accepted precedence; the runner had expected balance code
7. Runs `042-b0-p25-moderate-mixed-r2` and
`045-b0-p75-moderate-mixed-r2` failed and are preserved in
`research/evidence/phase_05/raw/20261026T054450Z-full/`. The runner was
stopped after its in-flight Go test cleaned up; no orphan node remained.
The corrected runner reserves V000 for late-invalid probes and uses V001
onward for all valid workload transfers. No production source was changed.
The exploratory root retains 45 ledger entries and is excluded from the
definitive statistical analysis.

The final runner also received a bounded timeout-cleanup correction. A
`subprocess.run` timeout could terminate the Go test launcher without
reaping its independently grouped validator descendants. The runner now
snapshots only descendants of the launched PID through `/proc`, records
their PID birth markers, signals only matching owned PIDs, and records any
timeout cleanup in the ledger. An automated test launched a child sleeper
and an unrelated sentinel; the timeout stopped the child and left the
sentinel alive. The definitive 122 runs had no timeout, so this safety fix
changes no included measurement. The corrected runner is exercised by a
subsequent real four-validator smoke at
`raw/20261026T071922Z-smoke/`, staged tree
`4277e04d851f247dff85f72c81e804fd46248a0e`, which passed six valid
and six invalid requests with one matching AppHash across four validators.
It overlapped a separate local regression test and is used only as a
correctness/cleanup check, not a timing comparison. Hosted CI is a separate
pending check.

## Results and statistical interpretation

The definitive matrix at
`research/evidence/phase_05/raw/20261026T062154Z-full/` completed **122/122**
fresh-network runs (80 primary, 12 secondary, 30 class-focused), with
**1,464/1,464** requests classified, **452/452** expected valid requests
confirmed, **1,012/1,012** expected invalid requests rejected, and no
aggregate failure. Every raw record is `complete=true`, references corpus
SHA-256 `242e5c...cd535` and staged source tree
`fc629192cb046befcdae2058fac675c92d6de1db`, and includes the execution
command, environment, exact responses, matching-height block/AppHash vectors,
account state and PID samples. The analyzer's `--require-complete` command
returned zero with 122 ledger entries, 122 complete records, 30 groups and
40 matched B0/B1 pairs. Full raw-derived aggregate and 34 five-sample
microbenchmark rows are in
[`RESULT_TABLES.md`](../evidence/phase_05/RESULT_TABLES.md). A standalone
raw-derived [admission-latency figure](../evidence/phase_05/ADMISSION_LATENCY.svg)
shows the high-offered-rate condition without smoothing or extrapolation.

At **2 offered tx/s**, valid-only mean of per-run median RPC admission was
787.5 µs for B0 and 1,085.3 µs for B1 (five runs each); mean confirmed valid
throughput was 1.911 and 1.942 tx/s. At **8 offered tx/s**, the corresponding
medians were 826.5 and 1,047.6 µs, and confirmed throughput 5.417 and
5.209 tx/s. The higher rate produced fewer than 8 confirmed tx/s because a
12-request burst and subsequent commit/inclusion time are included in the
short measurement window; these are not saturation capacity estimates. For
the matched-seed valid-only pairs, the mean of five B1-minus-B0 per-run
median admission differences was 297.8 µs at 2 tx/s and 221.1 µs at
8 tx/s (mean paired ratios 1.378 and 1.271). Pairing matches the public
workload order and offered rate, not validator keys, genesis bytes or host
background activity. For
the 50% and 90% invalid 8-tx/s conditions, B0 confirmed 2.707 and 0.559
valid tx/s; B1 confirmed 2.686 and 0.559. At 100% invalid, valid confirmed
throughput and valid inclusion latency are respectively zero and unavailable.
The reduction is primarily the prescribed change in valid share; these short
bounded runs did not establish additional legitimate-service degradation.
No final-matrix client request reported an RPC error or timeout. At the
8-tx/s valid-only target, the observed mean incoming rates were 5.417 B0
and 5.209 B1 tx/s, carrying 0.820 and 19.876 KiB/s respectively. Accepted
admission and confirmed-valid rates happened to agree numerically in those
runs because every admitted valid request was later confirmed; they are
separate events and are recorded separately. Rejected throughput and input
size distributions for all conditions are in the raw-derived tables.
Across all matrix conditions, B0 returned code 0 for 226 valid requests and
rejection-code counts 1:55, 4:369 and 7:46. B1 returned code 0 for 226 valid
requests and rejection-code counts 1:55, 4:49, 7:46, 11:52 and 13:340.
These are workload-mix counts, not an estimate of field attack prevalence.

Direct five-sample benchmark medians were **125.4 µs/op** for B0 full
validation and **251.8 µs/op** for B1; secp256k1 verification was
112.5 µs/op in the B1 benchmark, ML-DSA-44 verification 49.8 µs/op, B1
key binding 42.1 µs/op, and B1 decode/digest 2.3 µs/op. These separately
constructed operations cannot be summed to explain one full call. Valid
ABCI CheckTx was 121.8 µs/op B0 and 252.5 µs/op B1. B1's late invalid ML
signature reached 237.6 µs/op, versus 0.135 µs/op for an empty structural
reject. The network's focused invalid-class RPC median was 1,066.5 µs for
B1 ML rejection versus 777.7 µs for B1 structural rejection; network
transport and full envelope size narrow the relative difference. Empty
structural bytes and a ~3.9 KiB signed late-invalid envelope are not equal
offered-byte workloads. A modified observed signed transaction can reach
the latter class; an unauthenticated sender without a registered B1 account
cannot freely create new valid classical proofs for arbitrary intents.

Empty-block `Commit` benchmark medians were 2.21 ms/op B0 and 4.90 ms/op
B1, including synchronous Pebble persistence. The B1 state format and
registry differ from B0; the delta is not solely ML-DSA. The research-only
recorder's disabled and enabled full-validation medians were 266.5 and
268.7 µs/op with equal median allocation counts; the roughly 0.85% timing
gap is an observation from five command repetitions, not a calibrated
overhead guarantee. The initial microbenchmark using self-calibrated
`-benchtime=500ms` was terminated after its empty-block setup caused over
44,000 durable commits per repetition. Its partial log is retained as
`micro-five.log` and excluded. The final command used fixed `-benchtime=200x`
and completed 34 benchmarks with five samples each in `micro-five-200x.log`.

The `/proc` samples for the 8-tx/s valid-only condition averaged 0.098 CPU
seconds B0 and 0.190 CPU seconds B1 across four `plutod` processes during
the measured window; the sum of each process's maximum sampled RSS averaged
110.0 and 117.7 MiB per four-validator network. Each `plutod` embeds CometBFT, so those
figures cannot isolate application authentication from consensus work.
Sampled RSS is not a proven peak. The short windows and 10 ms CPU tick
granularity limit precision. A run's admission p95 is mathematically
computed from only 12 requests and often reflects the first RPC request;
the 8-tx/s valid-only means of per-run p95 were 6.59 ms B0 and 6.58 ms B1.
**All 122 run-level p99 values are INSUFFICIENT SAMPLES** under the
predeclared 100-observation threshold. The 3–5 independent-run bootstrap
intervals are descriptive and weak for formal inference.

The definitive matrix's PID samples retained combined user+system ticks,
so user/system split is `UNAVAILABLE` for those 122 runs. A bounded sampler
correction added separate `/proc` tick fields without changing transaction
or measurement validity. Its **supplementary** B1 50%-invalid smoke run
`raw/20261026T070850Z-smoke/`, staged tree
`118d2bdd4451ad363bcb42027164a146b4fbc615`, passed 1/1 real network
run with 0.21 CPU seconds total, comprising 0.17 user and 0.04 system CPU
seconds across the four `plutod` PIDs. The 100 Hz tick source and one-run
sample prevent extrapolation to the principal B0/B1 matrix. An earlier
supplementary smoke at `raw/20261026T070759Z-smoke/` passed but carried the
pre-correction staged tree ID while working code had changed; it is retained
as an identity-mismatched exploratory run and excluded from reported CPU
split evidence. The corrected smoke staged the relevant source before
execution and records the matching tree.

For a concrete same-height result, B1 run
`059-b1-p0-low-ml-r1` reached H7 on all four validators with block hash
`D17FED0C0B720D2137346F22DF521D1B4FA52C08CA06C03ED18AFA264238B762`,
AppHash `F2A570A1B7F864BA59A20EC9966A68C903E90D28C4596E45FE46FB5774902091`,
ordered transaction digest
`29ad34e8078535899c8a29da1d280953042094fdc29fdc94055a7d9fd80e74e2`,
sender 99/nonce 1, recipient 12/nonce 0, supply 6,400. B0 run
`001-b0-p0-low-classical-r1` reached H7 on all four with block hash
`FAC054C928F5074FD67A90EE56C625200E3D1291E23461511F94F0DAE0B8BC85`,
AppHash `A633203BE3CA076D3907C977EF2D6A2B16F590026EA1D4BA1406A24872DB15D5`,
sender 99/nonce 1, recipient 13/nonce 0, supply 6,401. The profiles have
different genesis and AppHash formats, so cross-profile hash equality is not
expected. The matrix compares only validators within one run at matching
height. Every failed exploratory run remains locally auditable and is absent
from these aggregate numbers.

## Safety and limitations

No validator/private transaction key, last-sign state, runtime home or raw
unsanitized log is intended for Git. `.gitignore` excludes the Phase 05 raw
directory. Publication-safe manifests and aggregate tables are reviewed
before staging. The work cannot prove a FIPS-certified module, physical
power-loss durability, Byzantine behavior, multi-host resource independence,
production capacity or DoS resistance. Controller review and Owner decision
remain required before any acceptance, merge or later phase.

## Correctness, regression and gate disposition

The corpus oracle checks both B0 and B1 evaluator codes, ABCI CheckTx
codes, unchanged state after invalid input, and direct ProcessProposal
rejection for `Q01`–`Q05` in each profile. Proposal cases include an input
that bypasses local admission, valid-then-invalid ordering, same-nonce
conflict, an oversized envelope and mixed invalid contents. The 122 real
network records independently check all four nodes' matching-height block
hashes, ordered transaction digests and AppHashes; all 65 account balances
and nonces and each profile's total supply are verified. The 452 valid
transactions were included once; the 1,012 rejected inputs made no
unauthorized committed state change. Recorder ON/OFF tests compare code,
proposal result and AppHash. No accepted B0/B1 production package was
modified. The same-host consistency result does not prove Byzantine safety.

| Gate | Disposition | Evidence |
|---|---|---|
| G01–G03 corpus, reproducibility, class codes | PASS | `public_corpus.json`, `make benchmark-corpus`, 25 B0 + 34 B1 cases |
| G04 CheckTx and direct ProcessProposal | PASS | corpus oracle, ten Q cases |
| G05 cross-validator state | PASS | 122 complete same-height records, zero divergence |
| G06–G07 B0/B1 preserved | PASS | accepted integration regressions and protocol source diff |
| G08 crypto micro | PASS | 34 benchmark names, five 200-iteration samples each |
| G09 ABCI/application benchmarks | PASS | callback and empty-block Finalize/Commit samples |
| G10–G11 four-validator workloads and ratio matrix | PASS | 122/122 runs, 0/25/50/75/90/100% conditions |
| G12 CPU/memory/latency | PASS with limitation | PID CPU and sampled RSS; final matrix user/system split unavailable |
| G13–G14 raw-derived analysis and repetitions | PASS | 30 groups, 40 matched pairs, 3–5 independent runs; p99 insufficient |
| G15–G17 confounders, interpretation, decision | PASS | D-030, preregistered protocol, separate NO-GO document |
| G18 reproduction commands | PASS | public corpus, smoke, micro, full, analyze and Python unit; full matrix ran |
| G19 SHA-256 source/raw manifests | PASS after manifest check | `SOURCE_SHA256.txt`, `TESTED_SOURCE_SHA256.txt`, `RAW_SHA256.txt` |
| G20 private publication boundary | PASS for reviewed payload | raw ignored, explicit staged-path and secret scan |
| G21 build/unit/integration | PASS | verification command matrix below |
| G22 race/fuzz | PASS | Phase 05/state races, real workload race, corpus/B1 decoder fuzz |
| G23 final hosted CI | PENDING | Draft PR verification after push |
| G24 report/handoff | CANDIDATE | this report, decision, tables, figure, private raw audit |

`PASS` above means the stated implementation or local execution gate passed;
it does not grant Independent Controller technical acceptance or Owner
closure. G23 stays open until final hosted checks are observed.

## Exact local verification

The definitive matrix command was `make benchmark-full`, exit **0**;
`PHASE05_RUN_ROOT=research/evidence/phase_05/raw/20261026T062154Z-full
make benchmark-analyze` exited **0** with 122 expected and 122 complete
records. `make benchmark-micro` with five fixed 200-iteration repetitions
exited **0**. `python3 research/benchmarks/phase05/analyze_micro.py
<micro-log> <micro-json>` returned 34 names with minimum five samples;
`render_tables.py` produced the published summary. `make benchmark-smoke`
on the timeout-safe final runner exited **0**, six valid and six rejected
requests. An earlier self-calibrated micro attempt exited **143** by
intentional SIGTERM and is retained, not counted as a pass.

The final local command ledger under the ignored definitive raw root records
each exact command and exit code. All of the following exited **0**:

| Command | Exit |
|---|---:|
| `make verify` | 0 |
| `make test-integration` | 0 |
| `make test-integration-4v` | 0 |
| `make test-integration-b1-4v` | 0 |
| `go build ./...` | 0 |
| `go test -count=1 ./...` | 0 |
| `go vet ./...` | 0 |
| `go mod verify` | 0 |
| `make fmt-check` | 0 |
| `python3 -m unittest discover -s research/benchmarks/phase05 -p test_analyze.py` | 0 |
| `go test -race -count=1 ./internal/phase05/...` | 0 |
| `go test -race -count=1 ./internal/b0app ./internal/b1app ./internal/b0state ./internal/b1state ./internal/b0store ./internal/b1store` | 0 |
| `go test -race -tags 'integration fourval' ./test/integration -run '^TestPhase05Workload$' -timeout 240s -count=1 -v` | 0 |
| `go test ./internal/phase05/corpus -run '^$' -fuzz '^FuzzCorpusDecoder$' -fuzztime=5s -parallel=2` | 0 |
| `go test ./internal/b1state -run '^$' -fuzz '^FuzzHybridDecode$' -fuzztime=5s -parallel=2` | 0 |
| `git diff --check` and `git diff --cached --check` | 0 each |

The integration and race logs had no test-level `SKIP`, `FAIL` or race
detector warning. Expected nonzero child-process results inside fail-closed
crash tests are part of passing tests and are preserved in their raw logs.
The same complete regression list was run after the final Go sampler change;
the Python runner's timeout cleanup was independently unit tested and
validated by a real four-validator smoke.

## Evidence identity, publication and Controller handoff

The public corpus is 1,390,541 bytes, with 155-byte valid B0 and 3,907-byte
valid B1 wire transactions. The definitive matrix's 122 run records all
declare the same frozen staged tree `fc629192cb046befcdae2058fac675c92d6de1db`.
`TESTED_SOURCE_SHA256.txt` contains 25 file digests recomputed directly
from that Git tree with zero mismatch. `SOURCE_SHA256.txt` records the final
reviewed source/report payload; `RAW_SHA256.txt` records hashes of retained
local raw and exploratory files. The raw directory is Git-ignored and must
not be uploaded as part of the Draft PR. The evidence audit describes
all superseded runs, including the identity-mismatched supplementary smoke.

The source branch is `phase/05-b1-experimental-evaluation`, based on the
Owner-merged Phase 04 commit `b861479982b0114be2ad60f1e7727362075a7f73`.
The exact result commit/tree and hosted CI conclusions are supplied with the
Draft PR final handoff after Git publication. A self-referential commit hash
cannot be embedded in the same Git commit. Phase 05 is an implementation
candidate and requires Independent Controller review and Owner decision.

**STOP — no merge, no DoS mechanism and no subsequent phase.**
