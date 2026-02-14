# ----- Build stage -----
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/server

# ----- Runtime stage -----
FROM alpine:3.21

# Add CA certificates for outbound HTTPS and a non-root user
RUN apk --no-cache add ca-certificates \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup

WORKDIR /home/appuser

COPY --from=builder /app .

USER appuser

EXPOSE 8081

ENTRYPOINT ["./app"]
