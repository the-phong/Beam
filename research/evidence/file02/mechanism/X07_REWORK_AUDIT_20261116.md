# X07 rework audit — transaction-linked ingress and queue segments

Disposition: **PARTIAL; master X07 remains unchecked**. The bounded
source-hash-checked Comet overlay now records transaction-hash-linked RPC,
mempool precheck, ABCI socket queue, CheckTx callback, accepted-mempool
insertion/removal, committed update, and local P2P send/receive events.
The dedicated app recorder records matching transaction-hash entry, lock and
result. Both processes read the same Linux `CLOCK_MONOTONIC`; the tool never
subtracts cross-process UTC timestamps. The current verifier is
`research/tools/file02_x07_trace.py`.

## Independently checked QA

Source/plan commit `a91aad7ada4034c885c2305b7a599163aab02687`; private
matched QA session `research/artifacts/file02/x07-qa-20261116T083347Z-86109889`;
immutable raw manifest SHA-256
`0da0d76e6242d1e27ee783718a8f8efd10c533df107f585f0f9812c901ac1b5e`.
The ON and OFF networks used the same signed fixture and genesis on separate
fresh four-validator networks. Each arm sent 241 H-sized RPC transactions at
20 tx/s: 73 valid H committed exactly once; 168 late-bad-ML rejected.
All-four AppHash was identical in both arms, app recorder drops were zero,
and `file02_x07_trace.py verify` returned `PASS_QA_ONLY`. The ON trace held
2,600/1,486/1,480/1,498 events across validators 0–3, with all 241 origin
transaction paths checked against the workload and app result codes.
Missing or repeated critical markers, PID/sequence drift and partial JSONL
files are verifier errors.

The ON run measured mean 52.34 µs for the Comet ABCI socket subqueue and
1.105 s accepted-mempool residence for valid transactions. Mean RPC-entry
to app-entry was 426.05 µs; **that interval includes processing and transfer
and is not a queue-only duration**. Median producer RPC round trip was
0.863 ms OFF and 0.965 ms ON. This is one engineering QA pair, so the
difference is descriptive, not an overhead CI or a service-effect estimate.
The new trace is measurement-only; it did not change validity or AppHash in
this pair.

## Exclusions and remaining gap

Two earlier QA sessions remain private and immutable. The first source
`70d43cd` lacked a shared cross-process clock. The second source `7a23c2e`
used independently generated ON/OFF fixtures; a later verifier correctly
rejects it for genesis/workload/AppHash mismatch. Their raw is retained and
none of their ON/OFF differences is used here. A preliminary derived field
in that second session called RPC-to-app time a queue; the corrected v2
derived result explicitly supersedes that label. Neither session is
confirmatory evidence.

**Full transaction queue residence remains `null`**: the local P2P send
handoff is not actual remote delivery or a measured P2P egress queue;
Comet→app socket transfer/dispatch and app→Comet response transit are
bounded intervals but not decomposed into queue versus service; proposal
and consensus scheduling queues are not individually instrumented. The
same-host monotonic clock permits interval subtraction but does not remove
timestamp-placement or tracer-overhead uncertainty. One ON/OFF pair cannot
provide a precise overhead bound. The supported interval timeline is useful
for mechanism diagnosis; it does not satisfy the master checklist's **full
queue observation** claim. Closing X07 requires validated additional queue
markers and repeated ON/OFF QA with an uncertainty bound, without changing
Comet consensus or S06 caps.
