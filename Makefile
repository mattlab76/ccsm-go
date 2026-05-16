APP_NAME := ccsm
VERSION := $(shell grep 'Version = ' internal/model/session.go | cut -d'"' -f2)
GOFLAGS := -trimpath
LDFLAGS := -s -w
BUILD := go build $(GOFLAGS) -ldflags '$(LDFLAGS)'

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 freebsd/amd64 windows/amd64

.PHONY: build build-all test test-v test-report lint install clean help

## build: Build for current platform
build:
	$(BUILD) -o $(APP_NAME) ./cmd/ccsm

## build-all: Cross-compile for all platforms
build-all:
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		ext=""; \
		[ "$$os" = "windows" ] && ext=".exe"; \
		echo "Building $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch $(BUILD) -o dist/$(APP_NAME)-$$os-$$arch$$ext ./cmd/ccsm || exit 1; \
	done
	@echo "Done. Binaries in dist/"

## test: Run all tests
test:
	go test ./... -count=1

## test-v: Run all tests with verbose output
test-v:
	go test ./... -v -count=1

## test-report: Run tests + coverage, append entry to TESTRESULTS.md, stage the file
test-report:
	@./scripts/test-report.sh

## lint: Run golangci-lint (install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
lint:
	@which golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found, skipping"; exit 0; }
	golangci-lint run ./...

## install: Install to ~/.local/bin
install: build
	@mkdir -p $(HOME)/.local/bin
	cp $(APP_NAME) $(HOME)/.local/bin/$(APP_NAME)
	@echo "Installed $(APP_NAME) v$(VERSION) to ~/.local/bin/"

## uninstall: Remove from ~/.local/bin
uninstall:
	rm -f $(HOME)/.local/bin/$(APP_NAME)
	@echo "Removed $(APP_NAME) from ~/.local/bin/"

## clean: Remove build artifacts
clean:
	rm -f $(APP_NAME)
	rm -rf dist/

## version: Print version
version:
	@echo "$(APP_NAME) v$(VERSION)"

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
