# Phase 05 raw-derived result tables

Each network row averages independent run summaries; it does not pool requests as independent trials.
`admission` is client RPC wall time. `confirmed/s` counts only valid included transactions.
`CPU s` is sampled process user+system time for all Pluto/Comet processes; it is not wall time.

## Primary network matrix

| Profile | Invalid target | Offered tx/s | Runs | Admission p50 mean (µs) | Admission p95 mean (µs) | p99 | 95% run bootstrap interval for p50 (µs) | Valid inclusion p50 mean (ms) | Valid confirmed/s mean | CPU s mean | Four-process RSS sum (MiB) |
|---|---:|---:|---:|---:|---:|---|---|---:|---:|---:|---:|
| B0 | 0% | 2 | 5 | 787.466 | 6263.753 | INSUFFICIENT SAMPLES | [777.476, 795.031] | 1031.006 | 1.911 | 0.114 | 109.223 |
| B0 | 0% | 8 | 5 | 826.490 | 6594.210 | INSUFFICIENT SAMPLES | [792.153, 845.728] | 1024.281 | 5.417 | 0.098 | 109.998 |
| B0 | 50% | 2 | 5 | 817.725 | 7151.421 | INSUFFICIENT SAMPLES | [767.368, 870.425] | 1000.700 | 0.995 | 0.092 | 108.870 |
| B0 | 50% | 8 | 5 | 806.426 | 6719.197 | INSUFFICIENT SAMPLES | [789.576, 822.428] | 963.470 | 2.707 | 0.084 | 108.812 |
| B0 | 90% | 2 | 5 | 878.254 | 6831.002 | INSUFFICIENT SAMPLES | [828.332, 931.418] | 1440.720 | 0.178 | 0.088 | 109.078 |
| B0 | 90% | 8 | 5 | 949.504 | 6872.579 | INSUFFICIENT SAMPLES | [857.416, 1070.384] | 1100.560 | 0.559 | 0.066 | 108.491 |
| B0 | 100% | 2 | 5 | 899.246 | 7127.936 | INSUFFICIENT SAMPLES | [803.677, 1059.660] | UNAVAILABLE | 0.000 | 0.042 | 97.102 |
| B0 | 100% | 8 | 5 | 882.744 | 6981.383 | INSUFFICIENT SAMPLES | [811.515, 1011.942] | UNAVAILABLE | 0.000 | 0.020 | 96.812 |
| B1 | 0% | 2 | 5 | 1085.296 | 6626.221 | INSUFFICIENT SAMPLES | [1017.379, 1193.943] | 1030.672 | 1.942 | 0.258 | 117.752 |
| B1 | 0% | 8 | 5 | 1047.609 | 6576.465 | INSUFFICIENT SAMPLES | [1025.588, 1063.857] | 1134.634 | 5.209 | 0.190 | 117.670 |
| B1 | 50% | 2 | 5 | 1062.618 | 6983.454 | INSUFFICIENT SAMPLES | [1046.827, 1078.409] | 1170.771 | 0.995 | 0.184 | 117.853 |
| B1 | 50% | 8 | 5 | 1025.313 | 6276.779 | INSUFFICIENT SAMPLES | [990.938, 1059.126] | 993.402 | 2.686 | 0.148 | 117.738 |
| B1 | 90% | 2 | 5 | 1049.792 | 6895.432 | INSUFFICIENT SAMPLES | [1005.552, 1095.977] | 1361.150 | 0.178 | 0.150 | 116.540 |
| B1 | 90% | 8 | 5 | 1110.801 | 6579.129 | INSUFFICIENT SAMPLES | [1079.720, 1143.154] | 1121.581 | 0.559 | 0.114 | 115.118 |
| B1 | 100% | 2 | 5 | 1090.698 | 6838.583 | INSUFFICIENT SAMPLES | [1035.547, 1167.767] | UNAVAILABLE | 0.000 | 0.122 | 109.566 |
| B1 | 100% | 8 | 5 | 1137.624 | 7348.307 | INSUFFICIENT SAMPLES | [1056.587, 1218.662] | UNAVAILABLE | 0.000 | 0.042 | 109.273 |

## Secondary moderate-rate conditions

