# X07 bounded queue timeline plan

Status: measurement-only source lock before QA. This plan does not alter
accepted transaction validity or Comet consensus. When the CheckTx
recorder is ON, record UTC at application method entry, immediately after
acquiring the application read lock, and after the validation call returns.
Keep existing stage spans and CheckTx service duration. No clock is read
on the recorder-OFF validation path. Compare ON/OFF results and AppHash
across the public correctness corpus, including NEW and RECHECK.

The pre-existing Comet overlay already records a histogram of socket
client CheckTx queue attempt to writer dequeue, with no per-transaction
identity. The RPC proxy records ingress time and round-trip, but its event
does not identify transactions. The app event has a SHA-256 transaction ID;
it can join its own stage and result fields. The mempool collector samples
accepted count/bytes and P2P relay envelope counters.

Acceptance: all recorder timestamps parse and preserve entry ≤ lock
acquisition ≤ validation result; ON/OFF correctness and four-validator
network QA pass. Report all unavailable intervals as `null` with reasons:
proxy-to-Comet enqueue, Comet precheck/admission residence, per-transaction
P2P relay and end-to-end transaction queue. A socket queue histogram must
never be relabeled as total queue wait. If these capabilities remain
unavailable, X07 receives a PARTIAL disposition and its master box stays
open, while later X work may use the supported subintervals.
