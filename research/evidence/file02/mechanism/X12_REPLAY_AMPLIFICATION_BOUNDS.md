# X12 — replay/cache controls and amplification bounds

Status: **PASS for the bounded RPC-origin replay and RECHECK distinction**.
The locked [A06 audit](../A06_REPLAY_DUPLICATE_AUDIT.md) supplies four
rotated fixture blocks and 16 fresh networks at 20 H-sized offered writes/s.
Each arm has six identical fixed legitimate H/s. A06 raw manifest is
`af82216b85b542c4ab6699e81617f3912658cb4439951a9a535f2d4a69b58025`;
its current verifier returned `A06_VERIFY=OK qc=PASS`. These are historical
locked cases reused for a mechanism analysis, not new confirmatory runs.

| Background, 56 requests/block | RPC duplicate-cache outcomes | CheckTx NEW calls | Secp-fail NEW calls |
| --- | ---: | ---: | ---: |
| Valid distinct | 0 | 56 | 0 |
| Valid repeated | 55 | 1 | 0 |
| Bad-secp distinct | 0 | 56 | 56 |
| Bad-secp repeated | 0 | 56 | 56 |

The same pattern holds in **all four blocks**. Repeating a valid accepted
hash reduced application NEW validation calls from 56 to 1 for the
background (56-fold call count ratio within this fixture). Repeating an
invalid secp hash left all 56 calls and secp failures: invalid-repeated minus
invalid-distinct NEW count **0**, paired block 95% CI **[0,0]**. The repeat
therefore did not receive a measured application-work saving through the
observed RPC duplicate-cache path. This is a call-count bound, not a CPU or
network amplification factor; wall service for the 56 failures was measured
separately, and no sustained service harm was tested here. The valid-repeated
mean wall service covers only its single NEW call and is not comparable as a
per-offered-request service cost without accounting for the 55 cache hits.

`RECHECK` is internal Comet revalidation of accepted pending transactions,
not a new RPC send or a replay-cache hit. A06 recorded RECHECK by arm;
historical [R14 audit](../R14_RECHECK_AUDIT.md) independently covered
16 networks across C/CP/P/H with 81 NEW per network, and H had 19–27
RECHECK per run (mean 23.75, 95% CI [20.50,26.25]), all code 0.
`file02_r14.py verify` on the current tree initially reported plan drift
because `go.mod` changed after the locked run. Re-running the exact verifier
at source commit `a97825bd2acad437e884afe51e92b5ee989f58f7` on a copied
private session returned `R14_VERIFY=OK qc=PASS n=16`, raw manifest
`5378c732f2e368ac9bb7dd0b30b8efdbe0bf1617a4ed2e7367e79da38cdc71e0`.
Original raw bytes were not changed.

The 56:1 bound applies to this RPC duplicate-cache behavior only. P2P origin,
other invalid types, cache lifetime, exact CPU saving and cross-host behavior
are unavailable. A06's original endpoint and semantics remain unchanged.
