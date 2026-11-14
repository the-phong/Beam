# A12 — many-peer scenario disposition

Date: 2026-11-14. A12 requires Actor D to originate transactions from
multiple controlled peers toward four validators on an isolated testnet,
then measure per-peer rates, ingress fairness, relay congestion and valid
goodput. The accepted [`file02_factors.json`](../../config/file02_factors.json)
currently marks `RPC_MULTI=UNSUPPORTED`,
`P2P_ORIGIN_INJECTION=UNSUPPORTED` and
`FOUR_VALIDATOR_MULTI_HOST_PRIMARY=UNSUPPORTED`. The bounded load sender
accepts only `http://127.0.0.1` RPC, and the COMMON plan validator accepts
only loopback endpoint hosts. The A01–A10 run harness therefore records one
RPC source at validator 0, even though transactions subsequently relay
among four Comet validators. Relay observation is not direct P2P-origin
injection or per-peer fairness evidence.

The prior T07 calibration verified two VMware guests but explicitly found
that both share one physical Windows hypervisor. Its clock mapping is
specific to that short calibration and cannot serve as an A12 run timing
bound. A read-only reachability probe on 2026-11-14 found the two private
guest SSH endpoints reachable but rejected noninteractive public-key
authentication; the short names `u1`/`u2` did not resolve in this workspace.
No guest was modified and no password was placed in a command or artifact.
Even with access, the repository lacks a controlled direct P2P-origin
injector, per-peer source/ingress attribution and a locked multi-source
safety/QC contract.

**A12 is UNSUPPORTED and unchecked.** No A12 network was sent, no per-peer
rate, fairness, relay-congestion, distributed goodput or multi-host effect
was manufactured. To close it, an Owner-reviewed source/topology extension
would need verified private guest access, a bounded per-peer P2P/RPC
injector, source labels and telemetry, per-run clock calibration where
cross-host latency is used, isolated network configuration, safety guard,
locked matched plan and fresh QC-passing runs. Physical-host independence
would remain unavailable while the guests share one workstation.
