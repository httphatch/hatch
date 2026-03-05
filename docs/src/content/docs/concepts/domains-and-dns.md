---
title: "Domains and DNS"
description: "How Hatch resolves .test domains to your local machine using a built-in DNS server."
category: "concepts"
order: 1
lastUpdated: 2025-03-05
---

# Domains and DNS

Every Hatch project gets a `.test` domain. Instead of remembering `localhost:3000` or `localhost:8080`, you visit `https://myapp.test` or `https://api.myapp.test`.

## Why .test domains?

Port numbers are hard to remember. When you work on multiple projects, they collide. And some browser features (cookies with `SameSite`, service workers, Web Crypto) behave differently on `localhost` than on a named host with HTTPS.

A `.test` domain gives each project a stable, human-readable address that works exactly like a production URL.

## How macOS resolves them

macOS checks `/etc/resolver/` for per-TLD nameserver overrides. During `hatch init`, Hatch creates a file at `/etc/resolver/test` containing:

```
nameserver 127.0.0.1
port 5053
```

Any DNS query for `*.test` is sent to `127.0.0.1:5053`, which is Hatch's built-in DNS server. All other DNS queries go through your normal DNS servers untouched.

The DNS server responds to every `*.test` query with `127.0.0.1`. This means any subdomain at any depth resolves to your local machine.

## Choosing a TLD

The default TLD is `test`, which is an IANA-reserved TLD that will never be used for real websites. You can change it in your config:

```yaml
settings:
  tld: test  # also supports: localhost, local, dev
```

Changing the TLD updates the resolver file, DNS server, and all project domains.

## What happens to a request

1. Browser asks "where is `myapp.test`?"
2. macOS sees the `/etc/resolver/test` file and sends the DNS query to `127.0.0.1:5053`
3. Hatch's DNS server responds with `127.0.0.1`
4. Browser connects to `127.0.0.1:443` (Hatch's HTTPS port)
5. Hatch's reverse proxy matches the `Host: myapp.test` header and forwards to your dev server

See [Architecture](/docs/advanced/architecture) for the full request flow and DNS server implementation details.
