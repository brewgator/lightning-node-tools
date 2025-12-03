#!/bin/bash

# validate-ci.sh - Test CI pipeline commands locally before pushing

set -e

echo "🧪 Validating CI pipeline locally..."

# Check Go version
echo "📋 Go version:"
go version

# Verify dependencies
echo "🔍 Verifying dependencies..."
go mod verify

# Check formatting
echo "✨ Checking code formatting..."
unformatted=$(gofmt -s -l .)
if [ -n "$unformatted" ]; then
  echo "❌ The following files need formatting:"
  echo "$unformatted"
  echo "Run 'make fmt' to fix"
  exit 1
fi
echo "✅ Code formatting is clean"

# Vet code
echo "🔎 Running go vet..."
go vet ./...
echo "✅ Go vet passed"

# Run tests
echo "🧪 Running tests..."
make test
echo "✅ Tests passed"

# Test with race detection
echo "🏃 Testing with race detection..."
make test-race
echo "✅ Race detection tests passed"

# Build all components
echo "🔨 Building all components..."
make build
echo "✅ Build successful"

# Generate coverage
echo "📊 Generating test coverage..."
make test-coverage
echo "✅ Coverage report generated"

echo ""
echo "🎉 All CI validation checks passed!"
echo "✅ Your code is ready to push"
echo ""
echo "Coverage report available at: coverage.html"