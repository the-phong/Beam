# X16 — bounded mechanism and resource decomposition

Historical X10-only disposition: **PARTIAL**. `X16_RESOURCE_DECOMPOSITION.json` independently
rederives the 72 QC-passing X10 runs from frozen raw manifest
`a97e1665deca67e747f30a1c6b03268140b716886e42fb24d9c1e1a6512233c8`.
It retains per-run values, paired 24-block CIs and 19-sample validator-0
mempool/Comet CPU curves by arm. `python3
research/tools/file02_x16_resources.py check` returns `X16_CHECK=OK`.
All comparisons below are **B−I** at matched offered 20 H-sized tx/s,
51,360 B/s and identical six fixed H/s. B has 14 additional **valid** H/s;
I has 14 additional late-bad-ML H/s. L has the six fixed H/s only.

| Per-run mean across 24 networks/arm | L | B | I | Paired B−I; 95% block CI |
| --- | ---: | ---: | ---: | ---: |
| Four-app process CPU interval sum, s | 1.598 | 2.415 | 1.721 | +0.694 [+0.671,+0.720] |
| Four-Comet process CPU interval sum, s | 5.478 | 5.870 | 5.465 | +0.404 [+0.293,+0.551] |
| Generator process CPU, s | 0.078 | 0.177 | 0.176 | +0.0008 [−0.0036,+0.0052] |
| Validator-0 relay-send envelope counter delta, B | 555,984 | 1,853,280 | 555,984 | +1,297,296 [1,297,296,1,297,296] |
| Validator-0 peak accepted mempool bytes, B | 24,931 | 86,884 | 24,824 | +62,060 [+59,492,+64,628] |
| Validator-0 CheckTx socket queue cumulative delta, s | 0.0035 | 0.0131 | 0.0102 | +0.0029 [+0.0024,+0.0033] |

The B side's **valid** extra transactions enter accepted mempool,
relay/proposal/commit; the I side's **invalid** extra transactions reach
late ML CheckTx and are rejected before normal accepted gossip/commit.
The paired application/Comet CPU, accepted-mempool and relay-envelope
differences are **consistent with** more downstream work on the valid B
side. They are total workload effects, not a causal decomposition of
individual stages. Relay counters include envelope traffic rather than
transaction-only wire bytes. Comet CPU uses `/proc` observer snapshots;
app CPU uses dedicated-process resource samples. Interval boundaries may
differ slightly and include idle/runtime/GC work. Generator CPU is measured
separately and its B−I interval includes zero. Observer self CPU in X10 is
`null/UNAVAILABLE_OBSERVER_SELF_CPU_IN_X10`; the historical T05 ON/OFF study
shows correctness at 2 tx/s but cannot bound observer cost at 20 tx/s.

All 241 valid B transactions per run committed exactly once; in each I run,
73 fixed valid committed and 168 bad-ML rejected with no invalid commit.
The observed invalid I application CheckTx wall mean is about 363.9 µs/tx.
Historical A01/A03/A04/A05/A08 class walls and the separate RQ1 valid
H−CP +81.47 µs/tx answer different cost questions; their networks are not
pooled. Isolated X06 primitive CPU profiling showed sampled secp and ML
work, but **network per-stage CPU remains `null`**. CheckTx stage spans
measure wall time, not CPU. The X07 socket queue delta is **only the
Comet-to-app socket subqueue**, not whole ingress→commit residence.

The valid B side used more process CPU and downstream work yet its fixed-H
observation upper bound was lower than I by 60.03 ms in X10. Shared-host
scheduling, block timing, app/Comet workload allocation and scanner phase
could contribute; no mediator is individually randomized or timed end to
end. Consequently the location of a bottleneck, true stage CPU shares,
observer process cost, transaction-specific relay work and full queue
residence are **UNAVAILABLE**. No sustained degradation, capacity knee,
DoS or multi-host mechanism is established. That historical analysis alone
left X16 PARTIAL; the prospective rework below supplies the remaining
stage/observer measurements and a separate overhead QA.

## Prospective X16 rework, 2026-11-16

