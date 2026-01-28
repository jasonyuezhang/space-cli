#!/bin/bash
# E2E Test Runner Script for space-cli
# Inspired by best practices from open source docker-compose e2e testing patterns

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$PROJECT_ROOT/bin"
BINARY="$BUILD_DIR/space"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi

    # Check Docker is running
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi

    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi

    log_info "All prerequisites met"
}

# Build the binary
build_binary() {
    log_info "Building space binary..."
    cd "$PROJECT_ROOT"
    make build

    if [ ! -f "$BINARY" ]; then
        log_error "Binary not found at $BINARY"
        exit 1
    fi

    log_info "Binary built successfully: $BINARY"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test containers..."

    # Stop any running e2e containers
    docker ps -a --filter "name=e2e-" -q | xargs -r docker rm -f 2>/dev/null || true

    # Remove e2e networks
    docker network ls --filter "name=e2e-" -q | xargs -r docker network rm 2>/dev/null || true

    log_info "Cleanup complete"
}

# Run tests
run_tests() {
    local test_pattern="${1:-}"

    log_info "Running e2e tests..."

    cd "$PROJECT_ROOT"

    # Set environment for tests
    export SPACE_BIN="$BINARY"

    if [ -n "$test_pattern" ]; then
        log_info "Running tests matching: $test_pattern"
        go test -v -tags=e2e -timeout 10m -run "$test_pattern" ./e2e/...
    else
        log_info "Running all e2e tests"
        go test -v -tags=e2e -timeout 10m ./e2e/...
    fi
}

# Main
main() {
    local cmd="${1:-run}"
    local test_pattern="${2:-}"

    case "$cmd" in
        check)
            check_prerequisites
            ;;
        build)
            check_prerequisites
            build_binary
            ;;
        clean)
            cleanup
            ;;
        run)
            check_prerequisites
            build_binary
            trap cleanup EXIT
            run_tests "$test_pattern"
            ;;
        quick)
            # Quick mode: skip build if binary exists
            check_prerequisites
            if [ ! -f "$BINARY" ]; then
                build_binary
            fi
            trap cleanup EXIT
            run_tests "$test_pattern"
            ;;
        *)
            echo "Usage: $0 {check|build|clean|run|quick} [test_pattern]"
            echo ""
            echo "Commands:"
            echo "  check  - Check prerequisites"
            echo "  build  - Build the space binary"
            echo "  clean  - Cleanup test containers"
            echo "  run    - Build and run all e2e tests"
            echo "  quick  - Run tests (skip build if binary exists)"
            echo ""
            echo "Examples:"
            echo "  $0 run                     # Run all e2e tests"
            echo "  $0 run TestSpaceUpSimple   # Run specific test"
            echo "  $0 quick TestSpacePs       # Quick run matching tests"
            exit 1
            ;;
    esac
}

main "$@"
