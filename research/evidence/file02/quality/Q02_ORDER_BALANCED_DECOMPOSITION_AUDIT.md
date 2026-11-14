# Q02 — corrected, order-balanced cross-profile decomposition

**Disposition: PASS within the tested structural-invalid, four-validator single-host RPC stratum.** The corrected `schedule.v2` source was committed at `42d9ac9` before any v2 send. The private session `research/artifacts/file02/q02-20261114T044327Z-e7f236e2` has frozen raw manifest SHA-256 `58265e00a8c0fc08a37cb4e2935633d85304a1e70d32142df5d9eaa4d1180f3f`. The integration run, `qc_report.json` and independent `python3 research/tools/file02_q02.py verify ...` all PASS. The plan, raw, 96 per-run results, QC, 12 paired blocks, aggregate, figure status and decision stay in the ignored private artifact tree.

The v2 schedule rotated C/CP/P/H positions across 12 blocks and assigned **six benign-first and six attack-first pairs to each profile**. Each block used eight fresh networks with the same genesis/seed and fixed legitimate intents; within each profile pair the 73 fixed legitimate hashes and offsets matched. All 96 arms had 241 successful socket sends. Every benign arm had 241 exact-once valid commits; every attack arm had 73 exact-once valid commits and 168 structural-invalid rejects at the expected stage. The four validators agreed on block/state; there were no QC exclusions, safety aborts or missing first-all-four observations. Offered load was 20 tx/s per arm with six fixed legitimate tx/s. Within each profile pair encoded bytes/s matched; CP/H both used 51,360 B/s, while C and P used 2,960 and 50,080 B/s respectively.

| Contrast on fixed-legitimate fraction observed within 3,000 ms | Mean, proportion points | 95% block-bootstrap CI |
|---|---:|---:|
| C benign−attack | 0.000 | [0.000, 0.000] |
| CP benign−attack | 0.000 | [0.000, 0.000] |
| P benign−attack | 0.000 | [0.000, 0.000] |
| H benign−attack | 0.000 | [0.000, 0.000] |
| `(H_attack−H_benign)−(C_attack−C_benign)` | 0.000 | [0.000, 0.000] |

C→CP, CP→H and P→H contrasts are also 0.000 with [0.000, 0.000] CIs in both conditions. CI resampled 12 independent fixture blocks 20,000 times with the locked seed 20261114; transfers within a network were not treated as independent replicates. Every steady fixed transfer was observed within the 3-s deadline, so the binary endpoint is ceiling-limited. These degenerate intervals are **not equivalence evidence**. C/P cross-profile comparisons are confounded by encoded size; only CP/H match bytes. This is structural-invalid traffic that rejects before crypto, so it cannot identify ML-DSA-specific effects, other attack families, P2P origin, independent-host effects, a knee or DoS.

**Transparent rework history.** The first Q02 attempt `q02-20261114T024019Z-8ccd9928` was frozen as `INVALID_FOR_INFERENCE` after a pre-send stale-observer safety abort. The first complete session `q02-20261114T025508Z-a4b258cb` passed technical QC but was [withdrawn for profile-order confounding](Q02_IMPACT_DECOMPOSITION_AUDIT.md): `schedule.v1` put C/P always benign-first and CP/H always attack-first. Neither session is pooled with v2 or overwritten. The v2 session repeated the **entire** locked contrast and changed only arm order to the intended 6/6 balance, with unchanged workload, endpoint, CI, topology and safety caps.
