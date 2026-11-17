# X05 — dedicated application process CPU validation

Status: **PASS for process separation and process-level measurement only**.
Source commit before the run: `5d512c01dfb28d698f3c7b65fc632a0abc060aa4`.
The locked QA plan is `research/protocol/FILE02_X05_PROCESS_QA_PLAN.md`.

```text
integration harness (one process)
  -> Comet validator 0 -> ABCI socket 127.0.0.1 -> profile-app PID 52108
  -> Comet validator 1 -> ABCI socket 127.0.0.1 -> profile-app PID 52132
  -> Comet validator 2 -> ABCI socket 127.0.0.1 -> profile-app PID 52158
  -> Comet validator 3 -> ABCI socket 127.0.0.1 -> profile-app PID 52183
```

All four app PIDs were distinct and separate from the harness. One valid H
transaction passed CheckTx and DeliverTx, appeared at height 3, and the
four validators agreed on block ID and header AppHash. Validator 0 emitted
a CheckTx recorder event. This verifies the accepted app code path and
correctness for this QA transaction, not performance equivalence.

| Validator | CPU interval user+system (ms) | peak RSS (MiB) | GC cycles | GC pause (µs) | threads | samples |
|---|---:|---:|---:|---:|---:|---:|
| 0 | 36.911 | 39.05 | 2 | 195.861 | 13 | 8 |
| 1 | 34.812 | 39.68 | 2 | 120.950 | 13 | 8 |
| 2 | 28.616 | 39.84 | 1 | 57.046 | 11 | 7 |
| 3 | 27.935 | 40.30 | 1 | 58.394 | 13 | 7 |

The process series also record user/system CPU separately, Go heap,
goroutines and thread status. Private raw under
`research/artifacts/file02/x05-20261115T000000Z-5d512c0/raw` was frozen
after all processes stopped; manifest SHA-256:
`e2e0d9385f6670912496374dae2118fdb51a266df811fc3870aa3ea2d3ad49df`.
Independent `file02_common.verify_raw` returned the same digest. The raw
directory is Git-ignored and contains runtime homes/keys; it must not be
published. `go test -tags 'integration fourval' ./test/integration -run
'^TestFile02X05DedicatedAppProcesses$' -count=1 -timeout 300s -v` passed.

`getrusage` process CPU covers all application work during each interval,
including idle/runtime/GC. `ru_maxrss` is a peak, not interval RSS. These
numbers cannot be divided by one request to infer per-stage CPU. Observer
overhead and attribution remain X06/X09 work. Four software processes on
one physical host remain one host, not independent hardware replication.
