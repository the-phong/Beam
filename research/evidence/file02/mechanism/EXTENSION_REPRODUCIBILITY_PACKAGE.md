# X17 — extension reproducibility and independent QA package

Status: **PASS for the bounded X10/X13 executable artifact contract**.
Both successful sessions contain a locked top-level `plan.json`, each
network's pre-send common `research_plan/plan.json`, ignored private raw
with SHA-256 manifest, `result_per_run.json/csv`, `qc_report.json`,
`result_aggregate.json`, `decision.md` and `figures/status.json`. The
per-run raw includes signed wire schedules, actual socket sends, result
codes, four-validator block/AppHash oracle, observer streams, dedicated app
resource/CheckTx events and generator resources. Runtime homes and keys
remain private; no raw path is published in Git. Public audit, derived
class/resource/secondary-CI JSON and four reproducible PNG figures are
under `research/evidence/file02/mechanism/`.

| Study | Locked source commit | QC | Immutable raw manifest SHA-256 |
| --- | --- | --- | --- |
| X10, 24 balanced L/B/I blocks, 72 fresh networks | `c3ce45b4adae9d6477ac2749469762468e38d938` | PASS 72/72, zero exclusions | `a97e1665deca67e747f30a1c6b03268140b716886e42fb24d9c1e1a6512233c8` |
| X13, 12 balanced 20/23/25 blocks, 36 fresh networks | `2c2608d08d35fadfe48062b701293d4b4fb08a8f` | PASS 36/36, zero exclusions | `9eaa4d990f43e5c9d1c38d2ab4f968db3dc7a3c4e94a6a0a2a8d60ed4e0ab0e3` |

Both plans bind Go `1.24.1 linux/amd64`, `go.mod` SHA-256
`f0ad5b5c685d5fb5c51006c62b21502677ed0b336ae87ad0ef11d97d64c2babe`,
S06 config SHA-256
`9bc640db8b8875af3f9a5e4f89faa1882e8f4bc10459997d1a891397768cf21a`,
host inventory, source tree, chain/genesis, endpoint, 25/65,536/16/60
safety caps, workload seeds and balanced orders. The tools independently
re-read all raw files and compare every hash to the manifest, rederive
per-run QC, matched effects and CIs, then compare every derived JSON/CSV/
decision byte or value. A missing run, tampered raw, wrong source/config,
wrong rate, failed oracle or changed derived output is rejected.
`test_file02_common.py` exercises missing availability, raw hash tampering,
extra raw files, source/runtime-plan mismatch, invalid QC and frozen
write-once behavior. X10 decision tests cover inside/crossing/outside
registered margin and precision/binary conflicts; X13 tests enforce
six balanced rate orders and the safety byte cap.

## Recompute commands

The commands below use the current layout and require the private ignored
sessions. For source-locked full rederivation, check out each study's locked
source commit in this same repository and use the paths and verifier from
that commit; the locked plan and raw bytes must remain unchanged.

```sh
python3 experiments/analysis/file02_x10.py verify research/artifacts/file02/x10-20261115T051140Z-3d69d6eb
python3 experiments/analysis/file02_x13.py verify research/artifacts/file02/x13-20261115T135447Z-a960b885
python3 experiments/analysis/file02_x11_historical.py check
python3 experiments/analysis/file02_x15_secondary.py check
python3 experiments/analysis/file02_x16_resources.py check
python3 experiments/analysis/file02_x17_figures.py check
python3 -m unittest discover -s experiments/analysis -p 'test_file02_common.py' -v
python3 -m unittest discover -s experiments/analysis -p 'test_file02_x10.py' -v
python3 -m unittest discover -s experiments/analysis -p 'test_file02_x13.py' -v
make verify
```

`file02_x17_figures.py write` generated four public PNGs from verified raw
and public X11/X16 derived data. Its `check` command rerenders each image
and compares exact bytes and the figure manifest. The figures show X10
fixed-H service, X13 safe-rate response, X10 descriptive resource curves
and historical invalid CheckTx wall spans. They do not depict exact commit,
stage CPU or an inferred knee. The private session `figures/status.json`
remains its original pending marker so its historical verifier and bytes
remain unchanged; the publication figures are separate outputs.

## Failed, excluded and interrupted session audit

| Attempt | Disposition | Evidence |
| --- | --- | --- |
| X10 one-block preflight | QA_ONLY, excluded | Raw manifest `9081a49c...`; excluded preflight record in the frozen private session. |
| X10 first 72-network attempt | `INVALID_FOR_INFERENCE`; stopped after 31 good arms due one `RATE_MISS`, no effect/CI | Raw manifest `b75610e8...`; [first-attempt QC](X10_FIRST_ATTEMPT_QC.md). |
| X10 successful second full attempt | PASS 72/72, no excluded block | Raw manifest `a97e1665...`; [audit](X10_THREE_ARM_AUDIT.md). The original derived `analysis-v1/` is frozen; the corrected decision label changes no raw/effect/CI. |
| X13 first preflight | `INVALID_FOR_INFERENCE`; schedule representation mismatch before sends | Raw manifest `5db79c07...`; excluded preflight record in the frozen private session. |
| X13 second preflight | QA_ONLY 3/3; excluded | Raw manifest `0e0a86a4...`; excluded preflight record in the frozen private session. |
| X13 first full attempt | **Quarantined** after Owner interruption: 24 complete run oracles, 12 missing/incomplete; manifest subsequently mismatches one detached generator write | Manifest file SHA `20c13714...` is **not an integrity PASS**; [interruption audit](X13_INTERRUPTED_ATTEMPT.md). No estimator uses its data. |
| X13 successful whole-session restart | PASS 36/36, no excluded block | Raw manifest `9eaa4d99...`; [audit](X13_SAFE_RANGE_SENSITIVITY_AUDIT.md). |

The successful X10/X13 sessions have no selective arm deletion, safety
abort, observer drop or invalid commit. The interrupted X13 attempt is not
silently repaired or counted. Checksums and source commits above are the
reproducible evidence boundary; exact event timestamps, private host homes
and signed transaction bytes are available only in the ignored private
bundle for an authorized reviewer. X17 establishes repeatability of this
single-host RPC study, not independent physical-host replication.
