# G00 and X01–X18 engineering closeout for independent review

Historical closeout dated 2026-11-15. The later X07/X16 bounded technical
rework is recorded in `X07_X16_GATE_REVIEW_ADDENDUM_20261117.md`; independent
Gate acceptance is still pending. The status table below is preserved as
the original engineering disposition.

The outside master checklist at
`/home/ntp/research/BEAM_MASTER_CHECKLIST_TONG_HOP_20261114.md` was
processed in order. G00.1–G00.4 completed after bounded Gate QUALITY
technical acceptance and PR #28 integration at
`da46f0cc4f87548825d9768d9242e346c17bb137`. The X work used a new
research branch from that merge. Its engineering status is **16 evidenced
X items and two PARTIAL/unchecked (X07, X16)**. No D, ML, R/A/Q rerun,
historical protocol revision or safety-cap increase occurred.

| Range | Disposition and main evidence |
| --- | --- |
| X01–X04 | Causal map, historical raw QC, exploratory continuous latency and locked prospective extension protocol. X03 never replaces the accepted 3-s endpoint. |
| X05–X06 | Four dedicated application PIDs and sampled process CPU; isolated primitive CPU profiles with network stage CPU still `null`. |
| X07 | **PARTIAL/open:** application entry/lock/result and socket subqueue measured; no linked whole queue residence or per-transaction P2P relay. |
| X08–X09 | Per-validator first block observation timestamps, 100-ms uncertainty contract, ON/OFF correctness and safety QA. Exact commit time unavailable. |
| X10 | 24 new matched L/B/I blocks, 72/72 QC PASS, raw verifier PASS; I−B +60.03 ms [39.51,81.54] upper-bound mean, inside registered ±100-ms margin; 3-s endpoint 0 [0,0]. |
| X11–X12 | Five separately verified historical invalid-class contrasts and A06/R14 replay/RECHECK bounds; no cross-class pooling or CPU conversion. |
| X13 | 12 new balanced 20/23/25 blocks, 36/36 QC PASS, raw verifier PASS; all fixed H timely by 3 s, knee not observed. Failed QA and interrupted/corrupted first full attempt retained but excluded. |
| X14–X15 | RPC/single-host scope matrix and statistical plan with paired p50/p95 CIs, missing-value reasons and conservative X10 margin decision. |
| X16 | **PARTIAL/open:** app/Comet/generator CPU, relay, mempool, resource curves and paired B−I CIs; stage/observer CPU, full queue and exact causal bottleneck unavailable. |
| X17–X18 | Reproducibility package and four byte-reproducible figures; service impact and five-question engineering Gate decision. Independent Gate acceptance remains pending. |

The new 3-s endpoint remains negative in every QC-passing X10/X13 arm.
The positive **direction** in X10's continuous observation upper-bound
metric is real within that estimator/CI, yet does not exceed the registered
research margin and cannot be read as exact commit latency or DoS. The
valid B arm's higher app/Comet CPU and relay/mempool work is consistent
with acceptance/commit cost, but stage CPU and causal bottleneck are not
located. No knee, independent-host or direct P2P-origin result is claimed.
The original early degradation RQ2 event was not established; RQ3 is
`NOT_RUN` and D01–D13 remain closed.

The [service report](SERVICE_IMPACT_REPORT.md),
[mechanism report](MECHANISM_ATTRIBUTION_REPORT.md),
[reproducibility package](EXTENSION_REPRODUCIBILITY_PACKAGE.md) and
[Gate decision](MECHANISM_GATE_DECISION.md) provide the review path.
Unit/schema/integrity tests, integration-tag compile, raw/derived/figure
recomputations and `make verify` passed at engineering closeout. The
source/result SHA, branch, PR URL, final CI and any review requests belong
in the PR handoff; this document is not independent acceptance.
