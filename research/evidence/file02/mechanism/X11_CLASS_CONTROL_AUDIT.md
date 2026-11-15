# X11 — early and late invalid controls

Status: **PASS within the historical single-host RPC-origin matched design**.
This is a frozen-evidence reanalysis of five separately locked A cases, not a
new confirmatory service experiment. The independent computation is
`research/tools/file02_x11_historical.py`; its machine-readable output is
`X11_CLASS_CONTRASTS.json`. Each original plan has 12 balanced fresh-network
pairs, six fixed valid H/s plus 14 H-sized background/s, 20 offered tx/s and
51,360 offered bytes/s. Fixed transaction hashes and genesis match within
each pair. All five historical raw verifiers and the new recomputation pass.

| Class | Case | Invalid CheckTx mean wall, µs | Exploratory attack−benign fixed-stream mean observation upper bound, ms; paired 95% CI |
| --- | --- | ---: | ---: |
| Early structural | A01 | 7.25 | +45.32 [20.97, 70.29] |
| Bad secp | A03 | 179.21 | +52.70 [25.15, 80.80] |
| Bad ML | A04 | 391.31 | +56.68 [37.43, 77.69] |
| State invalid | A05 | 438.28 | +55.83 [37.93, 72.44] |
| Mixed invalid | A08 | 287.28 | +80.08 [47.52, 111.69] |

The CheckTx numbers are application wall spans on different networks, not
paired per-stage CPU estimates. Each class has its own run-level bootstrap
CI in the JSON; classes are never pooled. The original preregistered 3-second
timely-fraction contrast remains **0 [0,0] in every class**. The continuous
observation estimates and CIs above are exploratory analyses of the old
100-ms block scanner. They are upper bounds on first all-four observation,
not exact commit times or a replacement for the original endpoint. No class
demonstrates a 3-second miss, sustained degradation, a knee, or DoS. Per-stage
CPU remains `null/UNAVAILABLE_STAGE_CPU_ATTRIBUTION`.

Recompute: `python3 research/tools/file02_x11_historical.py check`.
