# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies (required for compiling with CGO/SQLite)
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build API, Worker, and Migration binaries with stripped symbols for minimal size
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api cmd/api/main.go && \
    CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /app/bin/worker cmd/worker/main.go && \
    CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /app/bin/migrate cmd/migrate/main.go

# Runtime stage
FROM alpine:3.20

WORKDIR /app

# Install runtime dependencies: ca-certificates for TLS, tzdata for timezones, and wget for healthchecks
RUN apk add --no-cache ca-certificates tzdata wget && \
    adduser -D -u 10001 appuser && \
    mkdir -p /app/logs && \
    chown -R appuser:appuser /app

# Copy compiled binaries from builder stage
COPY --from=builder --chown=appuser:appuser /app/bin/api /app/api
COPY --from=builder --chown=appuser:appuser /app/bin/worker /app/worker
COPY --from=builder --chown=appuser:appuser /app/bin/migrate /app/migrate

# Run as non-privileged user for CIS container security
USER appuser:appuser

EXPOSE 8080

# Built-in Container Healthcheck
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health/live || exit 1

CMD ["/app/api"]
