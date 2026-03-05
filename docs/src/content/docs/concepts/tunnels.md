---
title: "Tunnels"
description: "Expose local services to the internet with Cloudflare Tunnels for sharing and webhook testing."
category: "concepts"
order: 5
lastUpdated: 2025-03-05
---

# Tunnels

Hatch can expose your local services to the internet using [Cloudflare Tunnels](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/). This is useful for sharing work with teammates, testing webhooks, and mobile device testing.

## Quick tunnels

Quick tunnels require no Cloudflare account. They give you a temporary public URL on `trycloudflare.com`:

```yaml
services:
  web:
    proxy: http://localhost:3000
    tunnel: true
```

Or start one on demand:

```bash
hatch tunnel start myapp/web
# URL: https://random-words.trycloudflare.com
```

Quick tunnel URLs are temporary. They change each time the tunnel restarts.

## Named tunnels

Named tunnels give you a persistent domain through your Cloudflare account. Create a tunnel in the Cloudflare dashboard, then reference it by name:

```yaml
# ~/.hatch/config.yml
settings:
  cloudflare_token: "your-api-token"

# hatch.yml
services:
  web:
    proxy: http://localhost:3000
    tunnel: my-tunnel-name
```

When a named tunnel starts, Hatch uses the API token to resolve the tunnel JWT automatically. Without an API token, cloudflared falls back to credential files in `~/.cloudflared/`.

## When to use each

| | Quick | Named |
|---|---|---|
| Setup | None | Cloudflare account + tunnel config |
| URL | Random, changes each time | Persistent custom domain |
| Use case | Temporary sharing, quick demos | Webhooks, stable preview URLs |
| Account | Not required | Required |

## Vite and HMR

Vite's hot module replacement works through quick tunnels without any Vite configuration. Hatch runs a local rewrite proxy that handles localhost URL rewriting and WebSocket forwarding automatically.

For the full guide on tunnel setup, CLI commands, and advanced configuration, see [Share a Dev Server](/docs/guides/share-dev-server).

For implementation details, see [Architecture](/docs/advanced/architecture).
