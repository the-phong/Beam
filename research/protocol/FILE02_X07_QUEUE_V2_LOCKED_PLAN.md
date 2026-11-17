# X07 queue-path rework — prospective engineering QA lock

Status: LOCKED FOR ENGINEERING QA before any queue-v2 send. This extends the
2026-11-16 X07 trace lock without altering the accepted application,
CometBFT consensus rules, historical endpoints, protocol or S06 caps.

The disposable hash-checked Comet v0.38.26 overlay may add transaction-hash
markers at the ABCI socket server read, server mutex attempt/acquisition,
application return and response handoff. Build the dedicated app binary
against that overlay only for this QA and write a distinct private trace per
app process. Add a privacy-preserving envelope-hash mapping at the P2P peer
send and per-channel queue attempt/accept/dequeue markers at MConnection.
Only hashes, event names, process/channel IDs and timestamps may be logged;
no payload, key or account address. Normal builds keep the overlay disabled.

For each of eight blocks, draw one fresh signed I fixture: six fixed valid
H/s plus fourteen late-bad-ML H/s, 20 total H-sized tx/s and 51,360 B/s.
Use one fixture and genesis for both arms, fresh four-validator single-host
networks, loopback RPC origin at validator 0. Four blocks run trace OFF→ON,
four ON→OFF. No replacement within a block. The run/network is the unit.
Both arms retain the app event/resource recorder and 100-ms block scanner.
The trace ON arm must link all 241 origin tx through RPC, precheck, ABCI
client/server, app result/callback and admission; all accepted valid tx
must have a deterministic per-peer envelope map and queue trajectory where
the peer send succeeded. Invalid tx must have no accepted-mempool or relay
markers. Verify AppHash, exact-once valid commit, invalid non-commit, process
identity, zero drops, monotonic ordering, source hash and immutable raw.

Report separate intervals with precise labels: ABCI client queue attempt to
writer dequeue (includes any send-blocking), socket transfer/dispatch,
server mutex wait, application lock/service, CheckTx return, accepted
mempool residence, P2P queue attempt-to-dequeue (includes send-blocking),
and receiver reactor arrival. Where a channel dequeue races an accepted
marker, do not call their difference a pure queue wait. Peer relay and
consensus branches overlap; never sum them into one "full queue" duration.
Use paired 95% block-bootstrap CI, 20,000 resamples, seed 20261117, for
trace ON−OFF app/Comet/observer CPU and producer RPC latency. If any marker
or clock/provenance check fails, withhold the affected estimate and leave
the X07 box open. Close X07 only if all required observable queue segments
are verified with a bounded overhead statement; any unsupported segment
stays `null` and is named explicitly.

Max 25 tx/s, 65,536 B/s, 16 outstanding and 60 seconds per run remain
unchanged. Raw stays ignored/private. This is QA, not a new RQ1b effect,
capacity search, P2P-origin experiment or multi-host claim.
