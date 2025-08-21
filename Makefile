# Aegis Authentication Library Makefile

.PHONY: help build test test-verbose test-race test-cover clean lint fmt vet mod-tidy mod-download install-tools

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build all modules
	@echo "Building all modules..."
	@go build ./...

build-core: ## Build core library only
	@echo "Building core library..."
	@go build .

build-adapters: ## Build all adapters
	@echo "Building adapters..."
	@go build ./adapters/...

build-plugins: ## Build all plugins
	@echo "Building plugins..."
	@go build ./plugins/...

build-examples: ## Build example applications
	@echo "Building examples..."
	@go build ./examples/...

# Test targets
test: ## Run tests for all modules
	@echo "Running tests for all modules..."
	@go test ./...

test-core: ## Run tests for core library only
	@echo "Running core library tests..."
	@go test -v .

test-adapters: ## Run tests for all adapters
	@echo "Running adapter tests..."
	@go test ./adapters/...

test-plugins: ## Run tests for all plugins
	@echo "Running plugin tests..."
	@go test ./plugins/...

test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	@go test -v ./...

test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	@go test -race ./...

test-cover: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Code quality targets
lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

# Dependency management
mod-tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	@go mod tidy

mod-download: ## Download go modules
	@echo "Downloading go modules..."
	@go mod download

# Development tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/golang/mock/mockgen@latest

# Clean targets
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -f coverage.out coverage.html
	@go clean ./...

# CI targets
ci: mod-download fmt vet lint test-race ## Run CI pipeline

# Development workflow
dev: fmt vet test ## Run development checks