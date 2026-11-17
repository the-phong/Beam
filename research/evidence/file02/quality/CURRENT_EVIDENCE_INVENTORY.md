# File 02 current evidence inventory — G00.3 snapshot

Snapshot date: 2026-11-14. Q engineering source/result commit:
`f629b47c67dd75d929936a98aebf4617f513d108`. This is a source
inventory, not a new scientific result or Gate QUALITY acceptance. PR #28
was still Draft at this original snapshot; its post-merge state is recorded
below. The raw sessions remain in the ignored, private
`research/artifacts/file02/` tree; this inventory publishes identifiers and
hashes only.

## Integration chain

| PR | Scope | State at snapshot | Merge commit / Q evidence head |
|---|---|---|---|
| [#21](https://github.com/the-phong/Beam/pull/21) | COMMON | MERGED | `6de41446272520809fbdcb84d3b1155831dc24d5` |
| [#22](https://github.com/the-phong/Beam/pull/22) | T preflight | MERGED | `a88e5d7eb5984f2dcddb5368248be2bff57e2aa4` |
| [#23](https://github.com/the-phong/Beam/pull/23) | RQ1/RQ1b plans | MERGED | `45495c56319b7f61619e7cd6b00208e2af7b0f3c` |
| [#24](https://github.com/the-phong/Beam/pull/24) | R result tree, stacked on #23 branch | MERGED into planning branch | `7ae6fa82aa56d8d39cc4a848b8f5dd1be90ebd24` |
| [#25](https://github.com/the-phong/Beam/pull/25) | R result tree into `test-mainet` | MERGED | `d67fef2598f278015968353763357f6aed28250d` |
| [#26](https://github.com/the-phong/Beam/pull/26) | bounded R gate acceptance | MERGED | `2df001620d799eafa6359603f1e6dedded1d8107` |
| [#27](https://github.com/the-phong/Beam/pull/27) | A result tree | MERGED | `2b453b3c14c410545d5699e03cff90d0eb21d63e` |
| [#28](https://github.com/the-phong/Beam/pull/28) | bounded A acceptance and Q audit | OPEN Draft at original snapshot; now MERGED per freeze below | `f629b47c67dd75d929936a98aebf4617f513d108` is the Q evidence head before review supplements |

**Post-merge freeze, 2026-11-14:** PR #28 was merged into `test-mainet` at
`da46f0cc4f87548825d9768d9242e346c17bb137` from exact head
`86d45346bf72ac3f7d62c0271620d6e78f4952dc`. All four final-head CI
checks passed: `scan`, `smoke`, and two `verify` jobs. D-044 records the
delegated bounded Gate QUALITY decision. The private pre/post-merge
comparison of all **31** frozen `raw/manifest.json` SHA-256 values was
identical; the public PR diff contained no private raw or key files. The
published [provenance index](DATA_PROVENANCE_INDEX.csv) remained byte-identical
and `file02_provenance_index.py check` exited 0 from the merge head. This
addendum updates the integration state; it does not revise historical
measurements or inferential eligibility.

Relevant decisions: [D-032](../../../DECISIONS.md) accepted the scientific
contract; D-038 accepted bounded COMMON and separated RQ1b; D-039 accepted
bounded T; D-040 opened R; D-041 accepted bounded R; D-042 opened A; D-043
accepted bounded ATTACK and opened Q. At the original Q evidence snapshot,
there was **no D-044 or Owner Gate QUALITY acceptance**; D-044 and the
delegated bounded acceptance are recorded in the post-merge freeze above.
The current authority is recorded
in `AGENTS.md` and the outside Checklist 02. Prior decision identifiers
remain historical; this inventory does not amend them.

## Frozen definitions and source identities

At Q evidence head `f629b47`, Git tree OIDs are:

| Tracked source tree | Git tree OID |
|---|---|
| `research/protocol/` | `8fd4507b6c9726c051935b6a8b56329e0faa2de9` |
| `research/config/` | `19322daf114f9d45d7968cad52a727012b33ecf3` |
| `research/evidence/file02/` | `c463dacd2d0ee89b346eaf93f02ea80bf7a48d80` |

Key file SHA-256 values at that same commit:

| File | SHA-256 |
|---|---|
| `research/config/file02_factors.json` | `813fefd50810c8beb6fa91d90f4905c0e51fc9089d13cd8781f2f6d7fcc34ac9` |
| `research/config/file02_metrics.json` | `d5c561a42e02820724dcaeac31aede98413b2c02c0a8cf2eafea0bbb9bd0db5e` |
| `research/config/manifest.yaml` | `5ae6ec762006b3fc7a594412d2b366bfbced7964382ed5675bcd01d3125890e0` |
| `research/protocol/FILE02_RQ1B_ADDENDUM.md` | `2643257c9d42da8dc7293c5ab92598db541626633d5a33ee5d0e51518935f239` |
| `research/protocol/FILE02_Q02_LOCKED_PLAN.md` | `20cbfd75db06b9b8ec2d8b54454a59cd21953089e1ebca86c29b5846f9f6a9bd` |

The protocol tree OID covers all File 02 case plans, including the R, A and
Q locked plans; the config tree covers factors, metric definitions, artifact
templates and the manifest. Each private `plan.json` records its own source
commit, module hash and config hash where applicable. The accompanying
`qc_report.json` binds the exact plan and `raw/manifest.json` SHA-256.

## Run and QC locator

[DATA_PROVENANCE_INDEX.csv](DATA_PROVENANCE_INDEX.csv) lists 30 frozen
sessions and 1,063 result/session records: 974 rows have a real `run_id`,
87 benchmark/diagnostic result rows lack one and are identified by a
`result_per_run.json` row reference, and two aborted sessions have no
result rows. Every indexed session's QC plan and raw-manifest hashes matched
the local frozen files when generated. CSV SHA-256:
`ef1e651f2de94e0c9a4f01f15a1fc56093b06bfd84c1a527634567e6f990485a`.
Regenerate and check with `python3 research/tools/file02_provenance_index.py
write` and `python3 research/tools/file02_provenance_index.py check` from the
repository root. A matching manifest hash proves local byte consistency,
not that each session is inferentially eligible.

The Q02 pre-send abort (`q02-20261114T024019Z-8ccd9928`) remains invalid;
the first complete Q02 schedule (`q02-20261114T025508Z-a4b258cb`) remains
technically QC-PASS but superseded for cross-profile inference because its
arm order was confounded. Only corrected Q02 v2
(`q02-20261114T044327Z-e7f236e2`, manifest SHA-256
`58265e00a8c0fc08a37cb4e2935633d85304a1e70d32142df5d9eaa4d1180f3f`)
supports the current Q02 contrast. Two early R15 sessions remain
`INVALID_FOR_INFERENCE`; the final R15 session is bounded historical
evidence. The current R15 verifier reports plan drift because current
`go.mod` differs from the locked source commit; the historical module hash
and raw integrity are distinguished in [Q05](Q05_BOTTLENECK_DISPOSITION.md).

The CSV covers sessions with frozen QC reports. It does not fabricate
`run_id` rows for report-only or unsupported cases. Their dispositions remain
in [R05](../R05_VERIFY_ORDER_UNSUPPORTED.md),
[R11](../R11_P2P_PROPAGATION_DISPOSITION.md),
[R12](../R12_RPC_P2P_ORIGIN_UNSUPPORTED.md),
[R13](../R13_CACHE_DEDUP_DISPOSITION.md),
[A11](../A11_MATCHED_RESOURCE_DISPOSITION.md) and
[A12](../A12_MANY_PEER_UNSUPPORTED.md). A11 and Q05/Q06 are PARTIAL, while
R05/R12/A12 remain UNSUPPORTED under their respective case rules.

## Supported results and boundaries

- [RQ1 resource report](../RQ1_RESOURCE_REPORT.md): application CheckTx
  H−CP at 20 valid tx/s is +81.47 µs/tx, paired 95% CI
  [+75.16, +87.26] µs/tx. Same wire size does not identify crypto CPU alone.
- A01/A04/A05 report invalid CheckTx *wall* means 7.25/391.31/438.28
  µs/tx in different case networks; they are not a paired CPU comparison.
- [Q01](Q01_MATCHED_VALID_SERVICE_AUDIT.md), [Q02](Q02_ORDER_BALANCED_DECOMPOSITION_AUDIT.md)
  and [Q06](Q06_REPLICATION_DISPOSITION.md): the fixed legitimate H
  3-second first-all-four endpoint shows no loss in tested matched RPC
  strata. Original A-family effects and corrected Q02 structural DiD are
  0 [0,0], but the binary endpoint is ceiling-limited and cannot prove
  latency equivalence or absence of smaller effects.
- [A10](../A10_MATCHED_BYTES_AUDIT.md): validator-0 relay-send envelope
  means 1,853,280 B/run benign and 555,984 B/run attack; this is not a
  per-peer propagation-latency measurement.
- [Q07](Q07_NO_INVALID_ACCEPTED_AUDIT.md): 248 QC-passing RQ1b networks,
  38,600 valid exact-once commits and 20,496 invalid transactions absent
  from committed blocks; full-state replay and observed four-validator
  agreement passed within the stated oracle scope.
- [R15](../R15_BENIGN_CAPACITY_AUDIT.md): no knee observed at
  20/23/25 valid tx/s; 25 tx/s is the safety cap, not a capacity estimate.

The testbed has four Comet processes on one physical host; the application
shares the integration process, with validator-0 loopback RPC origin. The
100-ms scanner yields first-all-four *observation upper bounds*, not exact
commit timestamps. Q03 p99 inference, stage crypto CPU, full queue,
independent-host and P2P-origin claims remain unavailable. Q05/Q06 stay
PARTIAL; D01–D13 and ML remain NOT_RUN. No new X hypothesis or run is
authorized by this inventory.

## CI and verification snapshot

PR #28 final-head checks: `scan` PASS, `smoke` PASS, and both `verify` jobs PASS.
The present Controller review independently reran Q01, Q02 v2, Q03,
Q04, Q06 and Q07 verifiers with exit code 0. Q03 reported 22/22 groups
`INSUFFICIENT_SAMPLES`; Q06 reported `PARTIAL_HOST_REPLICATION`.
The published index generator's hash check and tamper-rejection unit test
also passed. The original inventory was a pre-acceptance source snapshot;
the post-merge freeze above records the subsequent delegated decision and
integration without overwriting that history.
