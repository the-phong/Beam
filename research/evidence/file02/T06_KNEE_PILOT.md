# T06 bounded knee search: no defensible knee identified

Date: 2026-11-12. Owner authorized this pilot within the unchanged S06
single-host loopback limits. The [plan and knee rule](../../protocol/FILE02_T06_KNEE_PILOT_PLAN.md)
were committed before execution at producer commit `b7ada39`. The pilot
ran 36 fresh four-validator networks: C/CP/P/H at 20, 23 and 25 valid tx/s,
three independent networks per cell. All offered windows met target exactly;
all valid transactions eventually committed once, four validators agreed,
and no observer drop or safety abort occurred. The highest encoded rate was
64,325 bytes/s for CP/H, below the unchanged 65,536-byte/s cap. No safety
cap was raised. The [public per-run table](T06_KNEE_PILOT_RUNS.json)
contains all 36 QC-passing rows and the literal preregistered decision.

Values below are medians across three independent networks per cell. Goodput
`timely` counts steady-origin transactions first seen on all four validators
before the twelve-second steady window closes. Goodput `eventual` includes
those committed during drain. p50/p95 are **upper bounds** from 100-ms
block scanning, each based on 240, 276 or 300 steady transactions per run.
Queue is the average CheckTx **socket** wait from cumulative recorder
count/sum; full mempool/P2P queue wait and queue depth are unavailable.

| Profile | Offered tx/s | Encoded bytes/s | Timely goodput tx/s | Eventual goodput tx/s | p50 upper ms | p95 upper ms | Socket queue wait ms | Max accepted mempool count |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| C | 20 | 3,060 | 17.42 | 20 | 1,133 | 1,773 | 0.0402 | 36 |
| C | 23 | 3,519 | 20.00 | 23 | 1,160 | 1,785 | 0.0423 | 42 |
| C | 25 | 3,825 | 21.83 | 25 | 1,127 | 1,750 | 0.0419 | 44 |
| CP | 20 | 51,460 | 19.33 | 20 | 1,114 | 1,704 | 0.0495 | 33 |
| CP | 23 | 59,179 | 19.67 | 23 | 1,129 | 1,745 | 0.0556 | 38 |
| CP | 25 | 64,325 | 23.67 | 25 | 1,133 | 1,719 | 0.0482 | 40 |
| P | 20 | 50,180 | 17.17 | 20 | 1,119 | 1,706 | 0.0510 | 33 |
| P | 23 | 57,707 | 19.92 | 23 | 1,129 | 1,758 | 0.0517 | 42 |
| P | 25 | 62,725 | 23.92 | 25 | 1,117 | 1,711 | 0.0478 | 42 |
| H | 20 | 51,460 | 17.42 | 20 | 1,145 | 1,800 | 0.0481 | 36 |
| H | 23 | 59,179 | 19.92 | 23 | 1,138 | 1,770 | 0.0509 | 42 |
| H | 25 | 64,325 | 21.92 | 25 | 1,120 | 1,749 | 0.0480 | 42 |

## Knee decision and measurement limitation

The preregistered rule flagged C and H with `KNEE_OBSERVED_BRACKET_20_TO_23_TX_S`
and CP/P with `KNEE_NOT_OBSERVED_WITHIN_TESTED_RANGE`. The C/H flags came
**only** from the timely-goodput branch; no profile met the joint latency
and queue branch. Crucially, median timely goodput was already below 90% of
offered load at the **20 tx/s reference** for C, P and H. The fixed
approximately 1.7–1.8-second inclusion observation delay removes the last
part of a twelve-second steady window from that count even when the system
keeps up. Thus the goodput threshold confounds end-of-window censoring with
saturation. The initial threshold flags remain published in the JSON and
are not silently rewritten, but they cannot establish a capacity knee.

**Scientific disposition: knee not confirmed/observed in the tested
20–25 tx/s region.** Eventual goodput equals offered rate at every level;
p95 and socket queue wait do not show sustained growth from 20 to 25 tx/s.
This is a negative finding for this bounded single-host pilot, not evidence
that the system has no knee above 25 tx/s or on another topology. The
pre-registered operational criterion needs prospective correction before
any future knee claim. Section 2.1 numerical near/at/above-knee levels
remain unset, and Gate COMMON remains open.

## Reproduction and integrity

`python3 research/tools/file02_t06_knee.py smoke` exited 0; its Python
audit stored the literal decision and froze private raw. The first `verify`
attempt exited 2 because in-memory integer rate keys became JSON string
keys. A parser-only canonicalization fix made `verify` exit 0 against the
same frozen raw and summary. Frozen private session:
`research/artifacts/file02/t06-knee-20261112T161511Z-1a3754c9`.
Manifest SHA-256:
`17175da15a386943dfa9e836b3f2fca271b49379a29a610ac86917935dad4e00`.
No raw payload, key, runtime home or unsanitized log is tracked.

## Proposal for Owner review on section 2

1. Amend section 2.1 to list 2/6/12/20/23/25 tx/s as **tested pilot levels**
   by profile, with 25 tx/s the current safety boundary and near/at/above
   knee marked `NOT_OBSERVED_WITHIN_SAFE_RANGE`. Do not label 23 or 25 tx/s
   as a knee or assign a fabricated over-knee level.
2. Keep the factor and metric dictionary; explicitly mark full queue depth,
   P2P propagation and multi-host measures `UNAVAILABLE` until separately
   instrumented. Future case plans must choose their supported load region
   and QC it independently.
3. Keep section 2.3 per-case artifacts unchecked until the separately
   authorized T/R/A/Q/D program produces them. A future knee search would
   require a prospective time-aligned throughput rule and separate Owner
   authority for any changed safety cap or topology.

The pilot stops here for Controller/Owner review as requested.
