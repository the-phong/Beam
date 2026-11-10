# Phase 05 local reproduction

Paths in this guide use the consolidated B0/B1 folders as of 2026-11-10.
The historical report and source manifests keep the paths used at the time;
the public corpus bytes and recorded SHA-256 digests are unchanged.

## Supplementary matched-rate contention correction

The previous 122-run, 12-request experiment below remains a historical
**exploratory** dataset. For the Controller-requested matched legitimate
traffic experiment, use the new public corpus and separate commands:

```bash
make benchmark-contention-corpus
make benchmark-contention-smoke
make benchmark-contention-full
PHASE05_RUN_ROOT=research/artifacts/b0b1-runs/<UTC-run-id>-contention-full make benchmark-contention-analyze
python3 -m unittest discover research/benchmarks/b0b1 -p 'test_*.py' -v
```

`benchmark-contention-full` schedules 78 independent fresh networks. The
primary conditions keep 1.5 legitimate tx/s and add 0, 0.5, 1.5, 4.5 or
13.5 invalid tx/s. The 100%-invalid condition separately offers 15 invalid
tx/s; the secondary conditions offer 4 valid tx/s plus 0, 4 or 12 invalid
tx/s. Total offered rate never exceeds 16 tx/s. Each primary or secondary
run submits 300 valid public signed transactions, and primary conditions
have five independent repetitions. The program uses a bounded 360-second Go
test and 390-second outer timeout per run. Do not run competing benchmarks
on the shared host. A full run requires several hours and must retain the
exact `execution.jsonl`, raw JSON, command logs and validator logs locally.

The contention corpus is
`research/corpus/b0b1/public_corpus_contention.json`. It contains 384
sequenced valid signed transactions per profile, using 64 public senders and
nonce layers 0–5. Generator signing keys remain ephemeral. Verifying the
committed public bytes needs no private key. Re-running generation creates
new random identities and a new digest, so use the stored public corpus for
byte-identical replay. The reserved sender-zero state-invalid probe retains
its expected first-failure code throughout the run.

The new analyzer writes publication-safe `per_run.csv`, `per_run.json`,
`normalized.json`, `aggregate.json` and `failed_runs.json`. It verifies the
execution ledger's log/result hashes and refuses to call a full matrix
complete unless all 78 runs are present and successful. Valid client RPC
admission latency and valid inclusion latency are separate distributions;
100%-invalid runs have no valid latency. The block observer polls a second
validator every 150 ms and queries each committed block once, producing an
upper-bound inclusion timestamp. Its load and the shared-host environment
remain measurement confounders. A per-run p99 from 300 samples is descriptive
and still uncertain; run-to-run intervals resample independent runs.

Raw results remain ignored and local; only reviewed sanitized tables and
hash manifests may enter Git. The runner creates and stops only its own
process tree. Timeouts and failed assertions remain in the ledger and are
never relabeled successful.

If a run has a wall-clock discontinuity, retain its original ledger, result
and log. Rerun the exact condition in a new root with
`python3 research/benchmarks/b0b1/run_contention.py --mode repair34` for
the identified B0 secondary baseline case. Pass that new root as
`PHASE05_REPAIR_ROOT` to `benchmark-contention-analyze`. The analyzer verifies
both ledgers and hashes, records the original in `excluded_runs.json`, and
uses the replacement only for the matching condition. The repaired Go harness
measures inclusion and elapsed duration with monotonic time; Unix timestamps
remain diagnostic only. No original file is overwritten.

## Historical exploratory commands and artifacts

The repository runs B0 and B1 four-validator networks under Go test control.
Use Go 1.24.1 and the pinned `go.mod`; Python 3 standard library is required
for the matrix and analysis. Tests launch the repository's built `plutod` and
CometBFT v0.38.26 binaries, bind only localhost ports, create fresh homes in
Go temporary directories, track child PIDs and reap only their own processes.
The full matrix is intentionally a local, sequential, bounded experiment.

