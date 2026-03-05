---
title: "Share a Dev Server"
description: "Expose your local service to the internet using Cloudflare Tunnels."
category: "guides"
order: 4
lastUpdated: 2025-03-05
---

# Share a Dev Server

This guide walks through exposing a local service via a Cloudflare Tunnel so teammates, webhooks, or mobile devices can reach it.

## Prerequisites

Install cloudflared:

```bash
brew install cloudflared
```

Hatch checks your PATH and standard Homebrew locations (`/opt/homebrew/bin/cloudflared`, `/usr/local/bin/cloudflared`), so it works even when the daemon runs under launchd with a minimal PATH.

## Quick tunnel (no account needed)

Add `tunnel: true` to any service with a `proxy`:

```yaml
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

Quick tunnel URLs are temporary. They change each time the tunnel restarts.

## Named tunnel (persistent domain)

Named tunnels give you a stable domain through your Cloudflare account.

1. Create a tunnel in the [Cloudflare dashboard](https://one.dash.cloudflare.com/)
2. Set your API token in Hatch:

```yaml
# ~/.hatch/config.yml
settings:
  cloudflare_token: "your-api-token"
```

3. Reference the tunnel name in your service:

```yaml
services:
  web:
    proxy: http://localhost:3000
    tunnel: my-tunnel-name
```

Hatch resolves the tunnel JWT automatically using the API token. Tokens and JWTs are never exposed in process arguments.

You can also run named tunnels without an API token. In that case, cloudflared falls back to credential files in `~/.cloudflared/` (created by `cloudflared tunnel login`).

### Account ID

If your API token has access to multiple Cloudflare accounts, Hatch uses the first one. To target a specific account:

```yaml
settings:
  cloudflare_token: "your-api-token"
  cloudflare_account_id: "abcdef0123456789abcdef0123456789"
```

## Managing tunnels

```bash
hatch tunnel start myapp/web     # start a tunnel
hatch tunnel stop myapp/web      # stop a tunnel
hatch tunnel status              # list all active tunnels
```

Tunnels can also be managed from the dashboard. See [Dashboard](/docs/guides/dashboard) for the tunnel controls UI.

## Config-driven tunnels

When `tunnel` is set in the config, Hatch starts the tunnel automatically when the daemon starts or the config reloads. Removing or clearing the `tunnel` field stops the tunnel. This means you can manage tunnels entirely through config changes.

## Auto-detected tunnel domains

When a `cloudflare_token` is configured, Hatch queries the Cloudflare API for each named tunnel's ingress rules. It reads the hostnames and adds Caddy routes for each one. Requests arriving through the tunnel with external domains (e.g., `myapp.example.com`) are routed to the correct local service automatically.

Without an API token, Hatch cannot detect tunnel domains.

## Vite and HMR

Vite's hot module replacement works through quick tunnels without any Vite configuration. Hatch runs a local rewrite proxy that:

1. Rewrites absolute localhost URLs in HTML to relative paths
2. Retries 404'd asset requests against the Vite dev server
3. Forwards WebSocket upgrades to the Vite dev server

You may see a console warning about a failed WebSocket to `wss://localhost:PORT`. This is harmless; the primary WebSocket through the tunnel URL succeeds.

For quick tunnels, the URL changes each time. HMR may require a manual page refresh after restarting the tunnel.
