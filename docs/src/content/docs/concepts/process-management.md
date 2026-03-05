---
title: "Process Management"
description: "How Hatch starts, supervises, and restarts your dev server processes."
category: "concepts"
order: 4
lastUpdated: 2025-03-05
---

# Process Management

Hatch can start and supervise your dev servers directly. Instead of opening multiple terminal tabs to run `npm run dev` for each service, you define a `command` in your config and Hatch manages the process lifecycle.

## Defining a command

Add `command` to any service:

```yaml
domain: myapp.test
services:
  web:
    proxy: http://localhost:3000
    command: npm run dev
  worker:
    command: npm run worker
```

The `web` service runs `npm run dev`, and Hatch proxies `https://myapp.test` to its output on port 3000. The `worker` service runs a background process with no proxy route.

## Supervision and restarts

When a managed process exits, Hatch restarts it automatically with exponential backoff:

- First restart: 1 second delay
- Each subsequent restart: delay doubles
- Maximum delay: 30 seconds
- Backoff resets after the process runs stably for 60 seconds

The restart counter is visible in `hatch status` and the dashboard. A high restart count usually means the process is crashing on startup. See [Debug a Crashing Process](/docs/guides/debug-crashing-process) for diagnosis steps.

## Graceful shutdown

When Hatch stops a managed process (during `hatch down`, config change, or project removal):

1. Sends `SIGTERM` to the process group
2. Waits up to 5 seconds for the process to exit
3. Sends `SIGKILL` if still running

Each command runs in its own process group, so child processes spawned by the command are also cleaned up.

## Working directories

By default, commands run in the project's `path` directory. Use `dir` to run in a subdirectory:

```yaml
domain: myapp.test
services:
  api:
    proxy: http://localhost:8080
    command: npm run dev
    dir: packages/api
  web:
    proxy: http://localhost:3000
    command: npm run dev
    dir: packages/web
```

The `dir` value must be a relative path. It's resolved from the project's root directory.

## Environment files

Use `env_file` to load environment variables before starting the command:

```yaml
services:
  web:
    proxy: http://localhost:3000
    command: npm run dev
    env_file: .env
```

The file is loaded from the project directory (or the `dir` subdirectory if set). Variables from the env file are merged with the system environment; env file values take precedence on conflict.

## Example: monorepo with three services

```yaml
domain: myapp.test
services:
  api:
    proxy: http://localhost:8080
    command: npm run dev
    dir: packages/api
    subdomain: api
    env_file: .env
  web:
    proxy: http://localhost:3000
    command: npm run dev
    dir: packages/web
  worker:
    command: npm run worker
    dir: packages/worker
    env_file: .env.worker
```

Running `hatch link` in the monorepo root starts all three processes, routes `https://myapp.test` to the web frontend and `https://api.myapp.test` to the API, and runs the worker in the background.
