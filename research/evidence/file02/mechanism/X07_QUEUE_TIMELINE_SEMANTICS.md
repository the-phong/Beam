# X07 — bounded queue timeline semantics and disposition

Status: **PARTIAL; full transaction queue residence remains unavailable**.
The application CheckTx recorder now emits `app_ingress_utc`,
`app_lock_acquired_utc` and `app_result_utc` for recorder-ON requests.
The public ON/OFF corpus test passed for accepted and rejected cases,
NEW/RECHECK and unchanged AppHash. A new dedicated-process four-validator
QA run at source `7e7eaf6` passed a valid H CheckTx/commit and block/AppHash
agreement. Its private raw manifest SHA-256 is
`0ca42dbf54c7b0190c8cacad110d2fa9e80c192eaa79cd7347c7767c16687f8a`.
One validator-0 CheckTx event recorded ordered app entry, lock acquisition
and validation-result timestamps. Raw was frozen after all processes
stopped; it remains Git-ignored.

| Interval | Collection and semantics | Status |
|---|---|---|
| Producer socket-write → RPC response | Existing submitter monotonic round trip and UTC endpoints | AVAILABLE; includes all RPC/network work |
| RPC proxy receive → proxy response | Existing ingress proxy UTC/round-trip | AVAILABLE when proxy used; no transaction hash for exact join |
| Comet socket-client queue attempt → writer dequeue | Existing Comet overlay histogram count/sum | AVAILABLE as aggregate socket subqueue only |
| ABCI app method entry → app lock acquisition | New event timestamps and `app_lock_wait_ns` | AVAILABLE per recorded CheckTx |
| App lock acquisition → validation result | New timestamps; stage wall spans | AVAILABLE per recorded CheckTx; not stage CPU |
| Accepted mempool occupancy | Existing sampled count/bytes | AVAILABLE as snapshots, not residence |
| P2P relay | Existing envelope byte counters | AVAILABLE as aggregate traffic, not per-tx delay |
| RPC proxy → Comet precheck/admission, Comet mempool residence, per-tx relay, complete ingress-to-result queue | No safe correlated per-tx markers in the accepted overlay | `null`: `UNAVAILABLE_FULL_TRANSACTION_QUEUE_RESIDENCE` |

The socket histogram does not include RPC admission, Comet mempool
processing, ABCI service, app lock wait, relay or proposal. Its mean or
maximum cannot be called the whole transaction queue. The proxy event
does not carry a transaction ID, so even same-host UTC fields cannot
construct a sound per-transaction queue trajectory across layers.
Cross-process wall-clock uncertainty is not measured tightly enough to
subtract individual UTC stamps into a high-precision delay. Existing
accepted-mempool samples are point-in-time occupancy, not exact enqueue/
dequeue logs. Adding fully correlated Comet mempool and P2P queue tracing
would require a separately validated overlay with overhead and correctness
QA; this study records the limitation rather than inventing zero wait.
