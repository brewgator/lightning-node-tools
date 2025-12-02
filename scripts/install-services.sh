#!/bin/bash

# Bitcoin Lightning Node Tools - Service Installation Script
# Installs or updates systemd service files

set -e  # Exit on any error

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SYSTEMD_DIR="${PROJECT_ROOT}/systemd"
CURRENT_USER="${USER}"

echo "🔧 Installing systemd services..."
echo "📁 Project root: ${PROJECT_ROOT}"
echo "👤 User: ${CURRENT_USER}"

# Check if running as root (needed for systemd operations)
if [ "$EUID" -eq 0 ]; then
    echo "❌ Please run this script as a regular user (it will use sudo when needed)"
    exit 1
fi

# Ensure systemd directory exists
if [ ! -d "${SYSTEMD_DIR}" ]; then
    echo "❌ Systemd directory not found: ${SYSTEMD_DIR}"
    exit 1
fi

# Process each service file
for template_file in "${SYSTEMD_DIR}"/*.service; do
    if [ ! -f "${template_file}" ]; then
        continue
    fi
    
    service_name=$(basename "${template_file}")
    temp_file="/tmp/${service_name}"
    
    echo ""
    echo "📝 Processing ${service_name}..."
    
    # Replace template variables
    sed -e "s|{{USER}}|${CURRENT_USER}|g" \
        -e "s|{{WORKING_DIRECTORY}}|${PROJECT_ROOT}|g" \
        "${template_file}" > "${temp_file}"
    
    # Copy to systemd directory
    echo "  📋 Installing to /etc/systemd/system/${service_name}"
    sudo cp "${temp_file}" "/etc/systemd/system/${service_name}"
    
    # Clean up temp file
    rm "${temp_file}"
    
    echo "  ✅ ${service_name} installed"
done

# Reload systemd
echo ""
echo "🔄 Reloading systemd daemon..."
sudo systemctl daemon-reload

# Enable services
echo ""
echo "⚡ Enabling services..."
for template_file in "${SYSTEMD_DIR}"/*.service; do
    if [ ! -f "${template_file}" ]; then
        continue
    fi
    
    service_name=$(basename "${template_file}")
    echo "  Enabling ${service_name}..."
    sudo systemctl enable "${service_name}"
done

echo ""
echo "🎉 Service installation completed!"
echo ""
echo "💡 Next steps:"
echo "  - Run './scripts/deploy.sh' to build and start services"
echo "  - Check logs with: sudo journalctl -u bitcoin-dashboard-api -f"