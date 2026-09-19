# Gin Go Clean Architecture Starter Pack

A production-ready REST API starter pack built with **Go (Golang)** and the **Gin Web Framework**, architected according to **Clean Architecture (Uncle Bob)** principles. Modular, loosely coupled, and flexible: supports dynamic multi-database switching (PostgreSQL, MySQL, SQLite), optional Redis caching, JWT authentication, daily rotating logs, rate limiting, Swagger UI, and background workers.

---

## Table of Contents

1. [Key Features](#key-features)
2. [Clean Architecture Principles](#clean-architecture-principles)
3. [Project Directory Structure](#project-directory-structure)
4. [Database Configuration (Ultra Flexible)](#database-configuration-ultra-flexible)
5. [Redis Configuration](#redis-configuration)
6. [Daily Rotating Logger (Laravel-Style)](#daily-rotating-logger-laravel-style)
7. [Background Job Worker Pool](#background-job-worker-pool)
8. [Email Service (pkg/mail)](#email-service-pkgmail)
9. [Rate Limiting & CORS](#rate-limiting--cors)
10. [Swagger API Documentation](#swagger-api-documentation)
11. [Database Migrations (GORM AutoMigrate)](#database-migrations-gorm-automigrate)
12. [Environment Variables Reference](#environment-variables-reference)
13. [Getting Started](#getting-started)
14. [Step-by-Step Guide: Adding a New Feature](#step-by-step-guide-adding-a-new-feature)
15. [API Endpoints & Request Examples](#api-endpoints--request-examples)
16. [Testing & Mocking](#testing--mocking)
17. [Deployment & Docker](#deployment--docker)

---

## Key Features

- **Clean Architecture**: Strict Dependency Rule (`Delivery` -> `Usecase` -> `Repository` -> `Domain`). The Domain layer is pure Go with zero external framework dependencies.
- **Viper Configuration**: Unified configuration loading from `.env`, system environment variables, and defaults via `spf13/viper`.
- **Multi-Database Support**: Switch between `postgres`, `mysql`, and `sqlite` simply by modifying `DB_DRIVER` or providing a direct `DB_DSN` without altering application code.
- **Driver Registry Pattern**: Easily install and plug in third-party GORM drivers (e.g. SQL Server, ClickHouse) in just a few lines.
- **Optional Redis Cache**: Redis can be toggled on/off (`REDIS_ENABLED=true/false`). When disabled, queries safely bypass the cache with zero downtime or panics.
- **JWT Authentication**: Built-in HMAC-SHA256 JWT generation, validation, and Gin auth middleware (`Authorization: Bearer <token>`).
- **Dedicated Password Hashing**: Standalone `pkg/hash` module powered by Bcrypt.
- **Daily Rotating Logger (Laravel-Style)**: Powered by Go standard `log/slog` and `lumberjack`. Outputs to stdout (terminal) and rotating files in `logs/app.log` with automatic gzip archiving.
- **IP-Based Rate Limiting**: Token-bucket algorithm per client IP (`golang.org/x/time/rate`), returning `429 Too Many Requests` when limits are exceeded.
- **Configurable CORS**: Dynamic allowed origins, HTTP methods, and headers configurable via `.env`.
- **Interactive Swagger / OpenAPI Docs**: Auto-generated API documentation served at `/swagger/index.html`.
- **Email Service**: Standalone `pkg/mail` package supporting `smtp` (TLS/SSL) and `log` (console logging for dev) drivers.
- **Asynchronous Background Worker**: Concurrent goroutine worker pool with job queues and graceful shutdown (e.g., asynchronous welcome email dispatch via Mailer).
- **Database Migrations**: Native GORM `AutoMigrate` runs automatically on startup and can also be executed via standalone CLI (`make migrate`).
- **Air Hot Reload**: Instant live code reloading for development using `make dev`.
- **DevOps Ready**: Multi-stage `Dockerfile`, `docker-compose.yml`, and `Makefile` shortcuts.

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
│   └── migrate/
│       └── main.go                 # Database migration CLI runner (GORM AutoMigrate)
├── config/
│   └── config.go                   # Viper configuration parser & environment binder
├── docs/                           # Auto-generated Swagger / OpenAPI spec files
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── internal/
│   ├── domain/                     # Core domain entities & interface contracts
│   │   ├── user.go                 # User entity, DTOs, and interface definitions
│   │   └── errors.go               # Standard domain error variables
│   ├── repository/                 # Data access layer
│   │   └── gorm/
│   │       └── user_repository.go  # GORM multi-driver repository implementation
│   ├── usecase/                    # Business logic layer
│   │   ├── user_usecase.go         # User business logic implementation
│   │   └── user_usecase_test.go    # Unit tests with in-memory mock repository
│   └── delivery/
│       └── http/                   # Transport layer (Gin HTTP)
│           ├── handler/
│           │   ├── user_handler.go   # CRUD & authentication handlers
│           │   └── health_handler.go # System diagnostics health check handler
│           ├── middleware/
│           │   ├── auth.go         # JWT Bearer token authentication middleware
│           │   ├── cors.go         # Configurable CORS middleware
│           │   ├── logger.go       # slog request logging middleware
│           │   ├── ratelimit.go    # IP-based token bucket rate limiter
│           │   └── recovery.go     # Panic recovery middleware
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
│   │   └── logger.go               # Daily rotating slog logger with lumberjack
│   ├── mail/
│   │   ├── mail.go                 # SMTP & Log mailer implementations
│   │   └── mail_test.go            # Unit tests for mail service
│   ├── redis/
│   │   └── redis.go                # Redis client wrapper with graceful fallback
│   ├── response/
│   │   └── response.go             # Standardized JSON API response helpers
│   └── worker/
│       ├── worker.go               # Asynchronous worker pool & job definitions
│       └── worker_test.go          # Unit tests for worker pool
├── .air.toml                       # Air configuration for hot reload
├── .env.example                    # Environment variable template
├── .env                            # Active environment configuration (git-ignored)
├── Dockerfile                      # Multi-stage Alpine Docker build
├── docker-compose.yml              # Services: app, postgres, mysql, redis
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

## Daily Rotating Logger (Laravel-Style)

Structured logging using Go's standard `log/slog` coupled with `lumberjack.Logger` in `pkg/logger`.
Operates similarly to Laravel's logging system:
- Real-time logging to both the **terminal** and rotating files in `logs/`.
- Active file: `logs/app.log`.
- Automatically rolls over based on file size or retention age and compresses old logs into `.gz`.

Configuration in `.env`:
```env
LOG_LEVEL=info        # debug, info, warn, error
LOG_DIR=logs          # log file directory
LOG_FILENAME=app.log  # active log file name
LOG_MAX_SIZE_MB=100   # maximum file size before rotation (MB)
LOG_MAX_BACKUPS=30    # maximum number of archived files retained
LOG_MAX_AGE_DAYS=30   # maximum age to retain files (days)
LOG_COMPRESS=true     # gzip compression for old archives
```

---

## Background Job Worker Pool

A concurrent **Worker Pool** in `pkg/worker` processes heavy or deferred tasks in the background without blocking client HTTP responses.

### How it works:
1. Create a struct that implements the `worker.Job` interface:
   ```go
   type WelcomeEmailJob struct {
       UserID   string
       Email    string
       UserName string
   }

   func (j *WelcomeEmailJob) Name() string {
       return "WelcomeEmailJob:" + j.Email
   }

   func (j *WelcomeEmailJob) Execute(ctx context.Context) error {
       // Send email or perform heavy task
       return nil
   }
   ```
2. Dispatch the job from your usecase:
   ```go
   workerPool.Dispatch(&WelcomeEmailJob{UserID: "123", Email: "user@example.com", UserName: "User"})
   ```
3. The Worker Pool manages worker goroutines and ensures a **graceful shutdown** (drains and finishes in-flight jobs before the server terminates).

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

## Rate Limiting & CORS

### IP Rate Limiting (Token Bucket)
Guards against brute-force attacks and DDoS:
- Implemented with `golang.org/x/time/rate`.
- Assigns a rate and burst limit per client IP.
- Exceeding the quota triggers an immediate `429 Too Many Requests`.

Configuration in `.env`:
```env
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPS=20.0     # Average requests per second per IP
RATE_LIMIT_BURST=40     # Peak burst allowance
```

### CORS
Configure allowed origins, methods, and headers:
```env
CORS_ALLOWED_ORIGINS=*
CORS_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-Requested-With,Accept
```

---

## Swagger API Documentation

Interactive OpenAPI / Swagger UI is integrated out-of-the-box:
- Access the UI in your browser: **`http://localhost:8080/swagger/index.html`**
- Regenerate Swagger documentation after editing endpoint annotations:
  ```bash
  make swagger
  ```

---

## Database Migrations (GORM AutoMigrate)

Database schema migration is handled natively by **GORM AutoMigrate**.

### How It Works:
- **Automatic on Startup**: When you start the API server (`make dev` or `make run`), `cmd/api/main.go` executes `db.AutoMigrate(...)` for all registered entities.
- **Standalone CLI Migration**: You can run migrations without starting the HTTP server:
  ```bash
  make migrate
  # Or directly:
  go run cmd/migrate/main.go
  ```
- **Adding Entities**: Simply register new entity structs in `cmd/api/main.go` and `cmd/migrate/main.go`:
  ```go
  db.AutoMigrate(
      &domain.User{},
      &domain.Product{},
  )
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
| `CORS_ALLOW_CREDENTIALS` | bool | `true` | Allow browser cookies / credentials across origins |
| `RATE_LIMIT_ENABLED` | bool | `true` | Enable/disable rate limiter |
| `RATE_LIMIT_RPS` | float | `20.0` | Requests allowed per second per IP |
| `RATE_LIMIT_BURST` | int | `40` | Burst request allowance |
| `WORKER_CONCURRENCY` | int | `5` | Number of concurrent worker goroutines |
| `WORKER_QUEUE_SIZE` | int | `100` | Background job buffer queue size |

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

### 1. Health Check
- **URL**: `GET /health`
- **Response**:
```json
{
  "success": true,
  "message": "Service is healthy",
  "data": {
    "database": "connected",
    "redis": "disabled",
    "status": "up",
    "time": "2026-09-19T06:23:21Z"
  }
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
- **Response Headers**:
  `Set-Cookie: access_token=<token>; Path=/; HttpOnly; SameSite=Lax`
  `Set-Cookie: token=<token>; Path=/; HttpOnly; SameSite=Lax`
- **Response Body** (`200 OK`):
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
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
- **Description**: Clears `access_token` and `token` cookies.
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
- **Authentication**: Either `Cookie: access_token=<token>` or `Authorization: Bearer <token>`
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
- **Authentication**: Either `Cookie: access_token=<token>` or `Authorization: Bearer <token>`
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
- **Authentication**: Either `Cookie: access_token=<token>` or `Authorization: Bearer <token>`
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

## Deployment & Docker

### Makefile Command Shortcuts
| Command | Description |
|---|---|
| `make dev` | Run application with **Air Hot Reload** (auto rebuild on code changes) |
| `make run` | Run application directly via `go run` |
| `make build` | Compile binary into `bin/gin-starter-pack` |
| `make test` | Run all unit tests with `-v -race` |
| `make tidy` | Tidy up Go module dependencies (`go mod tidy`) |
| `make swagger` | Regenerate OpenAPI specification & Swagger UI docs |
| `make migrate` | Run GORM AutoMigrate standalone via CLI |
| `make air-install` | Install Air binary (`go install github.com/air-verse/air@latest`) |
| `make docker-up` | Spin up services with Docker Compose |
| `make docker-down` | Tear down Docker Compose services |
| `make clean` | Remove binaries, tmp files, logs, and temporary SQLite databases |

### Build Docker Image
```bash
docker build -t gin-starter-pack:latest .
```
The Docker build utilizes a multi-stage process producing a minimal Alpine Linux image.
