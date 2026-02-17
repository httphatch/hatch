# Hatch

Local HTTPS development for macOS.

Hatch gives every local project a clean `.test` domain with trusted HTTPS. It manages DNS resolution, TLS certificates, and reverse proxying so you can work with `https://myapp.test` instead of `localhost:3000`.

## Install

### Homebrew (recommended)

```bash
brew install httphatch/tap/hatch
```

### From source

Requires Go 1.25+.

```bash
git clone https://github.com/httphatch/hatch.git
cd hatch
make build
sudo ln -sf "$(pwd)/hatch" /usr/local/bin/hatch
```

## Quick start

```bash
# One-time setup — creates certs, trusts CA, installs DNS resolver
hatch init

# Start the daemon
hatch up

# Add a project
hatch add myapp --proxy http://localhost:3000

# Open it
hatch open myapp
# → https://myapp.test with a green lock
```

## Multi-service projects

Create a `.hatch.yml` in your project root:

```yaml
domain: myapp.test
services:
  frontend:
    proxy: http://localhost:3000
    command: npm run dev
  api:
    proxy: http://localhost:8080
    subdomain: api
  docs:
    proxy: http://localhost:4000
    route: /docs
```

Then link it:

```bash
cd ~/projects/myapp
hatch link
```

This gives you:
- `https://myapp.test` → frontend on port 3000
- `https://api.myapp.test` → API on port 8080
- `https://myapp.test/docs/*` → docs on port 4000

## Documentation

Full docs at [httphatch.com](https://httphatch.com):

- [Getting Started](https://httphatch.com/guide/getting-started) — install, init, first project
- [CLI Reference](https://httphatch.com/reference/cli) — all commands and flags
- [Configuration](https://httphatch.com/reference/config) — global config options
- [.hatch.yml](https://httphatch.com/reference/hatch-yml) — per-project config format
- [How It Works](https://httphatch.com/advanced/how-it-works) — architecture and internals
- [Troubleshooting](https://httphatch.com/advanced/troubleshooting) — common issues and fixes

## License

[MIT](LICENSE)
