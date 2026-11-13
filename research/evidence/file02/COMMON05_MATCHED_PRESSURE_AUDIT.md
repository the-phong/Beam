# COMMON-05 — matched benign and attacker offered pressure

Date: 2026-11-12. Scope: bounded single-host loopback QA, not a confirmatory
T/R/A/Q/D experiment or a resource/DoS inference.

Source commit: `dec443fdd1f65a52338d8c329a6cf1b711b951e9`;
source tree: `d5e048575f502275ea7e4c3c1fdae3fe47bcbd7b`.
Branch: `research/file02-common-foundation`.

`make test-file02-common05` exited 0. Four C/CP/P/H benign/attack pairs ran
on eight fresh isolated four-validator networks. The attack plan retained
the exact five legitimate signed bytes, hashes, schedules, accounts and
nonces of its benign partner and replaced four additional valid transactions
with same-size invalid signature mutations at identical offsets. The direct
oracle checked unique valid inclusion, invalid exclusion, account state and
same-height BlockID/AppHash agreement. The private auditor checked actual
send offsets, RPC codes, distinct wire hashes/account slots, plan/result
hashes and every steady-state offered window.

| Profile | Matched total tx/s, benign = attack | Matched total bytes/s, benign = attack | Valid commits, benign / attack | Invalid rejected, attack |
| --- | --- | ---: | ---: | ---: |
| C | 2 each steady second | 306 each steady second | 9 / 5 | 4 |
| CP | 2 each steady second | 5,146 each steady second | 9 / 5 | 4 |
| P | 2 each steady second | 5,018 each steady second | 9 / 5 | 4 |
| H | 2 each steady second | 5,146 each steady second | 9 / 5 | 4 |

**Matched variables:** profile, genesis, pinned network config, planned
send offsets, legitimate stream, total offered tx/s and total offered encoded
bytes/s. **Unmatched by design:** accepted tx/s, valid commit count, mempool
occupancy and verification path. Those outcomes must not be interpreted as
controlled resource pressure. Equal offered bytes do not imply equal CPU or
mempool cost. Between profiles, bytes/s differ; matching applies only within
each profile pair.

Frozen private session:
`research/artifacts/file02/common05-20261112T134041Z-ecbbbada`.
Manifest SHA-256:
`84c695d9e027a499ee8368dcdc22b3fd5a716a7090867d67b09b78b8905ee55f`.
`python3 research/tools/file02_common05.py verify research/artifacts/file02/common05-20261112T134041Z-ecbbbada`
exited 0 and rederived all four pairs from frozen raw. No keys, runtime
homes, signed payloads or unsanitized logs are tracked.

One pair per profile at a low safe rate establishes the matching invariant
for this QA path only. Later case-specific questions must choose and declare
the matched variable, calibrate rates and report independent-run uncertainty.
