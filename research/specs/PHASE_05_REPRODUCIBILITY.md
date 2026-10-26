# Phase 05 local reproduction

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
make benchmark-micro > research/evidence/phase_05/raw/micro-five.log 2>&1
make benchmark-full > research/evidence/phase_05/raw/full-matrix.log 2>&1
PHASE05_RUN_ROOT=research/evidence/phase_05/raw/<UTC-run-id>-full make benchmark-analyze
python3 research/benchmarks/phase05/analyze_micro.py research/evidence/phase_05/raw/micro-five.log research/evidence/phase_05/raw/micro-summary.json
python3 research/benchmarks/phase05/render_tables.py research/evidence/phase_05/raw/<UTC-run-id>-full/analysis research/evidence/phase_05/raw/micro-summary.json research/evidence/phase_05/raw/summary-tables.md
python3 -m unittest discover research/benchmarks/phase05 -p 'test_*.py' -v
```

`benchmark-b1` runs the same prescribed primary, secondary and focused
conditions for B1 only. `benchmark-full` runs both profiles, 122 fresh
networks. The smoke runs one 12-request B1 condition, then verifies inclusion,
ordered block history, matching AppHashes, all accounts and supply. Every
new command has an executable smoke form. `phase05-corpus -mode verify` checks
the public file hashes, runs its oracle against disposable B0/B1 stores and
directly sends invalid candidate proposals to ProcessProposal. Corpus replay
uses no private signing key. Generation is deliberately an exclusive-create
operation and gives a different corpus digest because signing identities are
fresh; never claim a regenerated file is byte-identical to the committed
corpus.

The public corpus lives at `research/corpus/phase_05/public_corpus.json`.
`valid_pool` contains 64 independently signed transfer bytes per profile.
`cases` hold exact hex bytes or a bounded zero-repeat recipe for oversized
input. Every case has a digest, size, attacker capability, expected code,
state context and reached validation stage. `proposals` name ordered byte
vectors for direct ProcessProposal. `genesis_cases` check corrupt B1 bindings.
Metadata and signed transactions are public; no secret key appears in the
file. The original generation process discards ephemeral keys.

The matrix driver writes a unique ignored directory under
`research/evidence/phase_05/raw/` with one raw JSON record, one exact command
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
