# Source guide

Start at [`cmd/plutod/main.go`](../cmd/plutod/main.go) for CLI commands, then [`internal/node/node.go`](../internal/node/node.go) for initialization and `Run`. The node's `b0.go` and `b1.go` choose the accepted transaction applications and their separate homes. Read [`docs/TRANSACTION_FLOW.md`](TRANSACTION_FLOW.md) before changing a callback.

| Area | Read first | Responsibility and consumers |
| --- | --- | --- |
| `internal/legacy/{app,store}` | `app/app.go`, `store/state.go` | Transaction-free Phase 02 application; node consumes it. |
| `internal/classical/tx` | `tx.go` | B0 wire, digest, address, validation codes; state consumes it. |
| `internal/classical/{genesis,state,store,app}` | `state/transition.go`, `app/app.go` | B0 genesis, balances/nonces/AppHash, Pebble, ABCI++; node consumes app and store. |
| `internal/hybrid/tx` | `tx.go` | B1 envelope, signed digest and secp/ML verification; hybrid state consumes it. |
| `internal/hybrid/{genesis,state,store,app,testutil}` | `state/state.go`, `app/app.go` | B1 binding/transition, persistence, callbacks and fixtures; node consumes app/store. |
| `internal/experiment/profiles` | `tx.go`, `state.go`, `app/app.go` | C/CP/P/H controlled transaction, evaluator, ABCI and persistence; integration harness consumes app. |
| `internal/experiment/load` | `plan.go`, `runner.go`, `guard.go` | S06 plans, deadlines, RPC sends, aborts; research workload CLI consumes it. |
| `internal/experiment/telemetry` | `recorder.go`, `collector.go`, `ingress.go` | S05 optional CheckTx event and controlled observations; profile app and observer/ingress CLIs consume it. |
| `internal/experiment/corpus` | `profiles/corpus.go`, `b0b1/oracle.go` | Public corpus verification and independent direct-ABCI oracle inputs. |
| `internal/experiment/benchmark` | `measure.go`, `bench_test.go` | Historical cost measurements; benchmarks and runners consume it. |
| `cmd/research` | `corpus/*/main.go`, `workload/main.go`, `observer/main.go` | Separate executable contracts grouped by function; `Makefile` preserves binary names. |
| `test/integration` | `s04_four_validator_test.go`, `s05_telemetry_test.go`, `s06_pilot_test.go` | Real localhost validators, replay, telemetry and load pilot. |

To change protocol behavior, identify the affected B0, B1 or C/CP/P/H `tx` and `state` pair, then its `app` callbacks and golden tests. To change node wiring, start in `internal/node`; to change measurement, start in `internal/experiment` and read [`research/protocol`](../research/protocol) contracts. Run `make verify` and the matching integration target. The [refactor mapping](REFACTOR_MAPPING.md) lists former paths.
