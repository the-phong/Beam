# X07/X16 measurement rework — prospective source and analysis lock

Status: LOCKED FOR ENGINEERING QA ONLY. Owner request: 2026-11-16.
This amendment supplements the accepted X extension without changing RQ1,
RQ1b, the X10/X13 results, transaction validity, Comet consensus or S06 caps.
It does not retrospectively change any accepted endpoint.

## X07 transaction trajectory

The disposable, source-hash-checked Comet v0.38.26 overlay may emit a private,
bounded transaction-hash trace when explicitly enabled per validator. Capture
RPC entry/return, Comet mempool CheckTx entry and precheck outcome, ABCI socket
queue attempt/dequeue, CheckTx callback, accepted-mempool insertion/removal,
and successful P2P reactor send. The existing dedicated app process records
CheckTx entry, lock and result with the same SHA-256 transaction identifier.
No raw transaction, key, account or payload may be logged by the new tracer.

Durations derived from Comet markers must use its *per-process monotonic*
clock. Cross-process app/Comet UTC stamps are join keys/order evidence only;
do not subtract them as precise delays without measured clock uncertainty.
Classify each interval separately: RPC/mempool precheck, socket subqueue,
app lock/service, accepted-mempool residence, and successful P2P send. A
successful `peer.Send` reports only the local send handoff, not remote
delivery. Missing/repeated markers, unknown removal reason, partial files or
trace write errors invalidate the affected trajectory; they are never zero.
The tracer must be disabled for ordinary builds and ordinary experiments.

Engineering QA: compare trace ON/OFF fresh four-validator networks using the
same valid/invalid fixture and bounded RPC-origin schedule; check application
codes, exact-once valid commit, invalid non-commit, four-validator AppHash,
source hashes, trace coverage and recorder drops. Quantify ON/OFF process
CPU/latency with run as independent unit before making an overhead claim.
Only after QA may a separate source-locked confirmatory run use the tracer.

## X16 attribution

Measure dedicated app, Comet, producer and observer *process* CPU separately
with PID/start-time identity. Stage wall spans remain wall time. Any stage
CPU estimate requires a validated CPU-specific method (for example labelled
CPU sampling with adequate sample counts and ON/OFF perturbation QA); no
process CPU/request division or wall-to-CPU substitution is allowed. Pair
new B and I networks with fixed legitimate stream, matched offered 20 H-sized
tx/s and bytes/s, balanced order and fresh state. Use run/network as unit;
report per-arm values and paired run-level 95% CIs. Analyze accepted/rejected,
queue, relay, proposal/commit, generator/observer and app/Comet work without
post-hoc pooling of different attack classes.

The X16 bottleneck is LOCATED only if a concrete stage/queue/resource shows
repeatable accumulation or saturation within the safe range with valid
service evidence and an uncertainty bound. If none appears, report
`NOT_OBSERVED_WITHIN_SAFE_RANGE`; do not label 25 tx/s as capacity or claim
DoS. Correlation between B/I and downstream work is not an isolated causal
mediation estimate. Missing stage CPU or observer CPU remains `null` with a
reason. X07/X16 boxes are ticked only when the corresponding full checklist
requirements have independently verified evidence; this lock alone is not
completion.

Topology remains four validators on one physical host, loopback RPC origin
at validator 0. Keep max 25 tx/s, 65,536 B/s, 16 outstanding and 60 s/run,
plus all other existing S06 safety aborts. Raw and trace files stay private
and immutable after process shutdown; publish hashes, plans, derived results
and audit only. No D/ML, higher caps, multi-host or P2P-origin study is opened.
