# Architecture

This document explains how tube works under the hood.

## System Overview

tube creates a local development environment that maps pretty domain names (like `myapp.test`) to your local development servers (like `localhost:3000`).

```
                        ┌─────────────────┐
                        │    Browser      │
                        │ myapp.test:80   │
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
│  myapp.test   ──►  proxy_pass http://127.0.0.1:3000         │
│  api.test     ──►  proxy_pass http://127.0.0.1:8080         │
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

## Request Flow

When you open `http://myapp.test` in your browser:

### 1. DNS Resolution

```
Browser → "What IP is myapp.test?"
    │
    ▼
macOS Resolver → Checks /etc/resolver/test
    │               → "Use nameserver 127.0.0.1"
    ▼
dnsmasq (127.0.0.1:53) → "*.test = 127.0.0.1"
    │
    ▼
Browser ← "myapp.test = 127.0.0.1"
```

### 2. HTTP Request

```
Browser → HTTP GET http://myapp.test/ (connects to 127.0.0.1:80)
    │
    ▼
nginx (127.0.0.1:80) → Checks server_name
    │                   → myapp.test → proxy to 127.0.0.1:3000
    ▼
Your App (127.0.0.1:3000) → Processes request
    │
    ▼
nginx ← Response
    │
    ▼
Browser ← Response
```

## Component Details

### macOS DNS Resolver

macOS has a powerful DNS resolver system that allows per-domain configuration.

**Location:** `/etc/resolver/<domain>`

**How it works:**
1. When macOS needs to resolve a domain, it checks `/etc/resolver/`
2. If a file exists matching the TLD (e.g., `test`), it uses that configuration
3. The file specifies which DNS server to use for that domain

**tube's resolver file:** `/etc/resolver/test`
```
nameserver 127.0.0.1
```

This tells macOS: "For any `.test` domain, ask the DNS server at 127.0.0.1"

**Why not /etc/hosts?**
- `/etc/hosts` doesn't support wildcards
- You'd need an entry for every subdomain
- Changes require manual editing

### dnsmasq

dnsmasq is a lightweight DNS server that handles wildcard resolution.

**Configuration:** `~/.tube/configs/dnsmasq.conf`

```
port=53
listen-address=127.0.0.1
no-resolv
address=/.test/127.0.0.1
```

**How it works:**
1. Listens on port 53 (DNS) at 127.0.0.1
2. Responds to all `.test` queries with `127.0.0.1`
3. Doesn't forward to upstream DNS (`no-resolv`)

**Why dnsmasq?**
- Lightweight and fast
- Supports wildcard domains
- Easy to configure
- Widely available (Homebrew, Linux packages)

### nginx

nginx is a high-performance HTTP server that handles reverse proxying.

**Configuration:** `~/.tube/configs/nginx.conf`

```nginx
worker_processes 1;
error_log ~/.tube/logs/nginx-error.log;
pid ~/.tube/pids/nginx.pid;

events {
    worker_connections 1024;
}

http {
    access_log ~/.tube/logs/nginx-access.log;

    server {
        listen 80;
        server_name myapp.test;

        location / {
            proxy_pass http://127.0.0.1:3000;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }

    # Additional servers for each project...
}
```

**How it works:**
1. Listens on port 80 for HTTP requests
2. Matches `server_name` to incoming `Host` header
3. Proxies request to the configured local port
4. Adds headers so your app knows the original request details

**Proxy headers explained:**
| Header | Value | Purpose |
|--------|-------|---------|
| `Host` | Original hostname | Your app sees `myapp.test`, not `localhost` |
| `X-Real-IP` | Client IP | Your app sees the browser's IP |
| `X-Forwarded-For` | Proxy chain | Lists all proxies in the chain |
| `X-Forwarded-Proto` | `http` or `https` | Original protocol |

---

## Code Architecture

### Package Structure

```
tube/
├── cmd/
│   ├── tube/              # CLI entry point
│   │   └── main.go
│   └── tube-gui/          # GUI entry point
│       └── main.go
│
├── internal/
│   ├── cli/               # CLI commands (cobra)
│   │   ├── root.go        # Root command and setup
│   │   ├── projects.go    # add, remove, list commands
│   │   ├── services.go    # start, stop, restart, status
│   │   └── setup.go       # setup, uninstall, dns-status
│   │
│   ├── config/            # Configuration management
│   │   ├── config.go      # Load, save, validate
│   │   ├── types.go       # Config struct definitions
│   │   └── defaults.go    # Default values
│   │
│   ├── dns/               # DNS resolver management
│   │   └── resolver.go    # macOS /etc/resolver setup
│   │
│   ├── gui/               # GUI components
│   │   ├── tray.go        # Menu bar app (systray)
│   │   └── dashboard.go   # Web dashboard server
│   │
│   ├── proxy/             # Service managers
│   │   ├── nginx.go       # nginx config generation and control
│   │   └── dnsmasq.go     # dnsmasq config generation and control
│   │
│   └── service/           # Process lifecycle
│       └── manager.go     # PID management, start/stop
│
├── templates/             # Config templates (if using)
│   ├── nginx/
│   └── dnsmasq/
│
├── docs/                  # Documentation
└── scripts/               # Build and test scripts
```

