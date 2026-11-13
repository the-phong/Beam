# Gate RQ1 — bounded Owner acceptance

Decision date: 2026-11-13. The Owner accepted the Checklist 02 section 4
R01–R16 group with the verified dispositions below. This is an acceptance
of bounded evidence and its declared gaps, not a claim that all case
requirements passed. The [resource report](RQ1_RESOURCE_REPORT.md) and
individual case audits remain the measurement record; no historical raw,
analysis result or accepted RQ1 hypothesis was rewritten for this decision.

| Disposition | Cases | Checklist treatment |
| --- | --- | --- |
| PASS (8) | R01, R02, R07, R08, R09, R14, R15, R16 | Checked within each audit's stated scope. R15 PASS is a valid negative knee observation in the tested safe range. |
| PARTIAL (6) | R03, R04, R06, R10, R11, R13 | Remain unchecked; missing endpoints, unmatched causal control or unobserved precondition are retained in the case audits. |
| UNSUPPORTED (2) | R05, R12 | Remain unchecked; policy control arms and direct P2P-origin injection/oracle are absent. |

The accepted RQ1 H-minus-CP estimate is **+81.47 microseconds per valid
transaction**, paired 95% block-bootstrap CI **[+75.16, +87.26]**
microseconds, from [R06](R06_EQUAL_RATE_AUDIT.md): 12 matched blocks,
48 fresh four-validator networks at 20 valid tx/s per profile, exact
2,568-byte CP/H transfers and RPC-origin injection on one physical host.
This supports a positive application CheckTx service-cost difference in
that measured stratum. R06 remains PARTIAL because the checklist's
"below knee" predicate could not be established. The independent
matched-byte [R07](R07_EQUAL_BYTES_AUDIT.md) low-rate contrast supports
the direction but is not a capacity estimate.

[R15](R15_BENIGN_CAPACITY_AUDIT.md) tested valid-only 20/23/25 tx/s levels
and concluded `NOT_OBSERVED_WITHIN_SAFE_RANGE` for C/CP/P/H. Neither 23
nor 25 tx/s is labeled a knee. The short runs do not establish
sustainable steady-state throughput. CheckTx socket wait is not total
transaction queue residence, and sampled inclusion latency is an upper
bound rather than an exact commit timestamp.

The acceptance makes **no DoS conclusion** and no claim about multi-host
performance, distributed behavior, direct P2P-origin effects, exact
cross-host latency or RQ1b legitimate-service degradation. RQ1b remains
a separate question. This decision does not open A/Q/D cases, ML training,
new R experiments or higher safety caps. The six PARTIAL and two
UNSUPPORTED requirements retain their explicit gaps until separately
authorized and evidenced work resolves them.

## Integration provenance

- PR #23 merged the RQ1/RQ1b planning baseline into `test-mainet` at
  `45495c56319b7f61619e7cd6b00208e2af7b0f3c`.
- PR #24 merged the R01–R16 result tree into the planning branch at
  `7ae6fa82aa56d8d39cc4a848b8f5dd1be90ebd24`.
- [PR #25](https://github.com/the-phong/Beam/pull/25) merged the result
  tree into `test-mainet` at
  `d67fef2598f278015968353763357f6aed28250d`. Its two `verify`,
  `scan` and `smoke` checks passed; the
  [post-merge verify](https://github.com/the-phong/Beam/actions/runs/43450981784)
  also passed. The merged tree matches the source tree.
- The acceptance was explicitly given by the Owner after integration.
  No additional confirmatory run was performed for this closeout.
