# Node Session Demo

A minimal Node.js server that demonstrates Hatch sessions. Each session runs its own server instance on a different port with a unique HTTPS subdomain.

## Setup

```bash
cd examples/node-session-demo
hatch link
```

This registers the project with Hatch. The base project is accessible at `https://demo.test`.

## Create sessions

Via the API:

```bash
# Create a session named "feature-a"
curl -X POST http://127.0.0.1:42824/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"project": "node-session-demo", "name": "feature-a"}'

# Create a second session
curl -X POST http://127.0.0.1:42824/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"project": "node-session-demo", "name": "feature-b"}'
```

Now you have three instances running:

- `https://demo.test` (base project, port 3000)
- `https://feature-a.demo.test` (session, dynamic port)
- `https://feature-b.demo.test` (session, dynamic port)

Each instance shows its own session name, port, and PID.

## List sessions

```bash
hatch session list
```

## Stop sessions

```bash
hatch session stop node-session-demo/feature-a
hatch session stop-all
```

## MCP integration

Add to your AI tool's MCP config:

```json
{
  "mcpServers": {
    "hatch": {
      "command": "hatch",
      "args": ["mcp"]
    }
  }
}
```

The AI agent can then use `create_session` to spin up new instances and get HTTPS URLs back.

## How it works

The `hatch.yml` uses `{{port}}` as a template variable:

```yaml
domain: demo.test
services:
  web:
    proxy: http://localhost:{{port}}
    command: PORT={{port}} node server.js
```

When Hatch creates a session, it allocates a free port, substitutes `{{port}}` in the command and proxy URL, and starts the server. The session gets a subdomain like `feature-a.demo.test` that routes to the allocated port.

Sessions auto-cleanup after 30 minutes of no HTTP traffic.
