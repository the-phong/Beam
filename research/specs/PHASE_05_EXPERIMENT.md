# Phase 05 experimental protocol and Controller correction

## Supplementary matched-legitimate-rate protocol (Controller correction)

The 122-run, 12-request matrix described below is retained as **exploratory**.
It cannot establish valid-service degradation: increasing its invalid fraction
reduced the legitimate offered count, and every per-run p99 was insufficient.
The supplementary experiment below tests contention while holding legitimate
traffic constant. It was specified before its full execution; failed and
superseded attempts remain in the local execution manifest.

The new public corpus is `pluto-phase05-corpus-contention-v1`. Each B0 and B1
profile contains 64 independent public sender identities and six correctly
signed nonce layers, 384 valid transfers in total. Sender zero is reserved
for the stable, fully signed insufficient-balance probe. The workload uses
300 distinct public signed valid transactions from the other 63 senders.
Private signing keys are never written by the corpus generator. The original
v2 corpus and its 122-run results are preserved unchanged.

| Series | Profiles | Legitimate tx/s | Added invalid tx/s | Invalid fraction | Valid / invalid per run | Repetitions |
|---|---|---:|---:|---:|---:|---:|
| Primary | B0, B1 | 1.5 | 0, 0.5, 1.5, 4.5, 13.5 | 0, 25, 50, 75, 90% | 300 / 0, 100, 300, 900, 2700 | 5 |
| Invalid only | B0, B1 | 0 | 15 | 100% | 0 / 3000 | 5 |
| Secondary | B0, B1 | 4 | 0, 4, 12 | 0, 50, 75% | 300 / 0, 300, 900 | 3 |

There are 78 planned independent runs, each with a new four-validator home.
All offered rates are at or below 16 tx/s. A nonsecret fixed seed records run
pairing; already-signed public bytes supply all traffic. Valid and invalid
streams are independently paced, then deterministically merged. Every run
waits until all four validators reach height 2, submits its complete bounded
schedule, observes committed blocks from a second validator, and verifies all
four validators' block history, matching-height AppHashes, sender balances,
nonces and total supply. It records achieved rates, client admission latency,
block-observation inclusion latency, per-process CPU ticks and sampled RSS.
The block observer polls every 150 ms and queries each new block once; its
inclusion timestamps are upper-bound observations, not exact finality times.
Rejected inputs have no inclusion latency. The sampler includes Pluto and its
embedded CometBFT in one PID on this runtime.

The primary comparison pairs each ratio with its own-profile zero-invalid
baseline at the same legitimate rate and repetition. Per-run p99 is reported
only with at least 300 confirmed valid samples and remains a descriptive,
uncertain tail estimate. Across-run uncertainty uses independent run values.
The 100%-invalid condition has no valid latency or goodput estimate. Mixed
invalid classes cycle through structural, binding, classical proof, ML proof
and state-invalid cases; the B0 binding slot uses a classical-proof case.
The attacker capability for every class remains as specified below.

The provisional screening signals are 2× valid p99 amplification, 20%
legitimate goodput degradation, and a repeatable B1/B0 normalized effect.
They are **not scientific standards or acceptance thresholds**. Without a
measured contention regime, the broader DoS-risk question is INCONCLUSIVE,
even if no degradation is observed. The single-host WSL2 setup cannot prove
multi-host or production capacity. No DoS mechanism is in scope.

## Historical exploratory protocol

**Authority:** Owner Phase 05 contract, 2026-10-26. This protocol characterizes
the accepted B0/B1 implementations. It changes no transaction, state,
consensus, or admission validity rule. The public corpus is version
`pluto-phase05-corpus-v2` with SHA-256
`242e5c848aa3c97f731a3e70f9ca0876a9ec68bffbbbab2ead8c222c005cd535`.
The implementation source commit is recorded after source freeze and before
the principal runs. Exploratory data from corpus v1 are retained separately
and never pooled with v2.

