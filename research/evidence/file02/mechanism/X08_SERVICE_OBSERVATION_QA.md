# X08 — per-validator valid-service observation QA

Status: **PASS for timestamp capture and observation semantics**. The
contract is `research/protocol/SERVICE_METRICS_CONTRACT.md`. Source
`4373748` was committed before the fresh QA run. The raw manifest
`67243d13ae31ae858611784add06f751b0f63de66f686098581b42fb7c53196e`
was frozen and independently verified; private homes remain Git-ignored.

One valid H transaction was observed in block height 3 at all four
validators; the test checked shared block ID and AppHash. Per-validator
first block scan monotonic stamps were 7775978975031, 7775979945684,
7775980875740 and 7775981859723 ns. The all-four stamp equaled the latest
one. The scanner emitted UTC stamps for all four. These values validate
order and completeness, not an exact commit moment or service effect.
The historical generator already records per-transaction actual send UTC
and monotonic stamps; X10 must join those with new per-validator observations
and verify censoring/clock continuity.

The nominal 100-ms scanner interval is an observation schedule, not a
guaranteed ±100-ms precision bound. Event-level Comet commit callbacks
were not installed, so exact commit latency remains `null` with reason
`UNAVAILABLE_EXACT_COMMIT_TIMESTAMP`. Historical 3-s endpoint and raw
remain untouched. This QA used one transaction, so p50/p95 and any
RQ1b effect are `PENDING_CASE_EXECUTION`.
