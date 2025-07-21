# Build variables
BINARY_NAME=makemcp
BUILD_DIR=build
MAIN_PATH=./cmd/makemcp.go

# Version info (can be overridden)
VERSION ?= dev
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT_HASH) -X main.buildDate=$(BUILD_DATE)"

.PHONY: help build clean test install dev-deps lint fmt vet tidy run local-config-test local-test integration-test

help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)

test: ## Run tests
	@echo "Running tests..."
	go test -v -race ./...

install: ## Install the binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) $(MAIN_PATH)

dev-deps: ## Install development dependencies
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

tidy: ## Tidy and verify dependencies
	@echo "Tidying dependencies..."
	go mod tidy
	go mod verify

run: build ## Build and run the binary
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME) --help

# Testing commands
local-openapi-test: ## Test config generation with local server
	$(BUILD_DIR)/$(BINARY_NAME) openapi -s 'http://localhost:8081/openapi.json' -b "http://localhost:8081"

local-file-test:
	$(BUILD_DIR)/$(BINARY_NAME) load makemcp.json

local-test: local-openapi-test ## Alias for local-openapi-test

integration-test: build ## Run OpenAPI integration tests
	@echo "Running OpenAPI integration tests..."
	cd testbed/openapi && go test -v

# Cross-compilation targets
build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "Cross-compilation complete. Binaries in $(BUILD_DIR)/"

# Development workflow
dev: clean fmt vet test build ## Run full development workflow

# Release commands
tag-release: ## Tag a new release (usage: make tag-release [major|minor|patch], defaults to patch)
	@./scripts/tag-release.sh $(filter-out $@,$(MAKECMDGOALS))

# This allows arguments to be passed to the tag-release target
%:
	@:

release: clean test build-all ## Prepare release builds
