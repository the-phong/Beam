# Phase 05 research decision — evidence-limited recommendation

**Decision authority:** This document recommends a research direction to the
Independent Controller and Owner. It does not authorize a DoS mechanism,
admission policy, EVM, merge, production deployment or another phase.

## Decision rule recorded before final analysis

The result may be **GO** only if multiple independent runs show a repeatable,
material expensive-invalid class that a stated attacker can construct and a
matched legitimate workload shows service degradation beyond composition.
It may be **CONDITIONAL GO** if the cost asymmetry is reproducible but
attacker feasibility, causal attribution or service impact remains uncertain.
It is **NO-GO** if the observed data do not justify a DoS/detection
contribution under the tested bounded conditions. A roadmap mention of a
future mechanism is not evidence.

**Recommendation: NO-GO for a new DoS detection or mitigation mechanism on
the present evidence.** This is an evidence-limited research recommendation,
not a claim that no higher-rate or distributed attack could matter. The
definitive 122-run local matrix and five-sample microbenchmarks are described
in [`PHASE_05_REPORT.md`](PHASE_05_REPORT.md) and the raw-derived
[`RESULT_TABLES.md`](../evidence/phase_05/RESULT_TABLES.md). Failed exploratory
runs were excluded by the preregistered rule and retained locally.

## Questions the final evidence must answer

1. What B0/B1 cost difference was observed at equal transaction rate, and
   what part is attributable to ML-DSA verification rather than transaction
   size, registry, state hashing, storage or Comet scheduling?
2. Which structurally early, binding, classical-proof, ML-proof and
   state-dependent invalid classes cost most, with run-level uncertainty?
3. Which expensive classes can capability A, B or C actually construct under
   the immutable B1 genesis registry?
4. At fixed offered rate, did legitimate inclusion latency or confirmed
   throughput worsen beyond the arithmetic reduction in valid workload
   share? Were CPU/RSS changes repeatable?
5. Did all checked validators agree at matching committed heights, and were
   rejected inputs state-neutral?
6. Which untested multi-host, higher-rate, byte-matched or attack-feasibility
   comparisons would be needed before proposing or claiming a novel defense?

## Observed answer

At equal offered transaction rate, B1 valid-only median client admission
was 1,085.3 versus 787.5 µs (2 tx/s) and 1,047.6 versus 826.5 µs
(8 tx/s), based on five independent runs per condition. Direct complete
validation was 251.8 versus 125.4 µs/op. Standalone B1 ML-DSA-44 verify
was 49.8 µs/op, but the B1 result also includes binding, larger bytes,
different account state and persistence. An equal offered-byte-rate network
condition was unavailable under the 16-tx/s safety ceiling: matching 2 B1
tx/s of roughly 3.9 KiB requests would require roughly 50 B0 tx/s of
155-byte requests. The network delta therefore cannot be assigned to
ML-DSA alone.

The most expensive B1 invalid direct class here was the state-dependent
late reject at 244.0 µs/op; the invalid ML proof was 237.6 µs/op and an
empty structural input was 0.135 µs/op. The corresponding focused network
RPC medians were 1,085.2, 1,066.5 and 777.7 µs, respectively. A fully
signed state-invalid intent needs a registered malicious account (C) or
becomes invalid after a prior commit; a mutated ML proof attached to an
observed valid classical signature is feasible for observer/modifier B.
Unauthenticated capability A can send arbitrary malformed bytes but cannot
freely construct a registered B1 sender with a valid classical proof.

The 8-tx/s primary run's valid confirmed throughput moved from 5.209 tx/s
at 0% invalid to 2.686 at 50% and 0.559 at the actual 11/12-invalid
condition. These roughly follow the scheduled valid share. Its five-run mean
of per-run median inclusion latency was 1.135 s valid-only, 0.993 s at 50%,
and 1.122 s at 11/12 invalid. Those observations do not show repeatable
legitimate latency degradation beyond workload composition. At 100%
invalid, no valid transaction was submitted and valid inclusion latency is
UNAVAILABLE. CPU samples are short and combine Pluto with embedded
CometBFT; they are not a clean ML-DSA utilization measure.

The correctness oracle rejected all 1,012 expected invalid submissions;
all 452 expected valid submissions were included, and the four validators
agreed on block history, AppHash, balances, nonce and supply at matching
heights in all 122 final runs. Direct invalid ProcessProposal tests and
recorder ON/OFF state-equivalence tests passed separately. These are
software experimental observations, not a formal proof.

## Why NO-GO, and what would change the evidence

The measured late-invalid asymmetry makes further **measurement** useful,
especially for observed signed transactions, but the bounded 2/8-tx/s
single-host experiments did not demonstrate service degradation requiring
a new mechanism. p99 is INSUFFICIENT SAMPLES in every 12-request run; 3–5
independent repetitions and one WSL2 host do not support a strong tail or
capacity claim. The B1 account registry limits the attack construction
space, and the B0/B1 formats are not byte matched. Before a future proposal
could claim a novel defense, it would need a literature comparison with
existing rate limits and mempool controls, longer matched-valid traffic,
safe higher-rate and byte-matched trials, independent hosts, component
profiling, and a measured advantage over a simpler baseline. None of that
authorizes implementation in Phase 05. The historical DoS research aim
remains a hypothesis subject to new Owner authorization.
