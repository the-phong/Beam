# Domain-oriented refactor mapping

Source baseline: `test-mainet` merge `ed6bc8f474c13ed709d5ab5f57b3856edcbf602a` (S06 PR #17). `master` was inspected for layout only. Paths below cover every active source package and executable; historical evidence stays in place. Package identifiers are retained where they identify B0/B1 semantics, while import paths move.

## Baseline inventory

`git ls-tree -r --name-only master` at `5af503cf22cf47a0cde9cc6cc764b338cda2cae6` showed `cmd/{corpus,plutod}`, `internal/{classical,experiment,hybrid,legacy,node,version}`, `test/integration` and historical `research/`. That branch lacks the active S03–S06 source. `git ls-tree -r --name-only test-mainet` at the source commit showed `cmd/{plutod,b0b1-corpus,profile-corpus,profile-vectors,profile-workload-corpus,profile-load,profile-ingress,profile-observer}`, `internal/{app,store,tx,b0app,b0genesis,b0state,b0store,b1app,b1genesis,b1state,b1store,b1test,b1tx,benchmarks,node,profiles,telemetry,version}`, `test/integration` and one `research/` tree. The full tracked tree has 613 files; source and fixture directories were inspected before movement. `make verify` exited 0 on the source commit. The old `master` tree supplied organization ideas only.

| Old path | New path | Responsibility | Dependencies | Verification | Notes |
| --- | --- | --- | --- | --- | --- |
| `internal/app` | `internal/legacy/app` | Transaction-free ABCI application | legacy store, version, CometBFT | legacy unit and integration | Move intact |
| `internal/store` | `internal/legacy/store` | Legacy V1 persistence | Pebble | legacy unit and integration | Move intact |
| `internal/tx` | `internal/classical/tx` | B0 canonical wire and secp256k1 | crypto libraries | B0 vectors and unit | Protocol identifier retained |
| `internal/b0genesis` | `internal/classical/genesis` | B0 Genesis V2 transport | classical state | B0 unit and integration | Relative fixture paths reviewed |
| `internal/b0state` | `internal/classical/state` | B0 account state, transition, AppHash | classical tx | B0 unit and integration | Cross-profile primitives used by B1 and experiment stay here |
| `internal/b0store` | `internal/classical/store` | B0 Pebble state | classical state, tx | B0 unit and recovery | Relative fixture paths reviewed |
| `internal/b0app` | `internal/classical/app` | B0 ABCI++ | classical genesis, state, store, tx | B0 unit and four-validator | Move intact |
| `internal/b1tx` | `internal/hybrid/tx` | B1 canonical wire and dual signatures | crypto libraries | B1 vector and unit | Move testdata bytes intact |
| `internal/b1genesis` | `internal/hybrid/genesis` | B1 genesis transport | hybrid state | B1 unit and integration | Move intact |
| `internal/b1state` | `internal/hybrid/state` | B1 account state and AppHash | classical state, hybrid tx | B1 unit and integration | Move intact |
| `internal/b1store` | `internal/hybrid/store` | B1 Pebble state | classical state, hybrid state, tx | B1 unit and recovery | Move intact |
| `internal/b1app` | `internal/hybrid/app` | B1 ABCI++ | hybrid genesis, state, store, tx; classical state | B1 unit and four-validator | Move intact |
| `internal/b1test` | `internal/hybrid/testutil` | B1 fixtures | B1 and classical packages | B1 unit and integration | Test support only |
| `internal/node` | `internal/node` | Lifecycle, CometBFT and profile selection | legacy, classical, hybrid | integration targets | Active node behavior retained |
| `internal/version` | `internal/version` | Build metadata | none | `make verify` version output | Unchanged |
| `internal/profiles` | `internal/experiment/profiles` | C/CP/P/H transaction and state | classical state, cryptography | vectors, corpus, four-validator | Separate from historical B0/B1 |
| `internal/profiles/app`, `genesis`, `store`, `testfixture`, `testdata` | `internal/experiment/profiles/` same subpaths | Controlled-profile ABCI, genesis, persistence, fixtures | experiment profiles; telemetry | profile unit and S04–S06 | Golden bytes unchanged |
| `internal/profiles/load` | `internal/experiment/load` | Bounded offered load, rate plan and safety guard | experiment profiles, telemetry | load unit and S06 pilot | Separate workload responsibility |
| `internal/profiles/corpus` | `internal/experiment/corpus/profiles` | Controlled-profile public corpus | experiment profiles and fixture | corpus verification and S04 | Own executable retained |
| `internal/benchmarks/corpus` | `internal/experiment/corpus/b0b1` | Historical B0/B1 corpus and oracle | classical, hybrid | corpus verification, benchmark smoke | Different protocol corpus retained |
| `internal/benchmarks/bench` | `internal/experiment/benchmark` | Historical micro cost measurement | corpus, classical, hybrid | micro benchmark smoke | No benchmark semantics change |
| `internal/telemetry` | `internal/experiment/telemetry` | Recorder, collector, ingress hooks | standard library | telemetry unit and S05 | Application instrumentation import is intentional |
| `cmd/plutod` | `cmd/plutod` | Main node CLI | node, genesis, version | build and integration | Stable entrypoint |
| `cmd/b0b1-corpus` | `cmd/research/corpus/b0b1` | B0/B1 corpus executable | experiment corpus/b0b1 | public corpus verification | Flags retained |
| `cmd/profile-corpus` | `cmd/research/corpus/profiles` | C/CP/P/H corpus executable | experiment corpus/profiles | S04 verification | Subcommands retained |
| `cmd/profile-workload-corpus` | `cmd/research/corpus/workload` | S06 scenario corpus executable | experiment load | S06 verification | Subcommands retained |
| `cmd/profile-vectors` | `cmd/research/vectors/profiles` | C/CP/P/H vectors | experiment profiles | vector comparison | Output bytes retained |
| `cmd/profile-load` | `cmd/research/workload` | Open-loop generator CLI | experiment load | S06 pilot | Build output name retained |
| `cmd/profile-ingress` | `cmd/research/ingress` | RPC ingress observer CLI | telemetry | S05 | Build output name retained |
| `cmd/profile-observer` | `cmd/research/observer` | Telemetry collector CLI | telemetry | S05 | Build output name retained |
| `test/integration` | `test/integration` | Real node, BFT, replay, workload tests | node and domain packages | all integration targets | Existing selectors retained |
| `research/protocol`, `config`, `corpus`, `benchmarks`, `checklists`, `tools`, `evidence`, `artifacts` | same paths | Protocol, fixtures, runners and immutable results | CLI path references | corpus, Python tests, hash audit | Historical evidence and raw/private bytes remain in place |
| `research/telemetry/comet_overlay.py` | `research/tools/comet_overlay.py` | S05 instrumentation overlay builder | pinned CometBFT source | S05 build and pilot | Script path references updated |
| `research/specs`, governance records and `research/PHASE_REPORTS` | same paths | Accepted specifications and historical record | documentation | hash audit | No historical rewrite |

## Dependency direction

`cmd/plutod -> node -> {legacy, classical, hybrid}`; `classical/{app,store,genesis,state} -> classical/tx`; `hybrid/{app,store,genesis,state} -> hybrid/tx`, with deliberate use of classical state primitives. `experiment/{profiles,corpus,benchmark,load,telemetry}` consumes the historical domains where required. The controlled-profile application imports telemetry for S05 instrumentation; historical production applications do not import experimental packages. Research CLIs and integration tests are consumers of the above packages.

## Baseline preservation

Before editing, the tree had unrelated untracked `research/evidence/phase_03d/d026/` and `d026_review/`, ignored runtime outputs under `research/artifacts/` and evidence folders, and stash `stash@{0}` (`7ec52bd` lineage). These are excluded from the refactor. The baseline hash inventory includes tracked corpus, evidence and testdata and is compared after relocation by old-to-new path.
