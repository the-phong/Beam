# Proposed separate question: legitimate service under invalid hybrid load

Status: **PROPOSAL FOR CONTROLLER/OWNER DECISION**, 2026-11-13. This is not
an amendment to the accepted RQ1 hypothesis, a locked case plan, or authority
to execute T/R/A/Q/D cases. It specifies the decision that would be needed
before a service-degradation claim.

## Question and causal contrast

Does adding deliberately invalid hybrid (H) transaction traffic degrade the
service received by a fixed stream of legitimate H transfers, compared with
a matched benign background stream? This is an attack-versus-benign effect,
separate from the accepted RQ1 **H minus CP valid CheckTx service-time** cost
contrast. The latter remains a mechanism study and cannot by itself answer
the service question.

The treatment and control use the same H application, four-validator
topology, genesis, legitimate transaction bytes/accounts/nonces, legitimate
offered schedule, collector, mempool/block settings and safety caps. Treatment
adds an explicitly named invalid-H family; control adds a benign H background
chosen before execution. Match total offered tx/s and encoded bytes/s where
feasible, and report which dimensions cannot be matched. Extra control
transactions need separate accounts so their valid state effects cannot
consume the fixed legitimate stream's nonces or balances. Do not pool attack
families or topology/source strata. These controls estimate the effect of
that specified workload contrast; they do not isolate cryptographic cost.

## Proposed primary endpoint and hypotheses

For each independent run, let `Y` be the fraction of distinct fixed-stream
legitimate transactions sent during the preregistered steady window that
are **observed included by a fixed deadline** `tau_ms` after each monotonic
send timestamp. The block-scan observation gives an inclusion **upper-bound
timestamp**, not exact commit time. The locked case plan must choose
`tau_ms` from a prior benign calibration, publish the scanner interval and
timestamp uncertainty, and apply the identical rule in treatment and
control. Transactions whose deadline cannot be observed through the drain
window are censored and make the run unusable for this endpoint; they must
never be counted as failures or successes by default. The observation window
must include the full deadline for every included send.

Define `D = mean(Y_control - Y_treatment)` across paired, independently reset
network runs. Proposed directional hypotheses are `H0: D <= 0` and
`H1: D > 0`. Report the effect size and a run-pair-level 95% confidence
interval using a method fixed in the locked plan. Reject H0 only if the
lower CI bound is greater than zero and every pair passes oracle, rate,
provenance, safety and measurement QC. An upper bound at or below zero
falsifies the proposed directional effect in the tested stratum; an interval
crossing zero is inconclusive. A positive result is evidence for this
bounded service contrast, **not** by itself a DoS or capacity-knee finding.
Any practical SLO threshold or minimum important drop must be fixed before
case execution and reported separately; it cannot be chosen from the result.

Required secondary outcomes are achieved legitimate/invalid tx/s and
bytes/s, eventual included count, observation-limited inclusion latency,
accepted mempool occupancy, CheckTx service time, CheckTx socket queue wait,
CPU and RSS. Eventual goodput cannot substitute for sustained steady-state
throughput; socket queue wait cannot represent the full transaction queue.
Unavailable outcomes remain `null` with reasons. Multi-host and P2P-origin
claims remain outside this proposed single-host RPC contrast.

## Decision needed before Gate COMMON or case planning

Controller/Owner may accept this as a separately numbered study question,
revise its comparator/endpoint or keep it deferred. Acceptance would need a
versioned protocol and Checklist 02 amendment, a case-specific locked plan
with `tau_ms`, independent-run count, CI method, workload family and matched
rate, then the later T/R/A/Q/D gates. This proposal does not silently change
the accepted RQ1 H0/H1 or grant a new experiment.
