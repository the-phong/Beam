# T06 bounded generator calibration pilot

Date: 2026-11-12. Owner authorized this **pilot only**. The plan was locked
before execution in [`FILE02_T06_PILOT_PLAN.md`](../../protocol/FILE02_T06_PILOT_PLAN.md).
This is a four-validator single-host loopback diagnostic, not a confirmatory
T/R/A/Q/D case, capacity knee, RQ result or DoS finding.

Source commit: `6870caa946ab76e18d3ee789a07daed5e6808840`;
source tree: `08072117a5786788105b19642fc17ada7d09b543`.
Unmodified S06 config SHA-256:
`9bc640db8b8875af3f9a5e4f89faa1882e8f4bc10459997d1a891397768cf21a`.

`make test-file02-t06-calibration` exited 0. Sixteen fresh four-validator
networks ran C/CP/P/H at the preregistered 2, 6, 12 and 20 valid tx/s,
in increasing rate order. Every one-second steady window achieved exactly
its target, every offered request was sent, and the direct block/state
oracle found all valid transactions committed exactly once with same-height
agreement. No run hit a guard or exceeded 25 tx/s or 65,536 encoded
bytes/s. There were no unsent/dropped requests.

| Target tx/s | Achieved tx/s on C/CP/P/H | H achieved encoded bytes/s | H generator CPU core fraction | H generator peak RSS |
| ---: | --- | ---: | ---: | ---: |
| 2 | 2 / 2 / 2 / 2 | 5,136 | 0.0034 | 67.2 MB |
| 6 | 6 / 6 / 6 / 6 | 15,408 | 0.0058 | 75.2 MB |
| 12 | 12 / 12 / 12 / 12 | 30,816 | 0.0072 | 82.1 MB |
| 20 | 20 / 20 / 20 / 20 | 51,360 | 0.0107 | 88.4 MB |

These are generator process CPU/RSS observations, not validator CPU or
crypto cost. A single four-second steady segment per cell cannot establish
safe capacity, a knee, long-run thermal stability or a p99. It only selects
20 tx/s as a feasible **pilot** rate under the unchanged S06 caps for a
separately preregistered precision extension; the extended run must still
pass all its own safety and oracle checks.

Frozen private session:
`research/artifacts/file02/t06-calibration-20261112T141031Z-b8ad1a5f`.
Manifest SHA-256:
`5c591674c86b69e081c61e1f360f6d538c3e21120461453cd294c8098ff90f81`.
`python3 research/tools/file02_t06.py verify research/artifacts/file02/t06-calibration-20261112T141031Z-b8ad1a5f`
exited 0 and rederived all 16 run summaries. Raw payloads, runtime homes,
keys and unsanitized logs remain ignored and private.
