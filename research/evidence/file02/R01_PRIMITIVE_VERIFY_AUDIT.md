# R01 — valid primitive verification audit

Date: 2026-11-13. The [R01 method](../../protocol/FILE02_R01_LOCKED_PLAN.md)
was committed at source `5a9aec1a0169c9fe79fa3c52c987b750ab6efa56`
before a private `LOCKED` plan was created and before any benchmark process
ran. The plan source tree was `c02e0f579e9ee0bc55844bacd7ba00c0c1e6c860`.
The private session is
`research/artifacts/file02/r01-20261113T025326Z-5a9aec1`, with plan
SHA-256 `f0415a778e0369c73ea5a31f1a8c54e542c39a259cd077c85f65da95a27e9834`
and immutable raw manifest SHA-256
`286afaaaf5d403257027fcdd383e37595c7c28452865124c888c973eb844c86d`.
The private plan records exact Go, OS, CPU features, source, module digest,
orders, command and safety configuration. No key or signed payload was
published.

Twelve separate Go benchmark processes measured a valid H intent's secp256k1
and ML-DSA-44 verification primitives on the same host and configuration;
six process pairs used each order. Go benchmark iterations are not CI
replicates. Both valid proofs passed fixture checks before timing. All 12
processes exited 0 and yielded both cases, with no missing or excluded pair.
Each process used `GOMAXPROCS=1`, `-cpu=1`, `-benchtime=200ms` and
`-benchmem`. The timed region excludes signing, parsing, digest and key
decoding. One process pair is the uncertainty unit.

| Metric | secp256k1 | ML-DSA-44 |
| --- | ---: | ---: |
| Mean verification ns/op across 12 processes | 104,762.17 | 18,294.00 |
| Process-run ns/op range | 98,518–110,525 | 17,831–18,812 |
| Mean B/op | 688 | 450 |
| Mean allocs/op | 16 | 3 |
| Cycles/op | `UNAVAILABLE` | `UNAVAILABLE` |

The preregistered paired ML-minus-secp mean was **−86,468.17 ns/op**, with
20,000-resample paired-bootstrap 95% CI **[−88,306.50, −84,548.65]
ns/op**. Mean within-process ML/secp ratio was 0.1748. These figures describe
the pinned Go primitive implementations on this host; they are not full
CheckTx service cost, CPU attribution, a network limit or a DoS result.
No verified cycle counter was available, so cycles/op is `null` with reason.

The first analyzer classified all 12 successful raw logs invalid because
its parser required a Go benchmark `-1` CPU suffix, while this Go output
omitted it. The parser fix was committed at
`466f2923225bb22f931b189a521b71e5524ccb93`. The initial derived files
were preserved in the private frozen `analysis-v1/` archive (manifest SHA-256
`69592aa5e1d26f8332e3e4bf559fdffabf8808fbd85af99ade25e1f48ff68dfc`);
**the original raw and locked plan bytes were not changed or rerun**.
Reanalysis produced `QC=PASS`, `n=12/12`, and `ANALYZED`. The per-process
JSON/CSV, QC report, aggregate, analysis provenance, figure-status manifest
and decision report are in the same private session.

`python3 research/tools/file02_r01.py verify
research/artifacts/file02/r01-20261113T025326Z-5a9aec1` exited 0 after
rehashing raw and both analysis versions and recalculating results. R01 is
**PASS for bounded primitive measurement only**. R02 full validation is next.
