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

You can **add**, **edit**, and **delete** projects directly from the dashboard using the dialog forms.

### Route Map

Each project shows a visual route map of how requests are routed:

```
myapp.test → http://localhost:3000
api.myapp.test → http://localhost:8080
myapp.test/docs → http://localhost:4000
```

This makes it easy to understand complex multi-service setups at a glance.

### Log Viewer

The log viewer streams daemon logs in real time via server-sent events. Each entry shows:

- Timestamp
- Log level
- Message
- Structured fields

Toggle the log panel with the button in the header.

### Settings

The settings dialog provides:

- **Config editor** - edit the full `config.yml` directly with a YAML editor
- **Certificate status** - view root CA and intermediate CA details including subject, expiry, and trust status
