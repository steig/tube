#!/usr/bin/env bash
# Test runner script for tube
# Usage: ./scripts/test.sh [unit|integration|coverage|all]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${GREEN}==>${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}WARNING:${NC} $1"
}

print_error() {
    echo -e "${RED}ERROR:${NC} $1"
}

# Check if Docker is available
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed"
        exit 1
    fi
    if ! docker info &> /dev/null; then
        print_error "Docker daemon is not running"
        exit 1
    fi
}

# Run unit tests locally (fast, no Docker needed)
run_unit_tests() {
    print_status "Running unit tests..."
    go test -v -short -race ./...
}

# Run unit tests in Docker
run_docker_unit_tests() {
    check_docker
    print_status "Running unit tests in Docker..."
    docker compose -f docker-compose.test.yml run --rm unit
}

# Run integration tests in Docker (requires nginx/dnsmasq)
run_integration_tests() {
    check_docker
    print_status "Running integration tests in Docker..."
    docker compose -f docker-compose.test.yml run --rm integration
}

# Run all tests in Docker
run_all_tests() {
    check_docker
    print_status "Running all tests in Docker..."
    docker compose -f docker-compose.test.yml run --rm test
}

# Generate coverage report
run_coverage() {
    check_docker
    print_status "Generating coverage report..."
    mkdir -p coverage
    docker compose -f docker-compose.test.yml run --rm coverage

    if [ -f coverage/coverage.html ]; then
        print_status "Coverage report generated: coverage/coverage.html"
        if command -v open &> /dev/null; then
            open coverage/coverage.html
        fi
    fi
}

# Clean up Docker resources
cleanup() {
    print_status "Cleaning up Docker resources..."
    docker compose -f docker-compose.test.yml down --volumes --remove-orphans
}

# Build test image
build() {
    check_docker
    print_status "Building test Docker image..."
    docker compose -f docker-compose.test.yml build
}

# Show usage
usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  unit          Run unit tests locally (fast, no Docker)"
    echo "  docker-unit   Run unit tests in Docker"
    echo "  integration   Run integration tests in Docker (requires nginx/dnsmasq)"
    echo "  all           Run all tests in Docker"
    echo "  coverage      Generate coverage report"
    echo "  build         Build test Docker image"
    echo "  clean         Clean up Docker resources"
    echo ""
    echo "Default: unit"
}

# Main
case "${1:-unit}" in
    unit)
        run_unit_tests
        ;;
    docker-unit)
        run_docker_unit_tests
        ;;
    integration)
        run_integration_tests
        ;;
    all)
        run_all_tests
        ;;
    coverage)
        run_coverage
        ;;
    build)
        build
        ;;
    clean)
        cleanup
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        print_error "Unknown command: $1"
        usage
        exit 1
        ;;
esac
