.PHONY: build build-comet-s05 clean test test-integration test-integration-4v test-integration-b1-4v test-s04-4v test-s05-telemetry test-s06-pilot test-s07-schema test-s07-smoke test-file02-common test-file02-common02 corpus-s04-verify corpus-s06-verify vet fmt-check verify benchmark-corpus benchmark-smoke benchmark-micro benchmark-b1 benchmark-full benchmark-analyze benchmark-contention-corpus benchmark-contention-smoke benchmark-contention-full benchmark-contention-analyze

GO ?= go
INTEGRATION_TIMEOUT ?= 900s
REVISION := $(shell git rev-parse --short HEAD 2>/dev/null || printf unavailable)

build:
	mkdir -p build
	$(GO) build -ldflags "-X example.com/pluto/internal/version.Revision=$(REVISION)" -o build/plutod ./cmd/plutod
	$(GO) build -o build/cometbft github.com/cometbft/cometbft/cmd/cometbft
	$(GO) build -o build/profile-observer ./cmd/research/observer
	$(GO) build -o build/profile-ingress ./cmd/research/ingress
	$(GO) build -o build/profile-load ./cmd/research/workload
	$(GO) build -o build/profile-workload-corpus ./cmd/research/corpus/workload

build-comet-s05:
	python3 research/tools/comet_overlay.py build/s05-comet-overlay
	$(GO) build -modfile=build/s05-comet.mod -o build/cometbft-s05 github.com/cometbft/cometbft/cmd/cometbft

# Remove only disposable local outputs; runtime homes and research evidence
# deliberately have no clean target.
clean:
	rm -rf -- build research/benchmarks/__pycache__ research/benchmarks/b0b1/__pycache__
	rm -f -- ./*.test

test:
	$(GO) test ./...

test-integration: build
	$(GO) test -tags integration ./test/integration -timeout $(INTEGRATION_TIMEOUT) -count=1 -v

test-integration-4v: build
	$(GO) test -tags 'integration fourval' ./test/integration -run '^TestB0FourValidator' -timeout 900s -count=1 -v

test-integration-b1-4v: build
	$(GO) test -tags 'integration fourval' ./test/integration -run '^TestB1Hybrid' -timeout 900s -count=1 -v

test-s04-4v: build corpus-s04-verify
	$(GO) test -tags 'integration fourval' ./test/integration -run '^TestS04FourValidatorProfiles$$' -timeout 600s -count=1 -v

test-s05-telemetry: build build-comet-s05 corpus-s04-verify
	$(GO) test -tags 'integration fourval' ./test/integration -run '^TestS05FourValidatorTelemetry$$' -timeout 600s -count=1 -v

test-s06-pilot: build build-comet-s05 corpus-s06-verify
	$(GO) test -tags 'integration fourval' ./test/integration -run '^TestS06IsolatedOpenLoopPilot$$' -timeout 900s -count=1 -v

test-s07-schema:
	python3 -m unittest discover research/tools -p 'test_s07_*.py' -v

test-file02-common:
	python3 -m unittest discover research/tools -p 'test_file02_*.py' -v

test-file02-common02:
	python3 research/tools/file02_common02.py smoke

.PHONY: test-file02-common03
test-file02-common03:
	python3 research/tools/file02_common03.py smoke

.PHONY: test-file02-common04
test-file02-common04:
	python3 research/tools/file02_common04.py smoke

.PHONY: test-file02-common05
test-file02-common05:
	python3 research/tools/file02_common05.py smoke

.PHONY: test-file02-common06
test-file02-common06:
	python3 research/tools/file02_common06.py smoke

.PHONY: test-file02-t06-calibration
test-file02-t06-calibration:
	python3 research/tools/file02_t06.py smoke

.PHONY: test-file02-t06-precision
test-file02-t06-precision:
	python3 research/tools/file02_t06_precision.py smoke

.PHONY: test-file02-t01
test-file02-t01:
	python3 research/tools/file02_t01.py run

.PHONY: test-file02-t04
test-file02-t04:
	python3 research/tools/file02_t04.py run

# Builds, verifies public corpora and runs four fresh bounded private networks.
# The script prints an ignored session path and a SHA-256 manifest for reruns.
test-s07-smoke:
	python3 research/tools/s07_orchestrate.py smoke

corpus-s04-verify:
	$(GO) run ./cmd/research/corpus/profiles verify research/corpus/s04_public_corpus.json

corpus-s06-verify:
	$(GO) run ./cmd/research/corpus/workload verify research/corpus/s06_scenarios.json

vet:
	$(GO) vet ./...

fmt-check:
	@test -z "$$(gofmt -l cmd internal test)" || { gofmt -l cmd internal test; exit 1; }

verify: build test vet fmt-check
	$(GO) mod verify
	./build/plutod version

benchmark-corpus:
	$(GO) run ./cmd/research/corpus/b0b1 -mode verify -file research/corpus/b0b1/public_corpus.json

benchmark-smoke: build benchmark-corpus
	python3 research/benchmarks/b0b1/run_matrix.py --mode smoke

benchmark-micro:
	$(GO) test ./internal/experiment/benchmark -run '^$$' -bench '^BenchmarkPhase05' -benchmem -count=5 -benchtime=200x

benchmark-b1: build benchmark-corpus
	python3 research/benchmarks/b0b1/run_matrix.py --mode b1

benchmark-full: build benchmark-corpus
	python3 research/benchmarks/b0b1/run_matrix.py --mode full

benchmark-analyze:
	@test -n "$(PHASE05_RUN_ROOT)" || { echo 'set PHASE05_RUN_ROOT to a completed raw run directory'; exit 2; }
	python3 research/benchmarks/b0b1/analyze.py "$(PHASE05_RUN_ROOT)" "$(PHASE05_RUN_ROOT)/analysis" --require-complete

benchmark-contention-corpus:
	$(GO) run ./cmd/research/corpus/b0b1 -mode verify -file research/corpus/b0b1/public_corpus_contention.json

benchmark-contention-smoke: build benchmark-contention-corpus
	python3 research/benchmarks/b0b1/run_contention.py --mode smoke

benchmark-contention-full: build benchmark-contention-corpus
	python3 research/benchmarks/b0b1/run_contention.py --mode full

benchmark-contention-analyze:
	@test -n "$(PHASE05_RUN_ROOT)" || { echo 'set PHASE05_RUN_ROOT to a completed raw run directory'; exit 2; }
	python3 research/benchmarks/b0b1/analyze_contention.py "$(PHASE05_RUN_ROOT)" "$(PHASE05_RUN_ROOT)/analysis" --require-complete $(if $(PHASE05_REPAIR_ROOT),--repair-root "$(PHASE05_REPAIR_ROOT)",)
