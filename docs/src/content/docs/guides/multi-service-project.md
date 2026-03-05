---
title: "Multi-Service Project"
description: "Set up a frontend, API, and background worker in a single project with Hatch."
category: "guides"
order: 1
lastUpdated: 2025-03-05
---

# Multi-Service Project

This guide walks through setting up a project with a React frontend, an API server, and a background worker, all managed by Hatch.

## The setup

You have a project directory with three services:

- **Frontend** — React dev server on port 3000
- **API** — Express server on port 8080
- **Worker** — background job processor (no HTTP)

## Create hatch.yml

In your project root, create `hatch.yml`:

```yaml
domain: myapp.test
services:
  web:
    proxy: http://localhost:3000
    command: npm run dev
    dir: packages/web
  api:
    proxy: http://localhost:8080
    command: npm run dev
    dir: packages/api
    subdomain: api
  worker:
    command: npm run worker
    dir: packages/worker
    env_file: .env
```

## Link the project

```bash
cd ~/projects/myapp
hatch link
```

Hatch reads the `hatch.yml`, validates it, and adds the project to `~/.hatch/config.yml`. All three processes start automatically.

## Verify

Check that everything is running:

```bash
hatch status
```

You should see all three services listed. The `web` and `api` services show health status (green = healthy). The `worker` service shows process status only since it has no proxy.

Open in your browser:

- `https://myapp.test` — React frontend
- `https://api.myapp.test` — API server

Both should load with a green lock and no certificate warnings.

## What's happening

1. Hatch starts all three commands as supervised processes
2. DNS resolves `myapp.test` and `api.myapp.test` to `127.0.0.1`
3. The reverse proxy routes requests by host: root domain to the frontend, `api` subdomain to the API server
4. The worker runs in the background with no HTTP routing

If any process crashes, Hatch restarts it with [exponential backoff](/docs/concepts/process-management). If you edit `hatch.yml` and run `hatch link` again, Hatch reconciles the config: new services start, removed services stop, and changed services restart.

## Adding path-based routes

If you prefer path-based routing instead of subdomains:

```yaml
domain: myapp.test
services:
  web:
    proxy: http://localhost:3000
    command: npm run dev
    dir: packages/web
  api:
    proxy: http://localhost:8080
    command: npm run dev
    dir: packages/api
    route: /api
```

Now the API is at `https://myapp.test/api/*` instead of `https://api.myapp.test`.

See [Routing](/docs/concepts/routing) for how Hatch handles route specificity and sorting.
