# Makefile for space-cli

# Binary name
BINARY_NAME=space

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOINSTALL=$(GOCMD) install
GOMOD=$(GOCMD) mod

# Build directory
BUILD_DIR=bin

# Version info
VERSION?=$(shell cat VERSION 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Linker flags
LDFLAGS=-ldflags "-X github.com/happy-sdk/space-cli/internal/cli.Version=$(VERSION) \
	-X github.com/happy-sdk/space-cli/internal/cli.BuildTime=$(BUILD_TIME) \
	-X github.com/happy-sdk/space-cli/internal/cli.GitCommit=$(GIT_COMMIT)"

.PHONY: all build install clean test test-e2e test-e2e-verbose test-all deps help version version-patch version-minor version-major e2e-clean

# Default target
all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/space
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## install: Install to GOBIN
install:
	@echo "Installing $(BINARY_NAME) to GOBIN..."
	$(GOINSTALL) $(LDFLAGS) ./cmd/space
	@echo "Installed successfully!"
	@echo "Run '$(BINARY_NAME) --version' to verify"

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

## lint: Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

## run: Build and run
run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

## version: Show current version
version:
	@./scripts/version.sh get

## version-patch: Bump patch version
version-patch:
	@./scripts/version.sh patch

## version-minor: Bump minor version
version-minor:
	@./scripts/version.sh minor

## version-major: Bump major version
version-major:
	@./scripts/version.sh major

## test-e2e: Run e2e tests (requires Docker)
test-e2e: build
	@echo "Running e2e tests..."
	@echo "Note: This requires Docker to be running"
	SPACE_BIN=$(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) $(GOTEST) -v -tags=e2e -timeout 10m ./e2e/...

## test-e2e-verbose: Run e2e tests with verbose output
test-e2e-verbose: build
	@echo "Running e2e tests (verbose)..."
	SPACE_BIN=$(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) $(GOTEST) -v -tags=e2e -timeout 10m -count=1 ./e2e/...

## test-e2e-single: Run a single e2e test (use TEST=TestName)
test-e2e-single: build
	@echo "Running single e2e test: $(TEST)"
	SPACE_BIN=$(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) $(GOTEST) -v -tags=e2e -timeout 5m -run '$(TEST)' ./e2e/...

## test-all: Run all tests (unit and e2e)
test-all: test test-e2e
	@echo "All tests complete!"

## e2e-clean: Clean up any leftover e2e test containers
e2e-clean:
	@echo "Cleaning up e2e test containers..."
	@docker ps -a --filter "name=e2e-" -q | xargs -r docker rm -f 2>/dev/null || true
	@docker network ls --filter "name=e2e-" -q | xargs -r docker network rm 2>/dev/null || true
	@echo "E2E cleanup complete"

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@sed -n 's/^##//p' Makefile | column -t -s ':' | sed -e 's/^/ /'
