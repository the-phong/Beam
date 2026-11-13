# R03 — C versus CP size-only control

Status: **LOCKED BEFORE R03 MEASUREMENT**, 2026-11-13. R01 and R02 were
completed in checklist order. This case repeats the direct validation
measurement on **twelve fresh process runs** with a new genesis per process,
one warm-up and 240 valid CheckTx calls per profile. The same 16-byte
`pluto-file02-r02` chain ID, transaction intents, accounts, nonce-zero
schedule, hardware, Go toolchain and recorder mode are used for C and CP
within each run. Both profiles use secp256k1 only; CP adds exactly 2,420
canonical zero-padding bytes. C is 148 bytes and CP is 2,568 bytes.

The R02 direct-ABCI harness is reused without changing validation semantics.
All four profiles may be measured in a process to maintain counterbalanced
position, but the preregistered R03 comparison uses only C and CP. Twelve
process runs use the seed `20261113`, the same cyclic four-profile order
schedule and `GOMAXPROCS=1`. Process timeout is 55 seconds; no network
traffic is sent. A fresh session and raw manifest distinguish R03 from R02.

For each run, report CP-minus-C wire bytes, mean structural-parse wall ns,
mean full CheckTx service ns, process CPU ns/tx, allocated B/tx and allocs/tx.
The paired effect of each metric is computed within the process. For
CheckTx, structural parse and CPU, report a two-sided 95% percentile
bootstrap over the 12 process-level differences, 20,000 draws, seed
`20261113`. Other metrics are descriptive with their run-level ranges.
No within-process transaction is a CI replicate. The same R02 p99 rank
and cluster precision rule applies if a p99 is presented.

QC requires the actual private pre-run source/tree, host, toolchain and
unchanged S06 safety digest; all 12 process exits and fresh geneses;
240 accepted CheckTx records per profile with expected stage set, exact
wire length, zero recorder drop and complete CPU/allocation deltas. Failed
processes remain immutable and invalidate the inferential comparison.
Raw and derived artifacts follow the File 02 write-once private contract.

The canonical C/CP contrast controls cryptographic *type* but changes wire
size and structural parsing work together; it cannot prove that any full
service delta is solely transport. Exact P2P relay propagation latency is
`UNAVAILABLE`: the existing RPC-origin single-host capture records P2P
traffic counters, but has no matched peer send/receive timestamp. Do not
substitute RPC latency or block observation. Thus R03 can close its direct
application size/parse contrast but cannot satisfy the checklist's relay
latency output; the final case disposition must explicitly remain partial
unless a valid relay capture is implemented and verified.
