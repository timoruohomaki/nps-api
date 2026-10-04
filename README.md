# nps-api

REST API to collect NPS (Net Promoter Score) feedback from the `rokdsk` and
`idefinity` desktop applications and store it in a local SQLite database.

Deployed at `api.ruohomaki.fi/nps` as a Docker container behind Nginx.

## Prerequisites

- Go 1.24+
- (Optional) Sentry account for error monitoring

No external database is required — SQLite is embedded via the pure-Go
`modernc.org/sqlite` driver (no CGO), so the service has no network dependency
for storage.

## Quick Start

```bash
# Clone and configure
git clone https://github.com/timoruohomaki/nps-api.git
cd nps-api

# Run locally (creates ./nps.db in the working directory)
go run ./cmd/server

# Or with Docker (database persisted in the nps-data volume)
docker compose up --build
```

The server starts on port `8081` by default. All routes are prefixed with `/nps`.

## Configuration

All configuration is via environment variables. See `.env.example` for reference.

| Variable | Required | Default | Description |
|---|---|---|---|
| `DB_PATH` | No | `nps.db` | SQLite database file path (container sets `/data/nps.db`) |
| `PORT` | No | `8081` | HTTP server port |
| `SENTRY_DSN` | No | — | Sentry DSN for error tracking |
| `SENTRY_ENVIRONMENT` | No | `development` | Sentry environment tag |

## API Reference

### Health Check

```
GET /nps/health
```

Returns `200 OK` with `{"status": "healthy", "timestamp": "..."}`.

### Submit NPS Feedback

```
POST /nps/api/v1/feedback
Content-Type: application/json
```

See [`docs/feedback-v1.json`](docs/feedback-v1.json) for the full JSON schema.

**Example request:**

```bash
curl -X POST https://api.ruohomaki.fi/nps/api/v1/feedback \
  -H "Content-Type: application/json" \
  -d '{
    "schema_version": "1.0",
    "app": "idefinity",
    "app_version": "0.1.0",
    "platform": "macOS",
    "timestamp": "2025-06-15T14:23:00+03:00",
    "nps_rating": 9,
    "nps_category": "promoter",
    "timezone": "Europe/Helsinki",
    "comment": "Great workflow, would like more export formats."
  }'
```

**Responses:**

| Status | Description |
|---|---|
| `201 Created` | Feedback stored successfully |
| `400 Bad Request` | Invalid JSON |
| `422 Unprocessable Entity` | Validation error (details in response body) |

## Development

```bash
# All tests (unit + integration). Integration tests use a temporary SQLite
# database — no external service required.
go test ./...
```

## CI/CD

Push to `main` triggers automated testing, Docker image build, push to `ghcr.io/timoruohomaki/nps-api`, and deployment to the server.

Server-side deployment is documented in the **backend01** repo at
`docs/runbook-deploy-nps.md`.

## License

[Apache License 2.0](LICENSE)
