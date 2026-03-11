---
title: "Architecture"
description: "Deep technical overview of how Hatch's DNS, TLS, proxy, daemon, and tunnel systems work."
category: "advanced"
order: 1
lastUpdated: 2025-03-05
---

# Architecture

Hatch is a single daemon that orchestrates four subsystems: DNS resolution, TLS certificate management, HTTPS reverse proxying, and a management API. This page explains each one.

## Architecture overview

```
Browser request: https://myapp.test
        |
        v
+-------------------+
|  macOS Resolver    |  /etc/resolver/test -> 127.0.0.1:5053
+--------+----------+
         v
+-------------------+
|   DNS Server       |  *.test -> 127.0.0.1
+--------+----------+
         v
+-------------------+
|   Caddy (HTTPS)    |  TLS termination + reverse proxy
|   port 443         |  cert signed by Hatch CA
+--------+----------+
         v
+-------------------+
|  Your Dev Server   |  http://localhost:3000
+-------------------+
```

## DNS

Hatch runs a DNS server on `127.0.0.1:5053` using the [miekg/dns](https://github.com/miekg/dns) library.

**Resolver file:** During `hatch init`, a file is created at `/etc/resolver/<tld>` containing:

```
nameserver 127.0.0.1
port 5053
```

macOS checks `/etc/resolver/` for per-domain nameserver overrides. Any DNS query for `*.test` is sent to `127.0.0.1:5053`.

**Query handling:**
- Queries matching `*.<tld>` return `127.0.0.1` (A record) or `::1` (AAAA record)
- All other queries are forwarded to the system's upstream DNS servers

## TLS certificates

Hatch maintains a two-tier CA hierarchy. See [HTTPS and Certificates](/docs/concepts/https-and-certificates) for the user-facing explanation.

### Root CA

- ECDSA P-256 key pair, generated during `hatch init`
- Self-signed, valid for 10 years
- Stored at `~/.hatch/certs/rootCA.pem` and `~/.hatch/certs/rootCA-key.pem`
- Added to macOS Keychain as a trusted certificate via `security add-trusted-cert`

### Intermediate CA

- ECDSA P-256, signed by the root CA
- Valid for 10 years
- Stored at `~/.hatch/certs/intermediateCA.pem` and `~/.hatch/certs/intermediateCA-key.pem`
- Not added to Keychain; trust chains through the root

### Site certificates

- Generated on-the-fly by Caddy's built-in PKI module
- Signed by the intermediate CA
- Cached in `~/.hatch/caddy/`
- Created automatically when a new domain is first accessed

## Reverse proxy

Hatch embeds [Caddy](https://caddyserver.com) as a library. Caddy handles TLS termination and reverse proxying.

**Config translation:** Hatch translates its YAML config into Caddy's JSON config format. Each service becomes a Caddy route:

- **Host matching** — routes are matched by domain (e.g., `myapp.test`)
- **Path matching** — services with a `route` field match a path prefix (e.g., `/api/*`)
- **Subdomain matching** — services with a `subdomain` field match `<sub>.<domain>`
- **WebSocket support** — services with `websocket: true` get `Connection`/`Upgrade` header forwarding and instant response flushing

Routes are sorted by specificity: subdomains first, then path routes, then catch-all.

**Error pages:** When an upstream returns 502 or 503, Caddy intercepts the response and serves a branded HTML page showing the failing domain and upstream URL. A catch-all fallback route handles requests that don't match any configured domain with a 404 page.

**HTTP -> HTTPS redirect:** All HTTP requests are redirected to HTTPS with a 302.

**Live reload:** When `config.yml` changes, Hatch re-translates the config and loads it into Caddy via its admin API (`localhost:2019`). No process restart needed.

## Daemon

The daemon is a single long-running process managed by macOS launchd.

### Launchd

A plist file is installed at `~/Library/LaunchAgents/dev.hatch.daemon.plist`:

| Property | Value |
|----------|-------|
| Label | `dev.hatch.daemon` |
| Program | Path to `hatch` binary |
| Arguments | `hatch _run` (hidden command) |
| RunAtLoad | `true` if `auto_start` enabled |
| KeepAlive | `true` if `auto_start` enabled |

`hatch up` installs and loads the plist. `hatch down` unloads it.

### Tray process

The system tray app runs as a separate process (`hatch _tray`). `hatch up` spawns it automatically when `tray_icon` is enabled. The tray communicates with the daemon over the local HTTP API at `127.0.0.1:42824`. A file lock at `~/.hatch/tray.lock` prevents duplicate tray instances.

### Startup sequence

1. Acquire PID lock (`~/.hatch/hatch.pid`)
2. Clean up orphaned processes from a previous daemon instance
3. Load and validate config
4. Verify ports are available
5. Start DNS server
6. Start Caddy with translated config
7. Start health checker
8. Start process manager (supervised commands)
9. Start tunnel manager (Cloudflare Tunnels)
10. Start API server (`127.0.0.1:42824`)
11. Start config file watcher
12. Block until shutdown signal

### Config watching

The daemon watches `~/.hatch/config.yml` for changes using filesystem notifications. When a change is detected:

1. Reload and validate the config
2. Re-translate to Caddy JSON
3. Push to Caddy via admin API
4. Update health checker targets
5. Reconcile tunnels (start new, stop removed)

No daemon restart is required for config changes.

A config reload also triggers automatically when a named tunnel finishes connecting. This ensures Caddy learns about the tunnel's external domains even if the Cloudflare API returned empty results at startup. Multiple tunnel starts within 3 seconds are coalesced into a single reload.

## API server

The daemon exposes a local HTTP API on `127.0.0.1:42824` for the CLI and dashboard.

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

The API is localhost-only.

### Tray API

The tray process serves additional endpoints for daemon lifecycle control:

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/tray/start` | Start the daemon via launchd |
| POST | `/api/tray/stop` | Stop the daemon via launchd |
| POST | `/api/tray/restart` | Restart the daemon via launchd |

## Tunnels

Hatch runs cloudflared as a subprocess, one per active tunnel. See [Tunnels](/docs/concepts/tunnels) for the user-facing explanation.

**Quick tunnels** use `cloudflared tunnel --url <upstream>`. Hatch scans stdout for a `trycloudflare.com` URL and waits up to 30 seconds.

**Named tunnels** use `cloudflared tunnel run <name>`. When a `cloudflare_token` is configured, Hatch resolves the tunnel JWT by calling three Cloudflare API endpoints: (1) `/accounts` to find the account ID, (2) `/accounts/{id}/cfd_tunnel?name=...` to find the tunnel UUID, (3) `/accounts/{id}/cfd_tunnel/{tunnel_id}/token` to get the JWT. The JWT is passed via the `TUNNEL_TOKEN` environment variable (never in process arguments).

**Rewrite proxy:** Quick tunnels route through a local rewrite proxy between cloudflared and the upstream. The proxy: (1) strips absolute localhost URLs from HTML responses for PNA compliance, (2) discovers secondary dev servers and retries 404'd requests against them, (3) retries failed WebSocket upgrades against the dev server for HMR support. Named tunnels skip the proxy.

**Binary discovery:** `exec.LookPath` with fallbacks to `/opt/homebrew/bin/cloudflared` and `/usr/local/bin/cloudflared` for launchd environments.

**Lifecycle:** Tunnels start in background goroutines on daemon boot or config reload. When a named tunnel finishes starting, it fires an `OnTunnelReady` callback that triggers a debounced config reload. This re-resolves the tunnel's external domains from the Cloudflare API and rebuilds the Caddy config so requests route correctly. If a tunnel process exits unexpectedly, it stays stopped until manually restarted or the config reloads.

**State persistence:** Running tunnel metadata is written to `~/.hatch/tunnels.json` for CLI status display. Tunnels are not restored from this file on startup.

## Health checking

Hatch periodically checks the health of each service's upstream target. Health status is exposed via the API and displayed in the dashboard and `hatch status` output.

## Process management

See [Process Management](/docs/concepts/process-management) for the user-facing explanation.

**PID tracking:** The process manager writes running process PIDs to `~/.hatch/processes.json`. On daemon startup, Hatch reads this file and kills any surviving processes from a previous instance (SIGTERM with 5-second timeout, then SIGKILL). This prevents port conflicts after a daemon crash or upgrade.

## File locations

| Path | Purpose |
|------|---------|
| `~/.hatch/config.yml` | Main configuration |
| `~/.hatch/certs/` | CA certificates and keys |
| `~/.hatch/logs/hatch.log` | Daemon log file |
| `~/.hatch/hatch.pid` | Daemon PID lock file |
| `~/.hatch/tray.lock` | Tray instance lock file |
| `~/.hatch/processes.json` | Managed process PID state file |
| `~/.hatch/tunnels.json` | Active tunnel metadata |
| `~/.hatch/caddy/` | Caddy data (cached site certs) |
| `/etc/resolver/<tld>` | macOS DNS resolver override |
| `~/Library/LaunchAgents/dev.hatch.daemon.plist` | Launchd service |
