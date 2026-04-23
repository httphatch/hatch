---
title: "config.yml Reference"
description: "Complete reference for the global Hatch configuration file at ~/.hatch/config.yml."
category: "reference"
order: 2
lastUpdated: 2026-04-18
---

# config.yml Reference

Hatch stores its configuration in `~/.hatch/config.yml`. You can edit it directly, use `hatch config` to open it in your editor, or manage it through the [dashboard](/docs/guides/dashboard).

Changes to the config file are picked up automatically. No daemon restart required.

## Full example

```yaml
version: 1
settings:
  tld: test
  http_port: 80
  https_port: 443
  auto_start: true
  tray_icon: true
  log_level: info
  cloudflare_token: "your-api-token"
  cloudflare_account_id: "abcdef0123456789abcdef0123456789"
  session_port_min: 49152
  session_port_max: 65535
  session_ttl: 1800
projects:
  myapp:
    domain: myapp.test
    path: /Users/me/projects/myapp
    enabled: true
    cloudflare_token: "project-token"
    services:
      web:
        proxy: http://localhost:3000
        tunnel: true
      api:
        proxy: http://localhost:8080
        subdomain: api
        tunnel: my-api-tunnel
      docs:
        proxy: http://localhost:4000
        route: /docs
      ws:
        proxy: http://localhost:9000
        subdomain: ws
        websocket: true
      worker:
        proxy: http://localhost:3000
        command: npm run worker
        dir: packages/worker
        env_file: .env.worker
      tasks:
        command: npm run tasks
```

## Settings

### `version`

- **Type:** `integer`
- **Required:** yes
- **Value:** `1`

Config schema version. Must be `1`.

### `settings.tld`

- **Type:** `string`
- **Default:** `test`
- **Allowed:** `test`, `localhost`, `local`, `internal`

The top-level domain used for all project domains. See [Domains and DNS](/docs/concepts/domains-and-dns) for how the TLD affects DNS resolution.

### `settings.http_port`

- **Type:** `integer`
- **Default:** `80`
- **Range:** 1-65535

Port for HTTP traffic. HTTP requests are automatically redirected to HTTPS.

### `settings.https_port`

- **Type:** `integer`
- **Default:** `443`
- **Range:** 1-65535

Port for HTTPS traffic. Must be different from `http_port`.

### `settings.auto_start`

- **Type:** `boolean`
- **Default:** `true`

When enabled, the daemon starts automatically on login via launchd and restarts on crash.

### `settings.tray_icon`

- **Type:** `boolean`
- **Default:** `true`

Show the Hatch tray icon and status menu (macOS only). When enabled, `hatch up` launches the tray as a separate process. Set to `false` to start without the tray. The daemon always runs headless regardless of this setting. You can launch the tray manually at any time with `hatch app`.

### `settings.log_level`

- **Type:** `string`
- **Default:** `info`
- **Allowed:** `debug`, `info`, `warn`, `error`

Controls the verbosity of daemon logs in `~/.hatch/logs/hatch.log`.

### `settings.cloudflare_token`

- **Type:** `string`
- **Optional**

Cloudflare API token for named [tunnels](/docs/concepts/tunnels). When set, Hatch uses this token to resolve tunnel JWTs and auto-detect external hostnames. Can be overridden per-project.

### `settings.cloudflare_account_id`

- **Type:** `string`
- **Optional**

Cloudflare account ID (32-character hex string). When set, Hatch skips the account lookup API call. When empty, Hatch auto-detects the account from the API token (uses the first account).

### `settings.session_port_min`

- **Type:** `integer`
- **Default:** `49152`
- **Range:** 1024-65535
- **Optional**

Lower bound of the port range used when allocating dynamic ports for [sessions](/docs/concepts/sessions).

### `settings.session_port_max`

- **Type:** `integer`
- **Default:** `65535`
- **Range:** 1024-65535
- **Optional**

Upper bound of the session port range. Must be greater than or equal to `session_port_min`.

### `settings.session_ttl`

- **Type:** `integer` (seconds)
- **Default:** `1800` (30 minutes)
- **Optional**

Default idle timeout for [sessions](/docs/concepts/sessions). A session is destroyed after this many seconds with no HTTP traffic. Set to `0` to disable auto-cleanup.

## Projects

Each key under `projects` is the project name. Projects are a map, so names must be unique.

### `projects.<name>.domain`

- **Type:** `string`
- **Required:** yes

The hostname for this project. Must be a valid hostname ending with `.<tld>`. Must be unique across all projects.

### `projects.<name>.path`

- **Type:** `string`
- **Required:** yes

Filesystem path to the project directory. Used by `hatch doctor` to detect stale projects.

### `projects.<name>.enabled`

- **Type:** `boolean`
- **Default:** `true`

When `false`, the project's routes are excluded from the proxy config.

### `projects.<name>.cloudflare_token`

- **Type:** `string`
- **Optional**

Project-specific Cloudflare API token. Overrides `settings.cloudflare_token` for this project's named tunnels.

### `projects.<name>.services`

A map of named services. Each project must have at least one service.

## Services

### `services.<name>.proxy`

- **Type:** `string` (URL)
- **Required:** yes, unless `command` is set

The upstream URL to proxy to. Must be a valid `http://` or `https://` URL. See [Routing](/docs/concepts/routing) for how proxy targets map to routes.

### `services.<name>.route`

- **Type:** `string`
- **Optional**

A path prefix for this service. Requests matching this path are routed to this service's proxy.

### `services.<name>.subdomain`

- **Type:** `string`
- **Optional**

A subdomain for this service. Creates a route at `<subdomain>.<domain>`. Must be a valid DNS label.

### `services.<name>.command`

- **Type:** `string`
- **Optional**

A shell command to run for this service. Hatch supervises the process with [exponential backoff](/docs/concepts/process-management). The command is executed via `sh -c` in the project's `path` directory.

### `services.<name>.dir`

- **Type:** `string`
- **Optional**

Working directory for the command. Must be a relative path (resolved from the project's `path` directory).

### `services.<name>.env_file`

- **Type:** `string`
- **Optional**

Path to an environment file to load before running the command. Relative paths are resolved from the project's `path` directory.

### `services.<name>.websocket`

- **Type:** `boolean`
- **Default:** `false`

Enables [WebSocket proxying](/docs/concepts/routing) for this service.

### `services.<name>.tunnel`

- **Type:** `string` or `boolean`
- **Optional**

Exposes this service to the internet via a Cloudflare Tunnel. Requires `proxy` to be set.

- `true` — starts a quick tunnel (temporary URL, no account needed)
- `"my-tunnel-name"` — starts a named tunnel

See [Tunnels](/docs/concepts/tunnels) for details.

## Validation rules

- `http_port` and `https_port` must be different
- Each project must have a unique `domain`
- Each project must have at least one service
- Each service must have at least one of `proxy` or `command`
- Service `proxy` must be a valid URL
- Service `subdomain` must be a valid DNS label
- `route` and `subdomain` require `proxy`
- `dir` must be a relative path without `..`
- `dir` requires `command`
- `tunnel` requires `proxy`
- `settings.cloudflare_account_id`, when set, must be a 32-character hex string
- `session_port_min` must be 1024-65535
- `session_port_max` must be 1024-65535 and >= `session_port_min`
- `session_ttl` must be >= 0
