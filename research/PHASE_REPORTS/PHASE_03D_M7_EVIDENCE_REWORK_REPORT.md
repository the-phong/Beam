# PLUTO PHASE 03D — FINAL M7 EVIDENCE REWORK

**2026-10-25. Controller handoff: M7 conditional source pass, bounded evidence corrections completed; Phase 03D remains PARTIAL. STOP.** No production source, transaction protocol, D-020–D-025 rule, D-024 record byte, or consensus validity changed. D-026 remains an unapproved proposal.

## Source identity and exact changed files

Branch `phase/03d-b0-abci-integration`; `HEAD` is still `b56c81ccce7ea4a0e7f52053c7459c4601c15b38`. The previous staged tree was `dd54117e1c03cb632763f80618dc4d49c0dfbe45`. The new staged tree **before this report** is `3414316c4ead4c5a010f0e7dc2ab6c4aa124662d`. No result commit exists; no PR, push or merge was made.

The only changed preexisting files are `test/integration/b0_asymmetric_test.go`, `test/integration/b0_m7_callbacks_test.go`, `test/integration/b0_recovery_test.go`, `research/decisions/D-026_EXPLICIT_B0_INITIALIZATION_PROPOSAL.md`, and `research/PHASE_REPORTS/PHASE_03D_FINAL_BOUNDED_CLOSURE_REPORT.md` (a link to this rework). This report and `research/evidence/phase_03d/m7_rework/` are new. The [source manifest](../evidence/phase_03d/m7_rework/source_manifest.sha256), SHA-256 `f137b3656609d002dbb81bbc1db3305bd23eb959d9deb8e2ab2fac84859029b5`, lists the exact SHA-256 of the five changed preexisting files and passes `sha256sum -c`. The [evidence manifest](../evidence/phase_03d/m7_rework/evidence_manifest.sha256), SHA-256 `5e9ba2e19071b67fe7bdb68ab283e34ef2bf286556db8ed4ca1f9d0dc2cda4d4`, checks all 223 evidence entries. The prior Controller-reviewed source bundle and historical logs remain intact.

## Fault method and no-progress assertion

Each marked callback writes a JSON file **from the separate application process** containing callback name, existence of the committed B0 snapshot, durable application height A and AppHash. The parent reads and validates that marker, confirms Comet is a separate live process at the marker, then sends `SIGKILL` only to the application. It leaves Comet alone for 1.2 seconds. If Comet RPC is available both at the marker and after the outage, the test rejects any increase in reported height. It then records whether Comet is alive or has exited. Before restarting either process, it reads durable B/C from Comet's stores and reopens B0 to assert that A and AppHash equal the marker, and that C has not advanced. The post-synced Commit case expects C=A−1 at its marker; all other initialized callbacks expect C=A. The post-Commit parent-kill case similarly compares the pre-kill Comet height with C and A after the outage. A Comet process that remains alive with stable height is accepted; no assertion requires it to exit. Comet is stopped only after that outage observation so its stores can be opened safely.

This proves no additional **durable C or A height** in the tested software outage interval. B may already contain an undecided block one height ahead of C. RPC `/status` can report that block height, so it is recorded as an additional progress signal, never substituted for the durable C check. The tests distinguish application `SIGKILL`, later coordinated Comet stop, Comet-only faults from earlier M8, and physical power loss (not tested).

## Reached M7 callbacks and outcomes

B/C/A are blockstore, durable Comet state and durable B0 heights immediately **before coordinated recovery**. The sender begins 100/nonce 0; receiver is absent. A successful transfer ends sender 75/nonce 1 and receiver 25/nonce 0. H0=`e480c8b300c5ac1d662628f90334aae185ad38ed791c59ab14e3aa42abf083da`; H1=`d3d33f8ab367c11dbd28ea1cffe47ba5c61988041b623e2725c7bf4305b79681`; H2=`c0305de0df961f7bb6b7b379a392fb96c3cb7ee1ca6b48c767bd833570d55ae0`; empty H3=`57434e9bf8b6df4fe6c14476a65efe33117efe7b58291158d5e7acf567338c5a`; transfer H3=`b8f217cdbeb9a716851387d5b0fb7319389bca0cb1acfac67ff35ad6e468b743`; transfer H4=`44295da9566bf288667d249e95cd301217abbf9b76b500f594110fd11f9e2ffa`; transfer H5=`ceaf56ac281e653a7ac09ee4d27e6cfd251242def3f14fd5c5f004841a8189b2`. AppHashes are compared only at matching heights.

