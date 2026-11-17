# X07/X16 rework addendum for independent Gate review

This is a **new technical evidence addendum**, not a rewrite or acceptance
of D-046. The 2026-11-15 X18 decision and D-046 correctly recorded X07/X16
as PARTIAL at that time. The Owner subsequently requested full X07/X16
work. The source-locked X16 48-run matched B/I and 16-run probe QA, and the
source-locked X07 16-run queue-path QA, now support bounded technical PASS
for the two checklist deliverables. Public aggregates and private immutable
raw manifests are indexed by `X16_CONFIRMATORY_PUBLIC_MANIFEST.json` and
`X07_QUEUE_V2_PUBLIC_MANIFEST.json`.

| Review question | New evidence | Remaining boundary |
| --- | --- | --- |
| X07 queue-path observation | ABCI client/server, app lock/service, Comet admission and accepted residence, P2P per-peer queue and receiver hash linkage; 16/16 QC PASS; 8 matched ON/OFF pairs | Composite response/consensus queue-only intervals unavailable; no additive end-to-end queue scalar. Instrumentation increases producer RPC median +0.0961 ms [0.0856,0.1085]. |
| X16 stage/resources | 48/48 B/I QC PASS, app/Comet/observer/generator CPU, instrumented stage thread CPU, accepted/rejected, relay/commit oracles, resource curves and run-level CIs | Probe increases app CPU +0.0489 s/run [0.0134,0.0823] in 8-pair QA; unperturbed per-stage CPU and exact relay/commit CPU shares remain null. |
| Bottleneck/service | B valid background costs more app CPU and downstream bytes; fixed valid 3-s endpoint remains 100% both arms | No observed sustained degradation or knee within the safe range; no exact bottleneck, DoS or distributed-network claim. |

The technical recommendation remains **CONDITIONAL GO for bounded
mechanism/service reporting**, pending independent Controller/Owner review.
The original early service-degradation RQ2 event is still NOT ESTABLISHED,
RQ3 is NOT_RUN and D01–D13/ML stay closed. Historical RQ1/RQ1b definitions,
the negative 3-second endpoint, S06 caps, older raw and the quarantined
wrong-wire-size QA raw were preserved. This addendum does not authorize
merging the branch or opening a new phase.
