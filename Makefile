.PHONY: help build test lint run-auth run-rides run-geo run-payments run-notifications docker-up docker-down migrate-up migrate-down

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build all services
	@echo "Building all services..."
	@go build -o bin/auth ./cmd/auth
	@go build -o bin/rides ./cmd/rides
	@go build -o bin/geo ./cmd/geo
	@go build -o bin/payments ./cmd/payments
	@go build -o bin/notifications ./cmd/notifications

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

test-coverage: test ## Run tests with coverage report
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run ./...

run-auth: ## Run auth service
	@echo "Starting auth service..."
	@go run ./cmd/auth

run-rides: ## Run rides service
	@echo "Starting rides service..."
	@go run ./cmd/rides

run-geo: ## Run geo service
	@echo "Starting geo service..."
	@go run ./cmd/geo

run-payments: ## Run payments service
	@echo "Starting payments service..."
	@go run ./cmd/payments

run-notifications: ## Run notifications service
	@echo "Starting notifications service..."
	@go run ./cmd/notifications

docker-up: ## Start all services with Docker Compose
	@echo "Starting services with Docker Compose..."
	@docker-compose up -d

docker-down: ## Stop all services
	@echo "Stopping services..."
	@docker-compose down

docker-build: ## Build Docker images
	@echo "Building Docker images..."
	@docker-compose build

migrate-up: ## Run database migrations
	@echo "Running migrations..."
	@migrate -path db/migrations -database "postgresql://postgres:postgres@localhost:5432/ridehailing?sslmode=disable" up

migrate-down: ## Rollback database migrations
	@echo "Rolling back migrations..."
	@migrate -path db/migrations -database "postgresql://postgres:postgres@localhost:5432/ridehailing?sslmode=disable" down

migrate-create: ## Create a new migration file (use: make migrate-create NAME=migration_name)
	@migrate create -ext sql -dir db/migrations -seq $(NAME)

tidy: ## Tidy go modules
	@go mod tidy

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.txt coverage.html
