# Reproduce the S07 bounded smoke

This command runs four **fresh, isolated** CometBFT validator networks on one
Linux/WSL2 host, one each for C, CP, P and H. It exercises a small mixture of
legitimate and deliberately invalid pre-signed transactions. It is a QA smoke,
not a confirmatory RQ1–RQ3 experiment or a throughput/DoS test.

## Prerequisites

Use Go 1.24.x, Python 3 and GNU Make from a checked-out Beam
`research/s07-reproducibility` revision. The application and Go module remain
Pluto. Run from the repository root on a machine you control. The harness
uses only `127.0.0.1` RPC/ABCI/P2P endpoints and the pinned safety limits in
[`research/config/s06_pilot.json`](research/config/s06_pilot.json). It refuses
public RPC targets, duplicate validator identities, stale/missing observers,
rate misses and safety violations. Do not copy private runtime homes or raw
logs into Git.

```bash
make test-s07-schema
make test-s07-smoke
```

The second command prints `S07_SESSION=...` beneath ignored
`research/artifacts/s07/`, then `S07_STATUS` and a manifest SHA-256. Every
invocation creates a new session ID. A failed session remains frozen as
`INVALID_FOR_ANALYSIS`; rerun the command for a new ID. Never edit or reuse a
failed run directory.

To recheck a completed session, use the exact printed path:

```bash
python3 research/tools/s07_orchestrate.py verify research/artifacts/s07/<session-id>
```

`verify` checks the manifest digest, every frozen file hash, schemas,
source/config/corpus/plan/genesis identities, phase order and recomputed
per-run metrics. It exits nonzero if a frozen file is missing, added or
modified. `derived.csv` and `derived.json` inside that private session
contain one row per profile; `exclusions.json` records every invalid run and
its reason. Raw files are created exclusively and made read-only at freeze.
Validator homes under each `network/` are disposable private runtime state;
their keys are never published or included in the public sample.

## Trace through one run

| Stage | Artifact or check |
| --- | --- |
| Source/config | `session.json`, frozen `config.json`, `corpus.json`, and `metadata.json` hold commit/tree, hashes, host, seed and offered rates. |
| Setup | A fresh four-validator network and four independent observer processes are created for each profile. |
| Warm-up → execute → drain | `plan.json` supplies absolute schedules; the separate `profile-load` process enforces them and writes `result.json`. `phases.jsonl` identifies delegated schedule boundaries. |
| Oracle | `oracle.json` records common-height block/AppHash agreement, one commit for each legitimate tx, no invalid commit, and matching sender/receiver state on four validators. `commit_observations.json` records the first indexed result seen on **all four** RPCs. |
| Collect | `observer-0.jsonl` … `observer-3.jsonl` retain raw S05 telemetry; validator and load logs remain private. |
| Analyze | The strict stdlib analyzer produces per-run size, process CPU/RSS, RPC latency, conservative inclusion-latency upper bound, queue, mempool, observed relay and a conservative steady-window goodput lower bound. Missing relay streams are counted, never assumed zero. |
| Freeze | `manifest.json` and `MANIFEST_SHA256` bind all raw and derived files except disposable `network/` homes. |

The `goodput_observed_lower_bound_tx_per_steady_second` field counts only
legitimate transactions first observed indexed on all four validators within
the steady window. Polling at 100 ms resolution can delay observation, so it
is a lower bound on true steady-window goodput. The corresponding inclusion
latency is an **upper bound**, not an exact commit timestamp. `rpc_latency`
is CheckTx RPC round-trip, not inclusion latency. Relay totals cover only
streams where the metric was observed; coverage is reported separately.

Each profile has one independent network run in this smoke. The output states
`independent_run_n=1` and leaves 95% confidence limits `null`: individual
transactions and telemetry samples within a run are not independent
replicates. The [public example](research/evidence/s07/smoke_sample.json)
contains only derived, labeled smoke metrics and hashes. Its source/config/
corpus/manifest digests identify the original private session; fresh keys and
network ports make a later run's raw digest different. This is expected.

The quick CI workflow checks build, Go units, public vectors/corpora, S07
schema/integrity tests and this four-validator smoke. The separate existing
secret-scan workflow checks PR commits. Full research experiments remain
outside CI and require File 02 authorization and an isolated testbed.
