# RQ1 and RQ1b single-host planning handoff

Date: 2026-11-13. Scope: **plans only**, after the Owner's conditional Gate T
acceptance. PR #21 merged into `test-mainet` at
`6de41446272520809fbdcb84d3b1155831dc24d5`; PR #22 followed at
`a88e5d7eb5984f2dcddb5368248be2bff57e2aa4` with two passing `verify`,
one `scan` and one `smoke` check. The planning branch starts from that exact
second merge commit. No R/A/Q/D case was opened or run.

| Proposal | Design | Current state |
| --- | --- | --- |
| [RQ1 valid H–CP](../../protocol/FILE02_RQ1_SINGLE_HOST_PLAN.md) | 20 valid tx/s, exact 2,568 B/tx on both profiles; 12 paired fresh-network repetitions; primary validator-0 valid CheckTx service-time difference; paired-run CI. | `DRAFT`; H0/H1 and endpoint retain the accepted S02 research protocol. |
| [RQ1b legitimate service](../../protocol/FILE02_RQ1B_SINGLE_HOST_PLAN.md) | Fixed 6 legitimate H tx/s plus 14 H background tx/s; matched benign versus late bad-ML background; 12 paired repetitions; 3,000-ms all-four observation deadline and paired-run CI. | `DRAFT`; separate Owner-directed question, no DoS inference. |

Both candidates use four validators on **one physical host**, loopback RPC
through validator 0 and the unchanged S06 caps (25 tx/s, 65,536 B/s, 60 s
per run). The 20 tx/s total was delivered in T06's bounded valid-load
calibration; it is not a proven knee or sustainable throughput. The 6+14
mixed RQ1b schedule still needs a passing pre-run fixture and safety check.
No exact commit timestamp, full transaction queue wait, P2P-origin effect,
multi-host performance or distributed DoS result exists. The first
all-four block-scan observation is only an inclusion upper bound.

Before formal execution, each case needs its own accepted authorization and
`LOCKED` private `plan.json` files binding actual source/tree, genesis and
config hashes, host inventory, exact RPC ports, software versions, seeds,
run order, workload bytes, CI implementation, safety and QC. Case-specific
producer, oracle, analyzer and negative-path tests must pass. A failed pair
is excluded without outcome-based replacement. All run/result/figure and
decision artifacts remain `PENDING_CASE_EXECUTION`; Checklist 02 R/A/Q/D
boxes and section 2.3 per-case boxes remain open. No raw evidence or
scientific figure was created by this planning step.

The 12-pair choice follows bounded T06 precision-pilot scale, not a power
guarantee for either effect. A null, negative or imprecise result must be
reported as such. A practical service-failure SLO/minimum important effect
would need a prospective separate decision. The Owner/Controller can review
these proposed values without reopening Gate T or claiming a case complete.

## Planning verification

On the planning branch, `make verify` exited 0 (Go 1.24.1, linux/amd64),
`make test-file02-common` exited 0 (33 tests),
`python3 research/tools/file02_section2.py validate-registry` exited 0
(63 metrics), and `git diff --check` exited 0. T01 and T06 evidence explains
the chain-ID-dependent wire sizes: 16 bytes of chain ID produce 2,568-byte
CP/H transactions, while T06 formal's 23-byte ID produced 2,575-byte
transactions. The proposed 16-byte IDs were counted explicitly; the future
locked plans still must verify actual canonical bytes from their corpus.
Only planning/governance documents changed. Safety config, protocol code,
historical evidence and private raw were not edited.
