# X10 three-arm harness dry run

Status: **QA_ONLY, excluded from confirmatory inference**. At source
`dd42aa650db3a8f14d68ba734e1debe39bd2bd0d`, one matched fixture
block ran B, I and L in its locked permutation. All three fresh
four-validator networks passed the integration oracle, sent their planned
73/241/241 transactions and recorded the expected valid/invalid commit
counts: B 241/0, I 73/168, L 73/0. Each had four separate ABCI app PIDs,
per-validator observation stamps, positive process CPU, zero recorder
drops and matched fixed-stream/genesis. The independent X10 run auditor
parsed each frozen QA run successfully. Private raw manifest SHA-256:
`9081a49c00ee418bcda25c533164436ac4c2dcd34f2c1e7f420fa16748253496`.

These three `scope=QA_ONLY` networks are not among the 24 confirmatory
blocks and never enter the CI, aggregate or service decision. The
preflight exposed no harness/schema defect requiring protocol amendment.
The raw is retained privately and was not modified after freezing.
