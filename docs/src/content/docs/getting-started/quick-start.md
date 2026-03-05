---
title: "Quick Start"
description: "Add your first project and visit it in the browser in under a minute."
category: "getting-started"
order: 3
lastUpdated: 2025-03-05
---

# Quick Start

After [installing](/docs/getting-started/installation) Hatch and running `hatch init` + `hatch up`, you're ready to add your first project.

## Add a project

There are two ways to register a project with Hatch.

### Option A: Quick add

```bash
hatch add myapp --proxy http://localhost:3000
```

This creates a project called `myapp` accessible at `https://myapp.test`, proxying to your local dev server on port 3000.

### Option B: Per-project config

Create a `hatch.yml` in your project directory:

```yaml
domain: myapp.test
services:
  web:
    proxy: http://localhost:3000
```

Then link it:

```bash
cd ~/projects/myapp
hatch link
```

See [Per-Project Config](/docs/guides/per-project-config) for when to use each approach.

## Visit your site

Open your project in the browser:

```bash
hatch open myapp
```

Or navigate directly to `https://myapp.test`. You should see a green lock with no certificate warnings.

## Check status

```bash
hatch status
```

This shows the daemon state and all configured projects with their health status.

## Next steps

- [Domains and DNS](/docs/concepts/domains-and-dns) — understand how `.test` domains work
- [Multi-Service Project](/docs/guides/multi-service-project) — set up a frontend + API + worker
- [CLI Reference](/docs/reference/cli) — all available commands
- [Dashboard](/docs/guides/dashboard) — manage projects visually
