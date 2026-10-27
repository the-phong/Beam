.PHONY: build test test-integration test-integration-4v test-integration-b1-4v vet fmt-check verify benchmark-corpus benchmark-smoke benchmark-micro benchmark-b1 benchmark-full benchmark-analyze benchmark-contention-corpus benchmark-contention-smoke benchmark-contention-full benchmark-contention-analyze

GO ?= go
REVISION := $(shell git rev-parse --short HEAD 2>/dev/null || printf unavailable)

build:
	mkdir -p build
	$(GO) build -ldflags "-X example.com/pluto/internal/version.Revision=$(REVISION)" -o build/plutod ./cmd/plutod
	$(GO) build -o build/cometbft github.com/cometbft/cometbft/cmd/cometbft

test:
	$(GO) test ./...

test-integration: build
	$(GO) test -tags integration ./test/integration -timeout 240s -count=1 -v

test-integration-4v: build
	$(GO) test -tags 'integration fourval' ./test/integration -run '^TestB0FourValidator' -timeout 900s -count=1 -v

test-integration-b1-4v: build
	$(GO) test -tags 'integration fourval' ./test/integration -run '^TestB1Hybrid' -timeout 900s -count=1 -v

vet:
	$(GO) vet ./...

fmt-check:
	@test -z "$$(gofmt -l cmd internal test)" || { gofmt -l cmd internal test; exit 1; }

verify: build test vet fmt-check
	$(GO) mod verify
	./build/plutod version

benchmark-corpus:
	$(GO) run ./cmd/phase05-corpus -mode verify -file research/corpus/phase_05/public_corpus.json

benchmark-smoke: build benchmark-corpus
	python3 research/benchmarks/phase05/run_matrix.py --mode smoke

benchmark-micro:
	$(GO) test ./internal/phase05/bench -run '^$$' -bench '^BenchmarkPhase05' -benchmem -count=5 -benchtime=200x

benchmark-b1: build benchmark-corpus
	python3 research/benchmarks/phase05/run_matrix.py --mode b1

benchmark-full: build benchmark-corpus
	python3 research/benchmarks/phase05/run_matrix.py --mode full

benchmark-analyze:
	@test -n "$(PHASE05_RUN_ROOT)" || { echo 'set PHASE05_RUN_ROOT to a completed raw run directory'; exit 2; }
	python3 research/benchmarks/phase05/analyze.py "$(PHASE05_RUN_ROOT)" "$(PHASE05_RUN_ROOT)/analysis" --require-complete

benchmark-contention-corpus:
	$(GO) run ./cmd/phase05-corpus -mode verify -file research/corpus/phase_05/public_corpus_contention.json

benchmark-contention-smoke: build benchmark-contention-corpus
	python3 research/benchmarks/phase05/run_contention.py --mode smoke

benchmark-contention-full: build benchmark-contention-corpus
	python3 research/benchmarks/phase05/run_contention.py --mode full

benchmark-contention-analyze:
	@test -n "$(PHASE05_RUN_ROOT)" || { echo 'set PHASE05_RUN_ROOT to a completed raw run directory'; exit 2; }
	python3 research/benchmarks/phase05/analyze_contention.py "$(PHASE05_RUN_ROOT)" "$(PHASE05_RUN_ROOT)/analysis" --require-complete $(if $(PHASE05_REPAIR_ROOT),--repair-root "$(PHASE05_REPAIR_ROOT)",)
