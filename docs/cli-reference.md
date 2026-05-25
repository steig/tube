# CLI Reference

Complete reference for all tube commands.

## Overview

```
tube - Local development proxy with .test domains

Usage:
  tube [command]

Available Commands:
  add         Add a new project
  remove      Remove a project
  list        List all projects
  start       Start all services
  stop        Stop all services
  restart     Restart all services
  status      Show service status
  setup       Setup macOS DNS resolver
  uninstall   Remove DNS configuration
  dns-status  Check DNS resolver status
  init        Initialize tube configuration
  ssl         Manage SSL certificates
  doctor      Diagnose common issues
  help        Help about any command
  version     Print version information

Flags:
  -h, --help      help for tube
  -v, --version   version for tube

Use "tube [command] --help" for more information about a command.
```

---

## Project Management

### tube add

Add a new project to tube.

```bash
tube add <name> <port>
```

**Arguments:**
| Argument | Description |
|----------|-------------|
| `name` | Project name (used as subdomain) |
| `port` | Local port your app runs on |

**Examples:**
```bash
# Add a frontend running on port 3000
tube add frontend 3000

# Add an API on port 8080
tube add api 8080

# Add a documentation site
tube add docs 3001
```

**Output:**
```
✓ Added project 'frontend' on port 3000
  URL: http://frontend.test
```

**Validation:**
- Name must be 1-63 characters
- Name can only contain lowercase letters, numbers, and hyphens
- Name cannot start or end with a hyphen
- Port must be 1-65535
- Port cannot conflict with reserved ports (80, 443, 53, 3249)

**Notes:**
- Projects are immediately added to the configuration
- Run `tube restart` to apply changes if services are running
- The nginx configuration is automatically regenerated

---

### tube remove

Remove a project from tube.

```bash
tube remove <name>
```

**Arguments:**
| Argument | Description |
|----------|-------------|
| `name` | Project name to remove |

**Examples:**
```bash
# Remove the frontend project
tube remove frontend
```

**Output:**
```
✓ Removed project 'frontend'
```

**Notes:**
- Run `tube restart` to apply changes if services are running

---

### tube list

List all configured projects with their status.

```bash
tube list
```

**Output:**
```
NAME                 PORT     URL                                           STATUS
─────────────────────────────────────────────────────────────────────────────────
api                  8080     https://api.test                              ●
frontend             3000     https://frontend.test                         ○
dashboard            4200     https://dashboard.test                        ●

● = running  ○ = stopped
```

**Status Indicators:**
| Symbol | Meaning |
|--------|---------|
| `●` | Port is responding (service is running) |
| `○` | Port is not responding (service is stopped) |

**Notes:**
- Status is determined by attempting to connect to the port
- A project can be configured but its underlying service may not be running

---

## Service Control

### tube start

Start nginx and dnsmasq services.

```bash
tube start
```

**Output:**
```
✓ Started nginx
✓ Started dnsmasq
Services are running!
```

**What this does:**
1. Generates nginx configuration from your projects
2. Generates dnsmasq configuration
3. Starts nginx on ports 80 (HTTP) and 443 (HTTPS)
4. Starts dnsmasq on port 53 (DNS)

**Notes:**
- May require elevated privileges for ports < 1024
- Creates PID files in `~/.tube/pids/`
- Logs are written to `~/.tube/logs/`

---

### tube stop

Stop all running services.

```bash
tube stop
```

**Output:**
```
✓ Stopped nginx
✓ Stopped dnsmasq
Services stopped.
```

**What this does:**
1. Sends SIGTERM to nginx and dnsmasq
2. Waits for graceful shutdown
3. Cleans up PID files

---

### tube restart

Restart all services.

```bash
tube restart
```

**Output:**
```
✓ Stopped nginx
✓ Stopped dnsmasq
✓ Started nginx
✓ Started dnsmasq
Services restarted!
```

**When to use:**
- After adding or removing projects
- After changing configuration
- When services are in an unknown state

---

### tube status

Show the current status of all services.

```bash
tube status
```

**Output (running):**
```
nginx:   running (pid 12345)
dnsmasq: running (pid 12346)
```

**Output (stopped):**
```
nginx:   stopped
dnsmasq: stopped
```

**Output (partial):**
```
nginx:   running (pid 12345)
dnsmasq: stopped
```

---

## System Setup

### tube setup

Configure macOS DNS resolver for `.test` domains.

```bash
sudo tube setup
```

**Output:**
```
✓ Created /etc/resolver/test
✓ DNS cache flushed
DNS resolver configured for .test domains
```

**What this does:**
1. Creates `/etc/resolver/test` with `nameserver 127.0.0.1`
2. Flushes the macOS DNS cache
3. Restarts mDNSResponder

**Notes:**
- Requires sudo (root privileges)
- Only needs to be run once
- Persists across reboots

---

### tube uninstall

Remove the DNS resolver configuration.

```bash
sudo tube uninstall
```

**Output:**
```
✓ Removed /etc/resolver/test
✓ DNS cache flushed
DNS configuration removed
```

**What this does:**
1. Removes `/etc/resolver/test`
2. Flushes the DNS cache
3. `.test` domains will no longer resolve locally

---

### tube dns-status

Check the status of the DNS resolver configuration.

```bash
tube dns-status
```

**Output (configured):**
```
DNS Resolver Status
────────────────────
Resolver file: /etc/resolver/test
Status: ✓ Configured
Content:
  nameserver 127.0.0.1
```

**Output (not configured):**
```
DNS Resolver Status
────────────────────
Resolver file: /etc/resolver/test
Status: ✗ Not configured
Run 'sudo tube setup' to configure DNS
```