The new source lock `47c44cc2cf9f10d6704de253d6133a61af920294` added a
Linux thread-clock probe at CheckTx stage boundaries and observer self-CPU
sampling, while retaining separate app and Comet PIDs. The locked B/I
contrast ran 48 fresh networks in 24 balanced blocks; every run and pair
passed independent provenance, raw manifest, trace, oracle, PID and drop QC.
Private raw manifest SHA-256:
`53848abd502b0d922277852df1f59ace910238270777943bcb40335dd329be09`.
`python3 research/tools/file02_x16_results.py verify
research/artifacts/file02/x16-20261116T084807Z-7988d71d` returns
`X16_VERIFY=OK QC=PASS runs=48 pairs=24`. Per-run JSON/CSV, QC, aggregate,
decision and two figures are private beside that raw; the checker reproduces
them byte for byte. A publication-safe copy of the aggregate and two figures
is indexed in `X16_CONFIRMATORY_PUBLIC_MANIFEST.json`; it contains no private
transaction bytes, account material, PIDs or run homes. This rework does not replace the historical X10/RQ1b
source or its 3-second endpoint.

| New paired B−I measure | B mean | I mean | Difference; 95% block CI |
| --- | ---: | ---: | ---: |
| Four-app CPU, s/run | 2.563 | 1.859 | +0.704 [+0.647,+0.755] |
| Four-Comet CPU, s/run | 7.128 | 6.937 | +0.191 [−0.130,+0.457] |
| Four-observer CPU, s/run | 0.118 | 0.114 | +0.0046 [−0.0058,+0.0150] |
| Generator CPU, s/run | 0.188 | 0.192 | −0.0040 [−0.0104,+0.0024] |
| Validator-0 relay envelope, B/run | 1,853,280 | 555,984 | +1,297,296 [1,297,296,1,297,296] |
| Validator-0 peak accepted mempool, B | 88,810 | 27,499 | +61,311 [+59,385,+63,130] |
| Validator-0 ABCI socket subqueue cumulative, s/run | 0.0184 | 0.0119 | +0.0065 [+0.0032,+0.0126] |
| Fixed-valid all-four observation upper bound, ms | 1,123.53 | 1,168.41 | −44.88 [−68.69,−23.04] |
| Fixed-valid timely by 3 s | 1.000 | 1.000 | 0 [0,0] |

The newly instrumented validator-0 CheckTx thread CPU means were, for B/I
respectively, structural 2.334/2.315 µs, binding 3.811/4.003 µs, digest
2.510/2.644 µs, secp 197.170/196.954 µs and ML 86.977/62.546 µs per
stage invocation. The state and state-copy stages execute only on valid
paths; their B/I means were 24.302/25.206 µs and 36.504/36.686 µs per
invocation. These values are **instrumented thread CPU**, not an unbiased
estimate of native per-stage crypto CPU. The B/I stage populations differ:
B includes 241 valid NEW transactions per run, I includes 73 valid and 168
late-bad-ML NEW transactions. The ML-stage difference therefore must not be
read as the CPU cost of one valid versus one invalid signature without a
matched per-transaction comparison. Raw stage wall spans remain distinct.

The B arm's additional valid transactions are accepted, relayed and
committed; I's invalid background reaches ML validation and is rejected.
The paired app CPU and downstream byte/occupancy differences are consistent
with that path difference, but do not isolate a numerical causal contribution
of relay or commit. Comet CPU and observer CPU CIs include zero. The fixed
valid observation interval is a 100-ms scanner upper bound, not exact commit
latency. All fixed valid transactions were observed within 3 seconds in
both arms; no sustained degradation or knee was established. The new
thread-clock ON/OFF overhead check is documented in
`X16_REWORK_AUDIT_20261117.md`: 16/16 runs and 8/8 pairs passed, and the
probe raised app CPU by +0.0489 s/run [0.0134,0.0823]. The X16 checklist
deliverable is technically complete for the bounded single-host RPC scope:
stage CPU is explicitly **instrumented**, observer CPU is measured, and
unperturbed stage CPU, exact stage/relay/commit contributions and an exact
bottleneck remain `null` with reasons. Independent Gate review is pending.
