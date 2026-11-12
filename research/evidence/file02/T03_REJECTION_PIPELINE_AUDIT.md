# T03 — rejection pipeline and stage oracle

Date: 2026-11-13. The [locked plan](../../protocol/FILE02_T03_PLAN.md)
uses the public 34-case corpus (SHA-256
`d0f4bfaea4c541ee6f71547b6ea47e44c30aef588704051459e57e04d6478c83`).
`make corpus-s04-verify` exited 0. Fresh direct ABCI++ tests ran against
each case with the stated precondition; they did not load a public node.

| Profile | Corpus cases | Rejection stages covered |
| --- | ---: | --- |
| C | 8 | structural, profile, chain, amount, binding, secp, state |
| CP | 9 | structural (including bad padding), profile, chain, amount, binding, secp, state |
| P | 8 | structural, profile, chain, amount, binding, ML, state; no ECDSA-invalid class |
| H | 9 | structural, profile, chain, amount, binding, secp, ML, state |

`TestS04CorpusABCIOracle`, `TestS05CheckTxTelemetryONOFFCorpus` and
`TestProfileNegativeBoundariesAndReachedStages` exited 0. The direct oracle
compared CheckTx NEW/RECHECK code and last stage with the corpus, required
ProcessProposal rejection for invalid input, and checked that rejected
FinalizeBlock/Commit input did not alter accounts or total supply. The
recorder test required matching code/stage and two non-dropped NEW/RECHECK
events with nonnegative spans. `TestFile02T03LateInvalidStageOrdering`
exited 0 and checked the exact visited stages: H bad-ML reaches secp then
ML; bad-secp stops before ML; unbound sender stops before crypto; nonzero
CP padding stops at structural. Rejected paths leave AppHash unchanged.

**T03 PASS for direct application rejection and recorder stage order.** This
is an oracle/correctness check; it does not measure network attack impact,
resource exhaustion or P2P ingress. T04 still requires a fresh real
four-validator valid/invalid network mix.
