#!/bin/bash

# Lightning Node Tools - Automated Service Installation Script
# Installs systemd services using copy-paste ready .example files

set -e # Exit on any error

# Get project root (go up two levels from scripts directory)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SYSTEMD_DIR="${PROJECT_ROOT}/deployment/systemd"
CURRENT_USER="${USER}"

echo "🔧 Installing systemd services automatically..."
echo "📁 Project root: ${PROJECT_ROOT}"
echo "👤 User: ${CURRENT_USER}"
echo ""

# Check if running as root (should not be)
if [ "$EUID" -eq 0 ]; then
    echo "❌ Please run this script as a regular user (it will use sudo when needed)"
    exit 1
fi

# Check if we're in the right directory
if [ ! -d "${PROJECT_ROOT}/bin" ] && [ ! -d "${PROJECT_ROOT}/services" ]; then
    echo "❌ This doesn't appear to be the lightning-node-tools directory"
    echo "   Expected to find 'bin' and 'services' directories in: ${PROJECT_ROOT}"
    exit 1
fi

# Ensure systemd directory exists
if [ ! -d "${SYSTEMD_DIR}" ]; then
    echo "❌ Systemd directory not found: ${SYSTEMD_DIR}"
    exit 1
fi

# List of services to install
services=(
    "bitcoin-dashboard-api"
    "bitcoin-dashboard-collector" 
    "bitcoin-forwarding-collector"
    "webhook-deployer"
)

echo "🔄 Installing services..."

# Install each service
for service in "${services[@]}"; do
    example_file="${SYSTEMD_DIR}/${service}.service.example"
    target_file="/etc/systemd/system/${service}.service"
    
    echo "📝 Installing ${service}.service..."
    
    # Check if example file exists
    if [ ! -f "${example_file}" ]; then
        echo "  ⚠️  Example file not found: ${example_file}"
        echo "  🔄 Falling back to template file..."
        template_file="${SYSTEMD_DIR}/${service}.service"
        
        if [ ! -f "${template_file}" ]; then
            echo "  ❌ Neither example nor template file found for ${service}"
            continue
        fi
        
        # Use template with variable substitution
        temp_file="/tmp/${service}.service"
        sed -e "s|{{USER}}|${CURRENT_USER}|g" \
            -e "s|{{WORKING_DIRECTORY}}|${PROJECT_ROOT}|g" \
            -e "s|__USER__|${CURRENT_USER}|g" \
            -e "s|__GROUP__|${CURRENT_USER}|g" \
            -e "s|__REPO_PATH__|${PROJECT_ROOT}|g" \
            "${template_file}" >"${temp_file}"
        
        sudo cp "${temp_file}" "${target_file}"
        rm "${temp_file}"
    else
        # Use example file with username substitution
        temp_file="/tmp/${service}.service"
        
        # Replace YOUR_USERNAME and expand $HOME variables
        sed -e "s|YOUR_USERNAME|${CURRENT_USER}|g" \
            -e "s|\$HOME|${HOME}|g" \
            "${example_file}" >"${temp_file}"
        
        sudo cp "${temp_file}" "${target_file}"
        rm "${temp_file}"
    fi
    
    echo "  ✅ ${service}.service installed"
done

echo ""
echo "🔄 Reloading systemd daemon..."
sudo systemctl daemon-reload

echo ""
echo "⚡ Enabling services..."
for service in "${services[@]}"; do
    echo "  Enabling ${service}.service..."
    sudo systemctl enable "${service}.service"
done

echo ""
echo "🎉 Service installation completed!"
echo ""
echo "🚀 Next steps:"
echo "  1. Build binaries: make build"
echo "  2. Start services: sudo systemctl start bitcoin-dashboard-api bitcoin-dashboard-collector bitcoin-forwarding-collector webhook-deployer"
echo "  3. Check status: sudo systemctl status bitcoin-dashboard-api"
echo "  4. View logs: sudo journalctl -u bitcoin-dashboard-api -f"
echo ""
echo "📊 Quick health check:"
echo "  curl http://localhost:8090/api/health  # Dashboard API"
echo "  curl http://localhost:9000/health      # Webhook deployer"
echo ""