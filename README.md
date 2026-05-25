# tube

<div align="center">

<img src="docs/assets/logo.svg" alt="tube logo" width="120">

### Local development proxy with `.test` domains

**Stop memorizing port numbers. Start using pretty URLs.**

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-Passing-brightgreen.svg)]()
[![macOS](https://img.shields.io/badge/macOS-Supported-black?logo=apple)]()

[Features](#-features) • [Quick Start](#-quick-start) • [Screenshots](#-screenshots) • [Documentation](#-documentation)

---

</div>

## The Problem

During development, you juggle multiple services:

```
Frontend:  http://localhost:3000
API:       http://localhost:8080
Admin:     http://localhost:4200
Docs:      http://localhost:3001
```

**tube** fixes this:

```
Frontend:  https://app.test
API:       https://api.test
Admin:     https://admin.test
Docs:      https://docs.test
```

Both HTTP and HTTPS work out of the box with trusted local certificates.

## ✨ Features

| Feature | Description |
|---------|-------------|
| 🌐 **Pretty URLs** | Access `myapp.test` instead of `localhost:3000` |
| 🔒 **HTTPS Support** | Auto-generated trusted certificates with mkcert |
| 🖥️ **Menu Bar App** | Control everything from your macOS menu bar |
| 📊 **Web Dashboard** | Beautiful UI at `localhost:3249` |
| ⚡ **Zero Config DNS** | One command sets up macOS resolver |
| 🔄 **Live Reload** | Add projects without restarting |
| 🛠️ **CLI + GUI** | Use whichever you prefer |

## 🚀 Quick Start

### 1. Install

The fastest path: a POSIX `sh` install script that detects your OS/arch,
verifies SHA256 against the published `checksums.txt`, and installs
`tube` to `/usr/local/bin`.

```bash
curl -fsSL https://raw.githubusercontent.com/steig/tube/main/scripts/install.sh | sh
```

Pin a specific version or install without sudo:

```bash
curl -fsSL https://raw.githubusercontent.com/steig/tube/main/scripts/install.sh \
  | TUBE_VERSION=v0.1.0 sh
curl -fsSL https://raw.githubusercontent.com/steig/tube/main/scripts/install.sh \
  | TUBE_PREFIX="$HOME/.local" sh
```

Runtime deps (nginx, dnsmasq, mkcert) aren't installed by the script — grab
them separately: `brew install nginx dnsmasq mkcert`.

> On macOS the install script also drops `tube-gui` (menu bar app)
> alongside `tube`. Run `tube-gui &` after installing.

<details>
<summary>Build from source instead</summary>

```bash
# Clone and build
git clone https://github.com/steig/tube.git
cd tube
make build-all

# Add to PATH
sudo cp bin/tube bin/tube-gui /usr/local/bin/
```
</details>

### 2. Initialize (one-time)

```bash
tube init
```

This configures tube and sets up HTTPS with trusted local certificates (requires [mkcert](https://github.com/FiloSottile/mkcert)).

### 3. Setup DNS (one-time)

```bash
tube setup
```

This creates `/etc/resolver/test` so `*.test` domains work automatically.

### 4. Add Your Projects

```bash
tube add myapp 3000
tube add api 8080
```

### 5. Start Services

```bash
tube start
```

### 6. Open Your App

```bash
open https://myapp.test
```

That's it! 🎉 Both `http://` and `https://` work.

---

## 📸 Screenshots

### Menu Bar App

Run `tube-gui` to get a persistent menu bar icon:

```
┌───────────────────────────────┐
│ tube ●                        │  ← Green = running
├───────────────────────────────┤
│ ● Services Running            │
├───────────────────────────────┤
│ Projects                     ▸│
│   ├─ myapp.test       :3000   │
│   ├─ api.test         :8080   │
│   └─ dashboard.test   :4200   │
├───────────────────────────────┤
│ Start Services                │
│ Stop Services                 │
├───────────────────────────────┤
│ Open Dashboard...             │
│ Quit tube                     │
└───────────────────────────────┘
```

### Web Dashboard

Access at `http://localhost:3249`:

```
╭──────────────────────────────────────────────────────────────────╮
│                                                                  │
│   tube Dashboard                              ● Running          │
│                                                                  │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│   Services                                                       │
│   ┌─────────────────────┐  ┌─────────────────────┐              │
│   │ ● nginx             │  │ ● dnsmasq           │              │
│   │   running (pid 123) │  │   running (pid 456) │              │
│   └─────────────────────┘  └─────────────────────┘              │
│                                                                  │
│   [Start Services]  [Stop Services]                              │
│                                                                  │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│   Projects                                                       │
│   ┌────────────────────────────────────────────────────────────┐│
│   │ ● myapp        http://myapp.test              :3000   [x]  ││
│   │ ○ api          http://api.test                :8080   [x]  ││
│   │ ● dashboard    http://dashboard.test          :4200   [x]  ││
│   └────────────────────────────────────────────────────────────┘│
│                                                                  │
│   [________name________] [__port__] [Add Project]                │
│                                                                  │
╰──────────────────────────────────────────────────────────────────╯
```

---

## 📖 CLI Reference

### Project Management

```bash
tube add <name> <port>    # Add a project (e.g., tube add api 8080)
tube remove <name>        # Remove a project
tube list                 # List all projects with status
```

### Service Control

```bash
tube start                # Start nginx + dnsmasq
tube stop                 # Stop all services
tube restart              # Restart all services
tube status               # Show service status
```

### System Setup

```bash
tube init                 # Initialize tube config + SSL certificates
tube setup                # Configure macOS DNS (run once)
tube uninstall            # Remove DNS configuration
tube dns-status           # Check DNS resolver status
tube doctor               # Diagnose issues
```

### SSL Management

```bash
tube ssl status           # Show SSL configuration
tube ssl install          # Install mkcert CA to trust store
tube ssl generate         # Generate/regenerate certificates
tube ssl enable           # Enable HTTPS
tube ssl disable          # Disable HTTPS (HTTP only)
```

### Example Session

```bash
$ tube add frontend 3000
✓ Added project 'frontend' on port 3000

$ tube add backend 8080
✓ Added project 'backend' on port 8080

$ tube list
NAME                 PORT     URL                                           STATUS
─────────────────────────────────────────────────────────────────────────────────
backend              8080     https://backend.test                          ○
frontend             3000     https://frontend.test                         ○

● = running  ○ = stopped

$ tube start
✓ Started nginx
✓ Started dnsmasq
Services are running!

$ tube status
nginx:   running (pid 12345)
dnsmasq: running (pid 12346)

$ open https://frontend.test  # 🎉 It works!
```

---

## ⚙️ Configuration

Configuration lives in `~/.tube/config.yaml`:

```yaml
# Domain settings
domain: example.com
tunnel_prefix: dev-

# Local proxy settings
proxy:
  local_domain: .test        # TLD for local domains
  dashboard_port: 3249       # Dashboard port

# Service binaries
nginx:
  binary: nginx
  http_port: 80
  https_port: 443

dnsmasq:
  binary: dnsmasq
  port: 53

# SSL/HTTPS settings
ssl:
  enabled: true
  cert_file: ~/.tube/ssl/wildcard.test.pem
  key_file: ~/.tube/ssl/wildcard.test-key.pem

# Your projects
projects:
  frontend: 3000
  backend: 8080
  admin: 4200
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TUBE_CONFIG` | Config file path | `~/.tube/config.yaml` |
| `TUBE_*` | Any config key, prefixed with `TUBE_` and uppercased | (per-key default) |

---

## 🏗️ Architecture

```
                        ┌─────────────────┐
                        │    Browser      │
                        │ myapp.test:443  │
                        └────────┬────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│                    macOS DNS Resolver                        │
│                                                              │
│  /etc/resolver/test  ──►  nameserver 127.0.0.1              │
│                                                              │
└────────────────────────────────┬────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│                        dnsmasq                               │
│                                                              │
│  *.test  ──►  127.0.0.1                                     │
│                                                              │
└────────────────────────────────┬────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│                         nginx                                │
│                                                              │
│  :80  (HTTP)   ──►  proxy_pass http://127.0.0.1:PORT        │
│  :443 (HTTPS)  ──►  proxy_pass http://127.0.0.1:PORT        │
│                     (SSL termination with mkcert certs)      │
│                                                              │
└────────────────────────────────┬────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│                    Your Application                          │
│                                                              │
│                    localhost:3000                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 🛠️ Development

### Prerequisites

- Go 1.23+
- nginx
- make

### Build

```bash
# Clone
git clone https://github.com/steig/tube.git
cd tube

# Enter nix shell (optional, but recommended)
nix develop

# Build CLI only
make build

# Build GUI only
make build-gui

# Build both
make build-all
```

### Test

```bash
# Run unit tests
make test

# Run tests in Docker (includes nginx/dnsmasq)
make test-docker

# Integration tests
make test-integration

# Coverage report
make coverage
```

### Project Structure

```
tube/
├── cmd/
│   ├── tube/              # CLI entry point
│   └── tube-gui/          # GUI entry point
├── internal/
│   ├── cli/               # CLI commands (cobra)
│   ├── config/            # Configuration (viper)
│   ├── dns/               # macOS resolver management
│   ├── gui/               # Menu bar + dashboard
│   ├── proxy/             # nginx/dnsmasq managers
│   ├── service/           # Process lifecycle
│   └── ssl/               # Certificate management (mkcert)
├── templates/
│   ├── nginx/             # nginx config templates
│   └── dnsmasq/           # dnsmasq config templates
├── docs/                  # Documentation
├── scripts/               # Build/test scripts
└── Makefile
```

---

## 📚 Documentation

| Document | Description |
|----------|-------------|
| [Installation](docs/installation.md) | Detailed install guide |
| [CLI Reference](docs/cli-reference.md) | All commands |
| [Configuration](docs/configuration.md) | Config options |
| [GUI Guide](docs/gui.md) | Menu bar & dashboard |
| [Architecture](docs/architecture.md) | How it works |
| [Troubleshooting](docs/troubleshooting.md) | Common issues |
| [Contributing](CONTRIBUTING.md) | How to contribute |

---

## 🗺️ Roadmap

- [x] Core CLI
- [x] nginx reverse proxy
- [x] macOS DNS resolver
- [x] Menu bar app
- [x] Web dashboard
- [x] HTTPS with mkcert
- [ ] Cloudflare Tunnel integration
- [ ] Linux support
- [ ] Homebrew formula

---

## 🤝 Contributing

Contributions welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) first.

```bash
# Fork, clone, branch
git checkout -b feature/my-feature

# Make changes, test
make test

# Commit and push
git commit -m "Add my feature"
git push origin feature/my-feature

# Open PR
```

---

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

---

<div align="center">

**Made for developers who are tired of remembering port numbers** 🚀

[Report Bug](https://github.com/steig/tube/issues) • [Request Feature](https://github.com/steig/tube/issues)

</div>
