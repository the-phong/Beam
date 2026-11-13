# R12 — RPC versus P2P origin disposition

Date: 2026-11-13. R12 requires matched transactions sent through separate
RPC and direct P2P origins, with end-to-end stages, timestamps and gossip
path/counters. [`file02_factors.json`](../../config/file02_factors.json)
lists `origins_available=["RPC"]` and
`P2P_ORIGIN_INJECTION=UNSUPPORTED`. The current bounded load sender is an
HTTP RPC client; the four-validator relay test begins with
`broadcast_tx_sync` at validator 0. Peers receiving that transaction by
gossip do not constitute a P2P-origin experimental arm.

There is no controlled direct P2P injector, paired origin oracle or
source-to-peer timestamp contract. The accepted common contract expressly
excludes interpreting RPC relay as P2P-origin injection. **R12 is
UNSUPPORTED and unchecked.** No case run or RPC-vs-P2P effect estimate was
created. Closing R12 requires a separately verified P2P injector and
measurement-only origin/path instrumentation, a locked matched plan and
fresh QC-passing networks within the same safety bounds.
