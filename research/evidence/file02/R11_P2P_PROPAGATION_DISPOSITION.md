# R11 — P2P propagation disposition

Date: 2026-11-13. R11 requests valid C/CP/P/H on four validators with
P2P ingress, peer receive/transmit bytes, per-transaction propagation delay
and forwarding counts. The accepted factor registry supports RPC origin
only. The existing four-validator R06/R07 networks observe P2P relay after
RPC injection, and the `TestResearchV2M01P2PRelayBoundary` oracle confirms
that a peer can receive a valid transaction before a block commits. These
are relay observations, not P2P-origin injection.

The Comet overlay counts decoded mempool P2P transaction ingress and bytes,
but does not record matched transaction IDs and timestamps at sender and
receiving peers or per-transaction forwarding counts. Thus it cannot yield
a defensible propagation-delay distribution or forwarding count for R11.
R03 already records relay propagation latency as `UNAVAILABLE` for this
reason. Four validators on one host also do not measure cross-host delay.

**R11 is PARTIAL and unchecked.** The observed relay bytes/counts are
bounded supporting evidence only. The requested P2P-ingress latency and
forwarding attribution are `UNAVAILABLE`; no zero or invented estimate is
entered. A future measurement-only peer send/receive hook with a matched
transaction oracle, frozen plan, and independent networks would be needed
to close the case. No R11 confirmatory run was started.
