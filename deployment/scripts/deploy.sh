#!/bin/bash

# Bitcoin Lightning Node Tools - Deployment Script
# Stops services, builds binaries, and restarts services

set -e # Exit on any error

# Configuration
SERVICES=(
    "bitcoin-dashboard-api"
    "bitcoin-forwarding-collector"
    "bitcoin-dashboard-collector"
)

BINARIES=(
    "portfolio-api:services/portfolio/api"
    "forwarding-collector:services/lightning/forwarding-collector"
    "portfolio-collector:services/portfolio/collector"
)

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="${PROJECT_ROOT}/bin"

echo "🚀 Starting deployment process..."
echo "📁 Project root: ${PROJECT_ROOT}"

# Create bin directory if it doesn't exist
mkdir -p "${BIN_DIR}"

# Step 1: Stop all services
echo ""
echo "🛑 Stopping services..."
for service in "${SERVICES[@]}"; do
    if systemctl is-active --quiet "${service}"; then
        echo "  Stopping ${service}..."
        sudo systemctl stop "${service}"
    else
        echo "  ${service} is not running"
    fi
done

# Step 2: Build binaries
echo ""
echo "🔨 Building binaries..."
cd "${PROJECT_ROOT}"

for binary_info in "${BINARIES[@]}"; do
    binary_name=$(echo "${binary_info}" | cut -d: -f1)
    source_path=$(echo "${binary_info}" | cut -d: -f2)

    echo "  Building ${binary_name}..."
    go build -o "${BIN_DIR}/${binary_name}" "./${source_path}"

    if [ -f "${BIN_DIR}/${binary_name}" ]; then
        echo "    ✅ ${binary_name} built successfully"
    else
        echo "    ❌ Failed to build ${binary_name}"
        exit 1
    fi
done

# Step 3: Start all services
echo ""
echo "▶️  Starting services..."
for service in "${SERVICES[@]}"; do
    echo "  Starting ${service}..."
    sudo systemctl start "${service}"

    # Wait a moment for service to start
    sleep 2

    if systemctl is-active --quiet "${service}"; then
        echo "    ✅ ${service} started successfully"
    else
        echo "    ❌ Failed to start ${service}"
        sudo systemctl status "${service}" --no-pager -l
        exit 1
    fi
done

# Step 4: Show status
echo ""
echo "📊 Service status:"
for service in "${SERVICES[@]}"; do
    status=$(systemctl is-active "${service}" 2>/dev/null || echo "inactive")
    if [ "${status}" = "active" ]; then
        echo "  ✅ ${service}: ${status}"
    else
        echo "  ❌ ${service}: ${status}"
    fi
done

echo ""
echo "🎉 Deployment completed!"
echo ""
echo "🔍 Quick health check:"
echo "  curl http://localhost:8090/api/health"
echo "  curl http://localhost:8090/api/lightning/fees?days=7"
echo "  curl http://localhost:8090/api/lightning/forwards?days=7"
