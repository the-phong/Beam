# X05 dedicated ABCI process QA plan

Status: locked before the evidence run. This is instrumentation validation,
not an X10 service contrast or a confirmatory RQ1/RQ1b result.

* Topology: four CometBFT v0.38.26 validator processes and four dedicated
  `profile-app` processes on one physical host; all ABCI/RPC/P2P listeners are
  loopback. The integration harness is a ninth process. Each app uses its own
  accepted controlled-profile database and the same fixture genesis and H
  validation path as the in-process harness.
* Workload: one pre-signed valid H transaction submitted to validator-0 RPC.
  No open-loop stream, no adversarial traffic, no change to S06 limits.
* Required checks: four distinct app PIDs different from harness PID; all
  four resource series bind to their PID and contain at least two samples;
  positive measured CPU delta, RSS peak, available thread count and GC
  counters; one validator-0 CheckTx event; valid CheckTx and DeliverTx;
  exact transaction in a shared block height and equal block ID/AppHash at
  all four validators. A failure is recorded as QA failure, not dropped.
* Resource semantics: `getrusage(RUSAGE_SELF)` user/system CPU are process
  cumulative nanoseconds; their difference across samples is a process
  interval, never crypto-stage CPU or CPU/request. Linux `ru_maxrss` is a
  process peak in KiB converted to bytes. Go heap/GC and `/proc/self/status`
  thread count are process snapshots. Reporter cadence is one second, with
  an initial and shutdown sample. Measurement overhead needs X09 ON/OFF QA.
* Evidence: the test writes private runtime homes and a keyless summary to
  an ignored `research/artifacts/file02/<session>/raw` directory. After the
  test exits and all processes stop, hash and freeze the full raw tree.
  Public report names only the manifest digest and aggregate process
  measurements; private keys, homes and unsanitized logs stay ignored.

Run command: `FILE02_X05_ROOT=<absolute ignored session/raw> go test -tags
'integration fourval' ./test/integration -run
'^TestFile02X05DedicatedAppProcesses$' -count=1 -timeout 300s -v`.
