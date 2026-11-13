# T01 — four-profile valid baseline correctness

Date: 2026-11-13. The [locked plan](../../protocol/FILE02_T01_PLAN.md)
was committed at `c5bb90929a1b845d179f3968868dc72a08857ef0` before
execution. Command `make test-file02-t01` exited 0; four new four-validator
single-host loopback networks ran C, CP, P and H at 2 legitimate tx/s.
Each network had its own genesis, seed, run ID and validator homes. The
Go oracle checked CheckTx admission, exact-once inclusion in all four
validators, sender balances/nonces, receiver balance, same-height block ID
and AppHash. The independent Python audit rechecked source/plan/config/
genesis hashes, all nine transaction bytes and account slots, two achieved
sends in each one-second steady window, four observer streams and all-four
inclusion observations. No invalid traffic was offered.

| Profile | Valid accepted | Valid committed exactly once | Invalid | Common height | QC |
| --- | ---: | ---: | ---: | ---: | --- |
| C | 9 | 9 | 0 | 8 | USABLE |
| CP | 9 | 9 | 0 | 8 | USABLE |
| P | 9 | 9 | 0 | 8 | USABLE |
| H | 9 | 9 | 0 | 8 | USABLE |

Private frozen session:
`research/artifacts/file02/t01-20261112T175703Z-e1e59c0b`.
Manifest SHA-256:
`2b55abb9e087152154f0deb14aed98ecc60c3ad2f32b98d8a05f1eb63eed9529`.
`python3 research/tools/file02_t01.py verify` exited 0 and recomputed the
summary from all frozen raw. Only the keyless summary, hashes and audit are
public. No keys, runtime homes or unsanitized logs are tracked.

**T01 correctness PASS** for the bounded four-profile single-host preflight.
One run per profile supplies no CI, high-load capacity or independent-host
claim. The next checklist item is T02; T03–T07 are not inferred from T01.
