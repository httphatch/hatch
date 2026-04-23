---
title: "Development Setup"
description: "Set up your development environment to contribute to Hatch."
category: "contributing"
order: 1
lastUpdated: 2026-04-19
---

# Development Setup

## Prerequisites

- **Go 1.25+**
- **Node.js** (for the frontend dashboard)
- **Wails v3** (for building the desktop app)

## Clone and build

```bash
git clone https://github.com/httphatch/hatch.git
cd hatch
make build
```

## Build targets

| Command | Description |
|---------|-------------|
| `make build` | Build binary to `./hatch` |
| `make build-test` | Build binary to `./testing/hatch` |
| `make app` | Build with embedded frontend (Wails) |
| `make frontend` | Build React frontend only |
| `make test` | Run `go test ./...` |
| `make docs` | Start the docs site locally (Astro) |
| `make dev-init` | Initialize a dev Hatch instance at `~/.hatch-dev` |
| `make dev` | Build and start the dev daemon |
| `make dev-down` | Stop the dev daemon |
| `make dev-status` | Show dev daemon status |
| `make clean` | Remove built binary |

## Running alongside production

If you have Hatch installed via Homebrew, you can run a dev build alongside it using isolated config and ports.

### First-time setup

```bash
make dev-init
```

This creates `~/.hatch-dev/` with its own CA, certs, and config. Edit the config to use non-conflicting ports:

```bash
$EDITOR ~/.hatch-dev/config.yml
```

Set these values to avoid conflicts with the production daemon:

```yaml
version: 1
settings:
  tld: internal
  http_port: 8080
  https_port: 8443
  api_port: 42825
  dns_port: 5054
  caddy_admin_port: 2020
  auto_start: false
  tray_icon: false
  log_level: debug
```

### Start the dev daemon

```bash
make dev
```

This builds the binary and starts it with `HATCH_HOME=~/.hatch-dev`. The dev daemon uses its own launchd job, PID file, and ports. Your production daemon at `~/.hatch/` is unaffected.

### Using the dev instance

All CLI commands target the dev instance when `HATCH_HOME` is set:

```bash
HATCH_HOME=~/.hatch-dev ./hatch status
HATCH_HOME=~/.hatch-dev ./hatch link        # in a project dir
HATCH_HOME=~/.hatch-dev ./hatch session list
```

Or use the Makefile shortcut:

```bash
make dev-status
```

### Stop the dev daemon

```bash
make dev-down
```

### How it works

- `HATCH_HOME` env var redirects all config, certs, logs, and state files to `~/.hatch-dev/`
- `api_port`, `dns_port`, and `caddy_admin_port` settings override the hardcoded defaults
- The launchd label is derived from `HATCH_HOME` so each instance gets its own launchd job
- A different `tld` (e.g., `internal` instead of `test`) creates a separate `/etc/resolver/` file

## Running tests

```bash
go test ./...       # run all tests
go vet ./...        # static analysis
```

Update golden files for Caddy config translation tests:

```bash
UPDATE_GOLDEN=1 go test ./internal/caddy/...
```

## Project structure

```
hatch/
├── cmd/                    # CLI commands (Cobra)
├── internal/
│   ├── api/               # HTTP API server
│   ├── app/               # Wails app service
│   ├── caddy/             # Caddy config translation
│   ├── certs/             # CA generation & trust
│   ├── config/            # Config loading, validation, watching
│   ├── daemon/            # Daemon lifecycle, launchd, PID
│   ├── dns/               # DNS server, resolver files
│   ├── health/            # Service health checking
│   ├── logging/           # Structured logging
│   ├── mcp/               # MCP server for AI agents
│   ├── process/           # Process supervision
│   ├── proxy/             # Proxy utilities
│   ├── session/           # Session management
│   ├── tray/              # macOS tray integration
│   └── tunnel/            # Cloudflare tunnel integration
├── frontend/              # React dashboard (TypeScript, Vite, Tailwind)
├── main.go                # Entry point
├── Makefile               # Build targets
└── go.mod                 # Go module
```

## Code conventions

- **Error wrapping:** `fmt.Errorf("doing thing: %w", err)`
- **File permissions:** certs `0o644`, keys `0o600`, directories `0o755`
- **Tests:** table-driven where appropriate, `t.Fatalf` for setup failures, `t.Errorf` for assertions
- **Style:** no unnecessary comments, docstrings, or type annotations on unchanged code

## Pull request workflow

1. Create a feature branch from `main`
2. Make your changes
3. Run tests: `go test ./...`
4. Run static analysis: `go vet ./...`
5. Open a pull request against `main`
