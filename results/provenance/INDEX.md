# Scientific provenance and verification

The values in `results/` summarize accepted case measurements; they are not newly estimated by the structural refactor. The source commits and raw-manifest hashes below identify the complete run packages. Private run directories and signed material remain Git-ignored. The public audits, aggregate JSON and figure manifests are retained in `research/evidence/file02/` because their original source-locked paths and hashes are part of the published provenance. Read [REPRODUCE](../../REPRODUCE.md) before claiming raw rederivation.

| Result | Source commit locked before run | QC unit and private raw manifest SHA-256 | Public source |
| --- | --- | --- | --- |
| R06 H−CP valid application CheckTx | `635dfd3` | 12 matched blocks, 48 networks; `a8d08f01e8a19bc80b568966fdc988f7e50c07a1fd8002840121eb384878dcb7` | [R06 audit](../../research/evidence/file02/R06_EQUAL_RATE_AUDIT.md) and [locked plan](../../research/protocol/FILE02_R06_LOCKED_PLAN.md) |
| X10 L/B/I service | `c3ce45b4adae9d6477ac2749469762468e38d938` | 24 blocks, 72 networks; `a97e1665deca67e747f30a1c6b03268140b716886e42fb24d9c1e1a6512233c8` | [X10 audit](../../research/evidence/file02/mechanism/X10_THREE_ARM_AUDIT.md) and [locked plan](../../research/protocol/FILE02_X10_THREE_ARM_LOCKED_PLAN.md) |
| X13 safe-range sensitivity | `2c2608d08d35fadfe48062b701293d4b4fb08a8f` | 12 blocks, 36 networks; `9eaa4d990f43e5c9d1c38d2ab4f968db3dc7a3c4e94a6a0a2a8d60ed4e0ab0e3` | [X13 audit](../../research/evidence/file02/mechanism/X13_SAFE_RANGE_SENSITIVITY_AUDIT.md) and [locked plan](../../research/protocol/FILE02_X13_SAFE_RANGE_LOCKED_PLAN.md) |
| X16 matched B/I resource contrast | `47c44cc2cf9f10d6704de253d6133a61af920294` | 24 pairs, 48 networks; `53848abd502b0d922277852df1f59ace910238270777943bcb40335dd329be09` | [X16 audit](../../research/evidence/file02/mechanism/X16_REWORK_AUDIT_20261117.md), [public manifest](../../research/evidence/file02/mechanism/X16_CONFIRMATORY_PUBLIC_MANIFEST.json) |
| X16 probe ON/OFF QA | `28fc05d68a9ae7b236dbc9004533c72e880e9909` | 8 pairs, 16 networks; `9cd3e1fd7250ab0edb4b4a1b7b335fba5ff3153256de583315c01f7434ffb860` | [X16 audit](../../research/evidence/file02/mechanism/X16_REWORK_AUDIT_20261117.md), [public manifest](../../research/evidence/file02/mechanism/X16_CONFIRMATORY_PUBLIC_MANIFEST.json) |
| X07 queue-path ON/OFF QA | `07dc11bb10dddda700b6f4c66f720fb2c7528bda` | 8 pairs, 16 networks; `8a2082525610b5615bed6753ef38add2a79ab41d368b459cbf55b8dd68099fc2` | [X07 audit](../../research/evidence/file02/mechanism/X07_QUEUE_V2_AUDIT_20261117.md), [public manifest](../../research/evidence/file02/mechanism/X07_QUEUE_V2_PUBLIC_MANIFEST.json) |

The earlier R06 attempt without a pre-send COMMON plan, the interrupted X13 attempt and the wrong-wire-size X16 overhead QA are excluded from these estimates and remain in private raw. Their hashes and reasons are in their case audits. Historical RQ1b 3-second paired effects and the Q07 correctness oracle are documented in the [service report](../SERVICE_IMPACT.md) and [Q07 audit](../../research/evidence/file02/quality/Q07_NO_INVALID_ACCEPTED_AUDIT.md).

## Public checks

From the repository root, run:

```bash
python3 experiments/validation/verify_public_artifacts.py
make corpus-s04-verify corpus-s06-verify benchmark-corpus benchmark-contention-corpus
```

The first command checks source-locked public aggregate/PNG SHA-256 values and the moved corpus bytes against the `test-mainet` baseline. The private raw verifiers must run at their locked source revisions, with their original private sessions. For example, at X16 source commit `47c44cc2cf9f10d6704de253d6133a61af920294`, use that revision's `research/tools/file02_x16_results.py verify <absolute-x16-session-path>`. The published derived values can be inspected in the JSON files indexed by the public manifests. A fresh public clone lacks private raw and cannot claim that full rederivation passed.
