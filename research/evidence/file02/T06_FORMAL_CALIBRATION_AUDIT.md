# T06 — formal bounded generator calibration

Date: 2026-11-13. The [locked plan](../../protocol/FILE02_T06_FORMAL_PLAN.md)
and runner were committed at `5e8998f11441e8ce6be6a9552ac17936a536b706`
before execution. `make test-file02-t06-formal` exited 0. It ran 16 fresh,
four-validator single-host loopback networks: one per C/CP/P/H profile and
2, 6, 12, 20 valid tx/s, in ascending rate order. Each network had a distinct
genesis and run ID. The one-second warm-up was excluded from rate calculation;
four one-second steady windows were measured, then three seconds allowed for
drain. Each run stayed below the unchanged S06 limits of 25 tx/s, 65,536
bytes/s and 60 seconds.

| Profile | Target tx/s | Achieved tx/s | Target bytes/s | Achieved bytes/s | Loadgen CPU fraction | Loadgen peak RSS (MiB) | Dropped/unsent | Valid committed |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| C | 2 | 2.0 | 310 | 310 | 0.278% | 57.3 | 0 | 9 |
| CP | 2 | 2.0 | 5,150 | 5,150 | 0.275% | 60.3 | 0 | 9 |
| P | 2 | 2.0 | 5,022 | 5,022 | 0.315% | 62.9 | 0 | 9 |
| H | 2 | 2.0 | 5,150 | 5,150 | 0.269% | 63.5 | 0 | 9 |
| C | 6 | 6.0 | 930 | 930 | 0.354% | 67.1 | 0 | 25 |
| CP | 6 | 6.0 | 15,450 | 15,450 | 0.452% | 68.8 | 0 | 25 |
| P | 6 | 6.0 | 15,066 | 15,066 | 0.419% | 69.1 | 0 | 25 |
| H | 6 | 6.0 | 15,450 | 15,450 | 0.419% | 71.2 | 0 | 25 |
| C | 12 | 12.0 | 1,860 | 1,860 | 0.490% | 71.7 | 0 | 49 |
| CP | 12 | 12.0 | 30,900 | 30,900 | 0.556% | 74.1 | 0 | 49 |
| P | 12 | 12.0 | 30,132 | 30,132 | 0.532% | 74.4 | 0 | 49 |
| H | 12 | 12.0 | 30,900 | 30,900 | 0.593% | 76.9 | 0 | 49 |
| C | 20 | 20.0 | 3,100 | 3,100 | 0.690% | 77.3 | 0 | 81 |
| CP | 20 | 20.0 | 51,500 | 51,500 | 0.786% | 78.7 | 0 | 81 |
| P | 20 | 20.0 | 50,220 | 50,220 | 0.718% | 80.7 | 0 | 81 |
| H | 20 | 20.0 | 51,500 | 51,500 | 0.893% | 82.0 | 0 | 81 |

Each steady one-second window hit its exact target count and byte volume.
No submission error, missing request, loadgen abort or safety abort occurred.
The independent four-validator oracle confirmed exact-once valid commits,
matching height/block/AppHash and account state for every cell. The maximum
loadgen process CPU fraction was 0.00893 core, below the predeclared 0.25-core
diagnostic limit. The CPU and RSS measurements describe the load generator
process, not total testbed resource cost.

The private frozen session is
`research/artifacts/file02/t06-formal-20261112T185103Z-e3c998ed`.
Its manifest SHA-256 is
`477765a706a65d2d9c5a94ef15ebe70ed2a09c24c297321eef1c088260883b75`.
`python3 research/tools/file02_t06_formal.py verify` exited 0, checked the
manifest and recalculated all 16 summary rows from raw. The private
`summary.json` holds unrounded values and per-window counts.

**T06 generator calibration: PASS within the registered bounded region.**
These rates demonstrate successful offered-load delivery, not a sustainable
throughput limit or a knee. `NEAR_KNEE`, `AT_KNEE` and
`SLIGHTLY_OVER_KNEE` remain `NOT_OBSERVED_WITHIN_SAFE_RANGE` after the
earlier pilot; no new knee search was performed. The short single-host
preflight supports no RQ or DoS effect estimate. T07 is next.
