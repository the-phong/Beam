# T04 — mixed-input four-validator consensus preflight

Date: 2026-11-13. The [locked plan](../../protocol/FILE02_T04_PLAN.md)
was committed at `4c493cdd330c5228bb55bc3e164b344e3e19753e` before
execution. `make test-file02-t04` exited 0. Four fresh four-validator
single-host loopback networks, one each for C/CP/P/H, sent one warm-up
valid transfer and then one valid plus one deliberately invalid transfer
in each of four one-second steady windows. Bad secp was used for C/CP;
bad ML for P/H. All events were distinct and matched their planned
CheckTx codes.

| Profile | Valid committed once | Invalid rejected/not committed | Common height | Supply on all four validators | QC |
| --- | ---: | ---: | ---: | --- | --- |
| C | 5 | 4 | 8 | 32/32/32/32 | USABLE |
| CP | 5 | 4 | 8 | 32/32/32/32 | USABLE |
| P | 5 | 4 | 8 | 32/32/32/32 | USABLE |
| H | 5 | 4 | 8 | 32/32/32/32 | USABLE |

The Go oracle checked the all-four block transaction sets, same-height
block ID/AppHash, expected sender nonce/balance and receiver balance. A
separate sum over every genesis account on every validator verified total
supply conservation. The Python audit independently rechecked hashes,
source/plan/config/genesis identity, actual one-valid/one-invalid per
steady window, no invalid inclusion and four observer streams.

Private frozen session:
`research/artifacts/file02/t04-20261112T180446Z-39c61177`.
Manifest SHA-256:
`664ef844423923ac1ac39e51ee89e53e2c98e2cb9072d5d28174b6c079f1e0e7`.
`python3 research/tools/file02_t04.py verify` exited 0 and recomputed the
summary from the frozen raw. No keys, runtime homes or unsanitized logs
are tracked.

**T04 PASS for bounded single-host consensus correctness.** This does not
measure multi-host independence, sustained service degradation or DoS.
T05 recorder ON/OFF perturbation remains next.
