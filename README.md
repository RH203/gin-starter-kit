# Gin Go Clean Architecture Starter Pack

A production-ready REST API starter pack built with **Go (Golang)** and the **Gin Web Framework**, architected according to **Clean Architecture (Uncle Bob)** principles. Modular, loosely coupled, and flexible: supports dynamic multi-database switching (PostgreSQL, MySQL, SQLite), optional Redis caching, JWT authentication, daily rotating logs, rate limiting, Swagger UI, pluggable background queues (database, redis, memory, sync), and dedicated workers.

---

## Table of Contents

1. [Key Features](#key-features)
2. [Clean Architecture Principles](#clean-architecture-principles)
3. [Project Directory Structure](#project-directory-structure)
4. [Database Configuration (Ultra Flexible)](#database-configuration-ultra-flexible)
5. [Redis Configuration](#redis-configuration)
6. [Pluggable Structured Logger (slog)](#pluggable-structured-logger-slog)
7. [Unified Background Queue Subsystem (Laravel-Style)](#unified-background-queue-subsystem-laravel-style)
8. [Email Service (pkg/mail)](#email-service-pkgmail)
9. [Rate Limiting, CORS & Security Middlewares](#rate-limiting-cors--security-middlewares)
10. [Observability & Kubernetes Health Probes](#observability--kubernetes-health-probes)
11. [Swagger API Documentation](#swagger-api-documentation)
12. [Database Migrations & Safety Guard](#database-migrations--safety-guard)
13. [Database Seeder & Faker (gofakeit)](#database-seeder--faker-gofakeit)
14. [Environment Variables Reference](#environment-variables-reference)
15. [Getting Started](#getting-started)
16. [Step-by-Step Guide: Adding a New Feature](#step-by-step-guide-adding-a-new-feature)
17. [API Endpoints & Request Examples](#api-endpoints--request-examples)
18. [Testing & Mocking](#testing--mocking)
19. [Deployment & Docker Hardening](#deployment--docker-hardening)

---

## Key Features

- **Clean Architecture**: Strict Dependency Rule (`Delivery` -> `Usecase` -> `Repository` -> `Domain`). The Domain layer is pure Go with zero external framework dependencies.
- **Viper Configuration**: Unified configuration loading from `.env`, system environment variables, and defaults via `spf13/viper`.
- **Multi-Database Support**: Switch between `postgres`, `mysql`, and `sqlite` simply by modifying `DB_DRIVER` or providing a direct `DB_DSN` without altering application code.
- **Driver Registry Pattern**: Easily install and plug in third-party GORM drivers (e.g. SQL Server, ClickHouse) in just a few lines.
- **Optional Redis Cache**: Redis can be toggled on/off (`REDIS_ENABLED=true/false`). When disabled, queries safely bypass the cache with zero downtime or panics.
- **JWT Authentication**: Built-in HMAC-SHA256 JWT generation, validation, and Gin auth middleware (`Authorization: Bearer <token>`), fully optimized for Mobile Apps (Flutter, React Native, iOS, Android) and Web Single Page Applications.
- **Dedicated Password Hashing**: Standalone `pkg/hash` module powered by Bcrypt.
- **Pluggable Structured Logger (slog)**: Powered by Go standard `log/slog` and `lumberjack`. Supports pluggable drivers (`stdout`, `file`, `stack`, `discard`) and formats (`json`, `text`) with automatic daily gzip log file rotation and extensible 3rd-party driver registration.
- **Unified Background Queue (Laravel-Style)**: Modular queue engine in `pkg/queue` supporting `database` (SQL table `jobs`, survives server crashes with zero extra infrastructure), `redis` (distributed high-throughput), `memory` (fast local dev), and `sync` (unit tests). Features atomic locking and exponential retry backoff.
- **Dedicated Worker Service**: Background jobs can run embedded in the API server or as a standalone dedicated worker container (`cmd/worker/main.go` / `make worker`) for independent horizontal scaling.
- **Enterprise Security & Tracing**: OWASP-recommended HTTP security headers, unique `X-Request-ID` generation & propagation, and correlated request logs in `slog`.
- **Kubernetes-Ready Health Probes**: Endpoints for `/health` (system diagnostics), `/health/live` (liveness probe), and `/health/ready` (readiness probe returning `503 Service Unavailable` if database is down).
- **Production Migration Guard**: Configurable `DB_AUTO_MIGRATE=false` toggle prevents DDL race conditions and lock contention in multi-replica deployments.
- **IP-Based Rate Limiting**: Token-bucket algorithm per client IP (`golang.org/x/time/rate`), returning `429 Too Many Requests` when limits are exceeded.
- **Configurable CORS**: Dynamic allowed origins, HTTP methods, and headers configurable via `.env`.
- **Interactive Swagger / OpenAPI Docs**: Auto-generated API documentation served at `/swagger/index.html`.
- **Email Service**: Standalone `pkg/mail` package supporting `smtp` (TLS/SSL) and `log` (console logging for dev) drivers.
- **Database Seeder & Faker**: Modular seeder registry (`database/seeder`) powered by `gofakeit/v6` to seed admin accounts and bulk realistic dummy data (`make seed`).
- **DevOps & Container Hardening**: Multi-stage `Dockerfile` running as non-root `appuser:appuser`, built-in container `HEALTHCHECK`, multi-service `docker-compose.yml`, and `Makefile` shortcuts.

---

## Clean Architecture Principles

The application flow strictly adheres to the Clean Architecture data-flow diagram:

```
[ HTTP Client ]
      │
      ▼
┌────────────────────────────────────────────────────────┐
│ Delivery Layer (Gin)                                   │
│ - router.go, user_handler.go, health_handler.go        │
│ - Middlewares: slog logger, CORS, rate limiter, auth   │
│ - Request DTO validation & HTTP status mapping         │
└───────────────────────┬────────────────────────────────┘
                        │
                        ▼
┌────────────────────────────────────────────────────────┐
│ Usecase Layer (Business Logic)                         │
│ - user_usecase.go                                      │
│ - Business rules, password hashing, cache orchestration│
│ - Async job dispatching                                │
└───────────────┬────────────────────────┬───────────────┘
                │                        │
                ▼                        ▼
┌──────────────────────────────┐ ┌──────────────────────┐
│ Repository Layer (Database)  │ │ Cache Layer (Redis)  │
│ - gorm/user_repository.go    │ │ - pkg/redis/redis.go │
│ (PostgreSQL, MySQL, SQLite)  │ │ (Optional toggle)    │
└───────────────┬──────────────┘ └──────────────────────┘
                │
                ▼
┌────────────────────────────────────────────────────────┐
│ Domain Layer (Core Entities & Contracts)               │
│ - domain/user.go, domain/errors.go                     │
│ - Pure Go structs, UserRepository & UserUsecase bounds │
└────────────────────────────────────────────────────────┘
```

### Dependency Rules:
1. **Domain**: Pure business entities and interface contracts. Must NOT import Gin, GORM, or Redis.
2. **Usecase**: Implements core business logic. Relies solely on domain interfaces.
3. **Repository**: Implements database access contracts using concrete libraries (GORM).
4. **Delivery**: Handles incoming HTTP requests, invokes usecase methods, and responds with standardized JSON.

---

## Project Directory Structure

```
gin-starter-pack/
├── cmd/
│   ├── api/
│   │   └── main.go                 # Application entrypoint, DI wiring, graceful shutdown
│   ├── worker/
│   │   └── main.go                 # Dedicated standalone Queue Worker runner
│   ├── migrate/
│   │   └── main.go                 # Database migration CLI runner (GORM AutoMigrate)
│   └── seed/
│       └── main.go                 # Database seeder CLI runner (admin & faker data)
├── config/
│   └── config.go                   # Viper configuration parser & environment binder
├── database/
│   └── seeder/
│       ├── seeder.go               # Seeder registry and runner contracts
│       ├── seeder_test.go          # Seeder unit tests and idempotency checks
│       └── user_seeder.go          # UserSeeder with gofakeit fake user generator
├── docs/                           # Auto-generated Swagger / OpenAPI spec files
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── internal/
│   ├── domain/                     # Core domain entities & interface contracts
│   │   ├── user.go                 # User entity, DTOs, and interface definitions
│   │   ├── job.go                  # JobRecord entity and Queue job payloads
│   │   └── errors.go               # Standard domain error variables
│   ├── repository/                 # Data access layer
│   │   └── gorm/
│   │       └── user_repository.go  # GORM multi-driver repository implementation
│   ├── usecase/                    # Business logic layer
│   │   ├── user_usecase.go         # User business logic implementation & queue handlers
│   │   └── user_usecase_test.go    # Unit tests with in-memory mock repository
│   └── delivery/
│       └── http/                   # Transport layer (Gin HTTP)
│           ├── handler/
│           │   ├── user_handler.go   # CRUD & authentication handlers
│           │   ├── health_handler.go # Diagnostic, liveness, & readiness handlers
│           │   └── health_handler_test.go
│           ├── middleware/
│           │   ├── auth.go         # JWT Bearer token authentication middleware
│           │   ├── cors.go         # Configurable CORS middleware
│           │   ├── logger.go       # slog access logging enriched with Request ID
│           │   ├── ratelimit.go    # IP-based token bucket rate limiter
│           │   ├── recovery.go     # Panic recovery middleware
│           │   ├── request_id.go   # X-Request-ID propagation middleware
│           │   └── security.go     # OWASP security headers middleware
│           └── router.go           # Gin engine setup & route group definitions
├── pkg/
│   ├── database/
│   │   └── database.go             # GORM connection factory & Driver Registry
│   ├── hash/
│   │   ├── hash.go                 # Bcrypt password hashing helper functions
│   │   └── hash_test.go            # Unit tests for password hashing
│   ├── jwt/
│   │   └── jwt.go                  # HMAC-SHA256 JWT token generation & verification
│   ├── logger/
│   │   ├── logger.go               # Pluggable slog logger (stdout, file, stack, discard)
│   │   └── logger_test.go          # Logger driver and format unit tests
│   ├── mail/
│   │   ├── mail.go                 # SMTP & Log mailer implementations
│   │   └── mail_test.go            # Unit tests for mail service
│   ├── queue/
│   │   ├── queue.go                # Unified Queue & Worker contracts & Handler Registry
│   │   ├── driver_database.go      # Laravel-style persistent DB queue with retries
│   │   ├── driver_redis.go         # Distributed Redis queue driver
│   │   ├── driver_memory.go        # In-memory buffered channel queue driver
│   │   ├── driver_sync.go          # Synchronous test execution driver
│   │   ├── factory.go              # Pluggable queue factory and driver registry
│   │   └── queue_test.go           # Comprehensive queue test suite
│   ├── redis/
│   │   └── redis.go                # Redis client wrapper with graceful fallback
│   ├── response/
│   │   └── response.go             # Standardized JSON API response helpers
│   └── worker/
│       ├── worker.go               # Legacy worker pool implementation
│       └── worker_test.go          # Unit tests for worker pool
├── templates/
│   ├── emails/
│   │   ├── reset_password.html     # Responsive password reset HTML email
│   │   └── welcome.html            # Responsive onboarding welcome HTML email
│   └── templates.go                # Go embed.FS embedding HTML templates
├── .air.toml                       # Air configuration for hot reload
├── .env.example                    # Environment variable template
├── .env                            # Active environment configuration (git-ignored)
├── Dockerfile                      # Hardened multi-stage non-root Alpine Docker build
├── docker-compose.yml              # Multi-container stack (api, worker, postgres, redis)
├── Makefile                        # Shortcuts for development commands
├── go.mod
├── go.sum
└── README.md
```

---

## Database Configuration (Ultra Flexible)

The database driver is controlled in `.env` via `DB_DRIVER`.

### 1. SQLite (Default - Zero External Dependency)
Fastest way to get started without installing any database server:
```env
DB_DRIVER=sqlite
DB_NAME=app.db
```

### 2. PostgreSQL
```env
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=app_db
DB_SSLMODE=disable
```

### 3. MySQL
```env
DB_DRIVER=mysql
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=root
DB_NAME=app_db
```

### 4. Direct DSN (Supabase, Neon, PlanetScale, Cloud SQL, etc.)
If you have a full connection string, set `DB_DSN` in `.env` (overrides individual host/port fields):
```env
DB_DRIVER=postgres
DB_DSN=postgres://user:password@ep-db-123.us-east-2.aws.neon.tech/neondb?sslmode=require
```

### 5. Installing & Using Custom GORM Drivers (e.g. SQL Server, ClickHouse)
This starter pack includes a **Driver Registry** in `pkg/database`. To add any other driver:

1. Install the driver:
   ```bash
   go get gorm.io/driver/sqlserver
   ```
2. Register the driver in `cmd/api/main.go` or an `init()` block:
   ```go
   import (
       "gin-starter-pack/pkg/database"
       "gorm.io/driver/sqlserver"
   )

   func init() {
       database.RegisterDriver("sqlserver", func(cfg *config.DBConfig) (gorm.Dialector, error) {
           return sqlserver.Open(cfg.DSN), nil
       })
   }
   ```
3. Update `.env`:
   ```env
   DB_DRIVER=sqlserver
   DB_DSN=sqlserver://username:password@localhost:1433?database=app_db
   ```

All drivers are automatically managed with connection pooling:
- `DB_MAX_OPEN_CONNS=25`
- `DB_MAX_IDLE_CONNS=10`
- `DB_CONN_MAX_LIFETIME=15` (minutes)

---

## Redis Configuration

Redis is completely optional. Ideal for development without local Redis or testing environments:

```env
# Disable Redis (Bypass mode: queries hit DB directly, no caching)
REDIS_ENABLED=false

# Enable Redis (Cache mode: queries cached automatically)
REDIS_ENABLED=true
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

> **Safety Notice**: When `REDIS_ENABLED=false`, the `pkg/redis` wrapper never panics or returns connection errors. Methods like `Get`, `Set`, and `Del` execute safely as no-ops.

---

## Pluggable Structured Logger (slog)

Structured logging powered by Go's standard library `log/slog` coupled with `lumberjack.Logger` in `pkg/logger`. Inspired by Laravel's logging channels, it supports pluggable drivers, customizable serialization formats, and 3rd-party driver registration.

### Built-in Drivers:
- **`stack`** (Default): Simultaneously writes structured logs to both **stdout (terminal)** and rotating files in `logs/app.log`.
- **`stdout`**: Outputs solely to standard output (recommended for Docker / Kubernetes containers where logs are ingested by collectors like Fluentd, Vector, or Promtail).
- **`file`**: Writes solely to rotating disk files with size/age retention and automatic `.gz` compression.
- **`discard`**: Silent mode (no-op writer) ideal for high-speed benchmark runs and unit testing.

### Supported Formats:
- **`json`** (Default): Produces standardized JSON lines (`{"time":"...","level":"INFO","msg":"...","request_id":"..."}`) perfectly formatted for Datadog, ELK, and Grafana Loki.
- **`text`**: Human-readable key-value text format ideal for local console debugging.

Configuration in `.env`:
```env
LOG_DRIVER=stack         # stack, stdout, file, discard
LOG_FORMAT=json          # json, text
LOG_LEVEL=info           # debug, info, warn, error
LOG_DIR=logs             # log directory (used when driver is file/stack)
LOG_FILENAME=app.log     # active log file name
LOG_MAX_SIZE_MB=100      # max size before rotation (MB)
LOG_MAX_BACKUPS=30       # max retained archived files
LOG_MAX_AGE_DAYS=30      # retention period (days)
LOG_COMPRESS=true        # gzip compression for old archives
```

### Registering Custom Drivers:
You can plug in third-party or cloud handlers (e.g. Sentry, Papertrail, CloudWatch) without editing core logger files:
```go
logger.RegisterDriver("custom-cloud", func(cfg *config.LogConfig) (slog.Handler, io.Closer, error) {
    handler := myCustomCloudHandler.New(...)
    return handler, closer, nil
})
```

---

## Unified Background Queue Subsystem (Laravel-Style)

A robust, enterprise-grade task queue in `pkg/queue`. Modeled after Laravel's Queue system, it decouples job dispatching from the storage backend via a pluggable driver interface.

### Supported Queue Drivers:
1. **`database` (Recommended Default for Persistence)**:
   - Persists jobs to an SQL table (`jobs`) managed by GORM.
   - **Zero additional infrastructure required**: Works automatically with PostgreSQL, MySQL, and SQLite.
   - **Crash-proof & Durable**: If the application server or container crashes/restarts, tasks remain safely stored in the database.
   - **Atomic Reservation & Exponential Backoff**: Uses transactional status claiming (`pending` -> `processing`) and automatically reschedules failed jobs with exponential retry delays (e.g. 2s, 4s, 8s...).
2. **`redis`**:
   - High-throughput distributed queue backed by Redis Lists (`LPUSH` / `BRPOP`).
   - Ideal for high-scale microservices processing thousands of background events per second.
3. **`memory`**:
   - Fast, non-blocking in-memory Go channel worker pool (`pkg/queue/driver_memory.go`).
   - Ideal for local development without any database overhead.
4. **`sync`**:
   - Executes jobs immediately in the calling goroutine. Ideal for deterministic unit tests.

### How to Register Handlers and Dispatch Jobs:

1. **Define Job Identifier & Payload** (`internal/domain/job.go`):
   ```go
   const JobWelcomeEmail = "email:welcome"

   type WelcomeEmailPayload struct {
       UserID   string `json:"user_id"`
       Email    string `json:"email"`
       UserName string `json:"user_name"`
   }
   ```

2. **Register the Handler** (`internal/usecase/`):
   ```go
   queueRegistry.Register(domain.JobWelcomeEmail, func(ctx context.Context, payload []byte) error {
       var data domain.WelcomeEmailPayload
       if err := json.Unmarshal(payload, &data); err != nil {
           return err
       }
       return mailer.SendTemplate(ctx, data.Email, "Welcome!", "welcome.html", data)
   })
   ```

3. **Dispatch from Business Logic** (`internal/usecase/`):
   ```go
   // Immediate background execution
   err := queueDispatcher.Dispatch(ctx, domain.JobWelcomeEmail, domain.WelcomeEmailPayload{
       UserID:   user.ID,
       Email:    user.Email,
       UserName: user.Name,
   })

   // Or delayed execution (e.g. process after 10 minutes)
   err := queueDispatcher.DispatchDelayed(ctx, domain.JobWelcomeEmail, payload, 10*time.Minute)
   ```

### Running Dedicated Worker Service (Production):
In production environments, background jobs can be processed on dedicated worker pods/containers so heavy processing does not degrade HTTP API latency:

```bash
# Run standalone worker runner via Makefile:
make worker

# Or directly:
go run cmd/worker/main.go
```

---

## Email Service (pkg/mail)

A dedicated, modular email service that supports both live remote SMTP delivery and local development logging.

### Supported Drivers:
- **`log`** (Default): Writes email contents to `slog` output (terminal and `logs/app.log`). Ideal for local development and integration tests without setting up an actual SMTP server.
- **`smtp`**: Direct delivery to remote SMTP servers (Mailtrap, SendGrid, Amazon SES, Gmail, Mailpit) supporting STARTTLS (`587`), direct SSL (`465`), or standard port (`25`/`2525`).

### HTML Email Templates (`templates/emails/`)
Templates are stored as pure responsive HTML in `templates/emails/` and compiled directly into the binary using Go's `embed.FS` via `templates.EmailFS`.

Default templates included:
- `templates/emails/welcome.html`: Responsive onboarding email with greeting, custom action button, and footer.
- `templates/emails/reset_password.html`: Responsive password reset email with secure token link and expiration notice.

### Sending an Email:
```go
// Direct HTML template sending (renders templates/emails/welcome.html)
err := mailer.SendTemplate(ctx, "recipient@example.com", "Welcome to Starter Pack", "welcome.html", map[string]interface{}{
    "UserName":  "Jane Doe",
    "AppName":   "Starter Pack",
    "ActionURL": "https://example.com/dashboard",
    "Year":      2026,
})

// Or simple text / inline HTML
err := mailer.SendSimple(ctx, "recipient@example.com", "Subject Here", "Hello from Gin Starter Pack", false)
```

---

## Rate Limiting, CORS & Security Middlewares

### 1. IP Rate Limiting (Token Bucket)
Guards against brute-force attacks and abuse:
- Implemented with `golang.org/x/time/rate`.
- Assigns a rate and burst limit per client IP.
- Exceeding the quota returns `429 Too Many Requests`.

```env
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPS=20.0     # Average requests per second per IP
RATE_LIMIT_BURST=40     # Peak burst allowance
```

### 2. Configurable CORS
Configure allowed origins (comma-separated or JSON array), methods, and headers:
```env
CORS_ALLOWED_ORIGINS=["http://localhost:3000","https://example.com"]
CORS_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-Requested-With,Accept
CORS_ALLOW_CREDENTIALS=true
```

### 3. Request ID Correlation Middleware (`X-Request-ID`)
Every incoming HTTP request receives a unique UUID tracking identifier:
- Preserves existing upstream client headers or generates a new UUID v4.
- Injected into the request context and returned in HTTP response header `X-Request-ID`.
- Automatically attached to every `slog` access log entry for end-to-end distributed tracing.

### 4. OWASP Security Headers Middleware
Hardened HTTP response headers protect against common web attacks:
- `X-Content-Type-Options: nosniff` (mitigates MIME-type confusion attacks)
- `X-Frame-Options: DENY` (prevents clickjacking attacks)
- `X-XSS-Protection: 1; mode=block` (legacy cross-site scripting filter)
- `Referrer-Policy: strict-origin-when-cross-origin` (prevents sensitive path leakage)
- `Strict-Transport-Security: max-age=31536000; includeSubDomains` (enforced when `APP_ENV=production`)

---

## Observability & Kubernetes Health Probes

Three diagnostic endpoints ensure smooth operation and zero-downtime rolling updates in Kubernetes, Docker Swarm, and AWS ECS:

| Endpoint | Probe Type | Purpose & Behavior | HTTP Status |
|---|---|---|---|
| **`/health`** | Diagnostic | Returns comprehensive JSON report with database latency, redis connectivity, and system timestamp. | `200 OK` (or `503` if degraded) |
| **`/health/live`** | Liveness Probe | Verifies the web server process is running and responding. Used by orchestrators to restart hung containers. | `200 OK` |
| **`/health/ready`** | Readiness Probe | Verifies the primary database connection is operational. **Returns `503 Service Unavailable` if database is down**, instructing the load balancer to halt routing traffic to the unready pod. | `200 OK` / `503 Service Unavailable` |

---

## Swagger API Documentation

Interactive OpenAPI / Swagger UI is integrated out-of-the-box:
- Access the UI in your browser: **`http://localhost:8080/swagger/index.html`**
- Regenerate Swagger documentation after editing endpoint annotations:
  ```bash
  make swagger
  ```

---

## Database Migrations & Safety Guard

Database schema migration is handled natively by **GORM AutoMigrate**.

### Production Safety Guard (`DB_AUTO_MIGRATE`):
- **Development (`DB_AUTO_MIGRATE=true`)**: Automatically runs `db.AutoMigrate(...)` when the server starts (`make dev` or `make run`).
- **Production (`DB_AUTO_MIGRATE=false`)**: In production deployments with multiple replica pods, auto-migrate should be disabled to prevent database lock contention and race conditions. Migrations should be executed in a CI/CD pipeline or Kubernetes InitContainer using the standalone CLI:
  ```bash
  make migrate
  # Or directly:
  go run cmd/migrate/main.go
  ```
- **Adding Entities**: Simply register new entity structs in `internal/domain/entities.go` via `domain.Entities()`.

---

## Database Seeder & Faker (gofakeit)

The starter pack includes a modular database seeder subsystem in `database/seeder/` integrated with `github.com/brianvoe/gofakeit/v6` to seed initial data and generate bulk realistic dummy records.

### Features:
- **Deterministic Admin User**: Automatically provisions a default administrator account (`admin@example.com` / `password123`) if not already present.
- **Idempotent Execution**: Safe to run repeatedly; skips existing records based on unique constraints to avoid duplicate key errors.
- **Faker Generation**: Generates realistic dummy names and email addresses using `gofakeit/v6`.
- **Extensible Registry Pattern**: Easily add new domain seeders by implementing the `seeder.Seeder` interface.

### Running Seeders:
```bash
# Via Makefile shortcut:
make seed

# Or directly via CLI:
go run cmd/seed/main.go
```

### Adding a Custom Seeder:
Create a new seeder struct implementing `seeder.Seeder`:
```go
package seeder

import (
    "gorm.io/gorm"
)

type ProductSeeder struct{}

func (s *ProductSeeder) Name() string {
    return "ProductSeeder"
}

func (s *ProductSeeder) Seed(db *gorm.DB) error {
    // Implement seeding logic using gofakeit
    return nil
}
```
Register the seeder inside `database/seeder/seeder.go`:
```go
func RunAll(db *gorm.DB) error {
    reg := NewRegistry(
        NewUserSeeder(),
        &ProductSeeder{},
    )
    return reg.Run(db)
}
```

---

## Environment Variables Reference

| Variable | Type | Default | Description |
|---|---|---|---|
| `APP_NAME` | string | `gin-starter-pack` | Application name |
| `APP_PORT` | string | `8080` | HTTP port |
| `APP_ENV` | string | `development` | `development`, `production`, `test` |
| `DB_DRIVER` | string | `sqlite` | `postgres`, `mysql`, or `sqlite` |
| `DB_DSN` | string | `""` | Direct connection string |
| `DB_HOST` | string | `localhost` | Database host (postgres/mysql) |
| `DB_PORT` | string | `5432` | Database port |
| `DB_USER` | string | `postgres` | Database username |
| `DB_PASSWORD` | string | `postgres` | Database password |
| `DB_NAME` | string | `app.db` | Database name / SQLite file |
| `DB_SSLMODE` | string | `disable` | SSL mode (PostgreSQL) |
| `DB_MAX_OPEN_CONNS` | int | `25` | Max connection pool size |
| `DB_MAX_IDLE_CONNS` | int | `10` | Max idle connections |
| `DB_CONN_MAX_LIFETIME` | int | `15` | Connection lifetime (minutes) |
| `DB_AUTO_MIGRATE` | bool | `true` | Run GORM AutoMigrate on boot (`true` for dev, `false` for multi-replica prod) |
| `REDIS_ENABLED` | bool | `false` | Enable/disable Redis (`true`/`false`) |
| `REDIS_HOST` | string | `localhost` | Redis server host |
| `REDIS_PORT` | string | `6379` | Redis server port |
| `REDIS_PASSWORD` | string | `""` | Redis password (optional) |
| `REDIS_DB` | int | `0` | Logical Redis database index |
| `JWT_SECRET` | string | `super-secret-key...` | JWT HMAC signing secret |
| `JWT_EXPIRY_HOURS` | int | `24` | Token expiration time (hours) |
| `MAIL_DRIVER` | string | `log` | Email driver (`log` or `smtp`) |
| `MAIL_HOST` | string | `smtp.mailtrap.io` | SMTP server hostname |
| `MAIL_PORT` | int | `2525` | SMTP port (`587`, `465`, `2525`, `1025`) |
| `MAIL_USERNAME` | string | `""` | SMTP username |
| `MAIL_PASSWORD` | string | `""` | SMTP password |
| `MAIL_FROM_ADDRESS` | string | `noreply@example.com` | Default sender email address |
| `MAIL_FROM_NAME` | string | `Gin Starter Pack` | Default sender name |
| `MAIL_ENCRYPTION` | string | `tls` | Encryption protocol (`tls`, `ssl`, `none`) |
| `LOG_DRIVER` | string | `stack` | Pluggable driver (`stdout`, `file`, `stack`, `discard`) |
| `LOG_FORMAT` | string | `json` | Structured format (`json` for ELK/Datadog/Loki or `text` for dev) |
| `LOG_LEVEL` | string | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `LOG_DIR` | string | `logs` | Directory for log files |
| `LOG_FILENAME` | string | `app.log` | Active log file name |
| `LOG_MAX_SIZE_MB` | int | `100` | Max file size before rotation (MB) |
| `LOG_MAX_BACKUPS` | int | `30` | Old log archives retained |
| `LOG_MAX_AGE_DAYS` | int | `30` | Log retention period (days) |
| `LOG_COMPRESS` | bool | `true` | Gzip compression for old logs |
| `CORS_ALLOWED_ORIGINS` | string / JSON array | `*` | Allowed CORS origins: JSON array `["http://localhost:3000","https://example.com"]` or comma-separated string |
| `CORS_ALLOWED_METHODS` | string | `GET,POST,...` | Allowed HTTP methods |
| `CORS_ALLOWED_HEADERS` | string | `Content-Type,...` | Allowed request headers |
| `CORS_ALLOW_CREDENTIALS` | bool | `true` | Allow credentials / authorization headers across origins |
| `RATE_LIMIT_ENABLED` | bool | `true` | Enable/disable rate limiter |
| `RATE_LIMIT_RPS` | float | `20.0` | Requests allowed per second per IP |
| `RATE_LIMIT_BURST` | int | `40` | Burst request allowance |
| `QUEUE_DRIVER` | string | `database` | Background queue driver: `database` (SQL table `jobs`), `redis`, `memory`, `sync` |
| `QUEUE_NAME` | string | `default` | Active queue partition name |
| `QUEUE_CONCURRENCY` | int | `5` | Number of concurrent queue worker goroutines |
| `QUEUE_MAX_ATTEMPTS` | int | `3` | Max retry attempts before marking task as failed |
| `QUEUE_POLL_INTERVAL_MS`| int | `1000` | Polling frequency for database queue (milliseconds) |
| `WORKER_CONCURRENCY` | int | `5` | Legacy in-memory worker concurrency |
| `WORKER_QUEUE_SIZE` | int | `100` | Legacy in-memory buffer queue size |

---

## Getting Started

### Prerequisites
- Go 1.21+ (built & tested with Go 1.24)

### 1. Run with Hot Reload (Recommended for Dev)
Automatically recompiles and restarts the server whenever `.go` or `.env` files are modified:
```bash
make dev
```
*(If `air` is not installed yet, `make dev` will automatically install it via `go install github.com/air-verse/air@latest`).*

### 2. Standard Local Run
```bash
# Download dependencies
go mod tidy

# Run server
make run
# or
go run cmd/api/main.go
```

### 3. Run with Docker Compose
To run the complete stack with PostgreSQL and Redis:
```bash
# Start containers in background
docker compose up -d

# View application logs
docker compose logs -f app

# Stop containers
docker compose down
```

---

## Step-by-Step Guide: Adding a New Feature

To add a new entity (e.g. `Product`), follow the 4 Clean Architecture layers:

### Step 1: Create Domain Contract (`internal/domain/product.go`)
Define the entity struct and interface contracts:
```go
package domain

import "context"

type Product struct {
    ID    string  `json:"id" gorm:"primaryKey"`
    Name  string  `json:"name" gorm:"not null"`
    Price float64 `json:"price" gorm:"not null"`
}

type ProductRepository interface {
    Create(ctx context.Context, p *Product) error
    FindByID(ctx context.Context, id string) (*Product, error)
}

type ProductUsecase interface {
    CreateProduct(ctx context.Context, name string, price float64) (*Product, error)
    GetProduct(ctx context.Context, id string) (*Product, error)
}
```

### Step 2: Implement Repository (`internal/repository/gorm/product_repository.go`)
Implement `domain.ProductRepository` using GORM:
```go
package gorm

import (
    "context"
    "gin-starter-pack/internal/domain"
    "gorm.io/gorm"
)

type productRepository struct {
    db *gorm.DB
}

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
    return &productRepository{db: db}
}

func (r *productRepository) Create(ctx context.Context, p *domain.Product) error {
    return r.db.WithContext(ctx).Create(p).Error
}

func (r *productRepository) FindByID(ctx context.Context, id string) (*domain.Product, error) {
    var p domain.Product
    if err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
        return nil, err
    }
    return &p, nil
}
```

### Step 3: Implement Usecase (`internal/usecase/product_usecase.go`)
Implement `domain.ProductUsecase`:
```go
package usecase

import (
    "context"
    "gin-starter-pack/internal/domain"
    "github.com/google/uuid"
)

type productUsecase struct {
    repo domain.ProductRepository
}

func NewProductUsecase(repo domain.ProductRepository) domain.ProductUsecase {
    return &productUsecase{repo: repo}
}

func (u *productUsecase) CreateProduct(ctx context.Context, name string, price float64) (*domain.Product, error) {
    p := &domain.Product{
        ID:    uuid.NewString(),
        Name:  name,
        Price: price,
    }
    if err := u.repo.Create(ctx, p); err != nil {
        return nil, err
    }
    return p, nil
}

func (u *productUsecase) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
    return u.repo.FindByID(ctx, id)
}
```

### Step 4: Add HTTP Handler, Routes, and DI
1. Create controller in `internal/delivery/http/handler/product_handler.go`.
2. Register routes in `internal/delivery/http/router.go`.
3. Wire dependencies in `cmd/api/main.go`.

---

## API Endpoints & Request Examples

### 1. Health & Kubernetes Probes

#### Diagnostic Overview (`/health`)
- **URL**: `GET /health`
- **Response Headers**:
  `X-Request-ID: 7a840e69-df42-4f36-8e50-934c9c1b3fbc`
  `X-Content-Type-Options: nosniff`
  `X-Frame-Options: DENY`
- **Response Body** (`200 OK` or `503 Service Unavailable` if dependencies fail):
```json
{
  "success": true,
  "message": "Service is healthy",
  "data": {
    "database": "connected",
    "redis": "disabled",
    "status": "up",
    "time": "2026-09-29T04:14:15Z"
  }
}
```

#### Kubernetes Liveness Probe (`/health/live`)
- **URL**: `GET /health/live`
- **Purpose**: Verifies that the HTTP server process is running and responsive.
- **Response** (`200 OK`):
```json
{
  "status": "alive",
  "time": "2026-09-29T04:14:15Z"
}
```

#### Kubernetes Readiness Probe (`/health/ready`)
- **URL**: `GET /health/ready`
- **Purpose**: Checks primary database connectivity.
- **Healthy Response** (`200 OK`):
```json
{
  "status": "ready",
  "time": "2026-09-29T04:14:15Z"
}
```
- **Degraded Response** (`503 Service Unavailable` - halts pod traffic routing):
```json
{
  "status": "unready",
  "reason": "database unavailable",
  "time": "2026-09-29T04:14:15Z"
}
```

### 2. User Registration (`/api/v1/auth/register`)
- **Method**: `POST`
- **Body**:
```json
{
  "name": "Jane Doe",
  "email": "jane@example.com",
  "password": "secretpassword123"
}
```
- **Response** (`201 Created`):
```json
{
  "success": true,
  "message": "User created successfully",
  "data": {
    "id": "55c5e397-faf2-44d2-bda1-b66760860385",
    "name": "Jane Doe",
    "email": "jane@example.com",
    "created_at": "2026-09-19T06:23:26.836Z",
    "updated_at": "2026-09-19T06:23:26.836Z"
  }
}
```

### 3. User Login (`/api/v1/auth/login`)
- **Method**: `POST`
- **Body**:
```json
{
  "email": "jane@example.com",
  "password": "secretpassword123"
}
```
- **Response Body** (`200 OK`):
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "55c5e397-faf2-44d2-bda1-b66760860385",
      "name": "Jane Doe",
      "email": "jane@example.com"
    }
  }
}
```

### User Logout (`/api/v1/auth/logout`)
- **Method**: `POST`
- **Description**: User logout acknowledgement (client app clears stored JWT token).
- **Response** (`200 OK`):
```json
{
  "success": true,
  "message": "Logout successful",
  "data": null
}
```

### Authenticated Profile (`/api/v1/auth/me`) [Protected JWT]
- **Method**: `GET`
- **Authentication**: `Authorization: Bearer <token>`
- **Response** (`200 OK`):
```json
{
  "success": true,
  "message": "Profile retrieved successfully",
  "data": {
    "id": "55c5e397-faf2-44d2-bda1-b66760860385",
    "name": "Jane Doe",
    "email": "jane@example.com"
  }
}
```

### Paginated User List (`/api/v1/users`) [Protected JWT]
- **URL**: `GET /api/v1/users?page=1&page_size=10`
- **Authentication**: `Authorization: Bearer <token>`
- **Response** (`200 OK`):
```json
{
  "success": true,
  "message": "Users retrieved successfully",
  "data": {
    "items": [
      {
        "id": "55c5e397-faf2-44d2-bda1-b66760860385",
        "name": "Jane Doe",
        "email": "jane@example.com",
        "created_at": "2026-09-19T06:23:26.836Z",
        "updated_at": "2026-09-19T06:23:26.836Z"
      }
    ],
    "meta": {
      "current_page": 1,
      "page_size": 10,
      "total_items": 1,
      "total_pages": 1
    }
  }
}
```

### Get User by ID (`/api/v1/users/:id`) [Protected JWT]
- **URL**: `GET /api/v1/users/:id`
- **Authentication**: `Authorization: Bearer <token>`
- *Note: If Redis is enabled, responses are cached for 10 minutes.*

### 7. Update User (`/api/v1/users/:id`) [Protected JWT]
- **URL**: `PUT /api/v1/users/:id`
- **Header**: `Authorization: Bearer <token>`
- **Body**:
```json
{
  "name": "Jane Doe Updated",
  "email": "jane.new@example.com"
}
```

### 8. Delete User (`/api/v1/users/:id`) [Protected JWT]
- **URL**: `DELETE /api/v1/users/:id`
- **Header**: `Authorization: Bearer <token>`

---

## Testing & Mocking

Thanks to Clean Architecture, Usecase logic is tested completely in-memory without requiring any real database connection:

```bash
# Run all unit tests with race detection
make test
# or
go test -v -race ./...
```

Unit test examples can be found in `internal/usecase/user_usecase_test.go`, `pkg/hash/hash_test.go`, `pkg/mail/mail_test.go`, and `pkg/worker/worker_test.go`.

---

## Deployment & Docker Hardening

### Makefile Command Shortcuts
| Command | Description |
|---|---|
| `make dev` | Run application with **Air Hot Reload** (auto rebuild on code changes) |
| `make run` | Run HTTP API server directly via `go run` |
| `make worker` | Run dedicated background **Queue Worker** runner (`cmd/worker/main.go`) |
| `make build` | Compile all binaries into `bin/` (`gin-starter-pack`, `worker`, `migrate`, `seed`) |
| `make test` | Run all unit & integration tests with `-v -race` |
| `make tidy` | Tidy up Go module dependencies (`go mod tidy`) |
| `make swagger` | Regenerate OpenAPI specification & Swagger UI docs |
| `make migrate` | Run GORM AutoMigrate standalone via CLI |
| `make seed` | Seed database with initial admin and fake users |
| `make air-install` | Install Air binary (`go install github.com/air-verse/air@latest`) |
| `make docker-up` | Spin up services with Docker Compose (API, Worker, Postgres, Redis) |
| `make docker-down` | Tear down Docker Compose services |
| `make clean` | Remove binaries, tmp files, logs, and temporary SQLite databases |

### Hardened Production Dockerfile
The production Docker build is hardened according to CIS benchmark standards:
- **Multi-stage compilation**: Strips debugging symbols (`-ldflags="-w -s"`) and packages `api`, `worker`, and `migrate` into a clean Alpine image (~25MB).
- **Non-root execution**: Runs under non-privileged user `appuser:appuser` (UID 10001) to protect container hosts.
- **Built-in Healthcheck**: Automatically polls `GET http://localhost:8080/health/live` to enable self-healing container orchestrators.

```bash
# Build production Docker image
docker build -t gin-starter-pack:latest .
```

### Docker Compose Multi-Service Topology
In production, background job processing scales independently from web traffic:
```yaml
services:
  app:    # Web API Server running cmd/api/main.go
  worker: # Dedicated Queue Worker running cmd/worker/main.go
  postgres: # Primary SQL Database with healthcheck
  redis:  # In-memory Cache & Queue Broker with healthcheck
```
```bash
# Start all containers in background with automated health checks
docker compose up -d

# View live API server logs
docker compose logs -f app

# View live background queue worker logs
docker compose logs -f worker
```