| Reached callback or boundary | Outage B/C/A, hash and state | Startup/replay and exact effect |
|---|---|---|
| Initial `Info`; `InitChain` before genesis write | Each 0/0/absent; no account or AppHash | Production `start` exited specifically with `B0 database is absent while CometBFT state exists; refusing implicit reset`. No tx; fail closed. |
| `InitChain` after synced H0 before response | 0/0/0, H0, sender 100/0 | Real identical InitChain retry; 2/2/2, H2, sender 100/0. No tx. |
| `CheckTx` NEW | 2/2/2, H2, sender 100/0 | Explicit readmission; indexed execution Code 0; 4/4/4, transfer H4, sender 75/1, receiver 25/0 once. |
| Real Comet `CheckTx` RECHECK | 3/3/3, empty H3, sender 100/0 | Test proposer omitted the admitted tx for one block; real RECHECK marker reached. Indexed Code 0 after restart; 5/5/5, transfer H5, one effect. |
| `PrepareProposal`; `ProcessProposal` | Each 2/2/2, H2, sender 100/0 | Fresh proposal after rebroadcast or retained mempool tx respectively; indexed Code 0, 4/4/4, transfer H4, one effect. |
| `FinalizeBlock` before execution; `Commit` before write | Each 3/2/2, H2, sender 100/0 | Actual Comet replay; indexed Code 0, 4/4/4, transfer H4, one effect. |
| `Commit` after synced B0 write before response | 3/2/3, transfer H3, sender 75/1, receiver 25/0 | Comet mock replay; indexed Code 0, 4/4/4, transfer H4, no second effect. |
| After successful Commit response | 3/3/3, transfer H3, sender 75/1, receiver 25/0 | Comet child was alive immediately after application-parent SIGKILL; C=A=3 before coordinated stop. Real restart reached 4/4/4, transfer H4, no second effect. |
| Read-only `Info`; `Query` | Final normal run: Info 2/2/2 H2; Query 2/1/1 H1 with an undecided H2 block. Sender 100/0, receiver absent | Comet may exit or remain alive without reported-height increase; both restarted to 3/3/3, empty H3, no tx effect. |

The earlier M4 process-kill evidence at blocked WAL `Write` and `SyncData` remains the separate in-write Commit coverage; it was not repeated. Its outcomes were B/C/A=3/2/2 or 3/2/3 followed by real or mock replay and one transfer effect. A `SyncData` call entry is not a hardware durability proof.

## Execution, raw evidence and D-026 amendment

Exact test commands (run from the repository root; shell output was also saved with `tee`):

```sh
PLUTO_RECOVERY_ARTIFACTS="$PWD/research/evidence/phase_03d/m7_rework/raw/normal" go test -tags integration ./test/integration -run '^(TestB0M7AsymmetricApplicationCrash|TestB0M7OtherApplicationCallbacks|TestB0ApplicationProcessKilledWhileCometChildAlive)$' -count=1 -v -timeout=180s
PLUTO_RECOVERY_ARTIFACTS="$PWD/research/evidence/phase_03d/m7_rework/raw/race" go test -race -tags integration ./test/integration -run '^(TestB0M7AsymmetricApplicationCrash|TestB0M7OtherApplicationCallbacks|TestB0ApplicationProcessKilledWhileCometChildAlive)$' -count=1 -v -timeout=240s
make verify
PLUTO_RECOVERY_ARTIFACTS="$PWD/research/evidence/phase_03d/m7_rework/raw" make test-integration
python3 research/evidence/phase_03d/m7_rework/audit_subtests.py
sha256sum -c research/evidence/phase_03d/m7_rework/source_manifest.sha256
sha256sum -c research/evidence/phase_03d/m7_rework/evidence_manifest.sha256
```

Exit codes and results:

| Command | Exit | Result |
|---|---:|---|
| Targeted normal | 0 | 13/13 required subtests PASS, 0 SKIP, 74.853 s |
| Targeted race | 0 | 13/13 PASS, 0 SKIP, 75.962 s |
| `make verify` | 0 | Go build, unit tests, vet and module verification PASS |
| `PLUTO_RECOVERY_ARTIFACTS=$PWD/research/evidence/phase_03d/m7_rework/raw make test-integration` | 0 | Full integration PASS, 175.167 s of 180 s timeout |
| `audit_subtests.py` | 0 | 13/13 required subtests and 12 callback markers plus post-Commit acknowledgment in each targeted run |
| `sha256sum -c` for both manifests | 0 | Every source and evidence entry verified |

The [no-SKIP audit](../evidence/phase_03d/m7_rework/subtest_audit_final.log) checks all 13 required names in both targeted logs. [Normal output](../evidence/phase_03d/m7_rework/targeted_normal_final.log), [race output](../evidence/phase_03d/m7_rework/targeted_race_final.log), [full integration](../evidence/phase_03d/m7_rework/make_test_integration.log), and [verification](../evidence/phase_03d/m7_rework/make_verify.log) are retained. Separate `raw/normal`, `raw/race` and `raw/regression` directories preserve child/Comet logs and every callback marker. Earlier exploratory targeted outputs are retained; their shared raw artifact path was overwritten before separate directories were introduced, so final evidence rests on the isolated reruns.

The [D-026 decision amendment](../decisions/D-026_EXPLICIT_B0_INITIALIZATION_PROPOSAL.md) identifies the specific accepted B0 `InitChain` requirement that would be superseded **if approved**: genesis H0 creation would move from `InitChain` to `init-b0`, while `InitChain` would validate the existing identical H0. D-025 transport, D-024 physical record bytes, AppHash and conflict rejection remain. Existing complete B0 homes remain compatible; missing or incomplete B0 homes require explicit operator disposition. No D-026 implementation is authorized.

**Verdict:** The two Controller-requested M7 evidence corrections are implemented and verified for the tested software faults. M7 is ready for Controller review; G3, D12, D17 and T7 remain PARTIAL pending the D-026 policy decision and separate hardware limitations. No Phase 03D completion claim is made. **STOP — no commit, push, merge or Phase 03E.**
