# M01-03/M01-04 — artifact classification and B0/B1 comparison

Baseline: `test-mainet` commit `0bdd869f2c6b3ed59b0ae825954d8667b3c7481e`.
Classification applies to the new RQ1–RQ3 program, not to historical phase
acceptance. No historical result is imported as a new experiment.

## Artifact classification

| Artifact group | New-program classification | Evidence and reason |
|---|---|---|
| Go source, pinned modules, unit tests | Independently reproduced | `make verify` exited 0 on baseline; `BASELINE.md` records source/tree and 80 enumerated test/benchmark/fuzz names. This establishes build and tested behavior only. |
| B1 four-validator RPC/consensus/commit path | Independently reproduced for its bounded claims | `go test -tags 'integration fourval' ./test/integration -run '^TestB1HybridFourValidator$' -timeout 900s -count=1 -v` exited 0; three fresh localhost networks each committed one valid hybrid transfer, rejected a bad ML proof, compared four states and rejected direct invalid proposals. Test output was not published because it includes raw signed transactions and temporary home paths. |
| Phase 01–04 reports and `research/evidence/phase_01*` through `phase_04` | Historical report/evidence only for new study | 401 tracked files before Phase 05, including 349 under `phase_03d`; no new-program independent rerun or privacy review. Their original phase decisions remain historical. |
| Phase 05 result tables, corpus, reports and analysis scripts | Historical report/evidence only for new study | `research/PHASE_REPORTS/PHASE_05_RESEARCH_DECISION.md` describes 78 usable matched-rate conditions and 122 older exploratory runs. New CP/P controls, multi-host attribution and detector baselines were absent, so these results cannot answer the new RQ1–RQ3 contract. |
| Phase 05 original 122-run matrix and direct microbenchmarks | Exploratory | Phase 05 decision explicitly calls the original matrix exploratory and notes insufficient short-run p99 and B0/B1 confounding. |
| Phase 05 disrupted baseline and other exclusion records | Excluded historical data | The decision report says one wall-clock-disrupted B0 secondary run was retained and replaced; `research/evidence/phase_05/CONTENTION_EXCLUDED_RUNS.json` records exclusions. Never overwrite or relabel an excluded run as usable. |
| Four-profile C/CP/P/H common-engine results; byte-matched CP; P-only; multi-host resource measures; direct P2P relay telemetry; preregistered RQ1–RQ3 matrix; feature/detector results | Not present | No implementation or validated new-program run at this baseline. Every associated checkbox remains open. |

The baseline has 420 tracked files under `research/evidence/`: Phase 01 1,
Phase 02 3, Phase 02R 5, Phase 03A 6, Phase 03B 9, Phase 03B closeout 2,
Phase 03C 15, Phase 03C closeout 2, Phase 03D 349, Phase 03E 4, Phase 04 5,
and Phase 05 19. The public manifest hashes selected summary artifacts only.
Existing tracked raw logs are not copied into `research-v2`.

## Existing B0/B1 differences relevant to cost attribution

| Dimension | B0 classical | B1 hybrid | Consequence for new comparisons |
|---|---|---|---|
| Transaction wire | `PLUTOTX1`, version 1; 146–195 bytes, 33-byte compressed secp key, 64-byte signature (`internal/tx/tx.go`) | `PLUTOTX2`, version 2/profile 1; both 33-byte secp and ML-DSA-44 public keys, 64-byte secp and ML signature on wire (`internal/b1tx/tx.go`) | Whole-path B0/B1 deltas include wire and parser cost. |
| Signed intent and address | B0 domain-separated digest and address from secp key | Different B1 digest/domain and address from secp+ML keys; both signatures verify the same B1 digest | Existing B0/B1 transactions are not the proposed C/CP/P/H controls. |
| Genesis | `PLUTOG2!` V2: ordered address/balance allocations; new receivers may be created up to 65,536 accounts (`internal/b0state`) | `PLUTOG3!` V3: at most 1,024 accounts with address, balance, secp key and ML key; immutable registry (`internal/b1state`) | Registry lookup and fixed account set confound state and business comparison. |
| Verification order | Parse, chain, amount, canonical scalars, secp key, ECDSA, state | Parse, chain, amount, both key encodings, derived and registered binding, ECDSA, ML-DSA, state | An invalid class reaches different stages; cost must be attributed by stage. |
| Account transition | `b0state.ApplyAuthenticatedTrusted` checks nonce/balance and updates accounts | Same balance/nonce primitive after B1 registry and AND authentication; receiver must already be registered | Shared primitive does not make complete state handling identical. |
| AppHash | SHA-256 over B0 V2 domain and complete logical state | SHA-256 over B1 domain, registry digest and B0 logical bytes | Committed hashes cannot be numerically compared as the same commitment. |
| Persistence | Pebble complete snapshot under `pluto/committed-state/v2`, magic `PLUTOS2!`, separate `pluto-b0-v2` home | Pebble complete snapshot under `pluto/committed-state/b1-v1`, magic `PLUTOS3!`, separate `pluto-b1-v1` home | Both sync writes, but record layout, registry binding and maximum account count differ. |
| Cache/admission | Pinned CometBFT v0.38.26 mempool cache; no B0 application verification cache | Same CometBFT mempool/cache implementation; no B1 application verification cache | Dedup and rejection before `CheckTx` depend on Comet config and traffic path, not just crypto profile. |

## Legacy component traceability for the new program

| Legacy component | Intent | Reason |
|---|---|---|
| `internal/tx`, `internal/b1tx` | Keep immutable as protocol references; build new profile codecs around a common logical intent | Accepted wire formats differ and cannot by themselves form fair four-profile controls. |
| `internal/b0state`, `internal/b1state` | Reuse proven balance/nonce logic where possible; replace divergent account policy with one common new-program policy | Existing B1 registry/fixed membership differs from B0. |
| `internal/b0app`, `internal/b1app` | Keep historical profiles; build a common experimental app with isolated profile authentication | Protect accepted semantics while holding proposal/state logic constant. |
| `internal/node` four-validator setup | Reuse for single-host diagnostic only; extend orchestration for multi-host | Four localhost processes share CPU, disk and fault domain. |
| `internal/phase05/corpus`, `internal/phase05/bench` | Keep as historical references; create new corpus/oracle and stage telemetry | Existing B0/B1 corpus cannot establish CP/P controls or early detector features. |
| `research/benchmarks/phase05` and Phase 05 results | Keep read-only historical; write a new preregistered analysis pipeline | Historical limits and exploratory run rules do not satisfy new RQ1–RQ3. |
| Phase 01–05 evidence | Preserve by immutable commit/tree, no raw copying | Maintain audit chain without exposing unreviewed raw logs or blending old data into new results. |

This inventory identifies existing code and gaps; it is not an implementation
or acceptance of the new profiles or experiments.
