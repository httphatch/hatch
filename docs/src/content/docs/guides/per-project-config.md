---
title: "Per-Project Config"
description: "When to use hatch.yml vs hatch add, and how to share project config across a team."
category: "guides"
order: 2
lastUpdated: 2025-03-05
---

# Per-Project Config

Hatch supports two ways to configure projects: `hatch add` for quick one-off setups, and `hatch.yml` files for repeatable, team-friendly configuration.

## hatch add vs hatch.yml

**Use `hatch add`** when:
- You want to proxy a single service quickly
- The project is personal and doesn't need to be shared
- You're experimenting and don't want to create files

```bash
hatch add myapp --proxy http://localhost:3000
```

**Use `hatch.yml`** when:
- The project has multiple services
- The project uses process management (`command`)
- You want teammates to use the same config
- You want the config versioned alongside the code

## Creating a hatch.yml

Create the file in your project root:

```yaml
domain: myapp.test
services:
  web:
    proxy: http://localhost:3000
```

Then register it with Hatch:

```bash
cd ~/projects/myapp
hatch link
```

The project name defaults to the directory basename. Override it with `--name`:

```bash
hatch link --name my-custom-name
```

## Team workflow

Commit `hatch.yml` to your repository. Each developer on the team runs:

```bash
git clone <repo>
cd <repo>
hatch link
```

Each developer's `~/.hatch/config.yml` is updated with the project, and the domain starts working immediately. The `hatch.yml` file is portable; it doesn't contain machine-specific paths or settings.

## Unlinking

To remove a project from Hatch without deleting the `hatch.yml` file:

```bash
cd ~/projects/myapp
hatch unlink
```

This removes the project from `~/.hatch/config.yml`. The `hatch.yml` file stays in the project directory. You can re-link at any time.

## What gets stored where

| Setting | hatch.yml | config.yml |
|---------|-----------|------------|
| Domain | Yes | Copied on link |
| Services | Yes | Copied on link |
| Project path | No (inferred from cwd) | Yes |
| Enabled state | No | Yes |
| Cloudflare token | No | Yes (project-level override) |

The `hatch.yml` file is the source of truth for domain and services. Running `hatch link` again after editing it updates the config.

See [hatch.yml Reference](/docs/reference/hatch-yml) for the full schema and examples.
