# A10 — matched-byte secondary analysis, locked before synthesis

Status: LOCKED ANALYSIS, 2026-11-14. Reuse exactly the six preregistered
A01–A05/A08 sessions from A09. Within every case pair, benign and attack
offered the same 20 H-sized writes/s and 51,360 B/s; every wire transaction
was 2,568 bytes. Equal tx/s is an additional control here, not a
requirement that rates differ. The fixed legitimate stream and genesis
remain identical. No additional payload or network run is created.

For each case, report the original paired 3,000-ms service effect and CI,
valid/invalid admission outcomes, sampled accepted mempool bytes, P2P
relay-send envelope bytes and CheckTx stage/service differences. Because
offered bytes are exactly equal, a stage-cost difference cannot be solely
caused by wire size; a service effect absent in both arms has no positive
degradation to explain away. Do not call P2P envelope counts exact
propagation latency, CheckTx socket wait a full transaction queue, or
eventual commit count sustainable throughput. Classifier FPR is outside
this case and remains unavailable.

QC requires upstream raw/derived integrity, all 144 arms and 72 pairs
present, exact byte-size/rate and fixed-stream matching, no failed upstream
QC and an immutable analysis-input manifest. This is a secondary synthesis,
so no pooled cross-case confirmatory CI or DoS conclusion is permitted.
