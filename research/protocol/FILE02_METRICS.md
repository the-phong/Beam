# File 02 metric dictionary v1

`null` means unmeasured, unavailable or not applicable and requires an
`availability` explanation in the per-run result. Zero is a measured zero.
All rates use the steady monotonic interval and actual sends. Percentiles
require a count and censoring note. The
[S05 dictionary](METRICS_DICTIONARY.md) gives the implemented capture points.

| Group | Canonical fields and unit | Source / limit |
| --- | --- | --- |
| Workload | `offered_valid_tx_s`, `offered_invalid_tx_s`, `offered_tx_bytes_s`, `achieved_valid_tx_s`, `achieved_invalid_tx_s`, `achieved_tx_bytes_s`, `burstiness`, `distinct_tx_count` | Generator planned deadlines and actual socket writes; do not substitute RPC results. |
| Crypto | `rejection_stage`, `verify_secp_us`, `verify_mldsa_us`, `total_checktx_us`, `cpu_ns_per_tx`, `alloc_B_per_tx` | Application spans; CPU/alloc attribution requires direct instrumentation and otherwise is null. |
| Ingress | `ingress_tx_s`, `ingress_bytes_s`, `queue_depth`, `queue_wait_ms`, `ingress_source` | Separate RPC and decoded P2P; current socket queue sample is not total enqueue-to-app wait. |
| Mempool | `accepted_tx_count`, `accepted_bytes`, `full_count`, `full_bytes`, `rejected_by_reason`, `eviction_count` | Native accepted list; pre-admission rejection is not occupancy. Unsupported eviction counter is null. |
| Relay | `peer_rx_bytes_s`, `peer_tx_bytes_s`, `relay_queue`, `propagation_p50_ms`, `propagation_p95_ms`, `propagation_p99_ms` | Native P2P counters; propagation and relay queue currently null. |
| Consensus | `valid_included_count`, `legitimate_goodput_tx_s`, `inclusion_latency_p50_ms`, `inclusion_latency_p95_ms`, `inclusion_latency_p99_ms`, `finality_measure_defined`, `blocks`, `matched_AppHash` | Independent four-node oracle; poll observation gives only an upper latency bound, not exact inclusion percentiles. |
| Node | `CPU_total`, `CPU_app`, `CPU_crypto`, `RSS_peak`, `GC_pause`, `process_count`, `thread_count` | Process/host sampler; total or app process CPU is not crypto CPU. |
| Detection | `TP`, `FP`, `TN`, `FN`, `PR_AUC`, `recall_at_fixed_FPR`, `FP_per_hour`, `lead_time_ms`, `inference_p99_ms`, `detector_CPU`, `detector_RSS` | Future held-out analysis only; all null in current foundation. |
| Quality | `n_samples`, `n_runs`, `CI95`, `exclude_reason`, `observer_drop_count`, `achieved_rate_error` | QC record and independent-run aggregate; no CI from transaction samples alone. |

`research/tools/file02_common.py` exports these exact fields and rejects
non-null detection outputs in a common-foundation QA envelope. The current
S07 sample is not backfilled with values it never measured.
