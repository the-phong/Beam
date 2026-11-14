# Q02 — cross-profile impact decomposition, locked before execution

Status: LOCKED DESIGN, 2026-11-14. Twelve independent fixture blocks; each
block runs C, CP, P and H profiles, with a matched benign/structural-invalid
pair per profile on eight fresh four-validator networks (96 networks total).
Profile order rotates by block and arm order alternates. The same genesis,
software, seed, topology, six fixed legitimate planned offsets/account
intents and 14 background planned offsets are used across a block. The
profile-specific canonical signatures and wire lengths differ by design.

Each arm offers 20 tx/s for 12 steady seconds, including six fixed valid
transfers/s. The benign background is 14 distinct valid transfers/s. The
attack background is 14 distinct malformed payloads/s formed by structural
mutation of otherwise valid profile-specific bytes; actor A needs no
signing key for those bytes. Every invalid must fail at the structural stage
before crypto and never commit. C/CP/P/H expected wire lengths at this
16-byte chain ID are 148/2568/2504/2568 B, so cross-profile bytes/s are
2,960/51,360/50,080/51,360. **Within a profile pair**, offered bytes/s
match exactly. Across profiles, C and P are not byte matched to H; report
this as a causal limitation. CP and H have exact byte match.

The primary endpoint is the same RQ1b fixed-legitimate fraction first
observed on all four validators within 3,000 ms of the measured monotonic
socket write, one value per fresh network. For each profile p and block b,
define `D[p,b] = Y[p,benign,b] - Y[p,attack,b]`. Report mean D and two-sided
95% percentile-bootstrap CI for each profile over **12 block units**. At
each condition report C→CP, CP→H and P→H differences in Y as secondaries.
The registered difference-in-differences is
`(H_attack-H_benign)-(C_attack-C_benign) = D[C]-D[H]`, in **proportion
points**, with a 12-block cluster bootstrap CI. Do not treat 72 fixed
transactions as CI replicates. Bootstrap 20,000 resamples, seed 20261114.
No post-data threshold, pooling across attack classes or replacement run.

Each arm uses one-second warm-up, 12-second steady, five-second drain and
two-second cooldown; 20 tx/s, at most 51,360 B/s, 16 outstanding and
60-s/run caps remain unchanged. RPC ingress is validator-0 loopback on one
physical host. QC requires 241 planned/actual sends, 73/241 exact-once
valid commits in attack/benign arms, 168 stage-correct invalid rejects in
each attack arm, 6 fixed plus 14 background sends in every steady window,
all four validators agreeing on BlockID/AppHash/state, full 3,000-ms
observation of all fixed transactions, zero recorder/observer drops, source
and config hashes, no safety abort and frozen raw verification. A failed arm
excludes its entire eight-arm block from the DiD; do not replace it.

This test checks profile interaction for **structural-invalid input only**.
It cannot attribute an effect to ML-DSA verification, explain late-invalid
A04/A05 behavior, establish p99 precision, sustainable throughput, knee,
P2P-origin, independent-host behavior or DoS.

## Order-balance correction, 2026-11-14

The first implementation of `schedule.v1` inadvertently made arm-first order
constant within each profile (C/P always benign-first, CP/H always
attack-first), despite the design above. Its frozen sessions and published
technical QC remain preserved, but the first complete session is superseded
for cross-profile inference. `schedule.v2` fixes the implementation so every
profile has six benign-first and six attack-first pairs across the same 12
rotated fixture blocks. The load, endpoint, contrasts, bootstrap, exclusion
rule and safety caps above are unchanged. A new, separately locked full
session is required; no historical raw is edited or silently substituted.
