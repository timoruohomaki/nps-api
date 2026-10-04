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
| `FEEDBACK_ENC_KEY` | Prod | — | base64 32-byte AES-256 key; encrypts `comment`/`timezone` at rest. Empty = unencrypted (dev only) |
| `API_KEYS` | No | — | Comma-separated accepted `X-API-Key` values for POSTing feedback. Empty = open |
| `READ_API_KEYS` | Analytics | — | Comma-separated consumer keys for the GET query (returns decrypted PII). Empty = read endpoint disabled (503) |
| `PORT` | No | `8081` | HTTP server port |
| `SENTRY_DSN` | No | — | Sentry DSN for error tracking |
| `SENTRY_ENVIRONMENT` | No | `development` | Sentry environment tag |

## Data protection

Feedback can contain personal information in the free-text `comment` (and coarse
location in `timezone`). Those two fields are encrypted at rest with AES-256-GCM
before being written to SQLite; the NPS metadata (rating, category, app, version,
platform, timestamps) stays in plaintext so it remains queryable.

- Set `FEEDBACK_ENC_KEY` to a base64-encoded 32-byte key:
  `openssl rand -base64 32`.
- If the key is unset the service runs but stores those fields **unencrypted**
  and logs a warning. A malformed key makes the service **fail to start** (it
  never silently falls back to plaintext).
- **Losing the key makes existing encrypted comments unrecoverable** — back it up
  separately from the database.

This protects leaked backups, stolen database files, and volume snapshots. It is
not a defense against an attacker with full access to the running host (which also
holds the key) — the same limitation as whole-file schemes like SQLCipher.

### Reading feedback back out

There is no read HTTP endpoint. Use the bundled `export` tool, which decrypts and
prints all feedback as JSON (run it with the same `DB_PATH`/`FEEDBACK_ENC_KEY`):

```bash
docker compose exec nps-api ./export > feedback.json
```

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
X-API-Key: <key>          # required only when API_KEYS is configured
```

See [`docs/feedback-v1.json`](docs/feedback-v1.json) for the full JSON schema.

When `API_KEYS` is set, requests to this endpoint must send a matching
`X-API-Key` header or receive `401`. The health check is always open. With
`API_KEYS` empty the endpoint accepts requests without a key (backward
compatible) and logs a warning at startup.

### Query Feedback (analytics)

```
GET /nps/api/v1/feedback?year=2026
X-API-Key: <consumer key>
```

Returns a JSON array of feedback with the PII fields **decrypted**, so it is
gated by a separate consumer key set (`READ_API_KEYS`) and is **fail-closed**:
with `READ_API_KEYS` empty the endpoint returns `503`. The optional `year`
filters by receipt year (`received_at`, UTC). Used by the R analytics project in
backend01's `analytics/`.

| Status | Description |
|---|---|
| `200 OK` | JSON array of feedback (possibly empty `[]`) |
| `400 Bad Request` | Invalid `year` |
| `401 Unauthorized` | Missing/invalid consumer `X-API-Key` |
| `503 Service Unavailable` | `READ_API_KEYS` not configured (endpoint disabled) |

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
| `401 Unauthorized` | Missing/invalid `X-API-Key` (only when `API_KEYS` is set) |
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
