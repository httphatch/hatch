# Tunnels

Hatch can expose your local services to the internet using [Cloudflare Tunnels](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/). This is useful for sharing work with teammates, testing webhooks, and mobile device testing.

## Prerequisites

Install cloudflared:

```bash
brew install cloudflared
```

Hatch runs cloudflared as a subprocess. It checks your PATH and standard Homebrew locations (`/opt/homebrew/bin/cloudflared`, `/usr/local/bin/cloudflared`), so it works even when the daemon runs under launchd with a minimal PATH. If cloudflared is not found, tunnel commands will return a clear error.

## Quick Tunnels

Quick tunnels require no Cloudflare account. They give you a temporary public URL on `trycloudflare.com` that routes to your local service.

Add `tunnel: true` to any service with a `proxy`:

```yaml
# hatch.yml
domain: myapp.test
services:
  web:
    proxy: http://localhost:3000
    tunnel: true
```

Or start one on demand:

```bash
hatch tunnel start myapp/web
```

The URL is printed when the tunnel starts:

```
Started tunnel myapp/web
URL: https://random-words.trycloudflare.com
```

Quick tunnel URLs are temporary. They change each time the tunnel is started.

## Named Tunnels

Named tunnels give you a persistent domain through your Cloudflare account. You need a Cloudflare API token and a pre-configured tunnel in the Cloudflare dashboard.

Set the token globally or per-project:

```yaml
# ~/.hatch/config.yml
version: 1
settings:
  cloudflare_token: "your-global-token"
projects:
  myapp:
    domain: myapp.test
    path: /Users/me/projects/myapp
    cloudflare_token: "project-specific-token"  # overrides global
    services:
      web:
        proxy: http://localhost:3000
        tunnel: my-tunnel-name
```

The tunnel name must match a tunnel configured in your Cloudflare dashboard. Tokens are never exposed in process arguments or API responses.

## CLI Commands

### `hatch tunnel start <project>/<service>`

Start a tunnel for a specific service.

```bash
hatch tunnel start myapp/web
```

### `hatch tunnel stop <project>/<service>`

Stop a running tunnel.

```bash
hatch tunnel stop myapp/web
```

### `hatch tunnel status`

List all tunnels with their type, URL, and uptime.

```bash
hatch tunnel status
```

Output:

```
PROJECT/SERVICE  TYPE   URL                                      UPTIME
myapp/web        quick  https://random-words.trycloudflare.com   5m ago
myapp/api        named  (configured in Cloudflare)                2h ago
```

## Config-Driven Tunnels

When `tunnel` is set in the config, Hatch starts the tunnel automatically when the daemon starts or the config reloads. Removing or clearing the `tunnel` field stops the tunnel.

Tunnels start in the background and do not block the daemon from booting. Quick tunnels can take up to 30 seconds to obtain a URL from cloudflared. During this time, the dashboard shows a "Starting tunnel..." indicator.

This means you can manage tunnels entirely through config changes, without running CLI commands.

## Dashboard

The dashboard shows tunnel controls next to each service:

- **Start button** (cloud icon) — starts a quick tunnel for any service with a proxy
- **Starting state** — shows a pulsing "Starting tunnel..." badge while cloudflared initializes
- **Running state** — displays the public URL as a clickable link, with a stop button
- **Error state** — shows a brief error message if the tunnel action fails (auto-clears after 5 seconds)

Tunnel output (cloudflared stdout/stderr) appears in the service's terminal viewer alongside process output. Click the terminal icon on any service row to view it.

## How It Works

For quick tunnels, Hatch runs a local rewrite proxy between cloudflared and your dev server. The proxy solves two problems that occur when accessing local dev servers through a public tunnel URL.

### URL rewriting

Dev servers like Vite inject absolute localhost URLs into HTML responses (e.g. `http://[::1]:5173/@vite/client`). Browsers block these under Private Network Access (PNA) policy when the page loads from a public origin. The rewrite proxy strips all `http://`, `https://`, `ws://`, and `wss://` URLs pointing to `localhost`, `127.0.0.1`, or `[::1]` on any port from HTML responses. This turns them into relative paths that resolve through the tunnel. Non-HTML responses pass through unchanged.

### Dev server fallback

In setups like Laravel + Vite, the app server and dev server run on different ports. Laravel serves HTML, Vite serves JS/CSS assets. When the proxy rewrites HTML, it discovers the dev server by finding localhost URLs on a port different from the upstream. For subsequent requests that 404 on the upstream (because Laravel doesn't serve Vite assets), the proxy retries against the discovered dev server.

The same fallback applies to WebSocket upgrades. When the upstream doesn't return 101 Switching Protocols (because it doesn't handle WebSocket), the proxy retries against the dev server. This allows Vite's HMR WebSocket to connect through the tunnel.

### The flow

Single-server setup (e.g. plain Vite):

```
Browser → trycloudflare.com → cloudflared → rewrite proxy → localhost:3000
```

Multi-server setup (e.g. Laravel + Vite):

```
Browser → trycloudflare.com → cloudflared → rewrite proxy → Laravel (port 8000)
                                                          ↘ Vite (port 5173) on 404/WS
```

### Tunnel processes

Hatch runs a cloudflared subprocess for each active tunnel:

- **Quick tunnels** run `cloudflared tunnel --url http://127.0.0.1:<proxy-port>` and parse the URL from stdout. Quick tunnels wait up to 30 seconds for cloudflared to provide a URL.
- **Named tunnels** run `cloudflared tunnel run <name>` with the token passed via environment variable. Named tunnels connect directly to the upstream without a rewrite proxy.

The tunnel manager tracks state in `~/.hatch/tunnels.json` for CLI status display. This file is informational only; tunnels are not restored on daemon restart.

If a tunnel process exits unexpectedly, it stays stopped until manually restarted or the config is reloaded. There is no automatic retry.

## Vite and HMR

Vite's hot module replacement works through tunnels without any Vite configuration. The rewrite proxy handles everything:

1. Absolute localhost script URLs in HTML are rewritten to relative paths
2. Asset requests (JS, CSS) that 404 on the app server are retried against the Vite dev server
3. WebSocket upgrade requests for HMR are forwarded to the Vite dev server

You may see a console warning about a failed WebSocket to `wss://localhost:PORT`. This is Vite's client attempting a direct localhost connection as a fallback. It is harmless because the primary WebSocket through the tunnel URL succeeds.

For quick tunnels, the URL changes each time. HMR live-reloading may require a manual page refresh after restarting the tunnel.
