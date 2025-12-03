# CI/CD Documentation

This directory contains GitHub Actions workflows for continuous integration and deployment.

## Workflows

### 1. `test.yml` - Basic Test Workflow
- **Triggers**: Push to `main`/`develop` branches, PRs to `main`
- **Actions**: 
  - Checkout code
  - Set up Go 1.21
  - Install dependencies
  - Run tests
  - Run tests with coverage
  - Build all components

### 2. `ci.yml` - Comprehensive CI Pipeline  
- **Triggers**: Push to `main`/`develop` branches, PRs to `main`
- **Go Versions**: Tests against Go 1.20 and 1.21
- **Actions**:
  - **Test Job**: 
    - Dependency verification
    - Unit tests
    - Race condition detection
    - Coverage reports
    - Build verification
  - **Lint Job**:
    - Code formatting checks
    - Linting with golangci-lint
    - Import organization

## Local Development

Before pushing code, run these commands locally to ensure CI passes:

```bash
# Format code
make fmt

# Run all tests
make test

# Run tests with race detection  
make test-race

# Run tests with coverage
make test-coverage

# Build all components
make build

# Verify dependencies
go mod verify

# Check formatting
if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
  echo "Code needs formatting - run 'make fmt'"
  exit 1
fi
```

## Configuration

- **golangci-lint**: Configured via `.golangci.yml`
- **Go modules**: Dependencies managed via `go.mod`
- **Test coverage**: Reports generated as `coverage.html`

## Status Badges

Add these badges to your README.md:

```markdown
[![Test](https://github.com/your-username/lightning-node-tools/actions/workflows/test.yml/badge.svg)](https://github.com/your-username/lightning-node-tools/actions/workflows/test.yml)
[![CI](https://github.com/your-username/lightning-node-tools/actions/workflows/ci.yml/badge.svg)](https://github.com/your-username/lightning-node-tools/actions/workflows/ci.yml)
```