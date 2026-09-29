.PHONY: all run dev build test tidy clean docker-up docker-down air-install migrate seed worker swagger

APP_NAME = gin-starter-pack
MAIN_FILE = cmd/api/main.go
MIGRATE_MAIN = cmd/migrate/main.go
SEED_MAIN = cmd/seed/main.go
WORKER_MAIN = cmd/worker/main.go
GOPATH = $(shell go env GOPATH)
AIR = $(GOPATH)/bin/air
SWAG = $(GOPATH)/bin/swag

all: build

# Normal HTTP API run
run:
	go run $(MAIN_FILE)

# Run dedicated background queue worker
worker:
	go run $(WORKER_MAIN)

# Hot reload development with Air
dev:
	@if ! which air > /dev/null 2>&1 && [ ! -f "$(AIR)" ]; then \
		echo "Installing Air for hot reload..."; \
		go install github.com/air-verse/air@latest; \
	fi
	@if which air > /dev/null 2>&1; then \
		air; \
	else \
		$(AIR); \
	fi

air-install:
	go install github.com/air-verse/air@latest

# Swagger API Docs Generation
swagger:
	@if which swag > /dev/null 2>&1; then \
		swag init -g $(MAIN_FILE); \
	elif [ -f "$(SWAG)" ]; then \
		$(SWAG) init -g $(MAIN_FILE); \
	else \
		go install github.com/swaggo/swag/cmd/swag@latest; \
		$(SWAG) init -g $(MAIN_FILE); \
	fi

# Database Migrations (GORM AutoMigrate)
migrate:
	go run $(MIGRATE_MAIN)

# Database Seeding
seed:
	go run $(SEED_MAIN)

build:
	@mkdir -p bin
	go build -o bin/$(APP_NAME) $(MAIN_FILE)
	go build -o bin/worker $(WORKER_MAIN)
	go build -o bin/migrate $(MIGRATE_MAIN)
	go build -o bin/seed $(SEED_MAIN)

test:
	go test -v -race ./...

tidy:
	go mod tidy

clean:
	rm -rf bin/ tmp/ logs/ *.db *.db-journal build-errors.log

docker-up:
	docker compose up -d

docker-down:
	docker compose down
