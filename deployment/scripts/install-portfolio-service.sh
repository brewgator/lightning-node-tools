#!/bin/bash

# Bitcoin Portfolio API Service Installation Script
# Installs the new real-time portfolio API as a systemd service

set -e # Exit on any error

# Get project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SYSTEMD_DIR="${PROJECT_ROOT}/deployment/systemd"
CURRENT_USER="${USER}"
HOME_DIR="${HOME}"

echo "🚀 Installing Bitcoin Portfolio API Service"
echo "📁 Project root: ${PROJECT_ROOT}"
echo "👤 User: ${CURRENT_USER}"
echo ""

# Check if running as root (should not be)
if [ "$EUID" -eq 0 ]; then
    echo "❌ Please run this script as a regular user (it will use sudo when needed)"
    exit 1
fi

# Check if we're in the right directory
if [ ! -f "${PROJECT_ROOT}/bin/portfolio-api" ]; then
    echo "❌ Portfolio API binary not found. Please run 'make portfolio' first."
    echo "   Expected: ${PROJECT_ROOT}/bin/portfolio-api"
    exit 1
fi

# Ensure systemd directory exists
if [ ! -f "${SYSTEMD_DIR}/bitcoin-portfolio-api.service.example" ]; then
    echo "❌ Service template not found: ${SYSTEMD_DIR}/bitcoin-portfolio-api.service.example"
    exit 1
fi

echo "🔄 Creating customized service file..."

# Create service file by replacing placeholders
service_content=$(cat "${SYSTEMD_DIR}/bitcoin-portfolio-api.service.example")
service_content="${service_content//YOUR_USERNAME/${CURRENT_USER}}"
service_content="${service_content//YOUR_GROUP/${CURRENT_USER}}"
service_content="${service_content//\/path\/to\/lightning-node-tools/${PROJECT_ROOT}}"

# Write service file
service_file="/tmp/bitcoin-portfolio-api.service"
echo "${service_content}" > "${service_file}"

echo "📝 Installing service file..."
sudo cp "${service_file}" /etc/systemd/system/bitcoin-portfolio-api.service
rm "${service_file}"

echo "🔄 Reloading systemd daemon..."
sudo systemctl daemon-reload

echo "✅ Enabling bitcoin-portfolio-api service..."
sudo systemctl enable bitcoin-portfolio-api

echo "🚀 Starting bitcoin-portfolio-api service..."
sudo systemctl start bitcoin-portfolio-api

echo ""
echo "✅ Service installation complete!"
echo ""
echo "📊 Service status:"
sudo systemctl status bitcoin-portfolio-api --no-pager -l

echo ""
echo "🔗 Portfolio Dashboard: http://localhost:8090"
echo ""
echo "📋 Useful commands:"
echo "  sudo systemctl status bitcoin-portfolio-api    # Check status"
echo "  sudo systemctl restart bitcoin-portfolio-api   # Restart service"
echo "  journalctl -f -u bitcoin-portfolio-api         # View logs"
echo "  sudo systemctl stop bitcoin-portfolio-api      # Stop service"
echo ""