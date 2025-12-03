.PHONY: build clean all channel-manager telegram-monitor dashboard-collector dashboard-api forwarding-collector dashboard deploy install-services test test-verbose test-coverage test-unit test-integration test-api test-collector test-forwarding test-db test-utils test-race test-clean

# Default target - build all tools
all: build

# Build all tools
build: channel-manager telegram-monitor dashboard-collector dashboard-api forwarding-collector

# Build channel-manager
channel-manager:
	@echo "Building channel-manager..."
	@mkdir -p bin
	go build -o bin/channel-manager ./cmd/channel-manager

# Build telegram-monitor
telegram-monitor:
	@echo "Building telegram-monitor..."
	@mkdir -p bin
	go build -o bin/telegram-monitor ./cmd/telegram-monitor

# Build dashboard-collector
dashboard-collector:
	@echo "Building dashboard-collector..."
	@mkdir -p bin
	go build -o bin/dashboard-collector ./cmd/dashboard-collector

# Build dashboard-api
dashboard-api:
	@echo "Building dashboard-api..."
	@mkdir -p bin
	go build -o bin/dashboard-api ./cmd/dashboard-api

# Build forwarding-collector
forwarding-collector:
	@echo "Building forwarding-collector..."
	@mkdir -p bin
	go build -o bin/forwarding-collector ./cmd/forwarding-collector

# Build and start complete dashboard (collector + api)
dashboard: dashboard-collector dashboard-api
	@echo "Dashboard components built successfully!"
	@echo "To start:"
	@echo "  1. ./bin/dashboard-collector --oneshot  # Test data collection"
	@echo "  2. ./bin/dashboard-api                  # Start web API"
	@echo "  3. Open http://localhost:8080           # View dashboard"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/

# Install tools to GOPATH/bin (optional)
install: build
	@echo "Installing tools to GOPATH/bin..."
	go install ./cmd/channel-manager
	go install ./cmd/telegram-monitor
	go install ./cmd/dashboard-collector
	go install ./cmd/dashboard-api

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run unit tests only (exclude integration tests)
test-unit:
	go test -short ./...

# Run integration tests only
test-integration:
	go test -run Integration ./test/...

# Run specific package tests
test-api:
	go test -v ./cmd/dashboard-api/...

test-collector:
	go test -v ./cmd/dashboard-collector/...

test-forwarding:
	go test -v ./cmd/forwarding-collector/...

test-db:
	go test -v ./pkg/db/...

test-utils:
	go test -v ./pkg/testutils/...

# Run tests with race detection
test-race:
	go test -race ./...

# Clean test artifacts
test-clean:
	rm -f coverage.out coverage.html
	go clean -testcache

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Install/update systemd service files
install-services:
	@echo "Installing systemd service files..."
	./scripts/install-services.sh

# Deploy: stop services, build, restart services
deploy:
	@echo "Deploying services..."
	./scripts/deploy.sh

# Validate CI pipeline locally
validate-ci:
	./scripts/validate-ci.sh

# Install pre-commit hook for CI validation
install-pre-commit-hook:
	./scripts/install-pre-commit-hook.sh

# Verify code is ready for CI
ci-ready: fmt test test-race build
	@echo "✅ Code is CI-ready!"

# Show help
help:
	@echo "Available targets:"
	@echo "  build (default)     - Build all tools"
	@echo "  channel-manager     - Build only channel-manager"
	@echo "  telegram-monitor    - Build only telegram-monitor"
	@echo "  dashboard-collector - Build only dashboard-collector"
	@echo "  dashboard-api       - Build only dashboard-api"
	@echo "  forwarding-collector - Build only forwarding-collector"
	@echo "  dashboard           - Build complete dashboard"
	@echo "  install-services    - Install/update systemd service files"
	@echo "  deploy              - Stop services, build, restart services"
	@echo "  clean              - Remove build artifacts"
	@echo "  install            - Install tools to GOPATH/bin"
	@echo "  test               - Run all tests"
	@echo "  test-verbose       - Run tests with verbose output"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  test-unit          - Run unit tests only"
	@echo "  test-integration   - Run integration tests only"
	@echo "  test-api           - Run API tests only"
	@echo "  test-collector     - Run collector tests only"
	@echo "  test-forwarding    - Run forwarding tests only"
	@echo "  test-db            - Run database tests only"
	@echo "  test-race          - Run tests with race detection"
	@echo "  test-clean         - Clean test artifacts"
	@echo "  fmt                - Format code"
	@echo "  lint               - Lint code (requires golangci-lint)"
	@echo "  validate-ci        - Validate CI pipeline locally"
	@echo "  install-pre-commit-hook - Install pre-commit hook for CI validation"
	@echo "  ci-ready           - Verify code is ready for CI"
	@echo "  help               - Show this help"
