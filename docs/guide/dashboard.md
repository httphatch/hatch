# Dashboard

Hatch includes a desktop GUI for managing projects, viewing logs, and monitoring service health.

## Launch

```bash
hatch app
```

This opens the Hatch dashboard as a native macOS window.

## Features

### Project Management

The main view shows all your projects in a grid layout. Each project card displays:

- Project name and domain
- Enabled/disabled toggle
- Service list with proxy targets
- Health indicators (green = healthy, red = unreachable)

You can **add**, **edit**, and **delete** projects directly from the dashboard using the dialog forms. The add and edit dialogs support `command` and `env_file` fields for process management — proxy is optional when a command is set.

### Process Status

Services configured with a `command` show a process status indicator:

- **Running** (teal) — the process is alive
- **Stopped** (coral) — the process has exited

A restart count badge appears when the process has been restarted automatically.

### Tunnel Controls

Services with a `proxy` can be exposed to the internet via Cloudflare Tunnels. Each service row shows a tunnel button:

- **Start** (cloud icon) — starts a quick tunnel for the service
- **Starting** — pulsing badge while cloudflared initializes (can take up to 30 seconds)
- **Running** — displays the public URL as a clickable link, with a stop button (cloud-off icon)
- **Error** — brief error message if the action fails (auto-clears after 5 seconds)

Tunnel output from cloudflared appears in the service's terminal viewer alongside process output. Tunnel status updates automatically via polling. See [Tunnels](/guide/tunnels) for setup.

### Route Map

Each project shows a visual route map of how requests are routed:

```
myapp.test → http://localhost:3000
api.myapp.test → http://localhost:8080
myapp.test/docs → http://localhost:4000
```

This makes it easy to understand complex multi-service setups at a glance.

### Log Viewer

The log viewer streams daemon logs in real time via server-sent events. When you open the panel, it pre-fills with up to 200 recent log entries so you have immediate context. Each entry shows:

- Timestamp
- Log level
- Message
- Structured fields

Toggle the log panel with the button in the header.

### Settings

The settings dialog provides:

- **Config editor** - edit the full `config.yml` directly with a YAML editor
- **Certificate status** - view root CA and intermediate CA details including subject, expiry, and trust status
