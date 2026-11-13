# R13 — cache and duplicate control disposition

Date: 2026-11-13. The focused four-validator telemetry regression
`go test -tags 'integration fourval' ./test/integration -run
'^TestS05FourValidatorTelemetry$' -timeout 240s -count=1` exited 0
(62.253 s) on this branch. Across C/CP/P/H, that oracle sends a valid
transaction, checks exact-once commit and state, resubmits the same bytes,
and accepts either Comet `duplicate_cache` or application invalid-nonce
rejection depending on cache state. The observed ingress-proxy path also
classifies `duplicate_cache`; application recorder events are not treated
as cache hits. The S04 app oracle independently verifies duplicate proposal
deduplication and state neutrality. This establishes correctness of the
accepted cache/dedup boundary in a bounded network.

R13 additionally asks for **matched repeated-identical versus distinct
invalid and valid transaction streams**, cache hit counts, verification
calls, rejection reasons and CPU per rejected transaction. The S04/S05
regressions are not a preregistered R13 experiment and do not supply those
matched streams or per-request Comet CPU. The observer has no native cache
hit counter; an RPC `duplicate_cache` response is a classified outcome,
not a calibrated CPU saving. The earlier R09 valid-only raw cannot supply
the missing repeated/invalid arms. No R13-specific network was run.

**R13 is PARTIAL and unchecked.** Cache-hit count and Comet CPU for the
requested matched comparison are `UNAVAILABLE`, not zero. The current
evidence prevents claiming that repeated invalid payloads are cheaply
discarded or that dedup changes attacker feasibility. A prospective
repeated-versus-distinct design, bounded new networks and a measurement-only
cache/CPU capture would be required to close the case.
