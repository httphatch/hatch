---
title: "Debug a Crashing Process"
description: "Diagnose and fix a managed process that keeps restarting."
category: "guides"
order: 3
lastUpdated: 2025-03-05
---

# Debug a Crashing Process

When a managed process keeps crashing, Hatch restarts it with exponential backoff. Here's how to find and fix the problem.

## Symptoms

- `hatch status` shows a restart count badge (e.g., "restarts: 5")
- Dashboard shows the process status flickering between running and stopped
- Service shows as unhealthy because the dev server never stays up long enough to listen on its port

## Step 1: Check status

```bash
hatch status
```

Look for services with a high restart count or "unhealthy" status. The status output includes diagnostics: whether the port is free (nothing listening) or held by another process.

## Step 2: Read the logs

```bash
hatch logs -f
```

Filter for your service name. The logs show process start, exit, and restart events along with the exit code. Common patterns:

- **Exit code 1** — the command failed (missing dependency, syntax error, wrong directory)
- **Exit code 127** — command not found (wrong PATH, missing binary)
- **Exit code 126** — permission denied
- **SIGKILL (exit 137)** — out of memory or killed by the system

## Step 3: Run the command manually

Copy the `command` from your config and run it yourself in the same directory:

```bash
cd ~/projects/myapp/packages/api
npm run dev
```

This often reveals the error immediately: missing `node_modules`, wrong Node version, or a port conflict with another process.

## Step 4: Check for port conflicts

If the command starts but the service stays unhealthy:

```bash
hatch doctor
```

Or check the port directly:

```bash
sudo lsof -i :3000
```

Another process may already be using the port. Kill it or reconfigure your service.

## Understanding backoff

Hatch uses exponential backoff to avoid hammering a broken process:

| Restart | Delay |
|---------|-------|
| 1st | 1 second |
| 2nd | 2 seconds |
| 3rd | 4 seconds |
| 4th | 8 seconds |
| 5th | 16 seconds |
| 6th+ | 30 seconds (max) |

The backoff resets after the process runs stably for 60 seconds. If you fix the underlying issue, the next restart happens immediately with a fresh backoff counter.

## Common causes

- **Missing dependencies** — run `npm install` in the service directory
- **Wrong working directory** — check that `dir` in your config is correct relative to the project path
- **Environment variables** — verify `env_file` exists and contains required variables
- **Port already in use** — another process or a previous instance is holding the port
- **Wrong command** — test the command manually in the expected directory

## Step 5: Run doctor

```bash
hatch doctor
```

This runs all diagnostic checks: config validity, DNS, certificates, ports, and stale projects. Fix any reported issues and the process should stabilize.
