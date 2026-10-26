.PHONY: build test test-integration test-integration-4v test-integration-b1-4v vet fmt-check verify

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
