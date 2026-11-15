# X16 — bounded mechanism and resource decomposition

Disposition: **PARTIAL**. `X16_RESOURCE_DECOMPOSITION.json` independently
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
DoS or multi-host mechanism is established. The `PARTIAL` box remains open.
