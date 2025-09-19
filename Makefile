.PHONY: build clean all channel-manager telegram-monitor

# Default target - build all tools
all: build

# Build all tools
build: channel-manager telegram-monitor

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

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/

# Install tools to GOPATH/bin (optional)
install: build
	@echo "Installing tools to GOPATH/bin..."
	go install ./cmd/channel-manager
	go install ./cmd/telegram-monitor

# Run tests (if any exist)
test:
	go test ./...

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Show help
help:
	@echo "Available targets:"
	@echo "  build (default)    - Build all tools"
	@echo "  channel-manager    - Build only channel-manager"
	@echo "  telegram-monitor   - Build only telegram-monitor"
	@echo "  clean             - Remove build artifacts"
	@echo "  install           - Install tools to GOPATH/bin"
	@echo "  test              - Run tests"
	@echo "  fmt               - Format code"
	@echo "  lint              - Lint code (requires golangci-lint)"
	@echo "  help              - Show this help"