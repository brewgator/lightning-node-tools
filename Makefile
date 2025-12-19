.PHONY: build clean all channel-manager telegram-monitor dashboard-collector dashboard-api forwarding-collector dashboard deploy install-services test test-verbose test-coverage test-unit test-integration test-api test-collector test-forwarding test-db test-utils test-race test-clean

# Default target - build all tools
all: build

# Build all tools
build: channel-manager telegram-monitor portfolio-collector portfolio-api forwarding-collector webhook-deployer

# Build channel-manager
channel-manager:
	@echo "Building channel-manager..."
	@mkdir -p bin
	go build -o bin/channel-manager ./tools/channel-manager

# Build telegram-monitor
telegram-monitor:
	@echo "Building telegram-monitor..."
	@mkdir -p bin
	go build -o bin/telegram-monitor ./tools/monitoring

# Build portfolio-collector
portfolio-collector:
	@echo "Building portfolio-collector..."
	@mkdir -p bin
	go build -o bin/portfolio-collector ./services/portfolio/collector

# Build portfolio-api
portfolio-api:
	@echo "Building portfolio-api..."
	@mkdir -p bin
	go build -o bin/portfolio-api ./services/portfolio/api

# Legacy aliases for backward compatibility (portfolio services ARE the dashboard services)
dashboard-collector: portfolio-collector
	@echo "Note: dashboard-collector is an alias for portfolio-collector"

dashboard-api: portfolio-api
	@echo "Note: dashboard-api is an alias for portfolio-api"

# Build forwarding-collector
forwarding-collector:
	@echo "Building forwarding-collector..."
	@mkdir -p bin
	go build -o bin/forwarding-collector ./services/lightning/forwarding-collector

# Build webhook-deployer
webhook-deployer:
	@echo "Building webhook-deployer..."
	@mkdir -p bin
	go build -o bin/webhook-deployer ./services/deployment/webhook-deployer

# Build complete portfolio system (collector + api)
portfolio: portfolio-collector portfolio-api
	@echo "Portfolio components built successfully!"
	@echo "To start:"
	@echo "  1. ./bin/portfolio-collector --oneshot  # Test data collection"
	@echo "  2. ./bin/portfolio-api                  # Start web API"
	@echo "  3. Open http://localhost:8080           # View dashboard"

# Build and start complete dashboard (collector + api) - alias for portfolio
dashboard: portfolio
	@echo "Note: dashboard is an alias for portfolio - they are the same services"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/

# Install tools to GOPATH/bin (optional)
install: build
	@echo "Installing tools to GOPATH/bin..."
	go install ./tools/channel-manager
	go install ./tools/monitoring
	go install ./services/portfolio/collector
	go install ./services/portfolio/api

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@# Only test packages that have test files
	go test -v -coverprofile=coverage.out ./internal/db ./services/portfolio/api ./services/portfolio/collector ./services/lightning/forwarding-collector
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with coverage (compatible mode for older Go versions)
test-coverage-compat:
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
	go test -v ./services/portfolio/api/...

test-collector:
	go test -v ./services/portfolio/collector/...

test-forwarding:
	go test -v ./services/lightning/forwarding-collector/...

test-db:
	go test -v ./internal/db/...

test-utils:
	go test -v ./internal/testutils/...

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
	./deployment/scripts/install-services.sh

# Install systemd services automatically (recommended)
install-services-auto:
	@echo "Installing systemd services automatically..."
	./deployment/scripts/install-services-auto.sh

# Install crontab automatically
install-crontab:
	@echo "Installing crontab jobs..."
	./deployment/scripts/install-crontab-auto.sh

# Complete installation: services + crontab + build + start
install-all:
	@echo "Running complete installation..."
	./deployment/scripts/install-all.sh

# Deploy: stop services, build, restart services
deploy:
	@echo "Deploying services..."
	./deployment/scripts/deploy.sh

# Validate CI pipeline locally
validate-ci:
	./deployment/scripts/validate-ci.sh

# Install pre-commit hook for CI validation
install-pre-commit-hook:
	./deployment/scripts/install-pre-commit-hook.sh

# Verify code is ready for CI
ci-ready: fmt test test-race build
	@echo "✅ Code is CI-ready!"

# Show help
help:
	@echo "Available targets:"
	@echo "  build (default)     - Build all tools"
	@echo "  channel-manager     - Build only channel-manager"
	@echo "  telegram-monitor    - Build only telegram-monitor"
	@echo "  portfolio-collector - Build only portfolio-collector"
	@echo "  portfolio-api       - Build only portfolio-api"
	@echo "  dashboard-collector - Build only dashboard-collector"
	@echo "  dashboard-api       - Build only dashboard-api"
	@echo "  forwarding-collector - Build only forwarding-collector"
	@echo "  portfolio           - Build complete portfolio system"
	@echo "  dashboard           - Build complete dashboard system"
	@echo "  install-services    - Install/update systemd service files"
	@echo "  install-services-auto - Install systemd services automatically (recommended)"
	@echo "  install-crontab     - Install crontab jobs automatically"
	@echo "  install-all         - Complete installation: services + crontab + build + start"
	@echo "  deploy              - Stop services, build, restart services"
	@echo "  clean              - Remove build artifacts"
	@echo "  install            - Install tools to GOPATH/bin"
	@echo "  test               - Run all tests"
	@echo "  test-verbose       - Run tests with verbose output"
	@echo "  test-coverage      - Run tests with coverage report (Go 1.22 compatible)"
	@echo "  test-coverage-compat - Run tests with coverage (fallback for older Go versions)"
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
