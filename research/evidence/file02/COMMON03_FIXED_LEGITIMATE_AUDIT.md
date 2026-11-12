# COMMON-03 — fixed legitimate stream with additive adversarial load

Date: 2026-11-12. Scope: bounded single-host loopback **QA only**, not a
T/R/A/Q/D confirmatory experiment or RQ1–RQ3 finding.

Source commit: `d94a84b13d9434a6e675b4cf1217e0372067e03f`;
source tree: `025c38b0fa68a1f24f4b3b1ca342751b4205cb6e`.
Branch: `research/file02-common-foundation`.

## Method and actual result

`make test-file02-common03` exited 0. It built the pinned S05-instrumented
Comet binary and ran `TestFile02Common03FixedLegitimate` on eight fresh
isolated four-validator networks: benign and attack for each C/CP/P/H. Each
profile pair used the same genesis, signed legitimate transaction bytes,
SHA-256 hashes, planned offsets, account slots and nonce zero. The attack plan
added one separately signed-and-modified invalid transaction in each of four
steady-state seconds; it did not replace a legitimate transaction.

| Profile | Benign legitimate/invalid offered tx/s by steady window | Attack legitimate/invalid offered tx/s by steady window | Valid commits benign/attack | Invalid rejected attack | Distinct hashes and account slots benign/attack |
| --- | --- | --- | ---: | ---: | --- |
| C | 1/0, 1/0, 1/0, 1/0 | 1/1, 1/1, 1/1, 1/1 | 5/5 | 4 | 5/9 |
| CP | 1/0, 1/0, 1/0, 1/0 | 1/1, 1/1, 1/1, 1/1 | 5/5 | 4 | 5/9 |
| P | 1/0, 1/0, 1/0, 1/0 | 1/1, 1/1, 1/1, 1/1 | 5/5 | 4 | 5/9 |
| H | 1/0, 1/0, 1/0, 1/0 | 1/1, 1/1, 1/1, 1/1 | 5/5 | 4 | 5/9 |

The direct four-validator oracle found each valid hash exactly once in the
common-height block history and every invalid hash absent; it also checked
same-height BlockID/AppHash and sender/receiver balances and nonces. The
private auditor rechecked every plan/result hash, achieved monotonic send
window, distinct wire hash/account slot, source/fixture identity, planned
legitimate equality and actual result codes. This excludes duplicate-cache
misclassification for these QA transactions.

Private frozen session:
`research/artifacts/file02/common03-20261112T132622Z-73ebf3ff`.
Manifest SHA-256:
`ebec5e3179e72fb6f8895bac315ecc8ca58627a3e55fef1427dac91c5ce1d511`.
`python3 research/tools/file02_common03.py verify research/artifacts/file02/common03-20261112T132622Z-73ebf3ff`
exited 0 and rederived four pairs from the frozen raw. No validator home,
key, signed payload or unsanitized log is in tracked evidence.

The first setup attempt exited 2 before sending because observer/RPC
metadata was absent. Its private, unfrozen diagnostics remain under
`research/artifacts/file02/common03-20261112T132446Z-bddc7365`;
the observer setup was corrected and the complete eight-network QA was rerun
from a new clean source commit. Only the successful session supports this
checklist tick.

## Limits

These are one bounded pair per profile on a single host. They establish the
COMMON-03 workload invariant and correctness for the QA path. They do not
establish independent-run precision, multi-host behavior, capacity, resource
effects, DoS outcomes or any confirmatory File 02 comparison.
