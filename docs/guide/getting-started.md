# Getting Started

## Prerequisites

- **macOS** (Hatch uses macOS-specific APIs for DNS, Keychain, and launchd)

## Install

### Homebrew

```bash
brew install httphatch/tap/hatch
```

### From Source

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

You'll be prompted for your password to install the DNS resolver in `/etc/resolver/` and trust the CA in your Keychain.

## Start the Daemon

```bash
hatch up
```

This starts the Hatch daemon via launchd. It will automatically start on boot if `auto_start` is enabled (the default).

## Add Your First Project

There are two ways to add a project:

### Option A: Quick Add

```bash
hatch add myapp --proxy http://localhost:3000
```

This creates a project called `myapp` accessible at `https://myapp.test`, proxying to your local dev server on port 3000.

### Option B: Per-Project Config

Create a `.hatch.yml` in your project directory:

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

## Visit Your Site

Open your project in the browser:

```bash
hatch open myapp
```

Or navigate directly to `https://myapp.test`. You should see a green lock — no certificate warnings.

## Check Status

```bash
hatch status
```

This shows the daemon state and all configured projects with their health status.

## Next Steps

- [Add multiple services](/reference/hatch-yml) with path and subdomain routing
- [Explore the CLI](/reference/cli) for all available commands
- [Launch the dashboard](/guide/dashboard) for a visual interface
- [Configure settings](/reference/config) like TLD, ports, and log level
