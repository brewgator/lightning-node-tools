#!/bin/bash

# Lightning Node Tools - Complete Installation Script
# Installs systemd services, crontab, builds binaries, and starts everything

set -e # Exit on any error

# Get project root (go up two levels from scripts directory)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "🚀 Lightning Node Tools - Complete Installation"
echo "================================================"
echo "📁 Project root: ${PROJECT_ROOT}"
echo "👤 User: ${USER}"
echo ""

# Check if running as root (should not be)
if [ "$EUID" -eq 0 ]; then
    echo "❌ Please run this script as a regular user (it will use sudo when needed)"
    exit 1
fi

# Step 1: Build binaries
echo "🔨 Step 1: Building binaries..."
cd "${PROJECT_ROOT}"
make clean
make build
echo "  ✅ Binaries built successfully"
echo ""

# Step 2: Install systemd services
echo "⚡ Step 2: Installing systemd services..."
"${SCRIPT_DIR}/install-services-auto.sh"
echo ""

# Step 3: Install crontab (with user confirmation)
echo "📅 Step 3: Installing crontab jobs..."
"${SCRIPT_DIR}/install-crontab-auto.sh"
echo ""

# Step 4: Create required directories
echo "📁 Step 4: Creating required directories..."
mkdir -p "${HOME}/lightning-node-tools/logs"
mkdir -p "${HOME}/backups"
echo "  ✅ Directories created"
echo ""

# Step 5: Start services
echo "🚀 Step 5: Starting services..."
echo "❓ Do you want to start all services now? (Y/n)"
read -r response
case "$response" in
    [nN][oO]|[nN])
        echo "⏭️  Skipping service startup"
        ;;
    *)
        echo "  Starting bitcoin-dashboard-api..."
        sudo systemctl start bitcoin-dashboard-api
        
        echo "  Starting bitcoin-dashboard-collector..."
        sudo systemctl start bitcoin-dashboard-collector
        
        echo "  Starting bitcoin-forwarding-collector..."
        sudo systemctl start bitcoin-forwarding-collector
        
        echo "  Starting webhook-deployer..."
        sudo systemctl start webhook-deployer
        
        echo "  ✅ All services started"
        ;;
esac
echo ""

# Step 6: Health check
echo "🔍 Step 6: Health check..."
sleep 2  # Give services time to start

# Check service status
echo "📊 Service Status:"
services=("bitcoin-dashboard-api" "bitcoin-dashboard-collector" "bitcoin-forwarding-collector" "webhook-deployer")
for service in "${services[@]}"; do
    if systemctl is-active --quiet "${service}"; then
        echo "  ✅ ${service}: Running"
    else
        echo "  ❌ ${service}: Not running"
    fi
done
echo ""

# Test API endpoints
echo "🌐 API Health Check:"
if curl -s http://localhost:8090/api/health > /dev/null 2>&1; then
    echo "  ✅ Dashboard API (port 8090): Responding"
else
    echo "  ❌ Dashboard API (port 8090): Not responding"
fi

if curl -s http://localhost:9000/health > /dev/null 2>&1; then
    echo "  ✅ Webhook deployer (port 9000): Responding"
else
    echo "  ❌ Webhook deployer (port 9000): Not responding"
fi
echo ""

echo "🎉 Installation completed!"
echo ""
echo "📋 What was installed:"
echo "  • Systemd services for portfolio tracking and auto-deployment"
echo "  • Cron jobs for backups, monitoring, and maintenance"  
echo "  • Required directories for logs and backups"
echo ""
echo "🔧 Useful commands:"
echo "  make build                    # Rebuild binaries"
echo "  sudo systemctl status bitcoin-dashboard-api  # Check service status"
echo "  sudo journalctl -u bitcoin-dashboard-api -f  # View logs"
echo "  crontab -l                    # View installed cron jobs"
echo "  curl http://localhost:8090/api/health        # Test dashboard API"
echo "  curl http://localhost:9000/health            # Test webhook deployer"
echo ""
echo "📖 For more information, see:"
echo "  deployment/DEPLOYMENT.md      # Detailed deployment guide"
echo "  deployment/systemd/README.md  # Service file documentation"
echo ""