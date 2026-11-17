# X04 — locked prospective mechanism/service extension protocol

Status: **LOCKED DESIGN BEFORE ANY NEW X MEASUREMENT**, 2026-11-15, under
[D-045](../DECISIONS.md). This protocol creates a new RQ-Mechanism and new
continuous service estimands. It does **not** amend accepted RQ1 H−CP,
RQ1b's fixed legitimate H stream, or the historical 3,000-ms endpoint and
its negative result. It is a design lock, not evidence that the new
instrumentation or workload has passed QA. Each X10–X13 case must bind its
exact committed source, plan, fixture, n, arm order and QC before sending.
Any design correction must be committed and fully disclosed before new
confirmatory data; no outcome-dependent alteration or selective replacement.

## Units, interventions and estimands

All service observations concern six **fixed valid H transactions/s** sent
through validator-0 loopback RPC on four validators sharing one physical
host. The independent statistical unit is a fresh network within a matched
fixture block. Transaction observations within a network and four
validators on the same host are correlated measurements, not independent
replicates. Let `Y` be a per-run metric for that fixed legitimate stream.

| Estimand | Intervention contrast and sign | Measurement / interpretation |
|---|---|---|
| `validation_cost_H_minus_CP` | H−CP on valid equal-wire-size transactions | Historical accepted RQ1 application CheckTx wall result is +81.47 µs/tx [75.16, 87.26]; remains separate. New isolated CPU/stage attribution requires X05/X06 and matched inputs. |
| `effect_L_to_I` | `Y(I) − Y(L)`; L = six valid H/s only, I = six fixed valid H/s + 14 late-bad-ML H/s | Total effect of **adding invalid background** and raising offered total from 6 to 20 tx/s. It is not rate-matched to L. |
| `effect_B_to_I` | `Y(I) − Y(B)`; B = six fixed valid H/s + 14 distinct valid H/s | Total **replacement effect** at matched 20 tx/s and 51,360 encoded B/s. It includes differences in CheckTx, admission, relay, proposal and commit work. It is not pure crypto cost. |
| `stage_cost_invalid_class` | Per-class invalid CheckTx wall and independently validated CPU if available | Structural, bad secp, bad ML, state and mixed are separate classes; no post-hoc pooling or conversion of wall spans to CPU. |
| `service_degradation` | Prospectively defined sustained miss event below | A candidate research SLO event; never labeled DoS without separate safety/mechanism evidence. |
| `early_detection` | Features timestamped before measured event onset | Eligibility only; `NOT_ESTABLISHED` and lead time `null` when no event occurs. D/ML are not opened by this protocol. |

For service `Y`, the prospective **primary continuous metric** is the
network-level arithmetic mean in milliseconds of each fixed valid H
transaction's first-all-four inclusion **observation upper bound** minus
its monotonic socket-write time. The primary contrast is `I−B` in paired
fixture blocks. `I−L`, per-run p50 and p95 upper bounds, and the unchanged
fraction observed by 3,000 ms are secondary. Report both the registered
binary endpoint and new continuous metrics without substituting one for
the other. A 100-ms scanner does not yield exact commit time; X08 must
record its uncertainty and any improved timestamp method before a new
continuous result is confirmatory.

## Practical margins, precision and prospective decision rules

The **research-relevant mean-latency margin** is +100 ms for `I−B`. This
is a prospective engineering decision bound of one nominal historical
scanner interval and about 3.3% of the accepted 3-s deadline; it is **not
a production SLO**. It was set after seeing X03 exploratory data, so any
claim of fully outcome-independent margin selection is prohibited. The
confirmation set must consist only of new, independently planned networks;
X03 contributes only to precision planning. A positive **research-relevant**
service-latency effect requires a paired 95% CI for `I−B` with lower bound
>+100 ms. A CI entirely above zero but crossing +100 ms supports a direction
of effect while leaving its practical size inconclusive. A claim of no
research-relevant mean impact requires
the entire two-sided 95% CI inside [−100, +100] ms on those **new** runs,
plus QC and no material discordance in the 3-s miss endpoint. Otherwise
the continuous result is inconclusive. These rules are research decisions,
not clinical/production equivalence or a DoS threshold. The three-case X03
intervals are not eligible for this decision.