## Questions and falsifiable hypotheses

- **RQ1/H1:** For equivalent one-unit transfers at the same offered
  transaction rate, median direct validation wall time for B1 exceeds B0.
  Microbenchmarks separate secp256k1, ML-DSA-44, parsing, binding, state work.
  Network differences include byte size, registry, persistence and consensus.
- **RQ2/H2:** An invalid B1 transaction that passes secp verification and
  reaches ML-DSA verification costs more direct validation time than a
  structural early reject. The oracle fixes the reached stage before timing.
- **RQ3/H3:** The ordering of cost by invalid class is structural < binding <
  classical proof < ML proof or state-invalid. This is a hypothesis, not a
  guaranteed monotone relation; state hashing and key parsing may change it.
- **RQ4/H4:** At fixed offered rate, higher invalid fractions reduce valid
  committed transactions per second. A compositional reduction is expected
  even without contention, so a service-degradation claim requires a
  matched legitimate offered rate and nontrivial latency/CPU effect.
- **RQ5/H5:** Under equivalent committed state, all four validators accept or
  reject the same direct proposal, and all committed blocks and AppHashes
  match at the same height. Any failure invalidates affected timings.
- **RQ6:** No GO decision is presumed. A mechanism hypothesis requires
  repeated cost asymmetry, feasible attacker capability, and evidence of
  legitimate service impact beyond workload composition.

## Attacker capabilities

| Capability | Can construct | Cannot assume |
|---|---|---|
| A: unauthenticated external sender | Arbitrary bytes, invalid size/magic, self-generated unfunded B0 key | A registered B1 sender binding or valid proof for a registered account |
| B: observer/modifier | Replays public signed bytes; mutates a proof or field, including a valid secp proof paired with invalid ML proof | Fresh signatures on changed intent |
| C: registered malicious account holder | Signs state-invalid nonce, amount or conflicting same-nonce intent using own genesis keys | Post-genesis account registration or another account's keys |

Only harness-owned localhost networks receive workload traffic. The B1
registry is genesis-bound and has no registration or rotation path. Public
signed bytes and genesis are replayable; private signing keys are discarded
after corpus generation and never serialized. The corpus ID is its file hash;
the traffic seed controls only ordering, not signing identities.

## Matrix and boundaries

`run_matrix.py --mode full` executes sequential fresh four-validator networks:

| Matrix | Profiles | Invalid % | Rate | Requests per run | Independent runs |
|---|---|---|---:|---:|---:|
| Primary | B0, B1 | 0, 50, 90, 100 | 2 or 8 tx/s | 12 | 5 per condition |
| Secondary | B0, B1 | 25, 75 | 4 tx/s | 12 | 3 per condition |
| Class-focused | B0: structural/classical/state/mixed; B1: structural/binding/classical/ML/state/mixed | 100 | 4 tx/s | 12 | 3 per class/profile |

This is 122 planned independent runs: 80 primary, 12 secondary, 30 focused.
Each run creates a fresh four-validator root, four distinct validator keys and
stores, and the same public B0 or B1 genesis. It waits for height 2 before
measurement (block-based warm-up), sends at paced intervals, polls inclusion,
then checks matching-height block and AppHash history, ordered transactions,
all 65 account balances/nonces and supply. Maximum rate is 16 tx/s; maximum
count is 63 because one sender is reserved for invariant late-invalid probes;
the Go test has a 180-second timeout and process cleanup is
restricted to its child processes. The 12-request primary run is a short
diagnostic window. Its p99 is **INSUFFICIENT SAMPLES**; no p99 precision is
claimed from that run. Warm-up is defined by block height rather than a
fixed duration. Exact observed duration and counts are in raw JSON.

