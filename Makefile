.PHONY: build run test clean docker-build docker-up docker-down

# Build the application
build:
	go build -o bin/server ./cmd/server

# Run the application locally
run:
	go run ./cmd/server

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Build docker image
docker-build:
	docker-compose build

# Start services with docker-compose
docker-up:
	docker compose up --build

# Stop docker services
docker-down:
	docker compose down

# Stop and remove volumes
docker-clean:
	docker compose down -v

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run linter
lint:
	golangci-lint run

# Fix linting issues automatically
lint-fix:
	golangci-lint run --fix
