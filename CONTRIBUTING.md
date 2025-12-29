# Contributing to tube

Thank you for your interest in contributing to tube! This guide will help you get started.

## Code of Conduct

Be respectful and constructive in all interactions. We're all here to build great software together.

## Getting Started

### Prerequisites

- **Go 1.23+**
- **nginx** (for testing)
- **dnsmasq** (for testing)
- **Docker** (for running tests)
- **make**

### Development Setup

```bash
# Clone the repository
git clone https://github.com/steig/tube.git
cd tube

# Option 1: Use Nix (recommended)
nix develop

# Option 2: Install dependencies manually
brew install go nginx dnsmasq

# Install development tools
make dev-deps

# Build
make build-all

# Run tests
make test
```

### Project Structure

```
tube/
├── cmd/
│   ├── tube/              # CLI entry point
│   └── tube-gui/          # GUI entry point
├── internal/
│   ├── cli/               # CLI commands
│   ├── config/            # Configuration
│   ├── dns/               # DNS resolver management
│   ├── gui/               # GUI components
│   ├── proxy/             # nginx/dnsmasq managers
│   └── service/           # Process management
├── docs/                  # Documentation
├── templates/             # Config templates
└── scripts/               # Build scripts
```

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/my-feature
# or
git checkout -b fix/my-bugfix
```

### 2. Make Changes

- Write clean, idiomatic Go code
- Follow existing patterns and conventions
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

```bash
# Run unit tests
make test

# Run tests in Docker (includes nginx/dnsmasq)
make test-docker

# Run integration tests
make test-integration

# Run linter
make lint
```

### 4. Commit Your Changes

We use conventional commits:

```bash
# Features
git commit -m "feat: add HTTPS support"

# Bug fixes
git commit -m "fix: resolve DNS cache issue"

# Documentation
git commit -m "docs: update installation guide"

# Refactoring
git commit -m "refactor: simplify config loading"
```

### 5. Push and Create PR

```bash
git push origin feature/my-feature
```

Then open a pull request on GitHub.

## Code Style

### Go Code

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Run `gofmt` on all code
- Run `golangci-lint` before committing

```bash
# Format code
gofmt -w .

# Run linter
make lint
```

### Naming Conventions

- Packages: lowercase, single word (`config`, `proxy`)
- Interfaces: verb + "er" suffix (`Reader`, `Manager`)
- Exported functions: PascalCase (`NewConfig`, `Start`)
- Unexported functions: camelCase (`parsePort`, `writeFile`)

### Error Handling

```go
// Good: wrap errors with context
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

// Good: use meaningful error messages
if port < 1 || port > 65535 {
    return fmt.Errorf("invalid port %d: must be 1-65535", port)
}
```

### Comments

```go
// Package proxy provides managers for nginx and dnsmasq.
package proxy

// NginxManager handles nginx configuration and lifecycle.
type NginxManager struct {
    // ...
}

// WriteConfig generates and writes the nginx configuration file.
// It creates a server block for each configured project.
func (nm *NginxManager) WriteConfig() error {
    // ...
}
```

## Testing

### Unit Tests

Place tests next to the code they test:

```
internal/config/
├── config.go
└── config_test.go
```

Use table-driven tests:

```go
func TestValidatePort(t *testing.T) {
    tests := []struct {
        name    string
        port    int
        wantErr bool
    }{
        {"valid port", 3000, false},
        {"min port", 1, false},
        {"max port", 65535, false},
        {"zero port", 0, true},
        {"negative port", -1, true},
        {"too high", 65536, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePort(tt.port)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidatePort(%d) error = %v, wantErr %v",
                    tt.port, err, tt.wantErr)
            }
        })
    }
}
```

### Integration Tests

Use the `integration` build tag:

```go
//go:build integration

package integration

func TestFullWorkflow(t *testing.T) {
    // Test that requires actual nginx/dnsmasq
}
```

Run with:
```bash
make test-integration
```

### Test Coverage

Generate coverage report:

```bash
make coverage
open coverage/coverage.html
```

Aim for >80% coverage on new code.

## Documentation

### Code Documentation

All exported functions, types, and packages should have doc comments:

```go
// Package config provides configuration loading and management for tube.
package config

// Config represents the tube configuration.
// It is loaded from ~/.tube/config.yaml by default.
type Config struct {
    // ...
}

// Load reads and parses the configuration file at the given path.
// If path is empty, it uses the default location.
func Load(path string) (*Config, error) {
    // ...
}
```

### User Documentation

Update relevant docs when changing behavior:

- `README.md` - Overview and quick start
- `docs/installation.md` - Installation instructions
- `docs/cli-reference.md` - CLI commands
- `docs/configuration.md` - Configuration options
- `docs/gui.md` - GUI features
- `docs/architecture.md` - How it works
- `docs/troubleshooting.md` - Common issues

## Pull Request Guidelines

### Before Submitting

- [ ] Tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Code is formatted (`gofmt -w .`)
- [ ] Documentation updated (if applicable)
- [ ] Commit messages follow conventions

### PR Description

Include:
1. **What** - What does this PR do?
2. **Why** - Why is this change needed?
3. **How** - How does it work?
4. **Testing** - How was it tested?

### Example PR

```markdown
## Summary

Add HTTPS support using mkcert for local development.

## Changes

- Added SSLManager to handle certificate generation
- Updated NginxManager to configure SSL server blocks
- Added `tube ssl enable` and `tube ssl disable` commands

## Testing

- Added unit tests for SSLManager
- Tested manually on macOS 14.0
- Verified certificates work in Chrome and Firefox

## Related Issues

Closes #42
```

## Areas for Contribution

### Completed

- [x] Core CLI functionality
- [x] nginx reverse proxy
- [x] dnsmasq DNS resolution
- [x] macOS DNS resolver setup
- [x] Menu bar application
- [x] Web dashboard
- [x] Project management (add/remove/list)
- [x] Service lifecycle (start/stop/restart/status)

### In Progress

- [ ] HTTPS support with mkcert
- [ ] Cloudflare Tunnel integration

### Planned

- [ ] Linux support
- [ ] Homebrew formula
- [ ] Auto-refresh dashboard
- [ ] Log viewer in dashboard
- [ ] Per-project SSL settings

## Reporting Issues

### Bug Reports

Include:
1. macOS version (`sw_vers`)
2. tube version (`tube version`)
3. Output of `tube doctor`
4. Steps to reproduce
5. Expected vs actual behavior
6. Error messages (full output)

### Feature Requests

Include:
1. Description of the feature
2. Use case / why it's needed
3. Proposed implementation (optional)
4. Examples from similar tools (optional)

## Release Process

(For maintainers)

1. Update version in code
2. Update CHANGELOG.md
3. Create release branch: `release/v1.0.0`
4. Run full test suite
5. Create GitHub release with notes
6. Tag the release: `v1.0.0`

## Getting Help

- Check existing issues and PRs
- Read the documentation
- Open a discussion for questions

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to tube! Your efforts help make local development easier for everyone.