```bash
make verify
make test-integration
make test-integration-4v
make test-integration-b1-4v
make benchmark-corpus
make benchmark-smoke
mkdir -p -- research/artifacts/b0b1-runs
make benchmark-micro > research/artifacts/b0b1-runs/micro-five.log 2>&1
make benchmark-full > research/artifacts/b0b1-runs/full-matrix.log 2>&1
PHASE05_RUN_ROOT=research/artifacts/b0b1-runs/<UTC-run-id>-full make benchmark-analyze
python3 research/benchmarks/b0b1/analyze_micro.py research/artifacts/b0b1-runs/micro-five.log research/artifacts/b0b1-runs/micro-summary.json
python3 research/benchmarks/b0b1/render_tables.py research/artifacts/b0b1-runs/<UTC-run-id>-full/analysis research/artifacts/b0b1-runs/micro-summary.json research/artifacts/b0b1-runs/summary-tables.md
python3 -m unittest discover research/benchmarks/b0b1 -p 'test_*.py' -v
```

`benchmark-b1` runs the same prescribed primary, secondary and focused
conditions for B1 only. `benchmark-full` runs both profiles, 122 fresh
networks. The smoke runs one 12-request B1 condition, then verifies inclusion,
ordered block history, matching AppHashes, all accounts and supply. Every
new command has an executable smoke form. `b0b1-corpus -mode verify` checks
the public file hashes, runs its oracle against disposable B0/B1 stores and
directly sends invalid candidate proposals to ProcessProposal. Corpus replay
uses no private signing key. Generation is deliberately an exclusive-create
operation and gives a different corpus digest because signing identities are
fresh; never claim a regenerated file is byte-identical to the committed
corpus.

The public corpus lives at `research/corpus/b0b1/public_corpus.json`.
`valid_pool` contains 64 independently signed transfer bytes per profile.
`cases` hold exact hex bytes or a bounded zero-repeat recipe for oversized
input. Every case has a digest, size, attacker capability, expected code,
state context and reached validation stage. `proposals` name ordered byte
vectors for direct ProcessProposal. `genesis_cases` check corrupt B1 bindings.
Metadata and signed transactions are public; no secret key appears in the
file. The original generation process discards ephemeral keys.

The matrix driver writes a unique ignored directory under
`research/artifacts/b0b1-runs/` with one raw JSON record, one exact command
log, per-validator process logs and an append-only execution ledger per run.
It refuses to overwrite a run ID. Failed runs remain there, marked
`complete=false` or missing data and with nonzero ledger exit. The analyzer
reads only complete records but lists all failed/incomplete records and can
exit nonzero with `--require-complete`. Repeating a failed condition uses a
new run ID and must be documented in the final report.

Local raw logs and node homes are publication-sensitive. `.gitignore`
excludes the raw directory; Git includes only sanitized summaries and
SHA-256 manifests after review. The user should avoid pointing `TMPDIR` or
the output root at an existing validator home. No public or third-party
endpoint is used. The benchmark runner caps offered rate at 16 tx/s, 63
requests, and a 180-second per-run Go timeout. The runner's outer timeout is
210 seconds. On timeout it snapshots `/proc` descendants of its exact Go
test PID, checks PID birth markers and stops only those owned processes;
the terminated PID list is retained in the execution ledger. Results are
not production capacity or DoS guarantees.

Interpret the outputs as follows:

- `admission_latency_ns` is client-side `broadcast_tx_sync` wall time.
- `inclusion_latency_ns` exists only for valid included transactions and is
  an observed upper bound under concurrent 100 ms polling from another
  validator. It does not wait until the submission batch is over.
- `process_samples` are per-PID CPU ticks and sampled RSS. Each `plutod`
  embeds CometBFT in the definitive run; application and consensus resource
  use cannot be separated by PID. The sampler can label child processes, but
  none were observed. `clock_ticks_per_second` is recorded by the analyzer.
  The definitive 122-run matrix records combined CPU ticks. A subsequent
  source-tied supplementary smoke additionally records user and system ticks;
  no split is inferred retroactively for the matrix. Sampled maximum RSS is
  not a proven peak.
- `matched_height` identifies the block height whose block hash and ordered
  history were compared. Its AppHash is read from the *next* block header,
  because CometBFT headers carry the preceding committed application hash.
- B0/B1 state, transaction and registry formats differ; equal tx/s network
  comparisons are not pure ML-DSA measurements.
- Each run is one statistical replicate. The analyzer does not treat
  individual requests as independent runs. A p99 needs at least 100 relevant
  observations; shorter runs report `INSUFFICIENT SAMPLES`.
