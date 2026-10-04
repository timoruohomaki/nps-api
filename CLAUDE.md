# CLAUDE.md — nps-api

## Project Overview

REST API to collect NPS (Net Promoter Score) feedback from the `rokdsk` and
`idefinity` desktop applications. Deployed as a Docker container at
`api.ruohomaki.fi/nps` behind an Nginx reverse proxy.

Infrastructure context lives in the `backend01` repository (Nginx configs, SSL,
runbooks, deployment scripts). Server-side deploy: `backend01/docs/runbook-deploy-nps.md`.

## Architecture

- **Language:** Go 1.24+
- **Router:** Standard library `net/http` (Go 1.22+ routing patterns)
- **Database:** Local SQLite via `modernc.org/sqlite` (pure-Go, no CGO). One
  `feedback` table; file persisted in a Docker volume at `/data/nps.db`. No
  external database or network dependency.
- **PII encryption:** the `comment` and `timezone` columns are encrypted at rest
  with AES-256-GCM (`internal/crypto`, key from `FEEDBACK_ENC_KEY`) before being
  written. Metadata columns stay plaintext/queryable. Decrypt via `cmd/export`.
- **Monitoring:** Sentry (`github.com/getsentry/sentry-go`) — optional, enabled via SENTRY_DSN
- **Deployment:** Docker container on port 8081, reverse-proxied by Nginx

## Project Structure

```
nps-api/
├── cmd/server/main.go            # Entry point — wires config, Sentry, crypto, DB, server
├── cmd/export/main.go            # Exports decrypted feedback as JSON (no read HTTP endpoint)
├── internal/
│   ├── config/config.go          # Environment-based configuration
│   ├── config/config_test.go
│   ├── crypto/crypto.go          # AES-256-GCM field encryption for PII columns
│   ├── crypto/crypto_test.go
│   ├── db/sqlite.go              # SQLite connection, schema, inserts, decrypting reads
│   ├── handler/
│   │   ├── routes.go             # Route registration under /nps prefix
│   │   ├── feedback.go           # POST (submit) + GET (analytics query) feedback
│   │   ├── health.go             # GET /nps/health + JSON helpers
│   │   └── handler_test.go       # Unit tests
│   ├── middleware/logging.go     # Request logging (method, path, status, duration)
│   ├── middleware/auth.go        # X-API-Key check for the feedback endpoint
│   └── model/
│       ├── feedback.go           # Data model and validation
│       └── feedback_test.go
├── test/integration/             # End-to-end tests (embedded SQLite, no external deps)
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
go run ./cmd/server     # creates ./nps.db in the working directory
```

Note: Go does not read `.env` files. Export variables manually or use Docker Compose.

### Docker

```bash
docker compose up --build
```

### Test

```bash
go test ./...   # unit + integration (integration uses a temp SQLite DB, no external deps)
```

## Environment Variables

| Variable            | Default       | Description                            |
|---------------------|---------------|----------------------------------------|
| PORT                | 8081          | HTTP listen port                       |
| DB_PATH             | nps.db        | SQLite file path (container: /data/nps.db) |
| FEEDBACK_ENC_KEY    | (empty)       | base64 32-byte AES-256 key; encrypts comment/timezone at rest. Empty = unencrypted + warning; malformed = fail to start |
| API_KEYS            | (empty)       | Comma-separated accepted X-API-Key values for POST. Empty = open |
| READ_API_KEYS       | (empty)       | Comma-separated consumer keys for GET analytics query. Empty = read endpoint disabled (503) |
| ALLOWED_PLATFORMS   | macOS,Windows | Comma-separated allowlist for the `platform` field (set via model.SetAllowedPlatforms at startup) |
| SENTRY_DSN          | (empty)       | Sentry DSN — empty = disabled          |
| SENTRY_ENVIRONMENT  | development   | Sentry environment tag                 |

## API Endpoints

All endpoints are prefixed with `/nps`:

| Method | Path                           | Description                        | Auth                          |
|--------|--------------------------------|------------------------------------|-------------------------------|
| GET    | /nps/health                    | Health check + timestamp           | open                          |
| POST   | /nps/api/v1/feedback           | Submit NPS feedback                | X-API-Key if API_KEYS set     |
| GET    | /nps/api/v1/feedback[?year=]   | Query feedback (decrypted) for analytics | consumer X-API-Key; 503 if READ_API_KEYS unset |

See `docs/feedback-v1.json` for the feedback payload schema.

## CI/CD Pipeline

**CI** (`.github/workflows/ci.yml`) — runs on every push and PR to `main`:
- Downloads Go dependencies
- Runs `go test -v -race ./...`
- Verifies the binary compiles

**CD** (`.github/workflows/cd.yml`) — runs on push to `main` only:
- Builds Docker image, pushes to `ghcr.io/timoruohomaki/nps-api` (SHA + `latest`)
- **Syncs `docker-compose.prod.yml` → `~/nps-api/docker-compose.yml`** (repo is the
  source of truth for the compose; the server-side `.env` is never touched)
- SSHs into server as `deploy`, pulls the new image, recreates the container

**Required GitHub Secrets:** `SERVER_HOST`, `SERVER_USER`, `SERVER_SSH_KEY`, `SERVER_PORT`
**Required GitHub Environment:** `production`

## Server-Side Setup

CD creates `~/nps-api/` and keeps `docker-compose.yml` in sync from the repo, so
the only manual, one-time step is the secrets file `~/nps-api/.env` (compose reads
it for `${...}` values). The SQLite database lives in the `nps-data` Docker volume
(`/data/nps.db`) and survives `docker compose down`.

```bash
mkdir -p ~/nps-api
echo "FEEDBACK_ENC_KEY=$(openssl rand -base64 32)" > ~/nps-api/.env
chmod 600 ~/nps-api/.env
# back up that key somewhere safe — losing it makes comments unrecoverable
# optionally also add API_KEYS / READ_API_KEYS / ALLOWED_PLATFORMS
```

Full procedure, backup, and troubleshooting: `backend01/docs/runbook-deploy-nps.md`.

## Related Repositories

- **backend01** — Server infrastructure: Nginx configs (including the
  `api.ruohomaki.fi` config with `/nps/` location block), SSL snippets,
  deployment runbooks, static sites.
- **docker-api-demo** — Demo API at `api.ruohomaki.fi/` on port 8080.