| Profile | Invalid target | Offered tx/s | Runs | Admission p50 mean (µs) | Valid confirmed/s mean | CPU s mean |
|---|---:|---:|---:|---:|---:|---:|
| B0 | 25% | 4 | 3 | 852.848 | 2.477 | 0.100 |
| B0 | 75% | 4 | 3 | 804.412 | 0.857 | 0.073 |
| B1 | 25% | 4 | 3 | 1139.208 | 2.454 | 0.180 |
| B1 | 75% | 4 | 3 | 1042.719 | 0.811 | 0.143 |

## Primary traffic rates

Rates are means of independent run rates over each run's observed window; offered pacing targets are listed above.

| Profile | Invalid target | Target tx/s | Incoming tx/s | Incoming KiB/s | Accepted admission tx/s | Rejected tx/s | Confirmed valid tx/s | Client errors/timeouts |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| B0 | 0% | 2 | 1.911 | 0.289 | 1.911 | 0.000 | 1.911 | 0/0 |
| B0 | 0% | 8 | 5.417 | 0.820 | 5.417 | 0.000 | 5.417 | 0/0 |
| B0 | 50% | 2 | 1.991 | 0.301 | 0.995 | 0.995 | 0.995 | 0/0 |
| B0 | 50% | 8 | 5.415 | 0.820 | 2.707 | 2.707 | 2.707 | 0/0 |
| B0 | 90% | 2 | 2.132 | 0.323 | 0.178 | 1.954 | 0.178 | 0/0 |
| B0 | 90% | 8 | 6.709 | 1.016 | 0.559 | 6.150 | 0.559 | 0/0 |
| B0 | 100% | 2 | 2.181 | 0.330 | 0.000 | 2.181 | 0.000 | 0/0 |
| B0 | 100% | 8 | 8.716 | 1.319 | 0.000 | 8.716 | 0.000 | 0/0 |
| B1 | 0% | 2 | 1.942 | 7.408 | 1.942 | 0.000 | 1.942 | 0/0 |
| B1 | 0% | 8 | 5.209 | 19.876 | 5.209 | 0.000 | 5.209 | 0/0 |
| B1 | 50% | 2 | 1.991 | 7.595 | 0.995 | 0.995 | 0.995 | 0/0 |
| B1 | 50% | 8 | 5.371 | 20.493 | 2.686 | 2.686 | 2.686 | 0/0 |
| B1 | 90% | 2 | 2.132 | 8.134 | 0.178 | 1.954 | 0.178 | 0/0 |
| B1 | 90% | 8 | 6.706 | 25.587 | 0.559 | 6.147 | 0.559 | 0/0 |
| B1 | 100% | 2 | 2.181 | 8.322 | 0.000 | 2.181 | 0.000 | 0/0 |
| B1 | 100% | 8 | 8.716 | 33.254 | 0.000 | 8.716 | 0.000 | 0/0 |

## Focused invalid classes

| Profile | Invalid class | Runs | Admission median mean (µs) | CPU s mean |
|---|---|---:|---:|---:|
| B0 | classical | 3 | 856.430 | 0.027 |
| B0 | mixed | 3 | 815.607 | 0.017 |
| B0 | state | 3 | 853.646 | 0.023 |
| B0 | structural | 3 | 987.288 | 0.007 |
| B1 | binding | 3 | 933.671 | 0.060 |
| B1 | classical | 3 | 998.212 | 0.057 |
| B1 | mixed | 3 | 1013.330 | 0.067 |
| B1 | ml | 3 | 1066.536 | 0.057 |
| B1 | state | 3 | 1085.170 | 0.067 |
| B1 | structural | 3 | 777.742 | 0.050 |

## Direct and callback microbenchmarks