The primary paired comparison matches B0/B1 ratio, load and repetition. B0
uses 65 accounts with 100-unit senders and a 1-unit receiver; B1 uses 65
registry-bound accounts with 100-unit senders and a 0-unit receiver. Both
transfer one unit; initial supply is 6401/6400 respectively because B0
Genesis V2 requires positive allocations. The different state formats,
transaction sizes and registry make raw network differences non-ML-only.

Equal offered **byte-rate** comparison is a separate focused analysis and
requires choosing B0 transaction rate to match B1 byte rate. At the bounded
maximum 16 tx/s and 12-request run size, exact equality may be infeasible;
if so it must be marked unavailable rather than implied by equal tx/s.

## Measurement and exclusion policy

Cryptographic and application callbacks use Go `testing.B` with `-benchmem`
and five command repetitions. The wall time around callbacks includes Go
scheduling; `Commit` includes synchronous Pebble persistence. The optional
research-only recorder uses Go monotonic `time.Now`/`time.Since`, a finite
buffer, and drops samples only. It never enters production code or changes
ABCI results. S1–S6 are separately measured component operations, not an
additive partition of one CheckTx call. S7–S10 are callback boundaries.

Network admission latency is client-side `broadcast_tx_sync` request/response
wall time. Valid inclusion latency ends when a separate validator's `/tx`
first reports a successful committed result. Polling starts after each valid
admission and uses a 100 ms interval, making the result an upper-bound
observation rather than the exact commit instant. Rejected transactions have
no inclusion latency.
CPU is per-process `/proc` user+system ticks, converted using the observed
kernel tick rate; RSS is sampled, not a reliable all-time peak. In this
deployment CometBFT runs within each `plutod` process, so its CPU and RSS
cannot be separated from application work using PID samples. The sampler
also detects child PIDs if a future runtime starts them, but none appeared
in the definitive matrix. Host activity remains a confounder. WSL2
and the Windows-mounted repository path may affect reads; test homes are
under `TMPDIR`. Every raw record includes environment and source metadata.

No run, request, timeout, slow observation or outlier is silently excluded.
Incomplete or failed runs remain raw and cannot enter successful aggregates.
Corrections require new run IDs and an explicit supersession note. Analysis
uses each run as the independent unit and bootstraps means over runs (3000
draws, fixed non-secret seed); individual request samples are not treated as
independent machines. A 95% interval from only 3–5 runs is descriptive, not
a strong significance test. p99 requires at least 100 relevant observations
within the reported unit; otherwise the output says INSUFFICIENT SAMPLES.

The experiment cannot establish Byzantine safety, physical power-loss
durability, independent host fault domains, DoS resistance or algorithm-only
causality from B0/B1 network deltas. No traffic leaves the local harness.

### Measurement correction before the final matrix

An initial five-run exploratory B0 valid-only series revealed that the first
runner polled inclusion only after sending all requests and ended its
throughput/CPU window after expensive account-verification queries. That
inflated inclusion latency and understated throughput. The exploratory root
`20261026T053931Z-full` and interrupted sixth-run log/home archive are kept
locally, excluded from final comparison. The corrected runner polls a second
validator throughout submission and marks a measurement end before
post-workload state verification. The analyzer uses only the new measurement
start/end fields and samples within that interval. The final matrix starts
fresh with a new run root and a new exact staged source tree identity.

### State-context correction before the definitive matrix

The next exploratory root `20261026T054450Z-full` exposed a corpus scheduling
error at run 042: an insufficient-balance transaction from sender V000 was
submitted after a valid V000 transfer. The accepted first-failure precedence
correctly returned invalid nonce (code 6), while the scheduled case expected
insufficient balance (code 7). The run failed and remains retained. The
runner now reserves V000 for state-invalid probes and uses V001 onward for
valid workload transfers. This keeps A03's committed-state context stable
without changing production validation or weakening the expected code. The
runner caps all-valid workloads at 63 requests. The exploratory root is
excluded from final performance aggregates; the definitive matrix uses a
fresh root and source tree after this correction.
