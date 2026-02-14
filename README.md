# nps-api

REST API to collect NPS (Net Promoter Score) feedback from the Idefinity desktop application and store it in MongoDB Atlas.

## Prerequisites

- Go 1.23+
- MongoDB Atlas cluster (or local MongoDB for development)
- (Optional) Sentry account for error monitoring

## Quick Start

```bash
# Clone and configure
git clone https://github.com/idefinity/nps-api.git
cd nps-api
cp .env.example .env
# Edit .env — set MONGODB_URI at minimum

# Run
go run ./cmd/server
```

The server starts on port `8080` by default.

## Configuration

All configuration is via environment variables. See `.env.example` for reference.

| Variable | Required | Default | Description |
|---|---|---|---|
| `MONGODB_URI` | Yes | — | MongoDB connection string |
| `MONGODB_DATABASE` | No | `nps` | Database name |
| `PORT` | No | `8080` | HTTP server port |
| `SENTRY_DSN` | No | — | Sentry DSN for error tracking |
| `SENTRY_ENVIRONMENT` | No | `development` | Sentry environment tag |

## API Reference

### Health Check

```
GET /health
```

Returns `200 OK` with `{"status": "healthy"}`.

### Submit NPS Feedback

```
POST /api/v1/feedback
Content-Type: application/json
```

**Request body** (see [`docs/feedback-v1.json`](docs/feedback-v1.json) for full JSON schema):

| Field | Type | Required | Description |
|---|---|---|---|
| `schema_version` | string | Yes | Must be `"1.0"` |
| `app` | string | Yes | Application identifier (e.g. `"idefinity"`) |
| `app_version` | string | Yes | Semantic version (`Major.Minor.Patch`) |
| `platform` | string | Yes | `"macOS"` or `"Windows"` |
| `timestamp` | string | Yes | ISO 8601 datetime |
| `nps_rating` | integer | Yes | 1–10 |
| `nps_category` | string | Yes | `"detractor"`, `"passive"`, or `"promoter"` |
| `timezone` | string | No | IANA timezone or OS timezone identifier |
| `comment` | string | No | Free-text feedback (max 2000 chars) |

**Example request:**

```bash
curl -X POST http://localhost:8080/api/v1/feedback \
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

## Project Structure

```
cmd/server/            Application entry point
internal/
  config/              Environment-based configuration
  db/                  MongoDB connection management
  handler/             HTTP request handlers
  model/               Data models and validation
  middleware/           HTTP middleware (request logging)
test/integration/      Integration tests (require live MongoDB)
docs/                  JSON schemas and API documentation
```

## Development

### Building

```bash
go build -o nps-api ./cmd/server
```

### Testing

```bash
# Unit tests
go test ./...

# Integration tests (requires MongoDB)
MONGODB_URI="mongodb://localhost:27017" go test ./test/integration/ -v

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Docker

```bash
docker build -t nps-api .
docker run -e MONGODB_URI="your-connection-string" -p 8080:8080 nps-api
```

## Monitoring

When `SENTRY_DSN` is set, the API reports errors and panics to Sentry. Performance traces are also captured. View events in the [Sentry dashboard](https://sentry.io).

## License

[Apache License 2.0](LICENSE)
