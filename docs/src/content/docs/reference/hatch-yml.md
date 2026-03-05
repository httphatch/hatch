---
title: "hatch.yml Reference"
description: "Per-project configuration file format for defining domains and services."
category: "reference"
order: 3
lastUpdated: 2025-03-05
---

# hatch.yml Reference

The `hatch.yml` file is a per-project config file that lives in your project's root directory. It defines the domain and services for that project, and is used with `hatch link` to register the project with Hatch.

## Schema

```yaml
domain: myapp.test
services:
  <service-name>:
    proxy: <upstream-url>        # required unless command is set
    command: <shell-command>      # optional
    dir: <relative-path>         # optional
    env_file: <path>             # optional
    route: <path-prefix>         # optional
    subdomain: <subdomain>       # optional
    websocket: <bool>            # optional, default: false
    tunnel: <bool|string>        # optional
```

### `domain`

- **Type:** `string`
- **Required:** yes

The hostname for this project. Must end with the configured TLD (e.g., `.test`).

### `services`

- **Type:** map
- **Required:** yes (at least one entry)

A map of named services. See [config.yml Reference](/docs/reference/config-yml#services) for full details on each field.

## Usage

### Link a project

```bash
cd ~/projects/myapp
hatch link
```

This reads `hatch.yml`, validates it, and adds the project to `~/.hatch/config.yml`. The project name defaults to the directory basename. Override it with `--name`:

```bash
hatch link --name my-custom-name
```

### Unlink a project

```bash
cd ~/projects/myapp
hatch unlink
```

This removes the project from the central config. The `hatch.yml` file is **not** deleted.

## Examples

### Simple web app

```yaml
domain: myapp.test
services:
  web:
    proxy: http://localhost:3000
```

### API + frontend

```yaml
domain: myapp.test
services:
  frontend:
    proxy: http://localhost:3000
  api:
    proxy: http://localhost:8080
    subdomain: api
```

Produces:
- `https://myapp.test` -> `http://localhost:3000`
- `https://api.myapp.test` -> `http://localhost:8080`

### Path-based routing

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

Produces:
- `https://myapp.test` -> `http://localhost:3000`
- `https://myapp.test/api/*` -> `http://localhost:8080`
- `https://myapp.test/docs/*` -> `http://localhost:4000`

### WebSocket service

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

### Process management

```yaml
domain: myapp.test
services:
  web:
    proxy: http://localhost:3000
    command: npm run dev
  worker:
    command: npm run worker
    env_file: .env
```

### Monorepo

```yaml
domain: myapp.test
services:
  api:
    proxy: http://localhost:8080
    command: npm run dev
    dir: packages/api
    subdomain: api
  web:
    proxy: http://localhost:3000
    command: npm run dev
    dir: packages/web
```

### Tunnel support

```yaml
domain: myapp.test
services:
  web:
    proxy: http://localhost:3000
    tunnel: true
```

See [Tunnels](/docs/concepts/tunnels) for quick vs named tunnels and [Share a Dev Server](/docs/guides/share-dev-server) for the full guide.
