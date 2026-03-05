---
title: "Routing"
description: "How Hatch maps domains, paths, and subdomains to your local services."
category: "concepts"
order: 3
lastUpdated: 2025-03-05
---

# Routing

Hatch routes HTTPS requests to your local services based on domain, path, and subdomain matching. Each project has a domain, and each service within a project defines where its traffic comes from and where it goes.

## Domain-to-service mapping

The simplest case: one project, one service, one domain.

```yaml
domain: myapp.test
services:
  web:
    proxy: http://localhost:3000
```

All requests to `https://myapp.test` forward to `http://localhost:3000`.

## Path-based routing

Use `route` to match a path prefix:

```yaml
domain: myapp.test
services:
  app:
    proxy: http://localhost:3000
  api:
    proxy: http://localhost:8080
    route: /api
  docs:
    proxy: http://localhost:4000
    route: /docs
```

This produces:

- `https://myapp.test` -> `http://localhost:3000`
- `https://myapp.test/api/*` -> `http://localhost:8080`
- `https://myapp.test/docs/*` -> `http://localhost:4000`

## Subdomain routing

Use `subdomain` to route traffic on a subdomain:

```yaml
domain: myapp.test
services:
  web:
    proxy: http://localhost:3000
  api:
    proxy: http://localhost:8080
    subdomain: api
```

This produces:

- `https://myapp.test` -> `http://localhost:3000`
- `https://api.myapp.test` -> `http://localhost:8080`

Subdomain values must be valid DNS labels: alphanumeric characters and hyphens (not at start or end), max 63 characters.

## Route sorting

When multiple services could match a request, Hatch applies a specificity order:

1. **Subdomain routes** — most specific, matched first
2. **Path routes** — matched by longest prefix
3. **Catch-all** — the service with no `route` or `subdomain`

You don't need to worry about ordering in your config file. Hatch sorts routes automatically.

## WebSocket proxying

Add `websocket: true` to a service to enable WebSocket support:

```yaml
domain: myapp.test
services:
  web:
    proxy: http://localhost:3000
  ws:
    proxy: http://localhost:9000
    subdomain: ws
    websocket: true
```

This adds the necessary `Connection` and `Upgrade` headers and enables instant response flushing so WebSocket frames pass through immediately.

## How routing connects to the proxy

Hatch translates your YAML config into Caddy's JSON route configuration. Each service becomes a Caddy route with host matching, optional path matching, and reverse proxy handlers. Changes to your config are pushed to Caddy via its admin API without restarting the daemon.

See [config.yml Reference](/docs/reference/config-yml) and [hatch.yml Reference](/docs/reference/hatch-yml) for the full field documentation.
