# Installation Guide

This guide covers all installation methods for tube.

## Prerequisites

### Required
- **macOS** (10.15 Catalina or later)
- **Go 1.23+** (for building from source)
- **nginx** (for reverse proxy)
- **dnsmasq** (for DNS resolution)

### Optional
- **Homebrew** (easiest way to install dependencies)
- **Nix** (for reproducible development environment)

## Quick Install

### Using Homebrew (Recommended)

```bash
# Install dependencies
brew install go nginx dnsmasq

# Clone and build
git clone https://github.com/steig/tube.git
cd tube
make build-all

# Install binaries
sudo cp bin/tube bin/tube-gui /usr/local/bin/

# Setup DNS (one-time)
sudo tube setup
```

### Using Nix

If you have Nix installed with flakes enabled:

```bash
# Clone the repository
git clone https://github.com/steig/tube.git
cd tube

# Enter development shell (includes all dependencies)
nix develop

# Build
make build-all

# Install
sudo cp bin/tube bin/tube-gui /usr/local/bin/
```

## Step-by-Step Installation

### 1. Install Go

**Using Homebrew:**
```bash
brew install go
```

**Manual installation:**
1. Download from https://go.dev/dl/
2. Follow the installation instructions
3. Ensure `go` is in your PATH

Verify installation:
```bash
go version
# Should output: go version go1.23.x darwin/arm64 (or amd64)
```

### 2. Install nginx

**Using Homebrew:**
```bash
brew install nginx
```

**Verify installation:**
```bash
nginx -v
# Should output: nginx version: nginx/1.x.x
```

Note: tube manages nginx configuration automatically. You don't need to configure nginx manually.

### 3. Install dnsmasq

**Using Homebrew:**
```bash
brew install dnsmasq
```

**Verify installation:**
```bash
dnsmasq --version
# Should output: Dnsmasq version 2.x
```

Note: tube manages dnsmasq configuration automatically.

### 4. Build tube

```bash
# Clone the repository
git clone https://github.com/steig/tube.git
cd tube

# Build CLI only
make build

# Build GUI only
make build-gui

# Build both CLI and GUI
make build-all
```

The binaries will be created in the `bin/` directory:
- `bin/tube` - Command-line interface
- `bin/tube-gui` - Menu bar app with web dashboard

### 5. Install Binaries

```bash
# System-wide installation (recommended)
sudo cp bin/tube bin/tube-gui /usr/local/bin/

# Or add to your PATH
export PATH="$PATH:$(pwd)/bin"
```

### 6. Setup DNS Resolution

This is the most important step. It configures macOS to resolve `.test` domains locally.

```bash
sudo tube setup
```

This creates `/etc/resolver/test` with:
```
nameserver 127.0.0.1
```

**What this does:**
- Creates a resolver configuration for the `.test` TLD
- Points all `.test` domain lookups to your local dnsmasq
- Flushes the DNS cache

**Verify DNS setup:**
```bash
tube dns-status
```

You should see:
```
DNS Resolver Status
────────────────────
Resolver file: /etc/resolver/test
Status: ✓ Configured
Content:
  nameserver 127.0.0.1
```

### 7. Initialize Configuration

```bash
tube init
```

This creates `~/.tube/config.yaml` with default settings.

## Uninstallation

### Remove DNS Configuration

```bash
sudo tube uninstall
```

This removes `/etc/resolver/test` and flushes the DNS cache.

### Remove Binaries

```bash
sudo rm /usr/local/bin/tube /usr/local/bin/tube-gui
```

### Remove Configuration

```bash
rm -rf ~/.tube
```

### Remove Dependencies (Optional)

```bash
brew uninstall dnsmasq nginx
```

## Troubleshooting Installation

### "command not found: tube"

The binary isn't in your PATH. Either:
1. Copy to `/usr/local/bin/` as shown above
2. Add the bin directory to your PATH

### "nginx: command not found"

nginx isn't installed or not in PATH:
```bash
brew install nginx
```

### "dnsmasq: command not found"

dnsmasq isn't installed or not in PATH:
```bash
brew install dnsmasq
```

### DNS Setup Fails

If `tube setup` fails:
1. Ensure you're running with `sudo`
2. Check if `/etc/resolver/` directory exists
3. Check disk permissions

Manual DNS setup:
```bash
sudo mkdir -p /etc/resolver
echo "nameserver 127.0.0.1" | sudo tee /etc/resolver/test
sudo dscacheutil -flushcache
sudo killall -HUP mDNSResponder
```

### Build Fails

If the build fails:
1. Ensure Go 1.23+ is installed: `go version`
2. Try cleaning first: `make clean`
3. Update dependencies: `go mod download`

### Permission Denied Errors

If you get permission errors when starting services:
1. nginx needs to bind to port 80 (requires root or capability)
2. dnsmasq needs to bind to port 53 (requires root or capability)

You may need to run tube with elevated privileges or configure your system to allow binding to privileged ports.

## Next Steps

After installation:
1. [Add your first project](cli-reference.md#project-management)
2. [Start services](cli-reference.md#service-control)
3. [Configure settings](configuration.md)
4. [Use the GUI](gui.md)
