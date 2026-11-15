# X13 interrupted full attempt — excluded

The first full X13 run started from source `ecb5225371f530b7a400754f3f1793088da9d7b2`
with the locked 36-network schedule. The Owner interrupted the turn while
the integration test was running. At stop, 25 network directories existed;
24 had complete oracles and the remaining run was incomplete. The runner
recorded `INVALID_FOR_INFERENCE`, `n_valid=24/36`, no rate estimates or CI.
The private session is
`research/artifacts/file02/x13-20261115T060132Z-ed05a21a`.

The manifest file SHA-256 is
`20c137140a560fed631b54a6295d6cfb1297d1eeb32f8e25b9b851dba25cceac`,
but this **is not a valid immutable raw verification**. After the interrupted
integration binary exited, its detached load process finished writing
`runs/X13-b08-r23/result.json`. The manifest recorded the empty-file hash
`e3b0c442...`; the eventual file hash was `cf97746a...`. The independent
verifier correctly reports `raw hash mismatch`. Neither the manifest nor
that raw file is altered to conceal the mismatch. This attempt is
**quarantined** and cannot contribute to any X13 estimate, QC PASS or claim.

The runner was updated before a fresh whole-session restart to quiesce its
own session generator before raw freezing. No 20/23/25 outcome from the
interrupted attempt was used to change rates, n, contrast, CI or decision
rule. The original QA-only sessions remain separately archived.
