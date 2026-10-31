#!/usr/bin/env bash
# Run from the repository root. Captures exact commands, output and exit codes.
set -u
log=research/evidence/phase_03b/verification.log
: > "$log"
printf 'UTC start: %s\n' "$(date -u '+%Y-%m-%d %H:%M:%S')" >> "$log"
printf 'Source baseline: 54bf720ac4625c17eaa6101a8f398ef7c0663432\n' >> "$log"
printf 'Working HEAD: %s\n' "$(git rev-parse HEAD)" >> "$log"
printf 'OS: %s\n' "$(uname -a)" >> "$log"
overall=0
run() {
  printf '\n$' >> "$log"
  printf ' %q' "$@" >> "$log"
  printf '\n' >> "$log"
  "$@" >> "$log" 2>&1
  code=$?
  printf 'EXIT CODE: %s\n' "$code" >> "$log"
  printf '%s: %s\n' "$code" "$*"
  if ((code != 0)); then overall=1; fi
}
run go version
run go build ./...
run go test -count=1 ./...
run go vet ./...
run make fmt-check
run go mod verify
run make verify
run make test-integration
run go test -race -count=1 ./internal/tx
run go test -coverprofile=research/evidence/phase_03b/coverage.out ./internal/tx
run go tool cover -func=research/evidence/phase_03b/coverage.out
run go test ./internal/tx -run '^$' -fuzz '^FuzzParse$' -fuzztime 10s -parallel 2
run go test ./internal/tx -run '^$' -fuzz '^FuzzCodec$' -fuzztime 10s -parallel 2
run python3 research/evidence/phase_03b/reference_oracle.py
run git diff --check
run git diff --exit-code 54bf720ac4625c17eaa6101a8f398ef7c0663432 -- cmd internal/app internal/node internal/store test .github Makefile
printf 'UTC end: %s\n' "$(date -u '+%Y-%m-%d %H:%M:%S')" >> "$log"
exit "$overall"
