# How It Works

Hatch is a single daemon that orchestrates four subsystems: DNS resolution, TLS certificate management, HTTPS reverse proxying, and a management API. This page explains each one.

## Architecture Overview

```
Browser request: https://myapp.test
        │
        ▼
┌─────────────────┐
│  macOS Resolver  │  /etc/resolver/test → 127.0.0.1:5053
└────────┬────────┘
         ▼
┌─────────────────┐
│   DNS Server     │  *.test → 127.0.0.1
└────────┬────────┘
         ▼
┌─────────────────┐
│   Caddy (HTTPS)  │  TLS termination + reverse proxy
│   port 443       │  cert signed by Hatch CA
└────────┬────────┘
         ▼
┌─────────────────┐
│  Your Dev Server │  http://localhost:3000
└─────────────────┘
```

## DNS

Hatch runs a DNS server on `127.0.0.1:5053` using the [miekg/dns](https://github.com/miekg/dns) library.

**Resolver file:** During `hatch init`, a file is created at `/etc/resolver/<tld>` (e.g., `/etc/resolver/test`) containing:

```
nameserver 127.0.0.1
port 5053
```

macOS checks `/etc/resolver/` for per-domain nameserver overrides. Any DNS query for `*.test` is sent to `127.0.0.1:5053`, which is Hatch's DNS server.

**Query handling:**
- Queries matching `*.<tld>` return `127.0.0.1` (A record) or `::1` (AAAA record)
- All other queries are forwarded to the system's upstream DNS servers

## TLS Certificates

Hatch maintains a two-tier CA hierarchy:

```
Root CA (self-signed, trusted in Keychain)
  └── Intermediate CA (signed by Root)
        └── Site certificates (signed by Intermediate, issued by Caddy)
```

### Root CA

- Generated during `hatch init` as an ECDSA P-256 key pair
- Self-signed, valid for 10 years
- Stored at `~/.hatch/certs/rootCA.pem` and `~/.hatch/certs/rootCA-key.pem`
- Added to the macOS Keychain as a trusted certificate via `security add-trusted-cert`

### Intermediate CA

- Also ECDSA P-256, signed by the root CA
- Valid for 10 years
- Stored at `~/.hatch/certs/intermediateCA.pem` and `~/.hatch/certs/intermediateCA-key.pem`
- **Not** added to Keychain - trust chains through the root

### Site Certificates

- Generated on-the-fly by Caddy's built-in PKI module
- Signed by the intermediate CA
- Cached in `~/.hatch/caddy/`
- Created automatically when a new domain is first accessed

Because the root CA is trusted in your Keychain, the full chain (root → intermediate → site cert) is valid and browsers show a green lock.

## Reverse Proxy

Hatch embeds [Caddy](https://caddyserver.com) as a library. Caddy handles TLS termination and reverse proxying.

**Config translation:** Hatch translates its YAML config into Caddy's JSON config format. Each service becomes a Caddy route:

- **Host matching** - routes are matched by domain (e.g., `myapp.test`)
- **Path matching** - services with a `route` field match a path prefix (e.g., `/api/*`)
- **Subdomain matching** - services with a `subdomain` field match `<sub>.<domain>`
- **WebSocket support** - services with `websocket: true` get `Connection`/`Upgrade` header forwarding and instant response flushing

Routes are sorted by specificity: subdomains first, then path routes, then catch-all.

**Error pages:** When an upstream returns 502 or 503, Caddy intercepts the response and serves a branded HTML page showing the failing domain and upstream URL. A catch-all fallback route with no host matcher is appended as the last HTTPS route. Any request that does not match a configured domain receives a 404 HTML page listing all configured domains and their upstreams.

**HTTP → HTTPS redirect:** All HTTP requests are redirected to HTTPS with a 302.

**Live reload:** When `config.yml` changes, Hatch re-translates the config and loads it into Caddy via its admin API (`localhost:2019`). No process restart needed.

## Daemon

The daemon is a single long-running process managed by macOS launchd.

### Launchd

A plist file is installed at `~/Library/LaunchAgents/dev.hatch.daemon.plist`. Key properties:

| Property | Value |
|----------|-------|
| Label | `dev.hatch.daemon` |
| Program | Path to `hatch` binary |
| Arguments | `hatch _run` (hidden command) |
| RunAtLoad | `true` if `auto_start` enabled |
| KeepAlive | `true` if `auto_start` enabled |

`hatch up` installs and loads the plist. `hatch down` unloads it.

### Tray Process

The system tray app runs as a separate process (`hatch _tray`). `hatch up` spawns it automatically when `tray_icon` is enabled. `hatch app` also spawns it on demand. The tray communicates with the daemon over the local HTTP API at `127.0.0.1:42824`. It does not share in-process state. Quitting the tray leaves the daemon running. A file lock at `~/.hatch/tray.lock` prevents duplicate tray instances.

### Startup Sequence

1. Acquire PID lock (`~/.hatch/hatch.pid`)
2. Load and validate config
3. Verify ports are available
4. Start DNS server
5. Start Caddy with translated config
6. Start health checker
7. Start process manager (supervised commands)
8. Start tunnel manager (Cloudflare Tunnels)
9. Start API server (`127.0.0.1:42824`)
10. Start config file watcher
11. Block until shutdown signal

### Config Watching

The daemon watches `~/.hatch/config.yml` for changes using filesystem notifications. When a change is detected:

1. Reload and validate the config
2. Re-translate to Caddy JSON
3. Push to Caddy via admin API
4. Update health checker targets
5. Reconcile tunnels (start new, stop removed)

No daemon restart is required for config changes.

## API Server

The daemon exposes a local HTTP API on `127.0.0.1:42824` for the CLI and dashboard to communicate with the running daemon.

Key endpoints:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/status` | Daemon info (PID, uptime, version) |
| GET | `/api/projects` | List all projects |
| POST | `/api/projects` | Add a project |
| PUT | `/api/projects/{name}` | Update a project |
| DELETE | `/api/projects/{name}` | Remove a project |
| PATCH | `/api/projects/{name}/toggle` | Toggle enabled state |
| GET | `/api/health` | Service health statuses |
| GET | `/api/logs` | Live log stream (SSE) |
| GET | `/api/settings` | Read global settings (JSON) |
| PUT | `/api/settings` | Write global settings (JSON) |
| GET | `/api/config` | Read config (YAML) |
| PUT | `/api/config` | Write config (YAML) |
| GET | `/api/certs` | Certificate status |
| GET | `/api/processes` | Process statuses |
| GET | `/api/tunnels` | Tunnel statuses |
| POST | `/api/tunnels/{project}/{service}/start` | Start a tunnel |
| POST | `/api/tunnels/{project}/{service}/stop` | Stop a tunnel |
| POST | `/api/restart` | Reload config |

The API is localhost-only and used by both the CLI commands and the React dashboard.

## Tunnels

Hatch can expose local services to the internet via [Cloudflare Tunnels](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/). It runs cloudflared as a subprocess, one per active tunnel.

**Quick tunnels** use `cloudflared tunnel --url <upstream>`. Hatch scans stdout for a `trycloudflare.com` URL and waits up to 30 seconds for it to appear. No Cloudflare account is needed.

**Named tunnels** use `cloudflared tunnel run <name>`. When a `cloudflare_token` (API token) is configured, Hatch resolves the tunnel JWT by calling three Cloudflare API endpoints: (1) `/accounts` to find the account ID (skipped if `cloudflare_account_id` is set), (2) `/accounts/{id}/cfd_tunnel?name=...` to find the tunnel UUID, (3) `/accounts/{id}/cfd_tunnel/{tunnel_id}/token` to get the JWT. The resolved JWT is passed via the `TUNNEL_TOKEN` environment variable (never exposed in process arguments). Without an API token, cloudflared falls back to credential files in `~/.cloudflared/`.

**Rewrite proxy:** Quick tunnels route through a local rewrite proxy that sits between cloudflared and the upstream. The proxy does three things: (1) strips absolute localhost URLs from HTML responses so browsers don't block them under PNA policy, (2) discovers a secondary dev server (e.g. Vite) by finding localhost URLs on a different port than the upstream, and retries 404'd GET requests against it, (3) retries failed WebSocket upgrades against the dev server so HMR works through the tunnel. Named tunnels skip the proxy and connect directly to the upstream.

**Binary discovery:** Hatch uses `exec.LookPath` to find cloudflared. On macOS, it also checks `/opt/homebrew/bin/cloudflared` and `/usr/local/bin/cloudflared` as fallbacks for launchd environments where Homebrew's bin directory is not in PATH.

**Lifecycle:** The tunnel manager starts tunnels in background goroutines on daemon boot or config reload. Tunnel startup does not block the daemon. Tunnels are stopped on shutdown or when removed from config. If a tunnel process exits unexpectedly, it stays stopped. There is no automatic retry.

**State persistence:** Running tunnel metadata (URLs, types, timestamps) is written to `~/.hatch/tunnels.json` for CLI status display. This is informational only; the manager does not restore tunnels from this file on startup.

## Health Checking

Hatch periodically checks the health of each service's upstream target. Health status is exposed via the API and displayed in the dashboard and `hatch status` output.

## Process Management

Services with a `command` field are managed as supervised processes.

**Supervisor model:** Hatch starts each command when the daemon starts (or when the config changes) and monitors it. If a process exits, it is restarted with exponential backoff.

**Exponential backoff:** Restarts begin at 1 second and double up to a maximum of 30 seconds. The backoff resets after the process has been running stably for 60 seconds.

**Command execution:** Commands are run via `sh -c` in the project's `path` directory. Each command runs in its own process group so that child processes are cleaned up on stop.

**Graceful shutdown:** On stop, Hatch sends `SIGTERM` to the process group, waits up to 5 seconds, then sends `SIGKILL` if the process is still running.

**Env file loading:** When `env_file` is set, Hatch loads the specified file (relative to the project path) and injects the variables into the command's environment.

## File Locations

| Path | Purpose |
|------|---------|
| `~/.hatch/config.yml` | Main configuration |
| `~/.hatch/certs/` | CA certificates and keys |
| `~/.hatch/logs/hatch.log` | Daemon log file |
| `~/.hatch/hatch.pid` | Daemon PID lock file |
| `~/.hatch/tray.lock` | Tray instance lock file |
| `~/.hatch/tunnels.json` | Active tunnel metadata |
| `~/.hatch/caddy/` | Caddy data (cached site certs) |
| `/etc/resolver/<tld>` | macOS DNS resolver override |
| `~/Library/LaunchAgents/dev.hatch.daemon.plist` | Launchd service |
