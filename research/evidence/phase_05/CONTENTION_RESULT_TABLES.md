# Phase 05 matched-rate contention: raw-derived result tables

78 usable independent four-validator runs: 77 original plus one clock-repaired replacement. The original anomalous run remains in private raw evidence and is excluded from measurement aggregates. Every valid-bearing run confirmed 300/300 valid transactions; every invalid submission was rejected. Means below are means of **per-run** summaries. p99 is descriptive even with 300 samples/run. CPU is summed process CPU as a percentage of one logical core across four Pluto/Comet processes; RSS is the sum of sampled process peaks. Client admission and block-observation inclusion are distinct timings.

Tested source trees: `29c16ad5f65c08a8922fcdba8448c6e84c0e4ea7, dfb1d32a821736ad029a7e77790577003186502f`. Public corpus SHA-256: `459148e8d350cecd1509130d88de909b9ce37c2420ef2999d224db9265ec1646`.

## Primary: fixed 1.5 legitimate tx/s

| Profile | Invalid % | Runs | Target valid / invalid tx/s | Achieved valid / invalid tx/s | Admission p50 / p95 / p99 (µs) | Inclusion p50 / p95 / p99 (ms) | Goodput tx/s | p99 amplification | Goodput degradation | CPU % | Sampled RSS MiB |
|---|---:|---:|---|---|---|---|---:|---:|---:|---:|---:|
| B0 | 0 | 5 | 1.5 / 0.0 | 1.50 / 0.00 | 858.20 / 1418.30 / 2663.20 | 1139.61 / 1705.94 / 1781.31 | 1.497 | 1.000× | 0.00% | 0.88 | 114.4 |
| B0 | 25 | 5 | 1.5 / 0.5 | 1.50 / 0.50 | 851.04 / 1320.00 / 2244.56 | 1131.42 / 1652.66 / 1799.09 | 1.497 | 1.012× | -0.00% | 0.86 | 114.6 |
| B0 | 50 | 5 | 1.5 / 1.5 | 1.50 / 1.50 | 862.75 / 1432.61 / 2107.58 | 1149.56 / 1671.89 / 1795.03 | 1.499 | 1.010× | -0.15% | 0.91 | 114.8 |
| B0 | 75 | 5 | 1.5 / 4.5 | 1.50 / 4.50 | 865.01 / 1437.13 / 2579.25 | 1128.24 / 1674.91 / 1771.83 | 1.497 | 0.997× | -0.03% | 1.00 | 114.2 |
| B0 | 90 | 5 | 1.5 / 13.5 | 1.50 / 13.50 | 882.63 / 1520.19 / 2606.64 | 1164.38 / 1668.56 / 1794.49 | 1.495 | 1.008× | 0.12% | 1.26 | 114.2 |
| B1 | 0 | 5 | 1.5 / 0.0 | 1.50 / 0.00 | 1118.20 / 1772.26 / 3351.07 | 1159.84 / 1728.18 / 1838.54 | 1.497 | 1.000× | 0.00% | 2.82 | 121.6 |
| B1 | 25 | 5 | 1.5 / 0.5 | 1.50 / 0.50 | 1149.49 / 1844.59 / 3680.71 | 1165.08 / 1631.99 / 1829.08 | 1.497 | 0.995× | -0.05% | 2.79 | 121.6 |
| B1 | 50 | 5 | 1.5 / 1.5 | 1.50 / 1.50 | 1135.00 / 1842.04 / 3655.52 | 1158.29 / 1671.86 / 1841.33 | 1.496 | 1.001× | 0.04% | 2.86 | 121.9 |
| B1 | 75 | 5 | 1.5 / 4.5 | 1.50 / 4.50 | 1113.05 / 1807.51 / 4057.25 | 1121.65 / 1624.92 / 1744.92 | 1.498 | 0.951× | -0.09% | 2.97 | 121.6 |
| B1 | 90 | 5 | 1.5 / 13.5 | 1.50 / 13.50 | 1117.12 / 1781.70 / 2903.93 | 1134.83 / 1678.18 / 1811.91 | 1.495 | 0.986× | 0.13% | 3.33 | 121.7 |

Uncertainty intervals and each validator's CPU/RSS are in `CONTENTION_AGGREGATE.json` and `CONTENTION_PER_RUN.csv`.

## Secondary: fixed 4 legitimate tx/s

