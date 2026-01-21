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
VERSION?=dev
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Linker flags
LDFLAGS=-ldflags "-X github.com/happy-sdk/space-cli/internal/cli.Version=$(VERSION) \
	-X github.com/happy-sdk/space-cli/internal/cli.BuildTime=$(BUILD_TIME) \
	-X github.com/happy-sdk/space-cli/internal/cli.GitCommit=$(GIT_COMMIT)"

.PHONY: all build install clean test deps help

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

## version: Show version
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@sed -n 's/^##//p' Makefile | column -t -s ':' | sed -e 's/^/ /'
