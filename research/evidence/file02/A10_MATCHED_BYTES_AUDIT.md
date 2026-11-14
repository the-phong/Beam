# A10 — matched offered-byte secondary audit

Date: 2026-11-14. The [analysis plan](../../protocol/FILE02_A10_LOCKED_ANALYSIS.md)
was committed at `f7fdced` before this synthesis; no new network was run.
Private session: `research/artifacts/file02/a10-20261113T184253Z-19389275`.
Plan SHA-256:
`1c5fadf8f54548b3239a81c79f44b10e31bfa6405d3ab5d4761a3da52d64f4e6`;
immutable input-manifest SHA-256:
`1d772ef47ea56a845e6c8af6b5b7d1f4a02de6eebcde8ded6b00ab6131604d02`.

Independent upstream raw and derived verification passed for A01–A05/A08.
All **72 pairs / 144 arms** offered 20 tx/s, exactly 2,568 bytes/tx and
51,360 B/s in every steady one-second window. The benign/attack pair also
shared six fixed legitimate H tx/s with identical bytes, hashes, genesis
and offsets. Equal tx/s is an additional control; A10 does not require the
two arm tx rates to differ. All six original primary effects were **0.000**
with original 95% CIs **[0.000, 0.000]** at the 3,000-ms endpoint. The
analysis does not pool dissimilar attack classes into a new CI.

| Measurement per case arm | Benign | Attack |
| --- | ---: | ---: |
| Offered wire bytes/s | 51,360 | 51,360 |
| Valid exact-once commits | 241 | 73 |
| Invalid admissions rejected | 0 | 168 |
| Validator-0 relay-send envelope bytes, mean run delta | 1,853,280 | 555,984 |

The admission and relay counts were the same across these six case types;
sampled mempool peak medians varied by case and remain in the private
per-run/aggregate files. Equal offered bytes do not imply equal **accepted**
or relayed bytes: invalid transactions were rejected before consensus
gossip, so lower attack-arm relay counts are expected. The original
CheckTx stage-service differences (structural ≈7 µs, bad ML ≈391 µs,
late-state ≈438 µs in their separate case means) persist when all wire
transactions are equal-sized; these processing differences cannot be
explained solely by wire size. There is **no positive service-degradation
effect at the registered endpoint to explain away as size-only**.

Relay envelope bytes are not exact propagation latency. Whole-process CPU
is not per-class CPU/tx, CheckTx socket wait is not a full transaction
queue, and eventual commit count is not sustainable throughput. Classifier
FPR and pooled cross-case CI remain `null` with explicit reasons.

`python3 research/tools/file02_a09_a10.py run A10` and independent
`verify research/artifacts/file02/a10-20261113T184253Z-19389275` exited
0 with QC PASS, six input cases, 72 pairs and 144 arms. Private plan,
immutable input manifest, per-run JSON/CSV, QC, aggregate, figure status
and decision remain preserved. **A10 PASS for matched-byte secondary
comparison.** It does not establish DoS, knee, P2P-origin or multi-host
effects.
