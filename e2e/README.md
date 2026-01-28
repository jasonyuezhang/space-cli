# E2E Testing for space-cli

This directory contains end-to-end tests for the space-cli tool. The test framework is inspired by best practices from open source projects like [alphagov/publishing-e2e-tests](https://github.com/alphagov/publishing-e2e-tests) and patterns described in [Robust end-to-end testing with Docker Compose](https://sundin.github.io/e2e-tests-with-docker-compose/).

## Directory Structure

```
e2e/
├── README.md              # This file
├── fixtures/              # Test fixtures (docker-compose projects)
│   ├── simple-app/        # Simple single-service app
│   │   ├── docker-compose.yml
│   │   ├── .space.yaml
│   │   └── html/
│   └── multi-service/     # Multi-service app with dependencies
│       ├── docker-compose.yml
│       ├── .space.yaml
│       ├── api/
│       └── frontend/
├── framework/             # Test framework utilities
│   ├── framework.go       # Core test framework
│   └── assertions.go      # Assertion helpers
├── scripts/
│   └── run-e2e.sh         # E2E test runner script
├── up_down_test.go        # Tests for space up/down commands
├── ps_test.go             # Tests for space ps command
├── dns_test.go            # Tests for DNS functionality
└── config_test.go         # Tests for config commands
```

## Prerequisites

- Docker installed and running
- Go 1.21 or later
- The space-cli binary (will be built automatically)

## Running E2E Tests

### Using Make (Recommended)

```bash
# Run all e2e tests
make test-e2e

# Run e2e tests with verbose output
make test-e2e-verbose

# Run a single test
make test-e2e-single TEST=TestSpaceUpSimple

# Run all tests (unit + e2e)
make test-all

# Clean up leftover test containers
make e2e-clean
```

### Using the Shell Script

```bash
# Run all e2e tests
./e2e/scripts/run-e2e.sh run

# Run specific test
./e2e/scripts/run-e2e.sh run TestSpaceUpSimple

# Quick run (skip build if binary exists)
./e2e/scripts/run-e2e.sh quick

# Check prerequisites
./e2e/scripts/run-e2e.sh check

# Clean up test containers
./e2e/scripts/run-e2e.sh clean
```

### Using Go Directly

```bash
# Build the binary first
make build

# Run e2e tests
SPACE_BIN=./bin/space go test -v -tags=e2e -timeout 10m ./e2e/...
```

## Test Framework

### Creating a New Test

```go
//go:build e2e
// +build e2e

package e2e

import (
    "context"
    "testing"
    "time"

    "github.com/happy-sdk/space-cli/e2e/framework"
)

func TestMyFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping e2e test in short mode")
    }

    // Create framework with fixture
    f := framework.New(t).WithFixture("simple-app")
    defer f.Cleanup()

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    // Run space commands
    result := f.RunSpaceCmd(ctx, "up")

    // Assert results
    assert := framework.Assert(t)
    assert.CmdSucceeds(result, "space up should succeed")
    assert.CmdOutputContains(result, "Starting services")

    // Wait for services
    err := f.WaitForService(ctx, "http://localhost:8080", 30*time.Second)
    assert.NoError(err, "service should be accessible")
}
```

### Available Assertions

```go
assert := framework.Assert(t)

// Command assertions
assert.CmdSucceeds(result, "message")
assert.CmdFails(result, "message")
assert.CmdOutputContains(result, "substring")
assert.CmdOutputNotContains(result, "substring")

// General assertions
assert.NoError(err, "message")
assert.Error(err, "message")
assert.True(condition, "message")
assert.False(condition, "message")
assert.Equal(expected, actual, "message")
assert.StringContains(s, substring, "message")
assert.StringNotEmpty(s, "message")
```

### Framework Methods

```go
f := framework.New(t)

// Setup with fixture
f.WithFixture("simple-app")

// Run space CLI commands
result := f.RunSpaceCmd(ctx, "up", "web")

// Wait for services
f.WaitForService(ctx, "http://localhost:8080", 30*time.Second)
f.WaitForContainer(ctx, "container-name", 30*time.Second)

// Get docker compose status
output, err := f.DockerComposePS(ctx, "project-name")

// Cleanup (called in defer)
f.Cleanup()
```

## Test Fixtures

### simple-app

A basic nginx web server for testing fundamental commands:
- Single service
- Health check endpoint
- Port binding

### multi-service

A more complex setup for testing service dependencies:
- Frontend (nginx)
- API (nginx serving JSON)
- PostgreSQL database
- Redis cache
- Service dependencies with health checks

## Writing Good E2E Tests

1. **Use timeouts**: Always use context with timeout to prevent hanging tests
2. **Clean up**: Always defer `f.Cleanup()` to stop containers
3. **Skip in short mode**: Allow skipping in `-short` mode for quick test runs
4. **Wait for health**: Use `WaitForService` or `WaitForContainer` instead of fixed sleeps
5. **Test one thing**: Each test should verify a specific behavior
6. **Use descriptive names**: Test names should describe what's being tested

## Debugging Failed Tests

1. Check Docker logs:
   ```bash
   docker logs <container-name>
   ```

2. List running containers:
   ```bash
   docker ps -a --filter "name=e2e-"
   ```

3. Run tests with verbose output:
   ```bash
   make test-e2e-verbose
   ```

4. Clean up orphaned containers:
   ```bash
   make e2e-clean
   ```

## CI/CD Integration

The e2e tests can be integrated into CI/CD pipelines. See the test patterns from:
- [Dockerized E2E Tests with GitHub Actions](https://lachiejames.com/elevate-your-ci-cd-dockerized-e2e-tests-with-github-actions/)
- [Docker + Cypress in 2025](https://dev.to/cypress/docker-cypress-in-2025-how-ive-perfected-my-e2e-testing-setup-4f7j)

Example GitHub Actions workflow:

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Run E2E Tests
        run: make test-e2e

      - name: Cleanup
        if: always()
        run: make e2e-clean
```
