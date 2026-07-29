.PHONY: build build-all clean test test-race lint vet fmt fmt-check check tidy neo morpheus oracle trinity release

# Build output directory
BIN_DIR := bin

# Version injection via ldflags
NEO_VERSION := 1.7.2-dev
MORPHEUS_VERSION := 1.7.2-dev
ORACLE_VERSION := 1.7.2-dev
TRINITY_VERSION := 1.7.2-dev

LDFLAGS_NEO := -ldflags "-X main.version=$(NEO_VERSION)"
LDFLAGS_MORPHEUS := -ldflags "-X main.version=$(MORPHEUS_VERSION)"
LDFLAGS_ORACLE := -ldflags "-X main.version=$(ORACLE_VERSION)"
LDFLAGS_TRINITY := -ldflags "-X main.version=$(TRINITY_VERSION)"

## build: Build all 4 binaries
build: build-all

build-all: neo morpheus oracle trinity

neo:
	go build $(LDFLAGS_NEO) -o $(BIN_DIR)/neo ./cmd/neo

morpheus:
	go build $(LDFLAGS_MORPHEUS) -o $(BIN_DIR)/morpheus ./cmd/morpheus

oracle:
	go build $(LDFLAGS_ORACLE) -o $(BIN_DIR)/oracle ./cmd/oracle

trinity:
	go build $(LDFLAGS_TRINITY) -o $(BIN_DIR)/trinity ./cmd/trinity

## test: Run all tests (with race detection)
test: test-race

## test-race: Run all tests with -race
test-race:
	go test ./... -race

## vet: Run go vet
vet:
	go vet ./...

## fmt: Format all Go source (mutating)
fmt:
	gofmt -w .

## fmt-check: Verify all Go source is gofmt-clean (non-mutating, for CI)
fmt-check:
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then \
		echo "gofmt found unformatted files:"; echo "$$out"; exit 1; \
	fi

## lint: Run golangci-lint (uses .golangci.yml). Falls back to vet if golangci-lint is not installed.
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed; falling back to go vet"; \
		go vet ./...; \
	fi

## check: Run fmt-check + lint + test-race (full pre-push gate)
check: fmt-check lint test-race

## tidy: Clean up go.mod
tidy:
	go mod tidy

## release: Tag and push a release (triggers CI workflow)
## Usage: make release VERSION=1.2.0 SUMMARY="one-line summary"
## Note: SUMMARY must not contain quotes or dollar signs.
release:
ifeq ($(strip $(VERSION)),)
	$(error VERSION is required. Usage: make release VERSION=1.2.0 SUMMARY="one-line summary")
endif
ifeq ($(strip $(SUMMARY)),)
	$(error SUMMARY is required. Usage: make release VERSION=1.2.0 SUMMARY="one-line summary")
endif
	@test "$$(git branch --show-current)" = "main" || (echo "Error: releases must be created from the main branch." && exit 1)
	@test -z "$$(git status --porcelain | grep -v '^??')" || (echo "Error: uncommitted changes. Commit or stash first." && exit 1)
	@echo "Running tests before tagging..."
	go test ./... -race
	@echo "Pushing commits..."
	git push origin HEAD
	@echo "Tagging v$(VERSION) and pushing..."
	git tag -a v$(VERSION) -m "Release v$(VERSION) — $(SUMMARY)"
	git push origin v$(VERSION)

## clean: Remove build artifacts
clean:
	rm -rf $(BIN_DIR)
