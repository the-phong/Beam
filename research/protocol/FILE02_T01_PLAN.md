# T01 four-profile baseline correctness plan

Status: locked before T01 execution, 2026-11-13. Scope: formal T-group
preflight correctness on four fresh, single-host, loopback four-validator
networks, one per C/CP/P/H profile. This is not an RQ effect estimate.

Use the pinned `s06_pilot.json` and `research/file02-t-preflight` source
commit. Each run gets a newly generated genesis and disposable keys, a
distinct run ID and seed, and an `equal_tx_s` plan at 2 valid tx/s. The
four-second steady interval has exactly two distinct valid transfers per
one-second window, after one warm-up transfer; drain is three seconds.
All nine transfers use separate genesis-funded senders with nonce zero and
one receiver. No invalid transaction is offered. Expected CheckTx code is
zero for every transfer.

Pass only if all four networks complete within unchanged S06 safety caps;
every planned send occurs in its phase and receives the expected code;
each valid transaction appears exactly once in the block set of all four
validators; same-height block ID and AppHash agree; sender balance/nonce
become 1/1 and receiver balance becomes 9 on every validator. The Go oracle
checks those state assertions independently of RPC admission. The Python
audit rechecks source/config/genesis/plan bindings, distinct bytes/accounts,
all-four inclusion observations, rate windows, observer streams and oracle
output. A missing or failed check makes T01 incomplete and retains raw.

Private raw is written only under ignored `research/artifacts/file02/t01-*`,
then frozen with a SHA-256 manifest. Public evidence may include the
manifest hash, run IDs, source/tree, counts and common-height hashes; no
keys, validator homes, unsanitized logs or raw transactions are published.
One run per profile supports correctness only, not a CI or throughput claim.
