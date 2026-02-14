# CLAUDE.md — nps-api

## Project Overview

REST API to collect NPS (Net Promoter Score) feedback from the Idefinity desktop
application. Deployed as a Docker container at `api.ruohomaki.fi/nps` behind an
Nginx reverse proxy.

Infrastructure context lives in the `backend01` repository (Nginx configs, SSL,
runbooks, deployment scripts).

## Architecture

- **Language:** Go 1.24+
- **Router:** Standard library `net/http` (Go 1.22+ routing patterns)
- **Database:** MongoDB Atlas (cloud) via `go.mongodb.org/mongo-driver/v2`
- **Monitoring:** Sentry (`github.com/getsentry/sentry-go`) — optional, enabled via SENTRY_DSN
- **Deployment:** Docker container on port 8081, reverse-proxied by Nginx

## Project Structure

```
nps-api/
├── cmd/server/main.go            # Entry point — wires config, Sentry, DB, server
├── internal/
│   ├── config/config.go          # Environment-based configuration
│   ├── config/config_test.go
│   ├── db/mongo.go               # MongoDB connection and lifecycle
│   ├── handler/
│   │   ├── routes.go             # Route registration under /nps prefix
│   │   ├── feedback.go           # POST /nps/api/v1/feedback
│   │   ├── health.go             # GET /nps/health + JSON helpers
│   │   └── handler_test.go       # Unit tests
│   ├── middleware/logging.go     # Request logging (method, path, status, duration)
│   └── model/
│       ├── feedback.go           # Data model and validation
│       └── feedback_test.go
├── test/integration/             # Integration tests (require MongoDB)
├── docs/feedback-v1.json         # JSON schema
├── Dockerfile                    # Multi-stage: golang:1.24-alpine → alpine:3.21
├── docker-compose.yml            # Local dev (builds locally, binds 127.0.0.1:8081)
├── docker-compose.prod.yml       # Production (pulls from ghcr.io)
├── .github/workflows/
│   ├── ci.yml                    # Test + build on push/PR to main
│   └── cd.yml                    # Build image → push to ghcr.io → deploy via SSH
└── .env.example                  # Template for required environment variables
```

## Conventions

- Files should not exceed ~100 lines of code
- Production-grade error handling (no silent failures)
- Graceful shutdown on SIGINT/SIGTERM with 10-second drain timeout
- Configuration via environment variables, never hardcoded
- Dates and times in ISO 8601 / RFC 3339 format
- Structured log output with `slog`
- Container ports bind to 127.0.0.1 (Docker bypasses UFW on 0.0.0.0)
- Use `docker compose` (v2, space not hyphen)
- Sentry is optional — runs fine without SENTRY_DSN set
- All routes use `/nps` prefix (Nginx proxies `api.ruohomaki.fi/nps/` to this container)

## Build & Run

### Local development (without Docker)

```bash
cp .env.example .env   # Fill in MONGODB_URI
go run ./cmd/server
```

Note: Go does not read `.env` files. Export variables manually or use Docker Compose.

### Docker

```bash
docker compose up --build
```

### Test

```bash
go test ./...                                                          # Unit tests
MONGODB_URI="mongodb://localhost:27017" go test ./test/integration/ -v  # Integration
```

## Environment Variables

| Variable            | Default       | Description                    |
|---------------------|---------------|--------------------------------|
| PORT                | 8081          | HTTP listen port               |
| MONGODB_URI         | (empty)       | MongoDB connection string      |
| MONGODB_DATABASE    | nps           | MongoDB database name          |
| SENTRY_DSN          | (empty)       | Sentry DSN — empty = disabled  |
| SENTRY_ENVIRONMENT  | development   | Sentry environment tag         |

## API Endpoints

All endpoints are prefixed with `/nps`:

| Method | Path                      | Description                   |
|--------|---------------------------|-------------------------------|
| GET    | /nps/health               | Health check + timestamp      |
| POST   | /nps/api/v1/feedback      | Submit NPS feedback           |

See `docs/feedback-v1.json` for the feedback payload schema.

## CI/CD Pipeline

**CI** (`.github/workflows/ci.yml`) — runs on every push and PR to `main`:
- Downloads Go dependencies
- Runs `go test -v -race ./...`
- Verifies the binary compiles

**CD** (`.github/workflows/cd.yml`) — runs on push to `main` only:
- Builds Docker image
- Pushes to `ghcr.io/timoruohomaki/nps-api` (tagged with commit SHA + `latest`)
- SSHs into server as `deploy`, pulls the new image, restarts the container

**Required GitHub Secrets:** `SERVER_HOST`, `SERVER_USER`, `SERVER_SSH_KEY`, `SERVER_PORT`
**Required GitHub Environment:** `production`

## Server-Side Setup

On the server, the deploy directory is `~/nps-api/`. Copy `docker-compose.prod.yml`
as `docker-compose.yml` and create a `.env` file with the MongoDB connection string:

```bash
mkdir -p ~/nps-api
# Copy docker-compose.prod.yml as docker-compose.yml
# Create .env with MONGODB_URI and SENTRY_DSN
```

## Related Repositories

- **backend01** — Server infrastructure: Nginx configs (including the
  `api.ruohomaki.fi` config with `/nps/` location block), SSL snippets,
  deployment runbooks, static sites.
- **docker-api-demo** — Demo API at `api.ruohomaki.fi/` on port 8080.
