.PHONY: build clean test test-integration test-integration-4v test-integration-b1-4v test-s04-4v test-s05-telemetry corpus-s04-verify vet fmt-check verify benchmark-corpus benchmark-smoke benchmark-micro benchmark-b1 benchmark-full benchmark-analyze benchmark-contention-corpus benchmark-contention-smoke benchmark-contention-full benchmark-contention-analyze

GO ?= go
INTEGRATION_TIMEOUT ?= 900s
REVISION := $(shell git rev-parse --short HEAD 2>/dev/null || printf unavailable)

build:
	mkdir -p build
	$(GO) build -ldflags "-X example.com/pluto/internal/version.Revision=$(REVISION)" -o build/plutod ./cmd/plutod
	$(GO) build -o build/cometbft github.com/cometbft/cometbft/cmd/cometbft
	$(GO) build -o build/profile-observer ./cmd/profile-observer
	$(GO) build -o build/profile-ingress ./cmd/profile-ingress

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

test-s05-telemetry: build corpus-s04-verify
	$(GO) test -tags 'integration fourval' ./test/integration -run '^TestS05FourValidatorTelemetry$$' -timeout 600s -count=1 -v

corpus-s04-verify:
	$(GO) run ./cmd/profile-corpus verify research/corpus/s04_public_corpus.json

vet:
	$(GO) vet ./...

fmt-check:
	@test -z "$$(gofmt -l cmd internal test)" || { gofmt -l cmd internal test; exit 1; }

verify: build test vet fmt-check
	$(GO) mod verify
	./build/plutod version

benchmark-corpus:
	$(GO) run ./cmd/b0b1-corpus -mode verify -file research/corpus/b0b1/public_corpus.json

benchmark-smoke: build benchmark-corpus
	python3 research/benchmarks/b0b1/run_matrix.py --mode smoke

benchmark-micro:
	$(GO) test ./internal/benchmarks/bench -run '^$$' -bench '^BenchmarkPhase05' -benchmem -count=5 -benchtime=200x

benchmark-b1: build benchmark-corpus
	python3 research/benchmarks/b0b1/run_matrix.py --mode b1

benchmark-full: build benchmark-corpus
	python3 research/benchmarks/b0b1/run_matrix.py --mode full

benchmark-analyze:
	@test -n "$(PHASE05_RUN_ROOT)" || { echo 'set PHASE05_RUN_ROOT to a completed raw run directory'; exit 2; }
	python3 research/benchmarks/b0b1/analyze.py "$(PHASE05_RUN_ROOT)" "$(PHASE05_RUN_ROOT)/analysis" --require-complete

benchmark-contention-corpus:
	$(GO) run ./cmd/b0b1-corpus -mode verify -file research/corpus/b0b1/public_corpus_contention.json

benchmark-contention-smoke: build benchmark-contention-corpus
	python3 research/benchmarks/b0b1/run_contention.py --mode smoke

benchmark-contention-full: build benchmark-contention-corpus
	python3 research/benchmarks/b0b1/run_contention.py --mode full

benchmark-contention-analyze:
	@test -n "$(PHASE05_RUN_ROOT)" || { echo 'set PHASE05_RUN_ROOT to a completed raw run directory'; exit 2; }
	python3 research/benchmarks/b0b1/analyze_contention.py "$(PHASE05_RUN_ROOT)" "$(PHASE05_RUN_ROOT)/analysis" --require-complete $(if $(PHASE05_REPAIR_ROOT),--repair-root "$(PHASE05_REPAIR_ROOT)",)
