# Justfile for tube project
set shell := ["bash", "-c"]

# Default recipe - show help
default:
    @just --list

# Build tube binary
build:
    go build -o bin/tube ./cmd/tube

# Build with version info
build-release:
    go build -ldflags="-s -w -X main.Version=$(git describe --tags --always --dirty) -X main.Commit=$(git rev-parse --short HEAD) -X main.Date=$(date -u +'%Y-%m-%dT%H:%M:%SZ')" -o bin/tube ./cmd/tube

# Run tube in development
run *ARGS:
    go run ./cmd/tube {{ARGS}}

# Run tests
test:
    go test -v ./...

# Run tests with coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    echo "Coverage report: coverage.html"

# Run linter
lint:
    golangci-lint run ./...

# Fix linter issues
lint-fix:
    golangci-lint run --fix ./...

# Format code
fmt:
    go fmt ./...

# Vet code
vet:
    go vet ./...

# Hot reload development (requires air)
dev:
    air

# Install locally
install: build
    cp bin/tube /usr/local/bin/tube
    echo "✓ tube installed to /usr/local/bin/tube"

# Clean build artifacts
clean:
    rm -rf bin/
    rm -rf dist/
    rm -f coverage.out coverage.html
    go clean

# Build for all platforms (requires goreleaser)
build-all:
    goreleaser build --clean --snapshot

# Generate release (create git tag first)
release:
    goreleaser release --clean

# Initialize git hooks
hooks:
    git config core.hooksPath .githooks
    mkdir -p .githooks
    chmod +x .githooks/*
    echo "✓ Git hooks configured"

# Generate mocks (if using mockery)
mocks:
    @echo "TODO: Add mock generation if needed"

# Check dependencies
doctor:
    go run ./cmd/tube doctor

# Show version info
version:
    go run ./cmd/tube --version

# Run a specific command
help:
    go run ./cmd/tube --help
