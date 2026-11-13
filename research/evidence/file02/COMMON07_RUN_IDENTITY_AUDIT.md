# COMMON-07 — per-run identity, rates and QC-before-analysis audit

Date: 2026-11-12. The audit used the Owner-authorized T06 bounded QA pilot,
not a confirmatory T06 case. `python3 research/tools/file02_common07.py
research/artifacts/file02/t06-precision-20261112T142307Z-1f61b257
research/evidence/file02/COMMON07_RUN_RECORDS.json` exited 0. It first
rehashes the frozen private raw manifest and re-executes the 48-run T06
validation. That validation checks metadata/source binding, schedule,
distinct sends, planned and actual rates, all-four oracle and inclusion
observations **before** the precision aggregate is calculated. The public
per-run records are written only after every run passes the gate. The
separate common File 02 `qc-run` also emits a QC report before an aggregate
accepts its result.

The resulting [48-run inventory](COMMON07_RUN_RECORDS.json), SHA-256
`f0700672e13992d28a3442268babd4b7c4b365e5b86d8484a68cac6f1284b760`,
has 12 runs/profile. Every row contains source commit and tree, profile,
`case_id=T06_BOUNDED_PRECISION_PILOT`, attacker capability `C` (registered
account holder, benign traffic), planned and achieved tx/s, planned and
achieved encoded bytes/s, run ID, transport, scope and QC status. The case
and capability are declared in this derived public record from the frozen
event plan; they are not retroactively written into immutable raw files.
The auditor rejects unexpected event capability/source/traffic or a mismatch
between encoded and actual wire byte length.

| Profile | Planned and achieved tx/s | Planned and achieved bytes/s | QC |
| --- | ---: | ---: | --- |
| C | 20 | 3,160 | 12 usable for pilot |
| CP | 20 | 51,560 | 12 usable for pilot |
| P | 20 | 50,280 | 12 usable for pilot |
| H | 20 | 51,560 | 12 usable for pilot |

`python3 -m unittest discover research/tools -p 'test_file02_common07.py' -v`
exited 0 (two tests, including wire-size and capability mutation rejection).
This verifies the available benign RPC pilot path. Every future attack or
confirmatory case must emit its own case/capability/rate record and pass QC
before analysis; these 48 rows do not serve as those future case results.
