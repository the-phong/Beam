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

**Proposed Controller/Owner decision:** keep the accepted RQ1 valid-cost
contrast as a prerequisite/mechanism study, and add a separately numbered
attack-impact question with preregistered matched benign control, fixed
legitimate offered stream, independent-run service endpoint, degradation
threshold, CI and safety gate; or explicitly amend RQ1's primary endpoint
and H0/H1. The present work makes neither normative change. No official
case, effect estimate or DoS conclusion is authorized by this review.
