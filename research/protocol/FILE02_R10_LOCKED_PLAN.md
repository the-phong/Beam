# R10 — canonical size ceiling and mempool transaction bound

Status: **LOCKED BEFORE EXECUTION**, 2026-11-13. R09 is complete. One
fresh four-validator single-host H network on loopback will receive three
distinct submissions at one-second spacing: a valid maximal-canonical H
transfer with a 50-byte chain ID (2,602 B), an 8,192-B malformed extension
of it, and an 8,193-B extension. All three fit the unchanged 25 tx/s,
65,536 B/s, 60-second and loopback S06 safety caps. The private plan and
actual endpoint/config hashes must be written before the first send.

The accepted codec has maximum chain-ID length 50 and H wire size 2,602 B;
there is no valid 8,192-B transaction. The case must record the valid/invalid
RPC result, whether validator-0 application CheckTx was reached, last stage,
service duration if reached, block inclusion and AppHash agreement. The
8,192-B malformed transaction should reach CheckTx and fail structural;
the 8,193-B one should be rejected by Comet's `max_tx_bytes=8192` before
CheckTx. Preserve an unexpected result as QC failure. CPU cost at the
Comet precheck boundary is `UNAVAILABLE` without calibrated per-request CPU
instrumentation, never zero. This controlled diagnostic does not claim a
valid transaction near 8,192 B or a DoS/capacity result.

The Go integration harness hosts the application in its own process, so the
S06 workload `HostGuard` refuses it by design. This case instead enforces
fixed three-send/byte/duration bounds, loopback-only HTTP transport, manual
STOP before each send and four fresh observer samples with zero drops before
each send. It does not claim the full S06 resource guard ran. Any guard or
observer failure stops the run and leaves frozen diagnostic raw.
