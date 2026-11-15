# X10 — locked L/B/I service experiment

Status: **LOCKED DESIGN BEFORE NEW CONFIRMATORY LOAD**, under X04 protocol.
The source, tool, schedule and per-run research plan must be committed and
bound before the first send. Historical A04/Q02 runs are planning evidence
only and cannot enter this confirmation set.

The unit is one fresh four-validator single-physical-host loopback network.
There are 24 matched fixture blocks, each with three separate networks:

| Arm | Fixed H stream | Background | Steady offered total |
|---|---:|---:|---:|
| L | 6 valid H/s | none | 6 tx/s, 15,408 B/s |
| B | 6 valid H/s | 14 distinct valid H/s | 20 tx/s, 51,360 B/s |
| I | 6 valid H/s | 14 distinct late-bad-ML H/s | 20 tx/s, 51,360 B/s |

The three arms of each block use the same 241-account genesis, fixed signed
H bytes and hashes, sender identities, planned offsets and chain ID. B/I
also use identical 20 tx/s offsets and wire size (2,568 B). L is a
no-background intervention and is intentionally lower-rate. The A04
late-bad-ML oracle is reused only for fixture construction; new X10 data
are separate. One warm-up second, 12 steady one-second windows and five
drain seconds are followed by two seconds of cooldown. No run exceeds
60 seconds, 25 tx/s, 65,536 B/s or 16 outstanding; S06 CPU/RSS/disk/
observer aborts remain active. RPC origin is validator 0 only.

All six arm permutations occur four times each. The schedule shuffle is
deterministic seed 20261115. The source commit, module/config SHA, host,
schedule, n and exclusion policy are locked in a private session plan;
each network binds a common schema plan after Comet config generation
and before sending. A failed network remains in frozen raw, excludes its
entire block and is not replaced after looking at outcomes.

Primary estimand: paired block mean `I−B` of fixed valid H send-to-first-
all-four block-index **observation upper bounds** in ms, using actual
monotonic socket-write timestamp and four per-validator monotonic scan
stamps. Secondary `I−L`, run p50/p95, original by-3,000-ms timely
fraction, rejection stages, app/Comet process CPU and sampled queues.
Do not infer exact commit time or whole-queue latency. Compute 20,000
paired bootstrap resamples seeded 20261115. The X04 +100-ms practical
margin and ≤40-ms target half-width apply only to this new set. A
relevant positive effect needs CI lower >100 ms; no relevant mean impact
needs full CI inside [-100,+100] ms and no binary-endpoint conflict.

QC requires 72/72 planned independent networks or a documented excluded
block, all planned socket writes and exact 6/0 or 6/14 one-second rates,
matched fixed stream/genesis and B/I bytes/offsets, all valid exact-once
commit, all 168 late-bad-ML transactions per I run rejected at ML stage
and absent from committed blocks, four-validator block/AppHash agreement,
zero observer/recorder drops, continuous local monotonic clocks, source/
config binding and unchanged safety caps. Derived results are withheld if
any required QC fails; no outcome-based rerun/replacement.

A one-block `FILE02_X10_QA_LIMIT=1` harness dry run may be executed first
with `scope=QA_ONLY` to validate the dedicated-process wiring and file
schema. Its data are frozen separately and excluded from all confirmatory
contrasts. The confirmatory runner never sets that variable and requires
all 72 networks.
