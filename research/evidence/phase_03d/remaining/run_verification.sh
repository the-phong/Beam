#!/usr/bin/env bash
set -uo pipefail

cd "$(dirname "$0")/../../../.." || exit 1
evidence_dir=research/evidence/phase_03d/remaining
log="$evidence_dir/verification.log"
manifest="$evidence_dir/source_manifest.sha256"
: > "$log"

printf 'UTC: %s\nbranch: %s\nHEAD: %s\naccepted D-025 tree: 7295b641254c91c53b3b80a0c2ec5f6c286e8dd1\nstaged tree before checks: %s\n' \
  "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$(git branch --show-current)" \
  "$(git rev-parse HEAD)" "$(git write-tree)" >> "$log"
printf 'environment: %s\n' "$(uname -srm)" >> "$log"
printf 'GOMAXPROCS: %s\nGOGC: %s\n' "${GOMAXPROCS:-Go default}" "${GOGC:-Go default}" >> "$log"
df -T . >> "$log" 2>&1
git ls-files -z cmd internal test go.mod go.sum Makefile .github research/specs \
  | xargs -0 sha256sum > "$manifest"
sha256sum "$manifest" >> "$log"

failures=0
run() {
  local label="$1"
  shift
  printf '\n$ %s\n' "$label" >> "$log"
  "$@" >> "$log" 2>&1
  local code=$?
  printf 'exit_code=%d\n' "$code" >> "$log"
  if ((code != 0)); then failures=$((failures+1)); fi
}

run 'go version' go version
run 'python3 --version' python3 --version
run 'go list -m pinned modules' go list -m github.com/cometbft/cometbft github.com/cockroachdb/pebble github.com/decred/dcrd/dcrec/secp256k1/v4
run 'go build ./...' go build ./...
run 'go test -count=1 ./...' go test -count=1 ./...
run 'go vet ./...' go vet ./...
run 'make fmt-check' make fmt-check
run 'go mod verify' go mod verify
run 'make verify' make verify
run 'make test-integration' make test-integration
run 'go test -race -count=1 ./internal/b0app ./internal/b0state ./internal/b0store ./internal/b0genesis' \
  go test -race -count=1 ./internal/b0app ./internal/b0state ./internal/b0store ./internal/b0genesis
run 'go test -cover -count=1 ./internal/b0app ./internal/b0state ./internal/b0store' \
  go test -cover -count=1 ./internal/b0app ./internal/b0state ./internal/b0store
run 'go test -count=1 -v ./internal/b0app -run ^TestB0Accepted' \
  go test -count=1 -v ./internal/b0app -run '^TestB0Accepted'
run 'go test -count=1 -v ./internal/b0app -run ^TestABCIPathsFailClosedAfterUncertainCommit$' \
  go test -count=1 -v ./internal/b0app -run '^TestABCIPathsFailClosedAfterUncertainCommit$'
run 'go test -count=1 -v ./internal/b0app -run ^TestB0V18GenesisSupplyOverflowAndV23CorruptRecord$' \
  go test -count=1 -v ./internal/b0app -run '^TestB0V18GenesisSupplyOverflowAndV23CorruptRecord$'
run 'go test -count=1 -v ./internal/b0app -run ^TestB0ProposalBudgetAndLimitFallback$' \
  go test -count=1 -v ./internal/b0app -run '^TestB0ProposalBudgetAndLimitFallback$'
run 'go test -tags integration ./test/integration -run ^TestB0Pinned -timeout 120s -count=1 -v' \
  go test -tags integration ./test/integration -run '^TestB0Pinned' -timeout 120s -count=1 -v
run 'go test -tags integration ./test/integration -run ^TestB0RealTransactionCommitAndRestart$ -timeout 120s -count=1 -v' \
  go test -tags integration ./test/integration -run '^TestB0RealTransactionCommitAndRestart$' -timeout 120s -count=1 -v
