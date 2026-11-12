# T04 mixed-input four-validator consensus plan

Status: locked before execution, 2026-11-13. Four new four-validator
single-host loopback networks run C, CP, P and H separately under the
unchanged S06 safety caps. Each fresh genesis has 16 funded senders, one
receiver and unique validator homes. One warm-up legitimate transaction is
followed by four steady one-second windows, each with one legitimate and
one distinct invalid transaction. The invalid class is bad secp for C/CP
and bad ML for P/H. Every planned transaction has a known CheckTx code and
last stage; no duplicates are sent.

Pass if each profile sends all nine planned events, exactly five valid
transactions commit once on all four validators, all four invalid events
receive their expected rejection code and commit zero times, same-height
block ID/AppHash and sender/receiver balances/nonces agree, and the sum of
all genesis account balances remains 32 on each validator. Record common
height, block ID, AppHash, supply oracle, actual phase rates, source/tree,
config/plan/genesis hashes and private raw manifest. This is correctness
preflight, not an attack impact or DoS experiment.