---

### tube init

Initialize tube configuration.

```bash
tube init
```

**Output:**
```
✓ Created ~/.tube/config.yaml
✓ Created directories
tube initialized!
```

**What this does:**
1. Creates `~/.tube/config.yaml` with default settings
2. Creates required directories:
   - `~/.tube/*.conf` - Generated service configurations
   - `~/.tube/logs/` - Service logs
   - `~/.tube/pids/` - PID files

**Notes:**
- Safe to run multiple times (won't overwrite existing config)
- Automatically run if config doesn't exist

---

### tube doctor

Diagnose common issues with tube setup.

```bash
tube doctor
```

**Output:**
```
tube Doctor
────────────────────

Checking dependencies...
  ✓ nginx found at /opt/homebrew/bin/nginx
  ✓ dnsmasq found at /opt/homebrew/bin/dnsmasq

Checking DNS resolver...
  ✓ /etc/resolver/test exists
  ✓ Resolves to 127.0.0.1

Checking services...
  ✓ nginx is running (pid 12345)
  ✓ dnsmasq is running (pid 12346)

Checking configuration...
  ✓ Config file exists at ~/.tube/config.yaml
  ✓ 3 projects configured

All checks passed!
```

**Checks performed:**
1. nginx and dnsmasq binaries exist and are executable
2. DNS resolver file exists and has correct content
3. Services are running
4. Configuration file is valid
5. Required directories exist

---

## SSL Management

### tube ssl

Parent command for SSL certificate management.

```bash
tube ssl [subcommand]
```

**Subcommands:**
| Command | Description |
|---------|-------------|
| `status` | Show SSL configuration and certificate status |
| `install` | Install mkcert CA to system trust store |
| `generate` | Generate/regenerate wildcard certificates |
| `enable` | Enable HTTPS support |
| `disable` | Disable HTTPS (HTTP only) |

---

### tube ssl status

Show the current SSL configuration and certificate status.

```bash
tube ssl status
```

**Output:**
```
SSL Status:
  Enabled:        yes
  CA Installed:   yes
  Cert Exists:    yes
  Local Domain:   test

Paths:
  mkcert:         /opt/homebrew/bin/mkcert
  Certificate:    /Users/you/.tube/ssl/wildcard.test.pem
  Private Key:    /Users/you/.tube/ssl/wildcard.test-key.pem
```

---

### tube ssl install

Install the mkcert Certificate Authority to your system's trust store.

```bash
tube ssl install
```

**Output:**
```
Installing mkcert CA certificate...
This may require your password for sudo access.

CA certificate installed successfully!
You can now generate certificates with: tube ssl generate
```

**Notes:**
- Requires administrator privileges (sudo)
- Only needs to be run once per machine
- The CA is trusted by browsers and curl

---

### tube ssl generate

Generate a wildcard SSL certificate for your local domain.

```bash
tube ssl generate [--force]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `-f, --force` | Force regeneration even if certificate exists |

**Output:**
```
Generating wildcard certificate for *.test...

Certificate generated successfully!
  Certificate: /Users/you/.tube/ssl/wildcard.test.pem
  Private Key: /Users/you/.tube/ssl/wildcard.test-key.pem

Restart services to apply: tube restart
```

**Notes:**
- Automatically installs CA if not already installed
- Creates a wildcard certificate covering all `*.test` domains
- Certificates are stored in `~/.tube/ssl/`

---

### tube ssl enable

Enable HTTPS support.

```bash
tube ssl enable
```

**Output:**
```
SSL enabled successfully!
Restart services to apply: tube restart
```

**Notes:**
- Automatically generates certificates if they don't exist
- Updates configuration to enable SSL
- Requires service restart to take effect

---

### tube ssl disable

Disable HTTPS support (HTTP only mode).

```bash
tube ssl disable
```

**Output:**
```
SSL disabled
Restart services to apply: tube restart
```

**Notes:**
- Services will only listen on HTTP (port 80)
- Certificates are preserved for later re-enabling
- Requires service restart to take effect

---

## Other Commands

### tube version

Print version information.

```bash
tube version
```

**Output:**
```
tube version 0.1.0
```

---

### tube help

Get help for any command.

```bash
tube help [command]
```

**Examples:**
```bash
# General help
tube help

# Help for a specific command
tube help add
tube help start
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Configuration error |
| 4 | Service error |

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TUBE_CONFIG` | Path to config file | `~/.tube/config.yaml` |
| `TUBE_<KEY>` | Override any config key (uppercased + underscore-separated) | (per-key default) |

**Examples:**
```bash
# Use a custom config file
TUBE_CONFIG=/path/to/config.yaml tube list

# Override a config key for one invocation
TUBE_PROXY_DASHBOARD_PORT=3300 tube status
```

---

## Common Workflows

### Initial Setup

```bash
# 1. Build and install
make build-all
sudo cp bin/tube /usr/local/bin/

# 2. Initialize config + SSL (one-time)
tube init

# 3. Setup DNS (one-time)
sudo tube setup

# 4. Add projects
tube add frontend 3000
tube add api 8080

# 5. Start services
tube start

# 6. Open in browser (HTTPS!)
open https://frontend.test
```

### Adding a New Project

```bash
# 1. Add the project
tube add newapp 4000

# 2. Restart services to apply
tube restart

# 3. Start your app on port 4000
npm run dev  # or whatever starts your app

# 4. Access at https://newapp.test (or http://newapp.test)
```

### Troubleshooting

```bash
# Check overall health
tube doctor

# Check DNS configuration
tube dns-status

# Check service status
tube status

# View current projects
tube list

# Restart everything
tube restart
```
