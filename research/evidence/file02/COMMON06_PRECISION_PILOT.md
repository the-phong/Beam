# COMMON-06 — counterbalanced independent QA and p99 precision gate

Date: 2026-11-12. **Disposition: PARTIAL; checklist box remains open.**
Scope: bounded single-host loopback QA only, not a T06 calibration or any
confirmatory File 02 run.

Source commit: `3367ba8f5aec9fa99f9ccfc7fb78ccb712b6ce97`;
source tree: `7eb15efa714712fdc98fabaeb290917a657d9696`.
Branch: `research/file02-common-foundation`.

`make test-file02-common06` exited 0. It built pinned binaries and ran 16
fresh four-validator networks, four independent runs per C/CP/P/H profile.
The recorded master seed `7162430642474591061` generated an initial random
profile permutation followed by four cyclic rotations. Each profile occupied
each order position exactly once. Each run had a unique seed, run ID, genesis
hash, validator set, private home, plan and direct correctness oracle. All
16 runs achieved 2 offered tx/s in every steady window and committed all
nine valid transactions exactly once. The auditor checked private hashes,
actual send offsets, all-four commit observations, source identity, schedule
replay and unique genesis hashes.

| Block | Position 1 | Position 2 | Position 3 | Position 4 |
| --- | --- | --- | --- | --- |
| 0 | P | C | H | CP |
| 1 | C | H | CP | P |
| 2 | H | CP | P | C |
| 3 | CP | P | C | H |

Each profile has `n_independent_runs=4`, `n_transaction_samples=36` and
`n_per_run=[9,9,9,9]`. The only latency sample is the upper bound from the
first RPC tx-index observation on **all four** validators after send; it is
not an exact commit timestamp. No p99 value or inferential confidence
interval is published.

## Precision finding and required gate

For an i.i.d. continuous sample, the exact binomial order-statistic formula
in `research/tools/file02_common06.py` gives only **25.45%** probability that
the empirical p99's population rank lies between 98.5% and 99.5% with
`n=36`. A conservative floor of `n=2,000` gives **96.42%** coverage under
that optimistic assumption. Within-run transactions are clustered, so this
floor cannot by itself establish run-level uncertainty. A later locked
cluster-aware method, enough independent runs, observed censoring and a
numerical CI-width target are still needed before a p99 claim.

The pinned S06 configuration caps a run at 25 tx/s and 60 seconds, giving
at most 1,500 sends even before warm-up and drain. Thus it cannot supply
2,000 observations inside one run. Increasing load or duration and choosing
profile-specific safe rates is T06 generator calibration, which the current
File 02 foundation authority explicitly does not open. The bounded QA pilot
therefore does **not** satisfy the p99 sufficiency clause of COMMON-06;
COMMON-07 and later items have not been started in response.

Frozen private session:
`research/artifacts/file02/common06-20261112T134703Z-45bf7598`.
Manifest SHA-256:
`deba0796f074d4ad66135d256484a938effefaee2295c3b470317822548a2026`.
`python3 research/tools/file02_common06.py verify research/artifacts/file02/common06-20261112T134703Z-45bf7598`
exited 0 and rederived all 16 runs. No keys, runtime homes, signed payloads
or unsanitized logs are tracked.

Owner decision needed: authorize a separately scoped T06 calibration and
precision pilot, including a bounded update to run duration/rate and a
cluster-aware p99 CI criterion, or revise the COMMON-06 acceptance wording
to a procedural gate that forbids p99 until enough samples exist. Neither
option is assumed here.

## Subsequent disposition

The Owner subsequently authorized the bounded T06 pilot. The locked
[calibration](T06_CALIBRATION_PILOT.md) and
[precision extension](T06_PRECISION_PILOT.md) completed. The latter ran 48
fresh networks and satisfied the preregistered p99 rank and cluster CI gates
for all four profiles with 12 independent runs and 2,880 steady samples per
profile. The original 16-run pilot remains an honest failed precision check;
its raw evidence and finding are unchanged. COMMON-06 is now verified for
the bounded single-host QA setup, with future confirmatory case runs required
to meet their own plan and QC gates.
