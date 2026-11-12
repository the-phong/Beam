# RQ1 wording consistency review — decision needed before RQ claims

`RESEARCH_PROTOCOL.md` defines the accepted primary RQ1 contrast as **H
minus CP mean application CheckTx service time for valid accepted transfers**
at matched offered tx/s and exact wire length. Queue wait is separate;
legitimate goodput, latency, CPU and mempool are secondary. Its H0/H1 and
run-level 95% CI decision concern that valid-transaction service time.
`FILE02_COMMON_CONTRACT.md` summarizes the same primary endpoint and adds
resource/service secondary outcomes. Checklist 02 section 0 calls RQ1
resource impact across four profiles. These three documents broadly align
on a controlled resource-cost question.

The Owner's current wording, **degradation of legitimate transaction service
under hybrid-invalid load**, is a distinct attack-versus-benign causal
question. It changes the treatment (invalid H traffic), comparator
(matched benign pressure), primary endpoint (legitimate service degradation)
and possibly the null hypothesis. The accepted RQ1 primary contrast cannot
answer that question by itself. Checklist 02's attack rows and RQ2 detection
work anticipate that scenario but do not replace the accepted RQ1 decision
rule. The bounded benign T06 pilot answers neither question.

| Contract | Treatment and comparator | Primary endpoint | Decision |
| --- | --- | --- | --- |
| Accepted RQ1 | Valid H minus valid CP at matched intent, offered rate and wire length | Application CheckTx service time for valid accepted transfers | Run-level 95% CI lower bound > 0, with equivalence and oracle QC |
| Proposed separate service question | Fixed legitimate H stream plus invalid-H background versus the same legitimate stream plus matched benign-H background | Legitimate deadline-inclusion fraction using the same observation bound in both arms | Paired independent-run 95% CI lower bound for control minus treatment > 0, with complete-window and run QC |

The [decision-ready proposal](SECTION2_SERVICE_DEGRADATION_PROPOSAL.md)
spells out the candidate hypotheses, endpoint, matching, censoring, CI rule
and limits. Its deadline, matched rate, run count and attack family still
require prospective case-specific locking; no result is inferred here.

**Proposed Controller/Owner decision:** keep the accepted RQ1 valid-cost
contrast as a mechanism study and accept, revise or defer the separately
numbered attack-impact proposal. An alternative would be an explicit RQ1
amendment, but that would change the accepted hypothesis and decision rule.
The present work makes neither normative change. No official case, effect
estimate or DoS conclusion is authorized by this review.
