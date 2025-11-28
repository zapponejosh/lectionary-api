# Lectionary API Makefile
# Prerequisites: Go 1.24+
# Run `make help` to see available commands

.PHONY: help build run test lint fmt clean migrate import docker-build docker-run

# Default target
.DEFAULT_GOAL := help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Binary names
API_BINARY=bin/api
IMPORT_BINARY=bin/import

# Build flags
LDFLAGS=-ldflags "-s -w"

## help: Show this help message
help:
	@echo "Lectionary API - Available Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build all binaries
build: build-api build-import

## build-api: Build the API server
build-api:
	@echo "Building API server..."
	@mkdir -p bin
	$(GOBUILD) $(LDFLAGS) -o $(API_BINARY) ./cmd/api

## build-import: Build the import tool
build-import:
	@echo "Building import tool..."
	@mkdir -p bin
	$(GOBUILD) $(LDFLAGS) -o $(IMPORT_BINARY) ./cmd/import

## run: Run the API server
run:
	@echo "Starting API server..."
	$(GORUN) ./cmd/api

## run-dev: Run with hot reload (requires air: go install github.com/air-verse/air@latest)
run-dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run linter (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

## tidy: Tidy and verify dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy
	$(GOMOD) verify

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@rm -f *.db *.db-journal *.db-shm *.db-wal

## migrate: Run database migrations
migrate:
	@echo "Running migrations..."
	$(GORUN) ./cmd/api -migrate

## import: Import lectionary data from PDF (usage: make import PDF=path/to/file.pdf)
import:
ifndef PDF
	$(error PDF is required. Usage: make import PDF=path/to/file.pdf)
endif
	@echo "Importing from $(PDF)..."
	$(GORUN) ./cmd/import -pdf $(PDF)

## setup: Initial project setup
setup: tidy
	@echo "Setting up project..."
	@mkdir -p data/pdfs
	@mkdir -p bin
	@cp -n .env.example .env 2>/dev/null || true
	@echo "Setup complete! Edit .env with your configuration."

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t lectionary-api:latest .

## docker-run: Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env lectionary-api:latest

## fly-deploy: Deploy to Fly.io
fly-deploy:
	@echo "Deploying to Fly.io..."
	fly deploy

## fly-logs: View Fly.io logs
fly-logs:
	fly logs