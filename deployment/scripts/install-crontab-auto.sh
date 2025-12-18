#!/bin/bash

# Lightning Node Tools - Automated Crontab Installation Script  
# Installs cron jobs using the crontab.example template

set -e # Exit on any error

# Get project root (go up two levels from scripts directory)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CRONTAB_TEMPLATE="${PROJECT_ROOT}/deployment/crontab.example"
CURRENT_USER="${USER}"

echo "📅 Installing crontab jobs automatically..."
echo "📁 Project root: ${PROJECT_ROOT}"
echo "👤 User: ${CURRENT_USER}"
echo ""

# Check if running as root (should not be)
if [ "$EUID" -eq 0 ]; then
    echo "❌ Please run this script as a regular user"
    exit 1
fi

# Check if we're in the right directory
if [ ! -d "${PROJECT_ROOT}/bin" ] && [ ! -d "${PROJECT_ROOT}/tools" ]; then
    echo "❌ This doesn't appear to be the lightning-node-tools directory"
    echo "   Expected to find 'bin' and 'tools' directories in: ${PROJECT_ROOT}"
    exit 1
fi

# Check if template exists
if [ ! -f "${CRONTAB_TEMPLATE}" ]; then
    echo "❌ Crontab template not found: ${CRONTAB_TEMPLATE}"
    exit 1
fi

echo "📋 Creating crontab from template..."

# Create temporary cron file
TEMP_CRON="/tmp/lightning-crontab-${USER}"

# Process template - expand $HOME variables to actual path
sed -e "s|\$HOME|${HOME}|g" \
    -e "/^#.*Usage:/,\$d" \
    "${CRONTAB_TEMPLATE}" > "${TEMP_CRON}"

echo "📝 Generated crontab jobs:"
echo "----------------------------------------"
grep -v "^#" "${TEMP_CRON}" | grep -v "^$" || echo "  (No active cron jobs found)"
echo "----------------------------------------"
echo ""

# Ask for confirmation
echo "❓ Do you want to install this crontab? (y/N)"
read -r response
case "$response" in
    [yY][eE][sS]|[yY])
        echo "💾 Backing up existing crontab..."
        
        # Backup existing crontab
        if crontab -l > "${HOME}/crontab-backup-$(date +%Y%m%d-%H%M%S).txt" 2>/dev/null; then
            echo "  ✅ Existing crontab backed up"
        else
            echo "  ℹ️  No existing crontab to backup"
        fi
        
        echo "📅 Installing new crontab..."
        crontab "${TEMP_CRON}"
        
        echo "  ✅ Crontab installed successfully!"
        ;;
    *)
        echo "❌ Installation cancelled"
        echo "💡 To install manually:"
        echo "   cp ${CRONTAB_TEMPLATE} /tmp/mycron"
        echo "   nano /tmp/mycron  # Edit as needed"
        echo "   crontab /tmp/mycron"
        rm "${TEMP_CRON}"
        exit 0
        ;;
esac

# Clean up
rm "${TEMP_CRON}"

echo ""
echo "🎉 Crontab installation completed!"
echo ""
echo "📋 Installed jobs:"
echo "  • Daily 2:00 AM - Channel backups"
echo "  • Weekly Sun 2:15 AM - Fee optimization"  
echo "  • Every 2 minutes - Telegram monitoring"
echo "  • Daily 3:00 AM - Log cleanup & backup rotation"
echo ""
echo "🔍 Verify installation:"
echo "  crontab -l"
echo ""
echo "📝 View cron logs:"
echo "  tail -f /var/log/syslog | grep CRON"
echo "  # Or check specific logs in: ${HOME}/lightning-node-tools/logs/"
echo ""
echo "⚠️  Important notes:"
echo "  • Ensure LND is running and accessible via 'lncli'"
echo "  • Telegram bot credentials should be configured in .env"
echo "  • Create logs directory: mkdir -p ${HOME}/lightning-node-tools/logs"
echo "  • Create backups directory: mkdir -p ${HOME}/backups"
echo ""