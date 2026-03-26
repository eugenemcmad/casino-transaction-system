.PHONY: help build-api build-processor test test-race test-integration cover docker-up docker-down docker-logs migrate-up

# Default shell
SHELL := /bin/bash

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build-api: ## Build the API binary
	go build -o bin/api ./cmd/api

build-processor: ## Build the Processor binary
	go build -o bin/processor ./cmd/processor

test: ## Run unit tests (standard)
	go test -v ./...

test-integration: ## Run unit and integration tests
	go test -v -tags=integration ./...

test-e2e: ## Run unit and end-to-end tests
	go test -v -tags=e2e ./...

test-race: ## Run unit tests with race detection (requires CGO)
	CGO_ENABLED=1 go test -v -race ./...

cover: ## Run all tests and show coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func coverage.out

docker-up: ## Start all infrastructure and apps in Docker
	docker-compose -f docker-compose.dev.yaml up -d --build

docker-down: ## Stop all containers and remove volumes
	docker-compose -f docker-compose.dev.yaml down -v

docker-logs: ## Show logs from all containers
	docker-compose -f docker-compose.dev.yaml logs -f

migrate-up: ## Run database migrations manually
	docker-compose -f docker-compose.dev.yaml run --rm migrate

lint: ## Run golangci-lint (if installed)
	golangci-lint run ./...

run-api: ## Run API locally
	go run cmd/api/main.go

run-processor: ## Run Processor locally
	go run cmd/processor/main.go
