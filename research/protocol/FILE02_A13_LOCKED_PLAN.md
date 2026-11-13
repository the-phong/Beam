# A13 — attacker start/stop dynamics, locked before execution

Status: LOCKED DESIGN, 2026-11-14. Four independent fixture blocks each
run a matched benign/attack pair in balanced order on eight fresh
four-validator single-host loopback-RPC networks. All arms have one H
warm-up transaction, 12 steady seconds, five-second drain and two-second
cooldown. Six fixed legitimate H transactions/s remain identical in bytes,
hashes, genesis and offsets throughout both arms. The background is off
for three seconds, on for six, then off for three. During the on phase,
benign adds 14 valid H tx/s; attack adds 4 structural-invalid, 5 bad-ML
and 5 signed nonce-invalid tx/s. Every on window offers 20 tx/s or
51,360 B/s; every off window offers six tx/s or 15,408 B/s. The scheduled
attack starts at 4 s after the run's load origin and stops at 10 s.
The Go plan's `step` waveform is scoped to A13 and still enforces unchanged
25 tx/s, 65,536 B/s, 16-outstanding and 60-second safety caps.

The primary effect is benign-minus-attack fraction of 36 on-phase fixed
legitimate transactions observed included on all four validators by
3,000 ms after socket write. Four paired fixture blocks yield a two-sided
20,000-resample percentile-bootstrap CI, seed 20261113. Baseline and
recovery-phase fractions (18 fixed transactions each), inclusion upper
bounds and sampled mempool/CPU/CheckTx socket-queue curves are secondary.
The 3,000-ms endpoint is a prospectively chosen research deadline, not a
production SLO. No new alert threshold is fitted after data.

The case records the **scheduled** attacker start/stop times separately
from observed service onset. If no endpoint degradation occurs, service
onset is `NOT_OBSERVED` and a recovery time from degradation is
`NOT_APPLICABLE`, never zero. No accepted detector or alert rule is active;
alert lead time and false-alert transients remain `null` with reason
`UNAVAILABLE_NO_ACCEPTED_DETECTOR`. This condition keeps A13 technically
partial even if the workload/phase/QC runs pass.

QC requires all planned per-second windows, 157 sends/arm, 157 benign or
73 attack exact-once valid commits, 84 attack invalid stage-correct rejects
only in on windows, identical fixed stream/genesis and offered bytes per
paired arm, all-four BlockID/AppHash and account-state agreement, no
observer/recorder drops, safety abort or censored fixed observation, and
intact immutable raw. No failed pair is replaced. No knee, DoS,
P2P-origin or multi-host inference follows.
