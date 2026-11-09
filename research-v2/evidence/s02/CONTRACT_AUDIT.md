# S02 contract drafting audit — 2026-11-09

## Provenance and result boundary

- Source baseline: `test-mainet` `0bdd869f2c6b3ed59b0ae825954d8667b3c7481e`; S01 checkpoint `484c64788883a482c0e8fdba49f80a0e2126617b` on `research/reproducible-testbed`.
- Result: three proposed contract documents and five completed drafting items. **Gate S02 remains pending independent review** under the current `AGENTS.md`; this author's audit is not Controller/Owner acceptance.
- No experimental profile, detector, high-load run, RQ result, multi-host measurement or S03 implementation is claimed here. Historical Phase 05 results are excluded from this study's inferential dataset.

## Item-by-item trace

| Item | Reviewable artifact | Audit question answered |
|---|---|---|
| S02.1 | [`RESEARCH_PROTOCOL.md`](../../protocol/RESEARCH_PROTOCOL.md#research-questions-and-falsifiable-decisions) | Exactly RQ1, RQ2, RQ3; primary contrast, H0/H1, reject/falsify/inconclusive criteria, independent-run inference and study-specific numeric thresholds. |
| S02.2 | [`THREAT_MODEL.md`](../../protocol/THREAT_MODEL.md#mechanisms-that-must-remain-separate) | Rejected verification work, actual accepted-list bytes/count and measured P2P relay each require distinct observation; rejection count alone is insufficient. |
| S02.3 | [`THREAT_MODEL.md`](../../protocol/THREAT_MODEL.md#actor-capabilities) | Actors A/B/C have feasible permissions and limitations; benign high/burst demand and accidental client errors remain possible. |
| S02.4 | [`RESEARCH_PROTOCOL.md`](../../protocol/RESEARCH_PROTOCOL.md#four-controlled-authentication-profiles) | C, CP, P and H share an intended transfer/state/consensus path; CP is valid canonical zero padding exactly size matched to H, excluded from its secp digest like H's ML proof; S03 must prove this implementation property. |
| S02.5 | [`MEASUREMENT_CONTRACT.md`](../../protocol/MEASUREMENT_CONTRACT.md) | Controls, measurement point and first availability for each metric, early alert/no-future rule, correctness/QC, FPR/cost limits and isolated-network safety stops are explicit. |

## Repeatable document checks

At the S01 source commit and current S02 working tree, `make verify` exited **0** on Go 1.24.1/linux-amd64. It built Pluto and CometBFT, ran `go test ./...` (all existing packages passed, cached where reported), `go vet ./...`, `go mod verify` (`all modules verified`) and `plutod version` (`Revision: 484c647`). No source or dependency changed in S02; this command establishes a regression check, not scientific validity.

`sha256sum research-v2/protocol/*.md`:

```text
403fb8ba968612686108a154080c06af891c629bba906456291c0e9544dc63fa  research-v2/protocol/MEASUREMENT_CONTRACT.md
e2481fd23f67934dc0480cfca72cb8e4feb52a1ec9056635e6e893942dac88de  research-v2/protocol/RESEARCH_PROTOCOL.md
1d3981c325e836b26be73abf5c9ea2a66d341519b9f35a269b1882c193afde96  research-v2/protocol/THREAT_MODEL.md
```

The local checklist check confirmed **five** checked S02 items, Gate `CHỜ DUYỆT ĐỘC LẬP` in both checklist copies and zero checked S03 items. The first local-link check reported only this audit file missing because it had not yet been written; rerun after creation is required. `git diff --check` exited 0 before this audit file was added. The proposed contract cites pinned CometBFT v0.38.26 and final NIST FIPS 204; its numerical alarm and safety limits are **study design choices**, not externally validated operational thresholds.

## Independent review request

The reviewer should check the causal contrast and CP byte equivalence, actor/corpus feasibility, detector target and confidence design, service-degradation calibration, no-future timing, and safety caps. A review decision can approve Gate S02 or identify precise corrections. No S03 task opens until the reviewed contract and both checklist copies record `Gate S02: OK`.
