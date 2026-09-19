# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies (required if compiling with CGO/SQLite)
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build binary
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api cmd/api/main.go

# Runtime stage
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/bin/api /app/api
COPY .env.example /app/.env

EXPOSE 8080

CMD ["/app/api"]
