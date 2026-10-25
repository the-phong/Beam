#!/usr/bin/env bash
set -uo pipefail

cd "$(dirname "$0")/../../../.." || exit 1
log=/tmp/pluto_phase03d_d025_rework_verification.log
: > "$log"
printf 'UTC timestamp: %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" | tee -a "$log"
printf 'Branch: %s\nHEAD: %s\nRemote integration: %s\nSystem: %s\n' \
  "$(git branch --show-current)" "$(git rev-parse HEAD)" \
  "$(git ls-remote origin refs/heads/test-mainet | cut -f1)" "$(uname -srm)" | tee -a "$log"
before_tree=$(git write-tree) || exit 1
printf 'Staged tree before verification: %s\n' "$before_tree" | tee -a "$log"

failures=0
run() {
  local label="$1"
  shift
  printf '\n$ %s\n' "$label" | tee -a "$log"
  "$@" 2>&1 | tee -a "$log"
  local code=${PIPESTATUS[0]}
  printf 'exit_code=%d\n' "$code" | tee -a "$log"
  if ((code != 0)); then failures=$((failures + 1)); fi
}

run 'go version' go version
run 'python3 --version' python3 --version
run 'go list -m github.com/cometbft/cometbft github.com/cockroachdb/pebble github.com/decred/dcrd/dcrec/secp256k1/v4' \
  go list -m github.com/cometbft/cometbft github.com/cockroachdb/pebble github.com/decred/dcrd/dcrec/secp256k1/v4
run 'go build ./...' go build ./...
run 'go test -count=1 ./...' go test -count=1 ./...
run 'go vet ./...' go vet ./...
run 'make fmt-check' make fmt-check
run 'go mod verify' go mod verify
run 'make verify' make verify
run 'make test-integration' make test-integration
run 'go test -tags integration ./test/integration -run TestPinnedCometGenesisTransportAndB0Restart -timeout 120s -count=1 -v' \
  go test -tags integration ./test/integration -run TestPinnedCometGenesisTransportAndB0Restart -timeout 120s -count=1 -v
run 'go test -count=1 -v ./internal/b0genesis ./internal/b0app' \
  go test -count=1 -v ./internal/b0genesis ./internal/b0app
run 'go test -race -count=1 ./internal/b0genesis ./internal/b0app ./internal/node ./cmd/plutod' \
  go test -race -count=1 ./internal/b0genesis ./internal/b0app ./internal/node ./cmd/plutod
run 'python3 research/evidence/phase_03d/d025/reference_transport.py' \
  python3 research/evidence/phase_03d/d025/reference_transport.py
run 'python3 research/evidence/phase_03c/reference_oracle.py' \
  python3 research/evidence/phase_03c/reference_oracle.py
run 'sha256sum research/evidence/phase_03c/state_vectors.json' \
  sha256sum research/evidence/phase_03c/state_vectors.json
run 'git diff --cached --check' git diff --cached --check
run 'git diff --exit-code b56c81ccce7ea4a0e7f52053c7459c4601c15b38 -- internal/app internal/tx internal/b0state internal/b0store internal/store go.mod go.sum Makefile .github research/evidence/phase_03c research/evidence/phase_03b research/specs/B0_CLASSICAL_TRANSACTION_PROTOCOL_V1.md research/specs/B0_TEST_VECTOR_PLAN_V1.md' \
  git diff --exit-code b56c81ccce7ea4a0e7f52053c7459c4601c15b38 -- \
    internal/app internal/tx internal/b0state internal/b0store internal/store go.mod go.sum Makefile .github \
    research/evidence/phase_03c research/evidence/phase_03b \
    research/specs/B0_CLASSICAL_TRANSACTION_PROTOCOL_V1.md research/specs/B0_TEST_VECTOR_PLAN_V1.md
run 'git diff --quiet (no unstaged tracked changes)' git diff --quiet
run 'test -z "$(git ls-files --others --exclude-standard)" (no untracked files)' \
  test -z "$(git ls-files --others --exclude-standard)"
after_tree=$(git write-tree) || exit 1
printf '\nStaged tree after verification: %s\n' "$after_tree" | tee -a "$log"
if [[ "$before_tree" != "$after_tree" ]]; then
  printf 'Staged tree changed during verification\n' | tee -a "$log"
  failures=$((failures + 1))
fi
printf 'Verification command failures: %d\n' "$failures" | tee -a "$log"
if ((failures != 0)); then exit 1; fi
