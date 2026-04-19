---
title: "Sessions"
description: "How Hatch creates ephemeral project instances with dynamic ports for parallel development."
category: "concepts"
order: 6
lastUpdated: 2026-04-18
---

# Sessions

Sessions let you run multiple instances of the same project at the same time. Each session gets its own ports and a unique HTTPS subdomain.

This is useful when multiple AI coding agents (or multiple terminal windows) need to preview different changes to the same project in the browser simultaneously.

## How sessions work

A session is an ephemeral copy of a project's services. When you create a session named `fix-auth` for project `myapp`:

1. Hatch allocates free TCP ports for each service that has a `command`
2. Each service starts on its allocated port using `{{port}}` template substitution
3. The session is accessible at `https://fix-auth.myapp.test`
4. After 30 minutes of idle time (no HTTP traffic), the session is automatically destroyed

Sessions are runtime-only. They are not saved to `config.yml`. They are persisted to `~/.hatch/sessions.json` so they survive daemon restarts.

## Template variables

Projects opt into session support by using `{{port}}` in their service commands:

```yaml
domain: myapp.test
services:
  web:
    proxy: http://localhost:3000
    command: npm run dev -- --port {{port}}
  api:
    proxy: http://localhost:8080
    command: go run ./cmd/api --port {{port}}
    subdomain: api
    env:
      PORT: "{{port}}"
```

When a session is created, Hatch substitutes `{{port}}` with the dynamically allocated port for each service.

Use `{{port:service_name}}` to reference another service's port. For example, a frontend that needs to know the API port:

```yaml
services:
  web:
    command: npm run dev -- --port {{port}} --api-port {{port:api}}
  api:
    command: go run ./cmd/api --port {{port}}
    subdomain: api
```

When running as a regular project (not a session), the `{{port}}` variable resolves to the port from the `proxy` field.

## Session domains

Each session gets a subdomain based on its name:

- Session `fix-auth` on `myapp.test` becomes `https://fix-auth.myapp.test`
- For services with an existing subdomain, the format is `https://fix-auth--api.myapp.test` (double-hyphen delimiter)

Session names must be valid DNS labels: alphanumeric characters and hyphens, 1-63 characters, no leading or trailing hyphen.

## Creating sessions

Sessions can be created through the REST API, CLI, or MCP server.

**REST API:**

```bash
curl -X POST http://127.0.0.1:42824/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"project": "myapp", "name": "fix-auth"}'
```

The response includes the allocated ports and HTTPS domains for each service.

**MCP server:**

AI agents connect to Hatch via the MCP server (`hatch mcp`). The `create_session` tool allocates ports and returns URLs. See [Architecture](/docs/advanced/architecture) for MCP setup.

## Auto-cleanup

Sessions are cleaned up automatically after a period of inactivity. The default TTL is 30 minutes. You can set a custom TTL per session (in seconds) when creating it, or change the global default with the `session_ttl` setting in [config.yml](/docs/reference/config-yml).

A TTL of `0` disables auto-cleanup. Sessions with no TTL persist until explicitly stopped or the daemon shuts down.

## Port allocation

Hatch allocates ports from the ephemeral range (49152-65535 by default). You can customize the range with `session_port_min` and `session_port_max` in your config.

Ports used by static project `proxy` URLs are excluded from allocation. A maximum of 20 sessions can be active at once.

## Lifecycle

When the parent project is disabled or removed from the config, all its sessions are automatically destroyed. Sessions are also destroyed when the daemon shuts down (`hatch down`).

On daemon restart, sessions are restored from `~/.hatch/sessions.json`. If the previously allocated ports are no longer available, Hatch allocates new ones and re-templates the service commands.
