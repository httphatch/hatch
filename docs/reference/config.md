# Configuration

Hatch stores its configuration in `~/.hatch/config.yml`. You can edit it directly, use `hatch config` to open it in your editor, or manage it through the [dashboard](/guide/dashboard).

Changes to the config file are picked up automatically. No daemon restart required.

## Full Example

```yaml
version: 1
settings:
  tld: test
  http_port: 80
  https_port: 443
  auto_start: true
  tray_icon: true
  log_level: info
projects:
  myapp:
    domain: myapp.test
    path: /Users/me/projects/myapp
    enabled: true
    services:
      web:
        proxy: http://localhost:3000
      api:
        proxy: http://localhost:8080
        subdomain: api
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
- **Allowed:** `test`, `localhost`, `local`, `dev`

The top-level domain used for all project domains.

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

Show the Hatch tray icon and status menu (macOS only). Set to `false` to run the daemon without a tray icon or window. Changes to this setting take effect after the daemon restarts.

### `settings.log_level`

- **Type:** `string`
- **Default:** `info`
- **Allowed:** `debug`, `info`, `warn`, `error`

Controls the verbosity of daemon logs in `~/.hatch/logs/hatch.log`.

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

### `projects.<name>.services`

A map of named services. Each project must have at least one service.

## Services

### `services.<name>.proxy`

- **Type:** `string` (URL)
- **Required:** yes, unless `command` is set

The upstream URL to proxy to. Must be a valid `http://` or `https://` URL.

```yaml
proxy: http://localhost:3000
```

### `services.<name>.route`

- **Type:** `string`
- **Optional**

A path prefix for this service. Requests matching this path are routed to this service's proxy.

```yaml
route: /api
```

### `services.<name>.subdomain`

- **Type:** `string`
- **Optional**

A subdomain for this service. Creates a route at `<subdomain>.<domain>`.

```yaml
subdomain: api
# Routes: api.myapp.test → proxy
```

Must be a valid DNS label: alphanumeric, hyphens allowed (not at start/end), max 63 characters.

### `services.<name>.command`

- **Type:** `string`
- **Optional**

A shell command to run for this service. Hatch supervises the process with exponential backoff (1s→30s, resets after 60s stable). The command is executed via `sh -c` in the project's `path` directory.

```yaml
command: npm run dev
```

### `services.<name>.env_file`

- **Type:** `string`
- **Optional**

Path to an environment file to load before running the command. Relative paths are resolved from the project's `path` directory.

```yaml
env_file: .env
```

### `services.<name>.websocket`

- **Type:** `boolean`
- **Default:** `false`

Enables WebSocket proxying for this service. Adds the necessary `Connection` and `Upgrade` headers and sets instant response flushing.

## Validation Rules

- `http_port` and `https_port` must be different
- Each project must have a unique `domain`
- Each project must have at least one service
- Each service must have at least one of `proxy` or `command`
- Service `proxy` must be a valid URL
- Service `subdomain` must be a valid DNS label
- `route` and `subdomain` require `proxy`
