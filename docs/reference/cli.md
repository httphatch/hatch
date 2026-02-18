# CLI Commands

## Daemon

### `hatch up`

Start the Hatch daemon.

Initializes the config, generates and trusts the CA if needed, installs the DNS resolver, and creates the launchd plist. Safe to run multiple times (idempotent).

```bash
hatch up
```

### `hatch down`

Stop the Hatch daemon.

Unloads the launchd plist and stops the daemon gracefully.

```bash
hatch down
```

### `hatch restart`

Restart the daemon (stops then starts).

```bash
hatch restart
```

### `hatch status`

Show daemon state, project list, and service health.

```bash
hatch status
```

### `hatch logs`

View daemon logs.

```bash
hatch logs
hatch logs -f          # follow (tail)
hatch logs -n 100      # show last 100 lines
```

| Flag | Default | Description |
|------|---------|-------------|
| `-f, --follow` | `false` | Continuously stream new log entries |
| `-n, --lines` | `50` | Number of lines to show |

## Projects

### `hatch add <name>`

Add or update a project.

```bash
hatch add myapp --proxy http://localhost:3000
hatch add myapp --proxy http://localhost:3000 --domain custom.test
hatch add myapp --proxy http://localhost:3000 --path ~/projects/myapp
```

| Flag | Default | Description |
|------|---------|-------------|
| `--proxy` | `http://localhost:3000` | Upstream target URL |
| `--domain` | `<name>.<tld>` | Override the project domain |
| `--path` | current directory | Project directory path |

### `hatch link`

Link a project using the `hatch.yml` file in the current directory.

```bash
cd ~/projects/myapp
hatch link
hatch link --name custom-name
```

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | directory basename | Override the project name |

### `hatch unlink`

Unlink a project from Hatch. Removes it from the central config but does not delete the `hatch.yml` file.

```bash
cd ~/projects/myapp
hatch unlink
hatch unlink --name custom-name
```

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | directory basename | Override the project name |

### `hatch list`

List all configured projects. Alias: `hatch ls`.

```bash
hatch list
```

Shows project name, domain, enabled/disabled status, and services.

### `hatch remove <name>`

Remove a project from the config. Alias: `hatch rm`.

```bash
hatch remove myapp
hatch remove myapp -f    # skip confirmation
```

| Flag | Default | Description |
|------|---------|-------------|
| `-f, --force` | `false` | Skip confirmation prompt |

### `hatch enable <name>`

Enable a disabled project.

```bash
hatch enable myapp
```

### `hatch disable <name>`

Disable a project without removing it.

```bash
hatch disable myapp
```

### `hatch open [name]`

Open a project in your default browser.

```bash
hatch open myapp
```

## System

### `hatch init`

Initialize Hatch for first-time use. Creates `~/.hatch`, generates the CA hierarchy, trusts the root CA in Keychain, and installs the DNS resolver. Does **not** start the daemon.

```bash
hatch init
```

### `hatch trust`

Trust the root CA in the macOS Keychain. Regenerates the CA if it's missing.

```bash
hatch trust
```

### `hatch doctor`

Run diagnostic health checks. Reports on config validity, DNS resolver, CA certificates, launchd plist, port availability, and stale projects.

```bash
hatch doctor
```

### `hatch clean`

Complete uninstall. Stops the daemon, removes the DNS resolver, untrusts the root CA, and deletes `~/.hatch` and Caddy data directories. Alias: `hatch uninstall`.

```bash
hatch clean
```

## Config

### `hatch config`

Open the config file in your `$EDITOR` (falls back to `$VISUAL`, then `vi`).

```bash
hatch config
```

### `hatch config validate`

Validate the config file without opening it.

```bash
hatch config validate
```

## Other

### `hatch version`

Print version, commit hash, and build date.

```bash
hatch version
```

### `hatch completion <shell>`

Generate shell completion scripts.

```bash
hatch completion bash
hatch completion zsh
hatch completion fish
```

### `hatch app`

Launch the GUI dashboard.

```bash
hatch app
```

## Global Flags

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable debug logging |
