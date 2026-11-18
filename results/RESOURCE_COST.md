# Resource cost

## Accepted RQ1 comparison

At 20 valid tx/s/profile on four Comet validators on one physical host, CP and H were both 2,568 bytes/transaction. The paired **H−CP application CheckTx service wall-time effect was +81.47 µs/tx**, 95% CI **[+75.16,+87.26] µs/tx**, from 12 matched blocks/48 fresh networks. This supports a positive service-cost difference in that stratum; it is not ML-DSA-only CPU time. The equal-byte-rate control at low rate found +102.13 µs/tx [66.67,148.84]. Direct ABCI C/CP/P/H valid CheckTx means were 155.05/157.09/76.80/201.36 µs; H−CP was +44.28 µs [40.15,48.51] there. Different settings and boundaries must not be pooled.

| Profile | Tested wire bytes | Proof requirement | Control role |
| --- | ---: | --- | --- |
| C | 148 | secp256k1 | Classical small wire |
| CP | 2,568 | secp256k1 plus canonical padding | Size-matched control for H |
| P | 2,504 | ML-DSA-44 | Post-quantum path; not H-sized |
| H | 2,568 | secp256k1 and ML-DSA-44 | Hybrid treatment |

R03 measured CP−C +2,420 bytes and +724 ns structural parse [635,834] ns, while its full direct CheckTx effect −1.38 µs [−7.93,+5.23] was inconclusive. R01 isolated primitive secp and ML verification means were 104,762 and 18,294 ns/op; those isolated operations are not the same as complete CheckTx. The P comparison is byte-confounded because P cannot be canonically padded to H size. See [RQ1 source report](../research/evidence/file02/RQ1_RESOURCE_REPORT.md) and [measurement definitions](../docs/METRICS.md).

## Rejection stage and downstream work

Separate A case networks measured structural-invalid CheckTx mean **7.25 µs/tx**, late bad-ML mean **391.31 µs/tx** and signed state-invalid mean **438.28 µs/tx**. They are different cases, not a paired CPU contrast. Accepted valid traffic enters mempool/relay/proposal/commit, while CheckTx-rejected traffic normally does not. In X16's new matched B/I study, B−I four-app process CPU was **+0.7035 s/run** [0.6466,0.7554] across 24 pairs; B sent 1,297,296 more validator-0 relay-envelope bytes/run and had 61,311 more peak accepted-mempool bytes. The four-Comet CPU interval included zero. Stage thread-clock values are probe-conditioned, and the probe itself raised app CPU by +0.0489 s/run [0.0134,0.0823]. Exact unperturbed stage CPU and relay/commit CPU shares remain null.

The earlier B0/B1 exploratory dataset and matched-rate contention result tables remain as public provenance under [`research/evidence/phase_05`](../research/evidence/phase_05/). They measured different applications, rates and boundaries; their CPU and RPC values do not replace the C/CP/P/H RQ1 contrast. The [provenance index](provenance/INDEX.md) names the locked sources and raw manifests. No natural capacity knee was observed through the 25-tx/s safety cap.