| Profile | Invalid % | Runs | Target valid / invalid tx/s | Achieved valid / invalid tx/s | Admission p50 / p95 / p99 (µs) | Inclusion p50 / p95 / p99 (ms) | Goodput tx/s | p99 amplification | Goodput degradation | CPU % | Sampled RSS MiB |
|---|---:|---:|---|---|---|---|---:|---:|---:|---:|---:|
| B0 | 0 | 3 | 4.0 / 0.0 | 4.00 / 0.00 | 833.58 / 1624.02 / 2423.97 | 1150.70 / 1767.68 / 1851.16 | 3.960 | 1.000× | 0.00% | 1.75 | 114.3 |
| B0 | 50 | 3 | 4.0 / 4.0 | 4.00 / 4.00 | 864.27 / 1677.18 / 2440.93 | 1151.66 / 1801.32 / 1885.98 | 3.960 | 1.019× | -0.01% | 1.91 | 114.8 |
| B0 | 75 | 3 | 4.0 / 12.0 | 4.00 / 12.00 | 853.01 / 1489.25 / 3083.82 | 1168.39 / 1784.84 / 1901.53 | 3.942 | 1.027× | 0.46% | 2.18 | 114.8 |
| B1 | 0 | 3 | 4.0 / 0.0 | 4.00 / 0.00 | 1154.85 / 2412.18 / 3548.30 | 1152.19 / 1768.78 / 1868.14 | 3.960 | 1.000× | 0.00% | 4.44 | 120.0 |
| B1 | 50 | 3 | 4.0 / 4.0 | 4.00 / 4.00 | 1117.67 / 1732.88 / 2942.37 | 1176.79 / 1801.21 / 1885.35 | 3.940 | 1.009× | 0.52% | 4.62 | 121.3 |
| B1 | 75 | 3 | 4.0 / 12.0 | 4.00 / 12.00 | 1133.66 / 1902.05 / 3649.18 | 1151.81 / 1753.42 / 1869.37 | 3.953 | 1.001× | 0.19% | 4.93 | 121.3 |

Uncertainty intervals and each validator's CPU/RSS are in `CONTENTION_AGGREGATE.json` and `CONTENTION_PER_RUN.csv`.

## Invalid-only and rejection classes

| Profile | Runs | Invalid offered / achieved tx/s | Invalid rejected per run | CPU % | Sampled RSS MiB | Valid inclusion latency |
|---|---:|---|---:|---:|---:|---|
| B0 | 5 | 15 / 15.00 | 3000 | 0.77 | 103.9 | UNAVAILABLE (zero valid submitted) |
| B1 | 5 | 15 / 15.00 | 3000 | 2.52 | 112.8 | UNAVAILABLE (zero valid submitted) |

| Profile | Case ID | Rejected samples per run | Mean of per-run client rejection p50 (µs) |
|---|---|---:|---:|
| B0 | A03 | 600–600 | 863.53 |
| B0 | P02 | 600–600 | 839.88 |
| B0 | P03 | 1200–1200 | 845.86 |
| B0 | S04 | 600–600 | 679.74 |
| B1 | A03 | 600–600 | 1089.67 |
| B1 | I01 | 600–600 | 843.37 |
| B1 | P03 | 600–600 | 1015.47 |
| B1 | P04 | 600–600 | 1057.19 |
| B1 | S04 | 600–600 | 750.74 |

Structural S04 is constructible by capability A. Proof mutation P04/P03 needs an observed signed transaction (B); fully signed A03 requires a registered malicious account (C) or a prior state change. Repeated identical public bytes may interact with mempool caching, and client RPC cost includes transport and scheduling; direct Go microbenchmarks isolate application verification more closely. The B0 mixed class repeats a classical-proof slot where B1 uses binding.

## Interpretation boundary

At 90% invalid, the means of paired within-profile p99 amplification are B0 1.008× and B1 0.986×. These bounded single-host rates did not locate a saturation threshold or establish DoS resistance. The 2× p99 and 20% goodput screening signals are provisional research heuristics, not standards.

The superseded B0 secondary baseline run `034-secondary-b0-p0-r2` had a wall-clock discontinuity: its outer duration was 2463.8 s while Go monotonic test duration was 83.85 s. It remains hashed in the raw ledger. The replacement used monotonic duration and was measured in a fresh network under the same workload configuration.
