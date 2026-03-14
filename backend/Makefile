.PHONY: build run up down migrate-up migrate-down test

# Build the Go binary
build:
	go build -o bin/server ./cmd/server

# Run the server locally
run:
	go run ./cmd/server

# Start all infrastructure
up:
	docker compose -f ../docker-compose.yml up -d

# Stop all infrastructure
down:
	docker compose -f ../docker-compose.yml down

# Run migrations (requires golang-migrate CLI)
migrate-up:
	migrate -path ./migrations -database "postgres://banking:banking123@localhost:5432/banking_db?sslmode=disable" up

migrate-down:
	migrate -path ./migrations -database "postgres://banking:banking123@localhost:5432/banking_db?sslmode=disable" down

# Run tests
test:
	go test ./... -v -race

# Format and vet
lint:
	go fmt ./...
	go vet ./...
