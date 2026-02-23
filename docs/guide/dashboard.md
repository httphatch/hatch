# Dashboard

Hatch includes a desktop GUI for managing projects, viewing logs, and monitoring service health.

## Launch

```bash
hatch app
```

This opens the Hatch dashboard as a native macOS window.

## Navigation

The dashboard uses a tab bar across the top with five tabs:

- **Projects** — project management grid
- **Routes** — full route table for all enabled projects
- **Logs** — real-time log table with filtering
- **Certs** — certificate status for root CA and intermediate CA
- **Settings** — settings form and raw YAML config editor

A status bar at the bottom of the window shows the daemon state (running/stopped), version, and uptime.

## Projects Tab

The main view shows all your projects in a grid layout. Each project card displays:

- Project name and domain
- Enabled/disabled toggle
- Service list with proxy targets and subdomain labels
- Health indicators (green = healthy, red = unreachable)

You can **add**, **edit**, and **delete** projects directly from the dashboard using dialog forms. The add and edit dialogs support `command` and `env_file` fields for process management. Proxy is optional when a command is set.

### Process Status

Services configured with a `command` show a process status indicator:

- **Running** (green) — the process is alive
- **Stopped** (red) — the process has exited

A restart count badge appears when the process has been restarted automatically.

### Tunnel Controls

Services with a `proxy` can be exposed to the internet via Cloudflare Tunnels. Each service row shows a tunnel button:

- **Start** (cloud icon) — starts a quick tunnel for the service
- **Starting** — pulsing badge while cloudflared initializes (can take up to 30 seconds)
- **Running** — displays the public URL as a clickable link, with a stop button (cloud-off icon)

Tunnel output from cloudflared appears in the service's terminal viewer alongside process output. Tunnel status updates automatically via polling. See [Tunnels](/guide/tunnels) for setup.

## Routes Tab

A full-width table of all active routes across all enabled projects. Each row shows:

- Domain
- Route path
- Proxy target
- Health status

Click any domain to open it in your browser.

## Logs Tab

The log viewer streams daemon logs in real time via server-sent events. When you switch to the Logs tab, it pre-fills with up to 200 recent log entries so you have immediate context.

The table has four columns:

- **Time** — timestamp formatted as HH:MM:SS.mmm
- **Level** — colored indicator (debug, info, warn, error)
- **Source** — the component or module that emitted the log entry
- **Message** — the log message with any structured key=value fields

Use the toolbar to filter by log level, search across log messages, or clear the log buffer. The view auto-scrolls to the latest entry; scroll up to pause, then click "Jump to latest" to resume.

## Certs Tab

Displays root CA and intermediate CA certificate details:

- Installed/missing status
- Trust status (root CA only)
- Subject (CN)
- Expiry date

Run `hatch trust` to update certificate trust if the root CA shows as not trusted.

## Settings Tab

The Settings tab has two sections:

- **Settings form** — TLD, HTTP/HTTPS ports, auto-start, tray icon, log level, and Cloudflare token. Changes are saved to `config.yml` and the daemon reloads automatically.
- **Advanced configuration** — raw YAML editor for `config.yml`, for direct editing of the full config including projects.

## Tray Icon Menu

The tray icon in the macOS menu bar provides quick access to all running projects and daemon controls.

- **Open Dashboard** — shows the dashboard window
- **Add Project...** — opens the dashboard to the add project dialog
- **Restart Hatch** — unloads and reloads the launchd plist; the daemon relaunches automatically
- **Stop Hatch** — removes the launchd plist and stops the daemon; it will not restart on boot until you run `hatch up` again

Each project appears as a submenu with its domain, an enable/disable toggle, and per-service health indicators.