run 'python3 research/evidence/phase_03c/reference_oracle.py' python3 research/evidence/phase_03c/reference_oracle.py
run 'python3 research/evidence/phase_03d/d025/reference_transport.py' python3 research/evidence/phase_03d/d025/reference_transport.py
run 'sha256sum accepted Phase 03B/03C vectors' sha256sum research/evidence/phase_03b/golden_vectors.json research/evidence/phase_03c/state_vectors.json
for target in FuzzGenesis FuzzLogical; do
  run "GOMAXPROCS=2 go test ./internal/b0state -run ^$ -fuzz ^${target}$ -fuzztime=2s -parallel=1" \
    env GOMAXPROCS=2 go test ./internal/b0state -run '^$' -fuzz "^${target}$" -fuzztime=2s -parallel=1
done
run 'GOMAXPROCS=2 go test ./internal/b0store -run ^$ -fuzz ^FuzzRecord$ -fuzztime=2s -parallel=1' \
  env GOMAXPROCS=2 go test ./internal/b0store -run '^$' -fuzz '^FuzzRecord$' -fuzztime=2s -parallel=1
for target in FuzzParse FuzzCodec; do
  run "GOMAXPROCS=2 go test ./internal/tx -run ^$ -fuzz ^${target}$ -fuzztime=2s -parallel=1" \
    env GOMAXPROCS=2 go test ./internal/tx -run '^$' -fuzz "^${target}$" -fuzztime=2s -parallel=1
done
run 'go test ./internal/b0state ./internal/b0app ./internal/b0store -run ^$ -bench BenchmarkB0 -benchtime=5x -count=3 -benchmem' \
  go test ./internal/b0state ./internal/b0app ./internal/b0store -run '^$' -bench '^BenchmarkB0' -benchtime=5x -count=3 -benchmem
run 'PLUTO_B0_SYNC_SAMPLES=1 go test ./internal/b0store -run ^TestB0SyncLatencySamples$ -count=1 -v' \
  env PLUTO_B0_SYNC_SAMPLES=1 go test ./internal/b0store -run '^TestB0SyncLatencySamples$' -count=1 -v
run 'python3 research/evidence/phase_03d/remaining/summarize_sync.py verification.log' \
  python3 "$evidence_dir/summarize_sync.py" "$log"
run 'go test ./internal/b0state -run ^$ -bench BenchmarkB0StateCost/K=65536/Reference/Malformed -benchtime=100x -cpuprofile reference_malformed.pprof' \
  go test ./internal/b0state -run '^$' -bench '^BenchmarkB0StateCost/K=65536/Reference/Malformed$' -benchtime=100x -cpuprofile "$evidence_dir/reference_malformed.pprof"
run 'go tool pprof -top reference_malformed.pprof' go tool pprof -top "$evidence_dir/reference_malformed.pprof"
run 'go test ./internal/b0state -run ^$ -bench BenchmarkB0StateCost/K=65536/Trusted/Malformed -benchtime=100000x -cpuprofile trusted_malformed.pprof' \
  go test ./internal/b0state -run '^$' -bench '^BenchmarkB0StateCost/K=65536/Trusted/Malformed$' -benchtime=100000x -cpuprofile "$evidence_dir/trusted_malformed.pprof"
run 'go tool pprof -top trusted_malformed.pprof' go tool pprof -top "$evidence_dir/trusted_malformed.pprof"
run 'git diff --check' git diff --check
run 'git diff --cached --check' git diff --cached --check
run 'accepted source and protocol preservation diff' git diff --exit-code 7295b641254c91c53b3b80a0c2ec5f6c286e8dd1 "$(git write-tree)" -- \
  internal/app internal/store internal/tx cmd/plutod internal/node go.mod go.sum Makefile .github \
  research/specs research/evidence/phase_03b research/evidence/phase_03c research/evidence/phase_03d/d025
run 'source manifest recheck' sha256sum -c "$manifest"

printf '\nverification_failures=%d\nstaged tree after checks: %s\n' "$failures" "$(git write-tree)" >> "$log"
printf 'Verification failures: %d\nEvidence: %s\n' "$failures" "$log"
if ((failures != 0)); then exit 1; fi