| Benchmark | Command repetitions | Median ns/op | Median B/op | Median allocs/op |
|---|---:|---:|---:|---:|
| `BenchmarkPhase05Application/b0_CheckTx_early` | 5 | 101.3 | 128 | 1 |
| `BenchmarkPhase05Application/b0_CheckTx_late` | 5 | 120831.0 | 1344 | 24 |
| `BenchmarkPhase05Application/b0_CheckTx_valid` | 5 | 121838.0 | 12096 | 28 |
| `BenchmarkPhase05Application/b0_PrepareProposal` | 5 | 122209.0 | 12016 | 29 |
| `BenchmarkPhase05Application/b0_ProcessProposal` | 5 | 121677.0 | 11968 | 28 |
| `BenchmarkPhase05Application/b1_CheckTx_early` | 5 | 219.4 | 128 | 1 |
| `BenchmarkPhase05Application/b1_CheckTx_late_ml` | 5 | 229656.0 | 74344 | 31 |
| `BenchmarkPhase05Application/b1_CheckTx_valid` | 5 | 252505.0 | 98985 | 43 |
| `BenchmarkPhase05Application/b1_PrepareProposal` | 5 | 267129.0 | 98905 | 44 |
| `BenchmarkPhase05Application/b1_ProcessProposal` | 5 | 255980.0 | 98865 | 43 |
| `BenchmarkPhase05Crypto/b0_full_validation` | 5 | 125395.0 | 11968 | 27 |
| `BenchmarkPhase05Crypto/b0_secp_validate` | 5 | 109648.0 | 1056 | 20 |
| `BenchmarkPhase05Crypto/b1_chain_profile` | 5 | 6.6 | 0 | 0 |
| `BenchmarkPhase05Crypto/b1_decode_digest` | 5 | 2323.0 | 5648 | 3 |
| `BenchmarkPhase05Crypto/b1_full_validation` | 5 | 251793.0 | 98857 | 42 |
| `BenchmarkPhase05Crypto/b1_key_binding` | 5 | 42132.0 | 22608 | 3 |
| `BenchmarkPhase05Crypto/b1_ml_verify` | 5 | 49800.0 | 23034 | 7 |
| `BenchmarkPhase05Crypto/b1_parse_only` | 5 | 396.5 | 16 | 1 |
| `BenchmarkPhase05Crypto/b1_secp_verify` | 5 | 112465.0 | 768 | 17 |
| `BenchmarkPhase05Crypto/b1_state_rules_after_auth` | 5 | 5983.0 | 10752 | 4 |
| `BenchmarkPhase05FinalizeCommit/b0_Commit` | 5 | 2210861.0 | 11626 | 5 |
| `BenchmarkPhase05FinalizeCommit/b0_FinalizeBlock` | 5 | 27184.0 | 11007 | 7 |
| `BenchmarkPhase05FinalizeCommit/b1_Commit` | 5 | 4904692.0 | 1681973 | 211 |
| `BenchmarkPhase05FinalizeCommit/b1_FinalizeBlock` | 5 | 35809.0 | 24557 | 12 |
| `BenchmarkPhase05InvalidClasses/b0_A03` | 5 | 124632.0 | 1216 | 23 |
| `BenchmarkPhase05InvalidClasses/b0_P03` | 5 | 113350.0 | 1056 | 20 |
| `BenchmarkPhase05InvalidClasses/b0_S01` | 5 | 42.6 | 0 | 0 |
| `BenchmarkPhase05InvalidClasses/b1_A03` | 5 | 244010.0 | 74664 | 33 |
| `BenchmarkPhase05InvalidClasses/b1_I01` | 5 | 72553.0 | 45232 | 7 |
| `BenchmarkPhase05InvalidClasses/b1_P03` | 5 | 206784.0 | 51632 | 26 |
| `BenchmarkPhase05InvalidClasses/b1_P04` | 5 | 237623.0 | 74216 | 30 |
| `BenchmarkPhase05InvalidClasses/b1_S01` | 5 | 135.3 | 0 | 0 |
| `BenchmarkPhase05RecorderOverhead/disabled` | 5 | 266470.0 | 98857 | 42 |
| `BenchmarkPhase05RecorderOverhead/enabled` | 5 | 268746.0 | 98857 | 42 |

## Sample sufficiency and raw link

Complete independent network runs: **122**. Runs with sufficient per-run admission samples for p99: **0**.
For short 12-request runs, p99 is `INSUFFICIENT SAMPLES`. Per-run p50/p90/p95, counts, standard deviation, inclusion and resource observations are in `per_run.json`.
The full raw records and execution logs remain local; `RAW_SHA256.txt` identifies them without publishing their contents.
