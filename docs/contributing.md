# Contributing

## Development Setup

### Prerequisites

- **Go 1.25+**
- **Node.js** (for the frontend dashboard)
- **Wails v3** (for building the desktop app)

### Clone and Build

```bash
git clone https://github.com/httphatch/hatch.git
cd hatch
make build
```

### Build Targets

| Command | Description |
|---------|-------------|
| `make build` | Build binary to `./hatch` |
| `make build-test` | Build binary to `./testing/hatch` |
| `make app` | Build with embedded frontend (Wails) |
| `make frontend` | Build React frontend only |
| `make test` | Run `go test ./...` |
| `make clean` | Remove built binary |

### Running Tests

```bash
go test ./...       # run all tests
go vet ./...        # static analysis
```

Update golden files for Caddy config translation tests:

```bash
UPDATE_GOLDEN=1 go test ./internal/caddy/...
```

## Project Structure

```
hatch/
├── cmd/                    # CLI commands (Cobra)
├── internal/
│   ├── api/               # HTTP API server
│   ├── app/               # Wails app service
│   ├── caddy/             # Caddy config translation
│   ├── certs/             # CA generation & trust
│   ├── config/            # Config loading, validation, watching
│   ├── daemon/            # Daemon lifecycle, launchd, PID
│   ├── dns/               # DNS server, resolver files
│   ├── health/            # Service health checking
│   ├── logging/           # Structured logging
│   ├── proxy/             # Proxy utilities
│   └── tray/              # macOS tray integration
├── frontend/              # React dashboard (TypeScript, Vite, Tailwind)
├── main.go                # Entry point
├── Makefile               # Build targets
└── go.mod                 # Go module
```

## Code Conventions

- **Error wrapping:** `fmt.Errorf("doing thing: %w", err)`
- **File permissions:** certs `0o644`, keys `0o600`, directories `0o755`
- **Tests:** table-driven where appropriate, `t.Fatalf` for setup failures, `t.Errorf` for assertions
- **Style:** no unnecessary comments, docstrings, or type annotations on unchanged code

## Pull Request Workflow

1. Create a feature branch from `main`
2. Make your changes
3. Run tests: `go test ./...`
4. Run static analysis: `go vet ./...`
5. Open a pull request against `main`
