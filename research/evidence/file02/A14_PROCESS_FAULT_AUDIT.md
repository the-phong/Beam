# A14 — cross-validator process-fault diagnostic

**Disposition: PASS for the locked single-host process-fault diagnostic.**
This is not a multi-host fault-isolation claim. The preregistered plan and
harness are at source commit `d2c3629`; four balanced control/fault pairs
ran on eight fresh four-validator loopback-RPC networks. Private session:
`research/artifacts/file02/a14-20261113T191055Z-3f26be2c`.
Immutable raw manifest SHA-256:
`4fb340c2d9447a7925bd8dea4869b7d689df56599f445f1d2e014f1b22bd9f66`.
`python3 research/tools/file02_a14.py verify <session>` returned
`A14_VERIFY=OK qc=PASS`; integration exited 0, 8/8 arms passed QC.

Every arm offered the **same** 25 distinct valid H transactions: one warm-up
and six/s for four seconds, 15,408 B/s, well inside the unchanged caps.
Validator-0 loopback RPC was the sender. The fault arm suspended only
validator-3's CometBFT process with SIGSTOP at 2,001–2,002 ms after load
process launch and resumed it with SIGCONT 1,500–1,501 ms later; control
arms received no signal. All fault signal calls succeeded. The application
instances were in one test process, and all validators shared one host.

| Paired block | Fault-minus-control local lag, ms | Validator-3 RPC during fault | Healthy RPC height progress during snapshot |
|---|---:|---|---|
| 0 | −3.86 | unavailable | 3→4 |
| 1 | +12.23 | unavailable | 3→3 |
| 2 | +12.88 | unavailable | 3→4 |
| 3 | +328.86 | unavailable | 3→4 |

The locked primary metric is the fault-minus-control difference in
validator-3 minus healthy-validator median first **block-index observation
upper-bound** latency. Its four-block mean was +87.53 ms, percentile
bootstrap 95% CI [+0.33, +249.71] ms. Block 3 contributes most of the
effect, block 0 is negative, and four pairs give weak precision. These
timestamps are polling upper bounds, **not exact commit or finality times**.
The private per-run CSV/JSON contain each validator's median, all-four
median, per-transaction observation times and signal/height traces.

During the pause, validator-3 RPC did not answer in all fault arms. Three
of four fault-arm snapshots show healthy nodes progressing by one block;
the fourth shows no height increase during the sampled interval. After
resume all eight runs reached a common height at least 8, and all four
validators agreed on BlockID/AppHash/account state. Each run committed
25/25 valid transactions exactly once, with no invalid acceptance,
recorder drop, safety abort or failed/replaced pair. The fault-trace
`after_heights` snapshot for one arm was at height 7 immediately after the
load process exited; the subsequent oracle observed all validators at
height 8. The two observation times must not be conflated.

This establishes that a short local process pause can be recorded and
separated from eventual chain-wide state agreement in this testbed. It
does not establish hardware fault isolation, distributed network behavior,
precise cross-host latency, sustained throughput loss, a knee, or DoS.
