---
title: "What is Hatch?"
description: "Local HTTPS development tool for macOS with .test domains, trusted certificates, and reverse proxying."
category: "getting-started"
order: 1
lastUpdated: 2025-03-05
---

# What is Hatch?

If you work on more than one project at a time, you've probably lost track of which one runs on port 3000 and which is on 3001. Or hit a conflict because two things want the same port. Or had to explain to a colleague how to set up their `/etc/hosts` and trust a self-signed certificate.

Hatch fixes all of that. It gives every project a real `.test` domain with trusted HTTPS, automatic DNS, and reverse proxying. No port numbers to remember, no certificate warnings, no manual config.

```
https://myapp.test       -> http://localhost:3000
https://api.myapp.test   -> http://localhost:8080
https://docs.myapp.test  -> http://localhost:4000
```

One binary, one config file, and everything just works.

## How it works

Hatch runs a lightweight daemon that manages three things:

1. **DNS server** — resolves `*.test` to `127.0.0.1` using a macOS resolver file
2. **Certificate authority** — generates a root CA trusted by your Keychain, then issues per-site TLS certificates automatically
3. **Reverse proxy** — routes HTTPS requests to your local services using an embedded [Caddy](https://caddyserver.com) server

All three run in a single process managed by launchd, with automatic startup on boot and live config reloading.

## What makes Hatch different

- **Single binary** — no Docker, no Nginx config files, no external processes
- **Embedded Caddy** — HTTP server with automatic TLS, built in
- **Live reload** — edit your config, changes apply instantly, no restart needed
- **macOS native** — integrates with launchd, Keychain, and the system resolver
- **GUI dashboard** — optional desktop app for managing projects visually
- **Process management** — start and supervise your dev servers directly
- **Tunnels** — expose local services to the internet with Cloudflare Tunnels