The target half-width for the new paired mean `I−B` interval is ≤40 ms.
X03's largest exploratory paired-mean SD among A04/A05/A08 was about
60 ms, yielding `ceil((1.96×60/40)^2)=9` pairs under an optimistic normal
approximation. **24 matched fixture blocks** are locked for the primary
L/B/I experiment to allow measurement-architecture variance and QC loss,
with **no outcome-based stopping or replacement**. Each block runs L, B and
I on fresh networks with one of the six arm permutations, four blocks per
permutation, and the same fixed legitimate hashes/intents/offsets/genesis.
If instrument QA reveals variance or clock uncertainty that makes the
40-ms goal unattainable, pause before confirmatory sends and publish a
prospective revision; do not expand n after inspecting confirmatory effects.
For X11 class controls, use separate preregistered 12-pair diagnostic
contrasts unless a later pre-send precision calculation locks a larger n;
their CIs are descriptive, not equivalence claims. X12/X13 require their
own pre-send plan and cannot borrow X10 blocks as if independent.

The candidate research service event for future RQ2 eligibility is:
in **two consecutive nonoverlapping 3-s send windows**, the fixed valid H
observed-by-3,000-ms fraction is ≤0.90 (18 planned fixed transactions per
window), with complete 3-s follow-up, oracle PASS and no observer/generator
failure. Event onset is the end of the second window's observation deadline,
not the attack start. This is a prospective measurement trigger, not an
accepted production SLO. If no event is measured in valid X runs,
early-degradation RQ2 is `NOT_ESTABLISHED`, lead time is `null`, and no D
training or efficacy number may be reported. Features must exist strictly
before event onset with their acquisition latency included.

## Workload, topology, safety, QC and inference lock

L offers 6 H-sized tx/s = 15,408 B/s; B and I offer 20 H-sized tx/s =
51,360 B/s (2,568 B each). The legitimate six/s use distinct funded
senders and fixed signed intents; 14 background/s use disjoint senders
or malformed bytes. Offered socket writes, RPC responses, achieved
one-second rate/bytes and all four validators' eventual inclusion are
separate fields. L is not equal-rate to B/I; B/I must match bytes and tx/s.
The same genesis and environment are reset in each fresh network; balanced
order controls drift. Reject any session that violates preregistered
paired-hash/offset, arm order, achieved rate, exact-once validity,
stage-specific invalid rejection, four-validator AppHash/state, observer
drops, clock continuity or safety guard. Failed/excluded sessions remain
frozen and are not silently replaced. An independent verifier must rederive
QC and aggregates from the immutable raw manifest. Missing CPU/queue/peer
latency is `null` with a reason, never zero.

Hard limits remain **≤25 tx/s, ≤65,536 B/s, ≤16 outstanding, ≤60 s/run**
plus all S06 CPU/RSS/disk/stale-observer aborts. Baseline 20 tx/s is first;
23/25 tx/s is only a separately locked sensitivity study after X10 QA,
never a knee search or a reason to raise caps. Use single-host private
loopback RPC validator-0 origin only. Direct P2P source, independent
physical hosts and cross-host latency require separate infrastructure and
authority; they are outside this extension.

The primary CI uses paired fixture-block resampling, 20,000 bootstrap
draws with a fixed seed recorded in each case plan. Report point estimate,
CI, n, arm order, excluded blocks and raw manifest. For X11's multiple
invalid-class comparisons, report each class separately and use a
predeclared Holm familywise adjustment for a joint positive claim; unadjusted
CIs remain labeled exploratory. No exact finality timestamp or sustainable
throughput claim follows from short runs. CPU process deltas, crypto-stage
profiles and queue stages require X05–X09 measurement validation before
mechanism attribution. X18 must be willing to conclude bounded negative,
partial or unavailable. D01–D13 remain closed.
