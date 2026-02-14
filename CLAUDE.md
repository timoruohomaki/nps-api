# NPS API

REST API to collect NPS (Net Promoter Score) feedback from the Idefinity desktop application.

## Tech Stack
- **Language**: Go 1.23+
- **Router**: net/http (standard library, Go 1.22+ routing patterns)
- **Database**: MongoDB Atlas (cloud) via `go.mongodb.org/mongo-driver/v2`
- **Monitoring**: Sentry (`github.com/getsentry/sentry-go`)

## Project Structure
```
cmd/server/          Entry point (main.go)
internal/config/     Environment-based configuration
internal/db/         MongoDB connection management
internal/handler/    HTTP handlers
internal/model/      Data models and validation
internal/middleware/  HTTP middleware (logging)
test/integration/    Integration tests (require MongoDB)
docs/                JSON schemas
```

## Running
```bash
cp .env.example .env   # Fill in MONGODB_URI
go run ./cmd/server
```

## Testing
```bash
go test ./...                                              # Unit tests
MONGODB_URI="mongodb://localhost:27017" go test ./test/integration/ -v  # Integration tests
```

## Building
```bash
go build -o nps-api ./cmd/server
docker build -t nps-api .
```

## API Endpoints
- `GET /health` — Health check
- `POST /api/v1/feedback` — Submit NPS feedback (see `docs/feedback-v1.json` for schema)

## Conventions
- Use `slog` for structured logging
- Validate input in model layer
- Unit tests alongside source files (`*_test.go`)
- Integration tests in `test/integration/`
- Configuration via environment variables (no config files)
