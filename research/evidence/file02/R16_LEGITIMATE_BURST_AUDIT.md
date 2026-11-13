# R16 — matched benign sustained, ramp and burst controls

Date: 2026-11-13. The [R16 plan](../../protocol/FILE02_R16_LOCKED_PLAN.md),
producer, validator and bounded ramp support were committed at `9947f2b`
before any R16 send. Private session:
`research/artifacts/file02/r16-20261113T122218Z-974de173`.
Top-level locked plan SHA-256:
`25ca8e2a211a90b187e744e79c4a2819b362a2e36f9ce8ad3f631cb2972bb3d7`.
Immutable raw manifest SHA-256:
`df7c8b5d34510f85faf577c50ff929f8e67a17e89d6acdc884734d265ff9e90b`.
`python3 research/tools/file02_r16.py run` exited 0: integration 0,
48/48 QC PASS, aggregate `BENIGN_WAVEFORM_CONTROL_MEASURED`.
Independent `python3 research/tools/file02_r16.py verify
research/artifacts/file02/r16-20261113T122218Z-974de173` exited 0,
rehashing raw and recomputing every output. The private session holds
top-level and per-run locked plans, immutable raw, per-run JSON/CSV, QC,
aggregate with registered run-level CI, numeric mempool curves, figure
status and decision.

Four independent logical fixture blocks each ran C/CP/P/H × sustained,
ramp and burst on a fresh four-validator single-host network. The three
arms in each matched profile/block used identical 41 transaction hashes,
genesis and wire size; only offsets changed. One warm-up and 40 steady
valid transfers were sent by validator-0 loopback RPC. Window counts
were `[10,10,10,10]`, `[4,8,12,16]` and `[0,20,0,20]`. Average offered
rate was 10 tx/s and the peak was 20 tx/s, within unchanged S06 caps.
Every scheduled window and zero window passed, all 41 transactions per
network committed exactly once with four-validator same-height
BlockID/AppHash, and no safety abort, observer or recorder drop occurred.

The registered primary effect is burst minus sustained **maximum sampled
accepted mempool bytes**, one difference per independent block. A positive
value means the benign burst had the larger sampled peak; a negative value
is retained as observed. The two-sided 95% intervals use 20,000 paired
block bootstrap resamples.

| Profile | Burst−sustained B, four blocks | Mean B | 95% CI B | Ramp−sustained mean B | Ramp 95% CI B |
| --- | --- | ---: | --- | ---: | --- |
| C | −148, 0, +296, 0 | +37 | [−111, +222] | +555 | [+370, +740] |
| CP | −5,136, +2,568, +5,136, +5,136 | +1,926 | [−2,568, +5,136] | +8,346 | [+2,568, +12,198] |
| P | −5,008, +12,520, −7,512, +5,008 | +1,252 | [−6,260, +8,764] | +10,642 | [+8,764, +12,520] |
| H | −2,568, −5,136, +2,568, −2,568 | −1,926 | [−4,494, +1,284] | +12,198 | [+10,272, +14,124] |

Across the 16 networks per waveform, mean p95 inclusion **upper bounds**
were 1,718/1,743/1,711 ms for sustained/ramp/burst; mean CheckTx
**socket** queue waits were 0.0420/0.0425/0.0442 ms. Mean sampled peak
mempool bytes were 30,216/38,151/30,538 B and mean peak sampled P2P
ingress was 2,657/4,226/2,756 B/s. These descriptive cross-profile
means are not causal contrasts because transaction sizes differ across
profiles. Per-run and per-profile values remain in the private artifacts.
The observed time from last send to first subsequent zero-mempool sample
was approximately 1.52–1.61 seconds, a sampling upper bound on drain,
not exact recovery time. No exact transaction queue residence or
peer propagation delay is available. Inclusion p99 is
`INSUFFICIENT_PRECISION`, represented as `null` with reason.

The ramp produced a higher sampled peak than sustained in at least one
block for each profile; the aggregate labels all four blocks/profile
as **benign false-alert candidates** under the registered descriptive
rule. Burst peak differences have CIs crossing zero. No classifier,
threshold false-positive rate or RQ2 result was evaluated. These
legitimate transients do not establish a knee, P2P-origin effect,
multi-host performance or DoS.

**R16 PASS for the bounded matched benign waveform control.**
