# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

JWTea is a Go-based OAuth2/OIDC Authorization Server for testing and development. It combines a fully functional OAuth2/OIDC server with an interactive Terminal UI (TUI) dashboard built with Bubble Tea.

## Common Commands

```bash
make build          # Compile binary to ./jwtea
make run            # Run with dev.yaml config (default)
make test           # Run all tests
make lint           # Run go vet and gofmt checks
make demo-flow      # Open browser to test OAuth authorization flow
```

Run with custom settings:
```bash
make run CONFIG=custom.yaml HOST=0.0.0.0 PORT=9000
```

Run a single test:
```bash
go test -v ./internal/service -run TestUserService_Create
```

## Architecture

```
CLI (Cobra) + TUI Dashboard (Bubble Tea)
              │
    HTTP Server (net/http)
    ├─ /authorize           - OAuth2 authorization endpoint
    ├─ /oauth2/token        - Token endpoint (returns JWT)
    ├─ /.well-known/openid-configuration
    ├─ /jwks.json           - Public keys for JWT validation
    └─ /callback            - Built-in callback UI
              │
    Service Layer (internal/service/)
              │
    Memory Store (internal/store/)
    └─ Thread-safe maps for clients, users, auth codes
```

**Key Directories:**
- `cmd/` - CLI commands (root.go, serve.go, dashboard.go)
- `internal/store/` - In-memory data store with mutex locking
- `internal/token/` - RS256 JWT generation
- `internal/tui/` - TUI components and tabs (generate, users, clients, logs, settings)
- `internal/config/` - YAML config loading with env var overrides (`JWTEA_*` prefix)
- `internal/chaos/` - Feature flags to inject failures for testing (expired tokens, invalid signatures)

**OAuth Flow:** Authorization Code flow with RS256-signed JWTs. Keys generated fresh on startup (no persistence). All state is in-memory.

## Configuration

See `config.example.yaml` for all options. Environment variables override YAML values with `JWTEA_` prefix (e.g., `JWTEA_SERVER_PORT=9000`).

## Code Style

Only add comments for the most complex logic. Code should be self-documenting.