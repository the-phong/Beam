# X13 preflight before confirmatory sensitivity runs

The prospective X13 plan and source were committed at `703c01e` before
any attempted send. The first QA-only attempt at source `703c01e` failed
**before sending** because the schedule verifier compared JSON lists with
Python tuples. Its private frozen raw manifest is
`5db79c0724d69d413f494bc2d7af3d918841854d08f88ad67b2dd7d5b54f0eec`;
QC is `INVALID_FOR_INFERENCE` and remains excluded. The schedule
representation fix was committed at `9bc055d` before a second QA session.

The second QA-only session ran **three new networks**, one each at 25, 23
and 20 tx/s, with all 3/3 QC PASS under QA scope. Its raw manifest is
`0e0a86a42ea1565d163ad2fb35de76118238ba8b3c4c3a2c5ff2ee6853de6cbb`;
`python3 research/tools/file02_x13.py verify
research/artifacts/file02/x13-20261115T055932Z-924b9b3c` returned
`X13_VERIFY=OK qc=QA_ONLY`. All three runs had 72/72 fixed valid H
observed within 3 s, exact one-second rate windows, the locked late-ML
rejection oracle and four-validator agreement. QA-only observations are
**excluded** from X13 effect estimates and future CI.
