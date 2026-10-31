#!/usr/bin/env bash
# Run from the repository root. Records the post-merge governance closeout checks.
set -u
log=research/evidence/phase_03b_closeout/verification.log
: > "$log"
printf 'UTC start: %s\n' "$(date -u '+%Y-%m-%d %H:%M:%S')" >> "$log"
printf 'Verified integration baseline: 8be07c87abed73763d40252e99bbf0573da487b4\n' >> "$log"
printf 'Working HEAD: %s\n' "$(git rev-parse HEAD)" >> "$log"
printf 'Branch: %s\n' "$(git branch --show-current)" >> "$log"
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
run git diff --check
run make verify
run make test-integration
run git diff --exit-code 8be07c87abed73763d40252e99bbf0573da487b4 -- cmd internal test go.mod go.sum Makefile .github research/specs research/evidence/phase_03b/golden_vectors.json
printf 'UTC end: %s\n' "$(date -u '+%Y-%m-%d %H:%M:%S')" >> "$log"
exit "$overall"
