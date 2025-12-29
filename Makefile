# Makefile for tube

.PHONY: build test test-unit test-integration test-all coverage lint clean help

# Default target
.DEFAULT_GOAL := help

# Build the CLI binary
build:
	go build -o bin/tube ./cmd/tube

# Build the GUI app
build-gui:
	go build -o bin/tube-gui ./cmd/tube-gui

# Build both
build-all: build build-gui

# Run unit tests locally (fast)
test: test-unit

test-unit:
	go test -v -short -race ./...

# Run unit tests in Docker
test-docker:
	docker compose -f docker-compose.test.yml run --rm unit

# Run integration tests in Docker
test-integration:
	docker compose -f docker-compose.test.yml run --rm integration

# Run all tests in Docker
test-all:
	docker compose -f docker-compose.test.yml run --rm test

# Generate coverage report
coverage:
	@mkdir -p coverage
	docker compose -f docker-compose.test.yml run --rm coverage
	@echo "Coverage report: coverage/coverage.html"

# Run linter
lint:
	golangci-lint run ./...

# Build Docker test image
docker-build:
	docker compose -f docker-compose.test.yml build

# Clean up
clean:
	rm -rf bin/ coverage/
	docker compose -f docker-compose.test.yml down --volumes --remove-orphans 2>/dev/null || true

# Install development dependencies
dev-deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest

# Run with hot reload (requires air)
dev:
	air

# Help
help:
	@echo "tube - Local development proxy"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build:"
	@echo "  build           Build the tube CLI binary"
	@echo "  build-gui       Build the tube GUI app (menu bar + dashboard)"
	@echo "  build-all       Build both CLI and GUI"
	@echo "  dev             Run with hot reload (requires air)"
	@echo ""
	@echo "Test:"
	@echo "  test            Run unit tests locally (fast)"
	@echo "  test-docker     Run unit tests in Docker"
	@echo "  test-integration Run integration tests in Docker"
	@echo "  test-all        Run all tests in Docker"
	@echo "  coverage        Generate coverage report"
	@echo ""
	@echo "Other:"
	@echo "  lint            Run linter"
	@echo "  clean           Clean up build artifacts and Docker resources"
	@echo "  dev-deps        Install development dependencies"
	@echo "  help            Show this help message"
