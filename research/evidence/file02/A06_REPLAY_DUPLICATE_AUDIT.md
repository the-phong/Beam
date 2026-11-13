# A06 — replay and duplicate-cache audit

Date: 2026-11-14. The [locked plan](../../protocol/FILE02_A06_LOCKED_PLAN.md),
scoped replay-plan exception, RPC duplicate classifier and oracle were
committed at `67115af` before any A06 send. Private session:
`research/artifacts/file02/a06-20261113T181233Z-7724bf59`.
Plan SHA-256:
`3cb6a715932986da4217fea56bfce2d71e2dece0482e538c56b96f60f9ae4241`;
immutable raw manifest SHA-256:
`af82216b85b542c4ab6699e81617f3912658cb4439951a9a535f2d4a69b58025`.

Four fixture blocks used rotated order across 16 fresh four-validator
single-host networks. Each arm offered 20 H-sized writes/s (51,360 B/s) for
four steady seconds, with six identical legitimate H tx/s per block. Actor B
replayed a signed valid H transaction or modified a captured signature and
replayed that invalid H transaction; distinct valid and distinct invalid
backgrounds were controls. Every arm wrote 81 RPC requests including
warm-up and met the one-second rate windows. The four validators agreed on
BlockID/AppHash, committed every distinct valid hash once, committed the
valid replay hash once and committed no bad-signature hash. Account-state
and raw integrity oracles passed, with no recorder/observer drop or safety
abort.

| Condition | Unique wire hashes | RPC `duplicate_cache` / 56 background requests | Background CheckTx NEW / 56 | Background secp-fail NEW / 56 | Mean background CheckTx wall service |
| --- | ---: | --- | --- | --- | ---: |
| Valid distinct | 81 | 0 in every block | 56 in every block | 0 | 288.12 µs |
| Valid repeated | 26 | 55 in every block | 1 in every block | 0 | 311.68 µs for the one NEW call/block |
| Bad-secp distinct | 81 | 0 in every block | 56 in every block | 56 in every block | 175.17 µs |
| Bad-secp repeated | 26 | 0 in every block | 56 in every block | 56 in every block | 185.06 µs |

The preregistered invalid-repeated minus invalid-distinct application NEW
count was **0 per block**, mean **0**, paired-block bootstrap 95% CI
**[0, 0]**. Thus the tested repeated bad-signature bytes were **not absorbed
by the Comet duplicate cache**: all 56 requests reached the application
secp failure stage in each block. Valid replay was absorbed after its first
accepted request. A `duplicate_cache` response is a classified RPC outcome,
not an application CheckTx call. The recorder's `RECHECK` counts are kept
separate from `NEW`; they do not measure replay cache hits.

Whole-application-process CPU per offered attempt was 6,172.84 / 2,222.22 /
3,425.93 / 3,672.84 µs for the four conditions in table order. This is a
shared process envelope including the test harness, **not** CPU per rejected
transaction. Class-specific CPU saving is `UNAVAILABLE`; CheckTx wall spans
and application-call counts provide the bounded mechanism evidence. The
16-network, four-block sample does not measure legitimate service degradation
with a confirmatory CI and does not imply a DoS or knee.

`make verify`, load unit tests and integration-tag compile passed before
execution. `python3 research/tools/file02_a06.py run` exited 0 with
16/16 QC PASS; `python3 research/tools/file02_a06.py verify
research/artifacts/file02/a06-20261113T181233Z-7724bf59` exited 0 with
`A06_VERIFY=OK`. Private per-run JSON/CSV, QC, aggregate, figure status,
decision and full four-validator committed-hash oracle remain preserved.
**A06 PASS for bounded replay/cache mechanism.** Single-host RPC-origin only.