### Key Interfaces

#### ProcessManager

Manages service lifecycles using PID files.

```go
type ProcessManager struct {
    pidDir string
}

func (pm *ProcessManager) Start(name string, cmd *exec.Cmd) error
func (pm *ProcessManager) Stop(name string) error
func (pm *ProcessManager) IsRunning(name string) (bool, int, error)
func (pm *ProcessManager) StartAll() error
func (pm *ProcessManager) StopAll() error
```

#### NginxManager

Generates and manages nginx configuration.

```go
type NginxManager struct {
    cfg     *config.Config
    pm      *ProcessManager
}

func (nm *NginxManager) WriteConfig() error
func (nm *NginxManager) Start() error
func (nm *NginxManager) Stop() error
func (nm *NginxManager) Reload() error
func (nm *NginxManager) IsRunning() (bool, error)
```

#### DnsmasqManager

Generates and manages dnsmasq configuration.

```go
type DnsmasqManager struct {
    cfg     *config.Config
    pm      *ProcessManager
}

func (dm *DnsmasqManager) WriteConfig() error
func (dm *DnsmasqManager) Start() error
func (dm *DnsmasqManager) Stop() error
func (dm *DnsmasqManager) IsRunning() (bool, error)
```

#### ResolverManager

Manages macOS DNS resolver files.

```go
type ResolverManager struct {
    domain    string
    dnsServer string
}

func (rm *ResolverManager) Setup() error
func (rm *ResolverManager) Remove() error
func (rm *ResolverManager) IsConfigured() bool
func FlushDNSCache() error
```

---

## Data Flow

### Configuration

```
~/.tube/config.yaml
        │
        ▼
    config.Load()
        │
        ▼
    *config.Config
        │
        ├──► NginxManager.WriteConfig()
        │           │
        │           ▼
        │    ~/.tube/configs/nginx.conf
        │
        └──► DnsmasqManager.WriteConfig()
                    │
                    ▼
             ~/.tube/configs/dnsmasq.conf
```

### Service Lifecycle

```
┌──────────────┐
│ tube start   │
└──────┬───────┘
       │
       ▼
┌──────────────────────┐
│ NginxManager         │
│ .WriteConfig()       │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ ProcessManager       │
│ .Start("nginx", cmd) │
└──────┬───────────────┘
       │
       ├──► Fork nginx process
       │
       └──► Write ~/.tube/pids/nginx.pid
```

### Request Handling (Dashboard API)

```
┌────────────────────┐
│ HTTP Request       │
│ POST /api/project  │
└──────┬─────────────┘
       │
       ▼
┌──────────────────────┐
│ Dashboard.handler()  │
│ .handleAddProject()  │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ Config.AddProject()  │
│ Config.Save()        │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ NginxManager         │
│ .WriteConfig()       │
│ .Reload()            │
└──────────────────────┘
```

---

## Security Considerations

### Privileged Ports

nginx (port 80) and dnsmasq (port 53) bind to privileged ports (< 1024).

**Options:**
1. Run tube with `sudo` (not recommended)
2. Use capability-based permissions
3. Use a reverse proxy that runs as root
4. Use higher ports and configure port forwarding

### Local-Only Binding

All services bind to `127.0.0.1`:
- nginx: `127.0.0.1:80`
- dnsmasq: `127.0.0.1:53`
- Dashboard: `127.0.0.1:3249`

This prevents external access to your development services.

### No Authentication (by default)

The dashboard doesn't require authentication by default. It's designed for local development use only.

---

## Future Architecture

### Planned: HTTPS Support

```
Browser ─────► nginx (HTTPS:443)
                    │
                    ▼
             mkcert certificates
                    │
                    ▼
              proxy_pass ─────► Your App (HTTP)
```

### Planned: Cloudflare Tunnel

```
Internet ─────► Cloudflare Edge
                    │
                    ▼
              Cloudflare Tunnel
                    │
                    ▼
              cloudflared ─────► nginx ─────► Your App
```

This will allow sharing your local development environment with external users.

---

## Testing Architecture

### Unit Tests

Test individual components in isolation:
- Config loading/saving
- Project validation
- Config generation

### Integration Tests

Test the full stack in Docker:
- nginx starts and proxies correctly
- dnsmasq resolves domains
- End-to-end request flow

```
┌─────────────────────────────────────┐
│           Docker Container          │
│                                     │
│  ┌─────────┐  ┌─────────┐          │
│  │ nginx   │  │ dnsmasq │          │
│  └─────────┘  └─────────┘          │
│         │          │                │
│         └────┬─────┘                │
│              ▼                      │
│       Test Application              │
│       (mock HTTP server)            │
│                                     │
│              ▲                      │
│              │                      │
│         Test Runner                 │
│         (Go tests)                  │
│                                     │
└─────────────────────────────────────┘
```
