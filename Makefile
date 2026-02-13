.PHONY: dev test test-integration migrate migrate-down build clean web

# Build the binary
build:
	go build -o bin/infraplane ./cmd/infraplane

# Run the server
dev:
	docker compose up -d postgres
	go run ./cmd/infraplane

# Run unit tests
test:
	go test ./... -v -short

# Run integration tests (requires Docker/Colima)
test-integration:
	DOCKER_HOST="unix://$(HOME)/.colima/docker.sock" TESTCONTAINERS_RYUK_DISABLED=true go test ./... -v -run Integration -timeout 180s

# Run all tests
test-all:
	go test ./... -v

# Run database migrations
migrate:
	go run ./cmd/infraplane migrate up

# Rollback last migration
migrate-down:
	go run ./cmd/infraplane migrate down

# Start frontend dev server
web:
	cd web && npm run dev

# Clean build artifacts
clean:
	rm -rf bin/

# Install Go dependencies
deps:
	go mod tidy

# Format code
fmt:
	go fmt ./...
	go vet ./...
