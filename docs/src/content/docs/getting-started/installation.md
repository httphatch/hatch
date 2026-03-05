---
title: "Installation"
description: "Install Hatch via Homebrew or from source, then run hatch init to set up DNS and certificates."
category: "getting-started"
order: 2
lastUpdated: 2025-03-05
---

# Installation

## Prerequisites

- **macOS** (Hatch uses macOS-specific APIs for DNS, Keychain, and launchd)

## Homebrew

```bash
brew install httphatch/tap/hatch
```

## From source

Requires **Go 1.25+**.

```bash
git clone https://github.com/httphatch/hatch.git
cd hatch
make build
sudo ln -sf "$(pwd)/hatch" /usr/local/bin/hatch
```

## Initialize

Run `hatch init` to set up Hatch for the first time. This creates the config directory, generates TLS certificates, trusts the root CA in your Keychain, and installs the DNS resolver.

```bash
hatch init
```

You'll be prompted for your password to install the DNS resolver in `/etc/resolver/` and trust the CA in your Keychain. See [HTTPS and Certificates](/docs/concepts/https-and-certificates) for what this does under the hood.

## Start the daemon

```bash
hatch up
```

This starts the Hatch daemon via launchd. It will automatically start on boot if `auto_start` is enabled (the default).
