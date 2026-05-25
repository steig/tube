---
hide:
  - navigation
---

# tube

**Local development proxy with `.test` domains and trusted HTTPS for macOS.**

Stop memorizing port numbers. Access your local services at `https://myapp.test` instead of `http://localhost:3000`.

<div class="grid cards" markdown>

-   :material-rocket-launch: **[Install](installation.md)**

    ---

    Build from source, set up DNS, and get your first project running in five minutes.

-   :material-console: **[CLI reference](cli-reference.md)**

    ---

    Every command: `add`, `start`, `doctor`, `ssl`, `config`, plus flags and examples.

-   :material-cog: **[Configuration](configuration.md)**

    ---

    The `~/.tube/config.yaml` schema, defaults, and environment variables.

-   :material-monitor-dashboard: **[GUI](gui.md)**

    ---

    The macOS menu bar app and the web dashboard at `localhost:3249`.

-   :material-sitemap: **[Architecture](architecture.md)**

    ---

    How nginx, dnsmasq, and mkcert combine into the proxy pipeline.

-   :material-tools: **[Troubleshooting](troubleshooting.md)**

    ---

    DNS not resolving, port 80 in use, certificate not trusted — the usual suspects.

</div>

## What you get

| Feature | Description |
|---------|-------------|
| Pretty URLs | `myapp.test` resolves locally to your running app |
| Trusted HTTPS | mkcert-issued wildcard certs your browser already trusts |
| CLI + GUI | A cobra-based CLI and a macOS menu bar app, both on the same state |
| Web dashboard | Add/remove projects and watch service health at `localhost:3249` |
| Zero-config DNS | `/etc/resolver/test` is generated for you by `tube setup` |
| Live reload | Add or remove projects without restarting services |

## How it fits together

```
Browser -> macOS resolver (/etc/resolver/test) -> dnsmasq (127.0.0.1) -> nginx (SSL termination) -> your app
```

See [Architecture](architecture.md) for the long version.

## Where to go next

If this is your first time, head to [Installation](installation.md). If you already have it running and want to know what a specific command does, [CLI reference](cli-reference.md) is the right page.
