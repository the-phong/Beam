# S02 threat model and workload labels — proposed contract

Status: S02 design candidate for independent review. This model is for an isolated Pluto/CometBFT research network, never a public node or third-party target. The defender consists of four validators running one experimental profile at a time, their RPC/P2P ingress, CometBFT's flood mempool and Pluto's ABCI++ application. The attacker controls only the sender capabilities listed below; it does not control consensus validity rules, validator keys, a majority of validators, the clock, or the collector. The new profile registry is a trusted genesis input, not an attacker-editable runtime service.

## Actor capabilities

| Actor | May do | Cannot do without a higher capability |
|---|---|---|
| **A — arbitrary sender** | Submit arbitrary transaction bytes and distinct malformed variants to allowed isolated RPC endpoints. P2P-origin input is an additional source only if the S06 testbed explicitly provides a controlled peer and records that path. | Produce a valid registered-account signature, spend another account's balance, or force an input past Comet precheck/cache. |
| **B — signed-transaction modifier** | Observe a valid signed experimental transaction and alter/replay its wire bytes or either proof while keeping the original canonical intent available. For H, changing only ML proof bytes can preserve a valid secp proof if S03's signature digest excludes proof bytes. | Sign a new intent, advance a nonce legitimately, or replace registered keys. |
| **C — registered key holder** | Generate fresh valid profile-specific transfers and unique nonces from accounts with the required registered key(s); deliberately sign a transaction that is state-invalid after its cryptographic checks; generate controlled high legitimate or adversarial offered load. | Change the fixed genesis registry, forge another account, or change validator consensus rules. |

An actor can use multiple client processes only if S06 records them as a distinct factor. Source address, account ID or signature type is **not** ground-truth maliciousness. Key ownership proves capability, not intent. A legitimate user producing a high-rate burst or accidentally malformed request may be benign.

## Mechanisms that must remain separate

| Mechanism | Required observation | What does **not** establish it |
|---|---|---|
| **Pre-admission verification exhaustion** | Distinct inputs pass structural/precheck/cache and reach an expensive verification stage; they are ultimately rejected by CheckTx or proposal validation. Measure actual stage service time, validator CPU and legitimate service effects. | Rejection count alone, malformed input rejected before crypto, or raw RPC throughput. Rejected transactions do not count as accepted mempool occupancy. |
| **Accepted-mempool byte/count pressure** | Unique valid transactions receive CheckTx `Code=0`, enter the accepted list, and raise the observed accepted count or bytes before inclusion/recheck. Actor C must supply enough distinct authorized intents and nonces. | Invalid traffic rejected by CheckTx, cache hits, or a large client send queue. |
| **P2P relay congestion** | Actual peer ingress/egress bytes and propagation/queue delay on the controlled P2P channel are observed while accepted transactions relay between validators. | RPC request size or latency, block gossip, or a successful localhost relay smoke without measured loaded P2P traffic. |

These mechanisms may co-occur, but every run states its primary mechanism and reports the others rather than merging them into one “mempool overflow” category. CometBFT v0.38.26 admits only successful CheckTx responses to its mempool, may recheck after Commit, and uses ProcessProposal as a separate validity boundary. [Pinned CometBFT ABCI++ requirements](https://github.com/cometbft/cometbft/blob/v0.38.26/spec/abci/abci%2B%2B_app_requirements.md).

## Workload feasibility and labels

| Workload class | Capability / profile | Expected first relevant boundary | Ground-truth treatment |
|---|---|---|---|
| Benign low, high and burst | C; each of C/CP/P/H | Valid CheckTx, accepted mempool, then commit | `BENIGN` when generated as ordinary demand, including high load. The validator must commit correctly or the run fails correctness QC. |
| Accidental malformed/client error | A; any profile | Structural or Comet precheck | `NATURAL_REJECTION` or `BENIGN` only when the scenario records a non-adversarial cause; do not infer attack from nonzero code. |
| Malformed/oversized/duplicate flood | A; any profile | Comet precheck/cache or Pluto structural check | `ATTACK_KNOWN` only for a preregistered adversarial sending scenario. Expected to be cheap if rejected early; cannot be called crypto exhaustion without stage evidence. |
| Bad secp proof | B modifying C/CP/H | Secp verify, if earlier canonical/binding checks pass | `ATTACK_KNOWN` when deliberately generated. Not a P-only case. The oracle must prove the secp stage was reached. |
| **Late bad ML proof after valid secp** | B modifying H; C may also produce its own bad proof | ML verify, after successful secp | Primary RQ2 adversarial class. S03 must prove secp still verifies for each mutated wire; otherwise the case is misclassified and excluded. Not a C/CP/P case. |
| Bad ML proof in P | B modifying P | ML verify | Separate P-only class; no secp stage exists and it cannot replace the H primary class. |
| Validly signed wrong nonce, insufficient balance or unregistered recipient | C; any profile where codec allows | State check after required proof(s) | `ATTACK_KNOWN` only under an adversarial scenario; accidental nonce errors are `NATURAL_REJECTION`. A/B cannot create a new signed late-state-invalid intent. |
| High valid transaction rate | C; any profile | Accepted mempool and commit | `BENIGN` for matched ordinary high demand; `ATTACK_KNOWN` only when a distinct preregistered adversarial scenario adds rate to a fixed legitimate stream. Bytes and signatures alone cannot distinguish the two. |

Every corpus case records `case_id`, actor capability, profile, immutable input bytes/hash, expected code and last stage reached, source path, prerequisite state, scenario label and oracle result. The oracle validates the **actual** stage and state outcome; a descriptive label is never derived solely from the error code or from future measurement outcomes. A duplicate/cache-hit case is not evidence of cryptographic work. No case may claim to reach ML verification unless the code path proves all earlier checks succeeded.

## Ground truth and service degradation

The generator owns the run schedule and records when the first adversarial input was **actually** sent. The independent oracle checks profile validity, transaction uniqueness, CheckTx/ProcessProposal/commit outcomes and state neutrality of rejected inputs. The scenario label is fixed before the run; the oracle can invalidate a mislabeled case but cannot relabel it after seeing performance. Mixed runs keep a fixed legitimate offered stream and add adversarial traffic; a separate benign high-load run matches the declared pressure variable.

An **attack attempt** is a preregistered adversarial scenario within A/B/C capabilities. A **measured service-degradation event** is the experimental threshold defined from disjoint benign calibration runs in `MEASUREMENT_CONTRACT.md`, sustained for its specified windows. A **DoS finding** needs both a controlled attack effect on legitimate service and evidence of saturation/causal contrast under safety limits; neither an attack label nor a CheckTx reject is sufficient. If no safe saturation point is reached, report the tested range and `NOT ESTABLISHED`.

## Boundaries and safety

The actor may target only a private, allowlisted loopback or VM network owned for this study. No public chain, public RPC/P2P endpoint, third-party IP, validator private key or real user data belongs in the generator or published artifacts. S06 must enforce endpoint allowlists, bounded rates/bytes/duration/queue, stop controls and private key handling before any high-load run. The registry and account balances are fixtures. A validator loss, state divergence, safety-cap trip or generator saturation invalidates the run for causal analysis and triggers the stop rules in `MEASUREMENT_CONTRACT.md`.
