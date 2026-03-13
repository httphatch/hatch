---
title: "Dashboard"
description: "Manage projects, view logs, and monitor services from the Hatch desktop app."
category: "guides"
order: 5
lastUpdated: 2025-03-05
---

# Dashboard

Hatch includes a desktop GUI for managing projects, viewing logs, and monitoring service health.

## Launch

```bash
hatch app
```

This opens the Hatch dashboard as a native macOS window.

## Navigation

The dashboard uses a tab bar across the top with three tabs:

- **Projects** — project management grid
- **Logs** — real-time log table with filtering
- **Settings** — routes, certificates, settings form, and raw YAML config editor

The top bar also shows daemon controls on the right side. When the daemon is running, restart and stop buttons appear. When stopped, a start button appears instead.

A status bar at the bottom shows the daemon state (running/stopped), version, and uptime.

## Projects tab

The main view shows all your projects in a grid layout. Each project card displays:

- Project name and domain
- Enabled/disabled toggle
- Service list with proxy targets and subdomain labels
- Health indicators (green = healthy, red = unreachable)

You can add, edit, and delete projects directly from the dashboard using dialog forms. The add and edit dialogs support `command`, `dir`, and `env_file` fields for process management. Proxy is optional when a command is set.

### Process status

Services configured with a `command` show a process status indicator:

- **Running** (green) — the process is alive
- **Stopped** (red) — the process has exited

A restart count badge appears when the process has been restarted automatically.

### Tunnel controls

Services with a `proxy` can be exposed to the internet via Cloudflare Tunnels. Each service row shows a tunnel button:

- **Start** (cloud icon) — starts a quick tunnel for the service
- **Starting** — pulsing badge while cloudflared initializes (can take up to 30 seconds)
- **Running** — displays the public URL as a clickable link, with a stop button (cloud-off icon)

Tunnel output from cloudflared appears in the service's terminal viewer alongside process output. See [Share a Dev Server](/docs/guides/share-dev-server) for setup.

## Logs tab

The log viewer streams daemon logs in real time via server-sent events. When you switch to the Logs tab, it pre-fills with up to 200 recent log entries.

The table has four columns: Time, Level, Source, and Message. Use the toolbar to filter by log level, search across messages, or clear the buffer. The view auto-scrolls to the latest entry; scroll up to pause, then click "Jump to latest" to resume.

## Settings tab

The Settings tab uses a sidebar with four sections:

- **Routes** — all active routes across enabled projects with domain, path, proxy target, and health status
- **Certificates** — root CA and intermediate CA details with trust status and expiry dates
- **General** — TLD, ports, auto-start, tray icon, log level, and Cloudflare token
- **Advanced** — raw YAML editor for `config.yml`

Changes in General are saved to `config.yml` and the daemon reloads automatically.

## Tray icon

The tray icon in the macOS menu bar provides quick access to all running projects and daemon controls. The tray runs as a separate process from the daemon. Running `hatch down` or `hatch restart` stops both the daemon and the tray. Running `hatch up` or `hatch restart` relaunches the tray automatically when `tray_icon` is enabled.

Menu items:
- **Open Dashboard** — shows the dashboard window
- **Add Project...** — opens the add project dialog
- **Restart Hatch** / **Stop Hatch** — daemon lifecycle controls
- **Quit** — closes the tray app; the daemon continues running

Each project appears as a submenu with its domain, an enable/disable toggle, and per-service health indicators.
