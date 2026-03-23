.PHONY: dev build run test lint migrate seed up down clean

# Run locally with hot reload (requires air: go install github.com/air-verse/air@latest)
dev:
	air

# Build binary
build:
	go build -o bin/server ./cmd/server

# Run binary
run: build
	./bin/server

# Run all tests
test:
	go test -race -cover ./...

# Run a single test (usage: make test-one T=TestTrustScore)
test-one:
	go test -race -run $(T) ./...

# Lint (requires golangci-lint)
lint:
	golangci-lint run ./...

# Vet
vet:
	go vet ./...

# Run migrations against local DB
migrate:
	psql "$(DATABASE_URL)" -f migrations/001_schema.up.sql

# Seed demo data
seed:
	psql "$(DATABASE_URL)" -f seed/seed.sql

# Reset DB: drop + recreate + seed
reset:
	psql "$(DATABASE_URL)" -f migrations/001_schema.down.sql
	psql "$(DATABASE_URL)" -f migrations/001_schema.up.sql
	psql "$(DATABASE_URL)" -f seed/seed.sql

# Docker compose up
up:
	docker compose up --build -d

# Docker compose down
down:
	docker compose down

# Docker compose down + remove volumes
clean:
	docker compose down -v
