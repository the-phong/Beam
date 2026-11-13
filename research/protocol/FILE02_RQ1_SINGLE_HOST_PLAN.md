# File 02 RQ1 — single-host H versus CP plan candidate

Status: **DRAFT FOR CONTROLLER/OWNER REVIEW**, 2026-11-13. This is a
pre-case design, not a `LOCKED` `plan.json`, a run authorization or a result.
It applies only to four real validators on one physical host. It implements
the accepted RQ1 contrast in [RESEARCH_PROTOCOL.md](RESEARCH_PROTOCOL.md)
without changing its hypothesis. The T06 calibration and precision pilots
motivate bounded values; they are not confirmatory observations.

## Case and estimand

Proposed case ID: `RQ1-H-CP-VALID-20-SH`. Compare valid, accepted H and CP
transfers at the **same 20 offered tx/s and exact 2,568 encoded bytes/tx**
(51,360 bytes/s per arm) using the proposed 16-byte chain ID
`pluto-file02-rq1`. The two arms use the same logical intents,
accounts, nonce schedule, canonical logical intent fields, genesis state,
network and collector configuration. Profile-separated signatures remain
different by design. CP's canonical padding and H's ML proof make the
wire lengths equal; both arms remain valid. The experiment identifies the
total H-versus-CP application validation cost under this bounded single-host
configuration. Stage spans are secondary decomposition; the contrast does
not by itself isolate an algorithmic ML-DSA cost or network saturation.

The primary endpoint for each network is the arithmetic mean of **valid,
accepted, steady-window application CheckTx service durations** at the
designated RPC ingress validator 0, from CheckTx entry to return. The
generator, Comet admission and CheckTx socket queue waits are excluded.
One observation per independent network enters inference. The paired effect
is `mean_H - mean_CP`, measured in microseconds per transaction. H0 is
`effect <= 0`; H1 is `effect > 0`, as already accepted. A positive effect
requires the paired-run 95% CI lower bound above zero and every equivalence,
rate and oracle check passing. A CI upper bound at or below zero opposes the
positive effect in this stratum; a crossing CI is inconclusive.

## Proposed parameters to lock before a formal run

| Field | Review candidate |
| --- | --- |
| Workload | Factor `BENIGN_HIGH`, capability `NONE`. One valid, pre-signed transfer per distinct funded sender at nonce 0; 1 warm-up and 240 steady sends per arm. Same logical intents, account registry and planned offsets within a pair. |
| Rate and schedule | 20 valid tx/s for 12 one-second steady windows; 0 invalid tx/s; 51,360 encoded bytes/s. One-second warm-up, 12-second steady, five-second drain and two-second cooldown; total 20 seconds, below the 60-second cap. |
| Independent replication | 12 preregistered matched H/CP pairs, each arm a fresh four-validator network (24 networks); pair seed and genesis matched within pair, distinct across pairs. Six pairs use H first and six CP first, with order assigned by a recorded master seed. No transaction is an independent CI replicate. |
| Origin and topology | One controlled generator sends `broadcast_tx_sync` through validator 0 RPC on `127.0.0.1`; four validators and collector run on one physical host. `FOUR_VALIDATOR_SINGLE_HOST_DIAGNOSTIC`; same pinned Comet/network limits and collector mode in both arms. Exact RPC ports, process placement and host inventory must be filled from the actual runtime before lock. |
| Primary capture | Validator 0 CheckTx entry/return spans joined to unique transaction hashes and steady-window sends. Record code/stage and any missing span; no zero filling. |
| CI and decision | Preselect a two-sided paired percentile bootstrap over the 12 pair differences, 20,000 resamples with recorded seed `20261113`, 2.5th/97.5th percentile. Report pair effects, n and interval even for null/negative results; no rate or method selection after seeing outcomes. The case analyzer must implement and test this procedure before locking. |
| Secondary | Stage times, RPC latency, process CPU/RSS, accepted mempool bytes, actual offered rates, eventual included count and 100-ms block-scan inclusion **upper bounds**. p99 is reported only if this case independently passes its registered rank/cluster precision gate; exact commit latency and sustainable throughput remain unavailable. |
| Safety | Keep `research/config/s06_pilot.json` unchanged: 25 tx/s, 65,536 bytes/s, 16 outstanding, 60 seconds, loopback allowlist, CPU/RSS/disk and observer guards. Any safety abort stops the sequence and preserves private raw. |
| QC | Require all planned socket writes; steady tx/s and bytes/s within ±5%, each one-second planned-send window at least 90%, no generator saturation, complete CheckTx spans, four live validators, exact-once valid inclusion, same-height BlockID/AppHash and correct account state. Verify run identity, immutable raw hashes, monotonic clock continuity and zero missing primary observations. A failed arm excludes its pair; no outcome-based replacement. |

The 20 tx/s rate was **delivered** in T06 for each profile and remained below
the fixed byte and duration caps. T06 formal used a 23-byte chain ID, so its
CP/H payloads were 2,575 bytes and it delivered 51,500 bytes/s at 20 tx/s.
T02's 2,568-byte CP/H size used a 16-byte chain ID. The locked case must
recompute actual canonical byte lengths and reject any deviation from its
declared byte schedule; the proposed schedule is below T06's tested byte
rate. T06 did not prove that 20 tx/s is below a
capacity knee or that it is sustainable. The proposed 12-pair count follows
the bounded precision pilot's independent-network scale; it is not a power
guarantee for a H–CP effect. A wide CI stays inconclusive.

## Pre-run lock and scientific boundary

The Owner's Gate T decision permits this planning document only. Before
execution, a separate case decision must authorize RQ1; the actual source
commit/tree, runtime inventory, software versions, exact endpoint ports,
genesis/config/corpus hashes, pair seeds/order and per-run IDs must populate
individual private `plan.json` files with status `LOCKED`. The case-specific
producer, span join, paired CI analyzer, oracle and QC tests must pass;
`file02_common.py validate-plan` and safety preflight must accept every plan.
No R item or per-case artifact box is checked by this draft.

Results, if later authorized, can describe only this RPC-origin single-host
stratum. They cannot establish P2P-origin effects, independent-host
performance, exact cross-host latency, a knee, DoS or distributed-network
behavior. If a required stream is absent, record `UNAVAILABLE`/`null` with
reason and withhold the affected endpoint.
