#!/bin/bash

# test-coverage-ci.sh - CI-friendly test coverage that handles Go version differences

set -e

echo "🧪 Running test coverage for CI..."

# Try the modern approach first (Go 1.25+)
if go test -v -coverprofile=coverage.out ./pkg/db ./cmd/dashboard-api ./cmd/dashboard-collector ./cmd/forwarding-collector 2>/dev/null; then
    echo "✅ Coverage generated using targeted packages"
else
    echo "⚠️  Falling back to compatible coverage mode..."
    # Fallback: run coverage on all packages but ignore covdata errors
    go test -v -coverprofile=coverage.out ./... 2>/dev/null || {
        # If that fails, just run tests without coverage profile
        echo "⚠️  Running tests without coverage profile..."
        go test ./...
        echo "coverage: N/A" > coverage.out
    }
fi

# Generate HTML report if coverage.out exists and has data
if [ -s coverage.out ] && ! grep -q "coverage: N/A" coverage.out; then
    go tool cover -html=coverage.out -o coverage.html
    echo "📊 Coverage report generated: coverage.html"
    
    # Show coverage summary
    echo ""
    echo "📈 Coverage Summary:"
    go tool cover -func=coverage.out | tail -1
else
    echo "⚠️  No coverage data available"
fi