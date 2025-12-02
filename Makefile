.PHONY: build clean all channel-manager telegram-monitor dashboard-collector dashboard-api forwarding-collector dashboard deploy install-services

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

# Run tests (if any exist)
test:
	go test ./...

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
	@echo "  test               - Run tests"
	@echo "  fmt                - Format code"
	@echo "  lint               - Lint code (requires golangci-lint)"
	@echo "  help               - Show this help"
