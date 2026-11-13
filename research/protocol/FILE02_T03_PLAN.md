# T03 rejection-pipeline oracle plan

Status: locked before T03 execution, 2026-11-13. Use the public S04
four-profile corpus and fresh direct ABCI++ application instances. Run
`TestS04CorpusABCIOracle` for expected CheckTx code/stage, NEW/RECHECK and
proposal behavior; `TestS05CheckTxTelemetryONOFFCorpus` for recorder
event/spans; and `TestProfileNegativeBoundariesAndReachedStages` for
structural, chain, binding, secp, ML and state boundaries. Verify the
public corpus checksum/shape first. No offered network load is needed.

Pass only if each corpus case returns its independent expected code and last
stage, a rejected transaction leaves state/AppHash unchanged and cannot
appear as a successful proposal transaction, and recorded spans never pass
an earlier rejected stage. H's deliberately invalid ML proof must reach ML
only after structural, binding and secp success; P-only has no ECDSA-invalid
class. Report per-profile code/stage coverage, corpus digest, exact commands
and test exit codes. No performance or attack-impact conclusion follows.
