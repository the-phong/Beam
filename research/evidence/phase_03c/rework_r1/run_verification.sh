#!/usr/bin/env bash
# Execute from repository root. This R1 runner never touches historical logs.
set -u
log=research/evidence/phase_03c/rework_r1/verification.log
: > "$log"
printf 'UTC start: %s\n' "$(date -u +%FT%TZ)" | tee -a "$log"
printf 'Host: %s\n' "$(uname -a)" | tee -a "$log"
printf 'Git HEAD before commit: %s\n' "$(git rev-parse HEAD)" | tee -a "$log"
printf 'Branch: %s\n' "$(git branch --show-current)" | tee -a "$log"
run() {
  printf '\n$' | tee -a "$log"
  printf ' %q' "$@" | tee -a "$log"
  printf '\n' | tee -a "$log"
  "$@" 2>&1 | tee -a "$log"
  result=${PIPESTATUS[0]}
  printf 'EXIT CODE: %d\n' "$result" | tee -a "$log"
  if (( result != 0 )); then failures=$((failures+1)); fi
}
failures=0
run go version
run python3 --version
run go test -count=1 -v -run '^Test(CreateCommitRestartAndIsolation|FaultBoundaries|LifecycleAndStorageErrorBranches|ConcurrentReadersFailClosedOnUncertainCommit)$' ./internal/b0store
run go test -count=1 ./internal/b0state ./internal/b0store
run go build ./...
run go test -count=1 ./...
run go vet ./...
run make fmt-check
run go mod verify
run make verify
run make test-integration
run go test -race -count=1 ./internal/b0state ./internal/b0store
run go test -race -count=10 -run '^TestConcurrentReadersFailClosedOnUncertainCommit$' ./internal/b0store
run go test -coverprofile=research/evidence/phase_03c/rework_r1/state.coverage.profile ./internal/b0state
run go tool cover -func=research/evidence/phase_03c/rework_r1/state.coverage.profile
run go test -coverprofile=research/evidence/phase_03c/rework_r1/store.coverage.profile ./internal/b0store
run go tool cover -func=research/evidence/phase_03c/rework_r1/store.coverage.profile
run go test -run '^$' -fuzz '^FuzzGenesis$' -fuzztime=5s -parallel=2 ./internal/b0state
run go test -run '^$' -fuzz '^FuzzLogical$' -fuzztime=5s -parallel=2 ./internal/b0state
run go test -run '^$' -fuzz '^FuzzRecord$' -fuzztime=5s -parallel=2 ./internal/b0store
run bash -c 'go run research/evidence/phase_03c/generate_vectors.go | cmp - research/evidence/phase_03c/state_vectors.json'
run sha256sum research/evidence/phase_03c/state_vectors.json
run python3 research/evidence/phase_03c/reference_oracle.py
run git diff --check
run git diff --exit-code 61bc65fdebf358fb0330cb9ebd4a332ca4a06aca -- internal/app internal/node internal/store cmd test go.mod go.sum Makefile .github research/specs
run git diff --exit-code 6459a1ad7ee5a17994f484c04f47658a976d9ea6 -- internal/b0state internal/tx research/evidence/phase_03c/state_vectors.json research/evidence/phase_03c/reference_oracle.py research/evidence/phase_03c/generate_vectors.go
printf '\nUTC end: %s\nFAILURES: %d\n' "$(date -u +%FT%TZ)" "$failures" | tee -a "$log"
exit "$failures"
