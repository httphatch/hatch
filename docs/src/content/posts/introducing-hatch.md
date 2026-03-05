---
title: "Introducing Hatch"
pubDate: 2026-03-05
description: "Why I built a local HTTPS development tool for macOS, and how it works."
author: "Paul Rose"
image:
  url: "./introducing-hatch-cover.svg"
  alt: "Hatch logo on a dark background"
tags: ["release", "announcement"]
---

# Introducing Hatch

I run a few projects at the same time. Different stacks, different frameworks, different ports. Every time I spin up a new service, it defaults to port 3000. Every time, it conflicts with something already running.

I got tired of remembering which project lives on which port. I got tired of typing `localhost:3847` into my browser. I got tired of dealing with CORS issues and cookie restrictions because my local environment runs on HTTP instead of HTTPS.

So I built Hatch.

## What it does

Hatch gives you real `.test` domains with trusted HTTPS certificates on your local machine. You configure a domain, point it at a port, and Hatch handles the rest.

Your setup looks like this:

```yaml
sites:
  myapp:
    host: localhost
    port: 3000
  api:
    host: localhost
    port: 8080
```

Now `myapp.test` serves your frontend over HTTPS. `api.myapp.test` serves your API. No port numbers to remember. No certificate warnings. No CORS hacks.

## How it works

Hatch runs as a background daemon on macOS. It manages three things:

- **DNS**: A local DNS server that resolves `.test` domains to `127.0.0.1`. Hatch creates resolver files in `/etc/resolver/` so macOS knows to route these domains locally.
- **TLS certificates**: Hatch generates a root CA and intermediate CA, installs them in your system keychain, and issues per-domain certificates on the fly. Your browser trusts them because the root CA is trusted.
- **Reverse proxying**: An embedded Caddy server accepts HTTPS requests on port 443 and forwards them to your local services. Caddy handles TLS termination, HTTP/2, and automatic certificate management.

One binary. One config file. Run `hatch up` and everything starts.

## The port problem

Here is the scenario that kept happening. I have a Next.js app on port 3000, a Go API on port 8080, and an admin panel on port 3000. Except the admin panel can not use 3000 because Next.js already has it. So I move it to 3001. Then I forget which is which.

With Hatch, every service gets a name. `frontend.test`, `api.frontend.test`, `admin.frontend.test`. The port mapping lives in one config file. I never think about port numbers again.

## Per-project config

You can also drop a `.hatch.yml` file in any project directory. When Hatch detects that directory in your linked paths, it picks up the config automatically.

```yaml
# .hatch.yml in your project root
domain: myapp
host: localhost
port: 3000
```

Link the project once:

```bash
hatch link /path/to/myapp
```

Hatch registers the domain and starts proxying. Unlink it when you are done.

## What is next

Hatch is open source and available now. Install it with Homebrew:

```bash
brew install httphatch/tap/hatch
```

The source code is on [GitHub](https://github.com/httphatch/hatch). Feature requests and bug reports are welcome. If you want to contribute, check out the [contributing guide](/docs/contributing/get-involved).

I built this because I needed it. If you juggle multiple local services, you might need it too.
