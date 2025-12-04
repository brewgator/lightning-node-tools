#!/bin/bash

# setup-auto-deploy.sh - Setup script for GitHub webhook auto-deployment

set -e

# Configuration
REPO_PATH="$(pwd)"  # Use current directory (where script is run from)
USER="${SUDO_USER:-$(whoami)}"    # Use current user
GROUP="$(id -gn)"   # Use current user's primary group
PORT="9000"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${BLUE}[SETUP]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   error "This script must be run as root (use sudo)"
fi

log "🚀 Setting up auto-deployment for Lightning Node Tools"

# Create user if doesn't exist
if ! id "$USER" &>/dev/null; then
    log "👤 Creating user: $USER"
    useradd --system --shell /bin/bash --home-dir /opt --no-create-home "$USER"
    success "User $USER created"
else
    log "👤 User $USER already exists"
fi

# Create directories
log "📁 Creating directories..."
mkdir -p "$REPO_PATH"
mkdir -p "$REPO_PATH/secrets"
mkdir -p "$REPO_PATH-backups"
mkdir -p "/var/log"

# Set permissions
chown -R "$USER:$GROUP" "$REPO_PATH"
chown -R "$USER:$GROUP" "$REPO_PATH-backups"
chmod 755 "$REPO_PATH"
chmod 700 "$REPO_PATH/secrets"

# Generate webhook secret if it doesn't exist
SECRET_FILE="$REPO_PATH/secrets/webhook.secret"
if [[ ! -f "$SECRET_FILE" ]]; then
    log "🔐 Generating webhook secret..."
    openssl rand -hex 32 > "$SECRET_FILE"
    chmod 600 "$SECRET_FILE"
    chown "$USER:$GROUP" "$SECRET_FILE"
    success "Webhook secret generated: $SECRET_FILE"
else
    log "🔐 Webhook secret already exists"
fi

# Build the webhook deployer
log "🔨 Building webhook deployer..."
cd "$REPO_PATH"
sudo -u "$USER" go build -o bin/webhook-deployer ./cmd/webhook-deployer
success "Webhook deployer built successfully"

# Install systemd service
log "⚙️  Installing systemd service..."
# Create service file with dynamic paths
sed -e "s|__USER__|$USER|g" \
    -e "s|__GROUP__|$GROUP|g" \
    -e "s|__REPO_PATH__|$REPO_PATH|g" \
    systemd/webhook-deployer.service > /etc/systemd/system/webhook-deployer.service
systemctl daemon-reload
systemctl enable webhook-deployer
success "Systemd service installed"

# Setup logrotate
log "📝 Setting up log rotation..."
cat > /etc/logrotate.d/lightning-deploy << EOF
/var/log/lightning-deploy.log {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    copytruncate
    su $USER $GROUP
}
EOF
success "Log rotation configured"

# Configure firewall (if ufw is available)
if command -v ufw &> /dev/null; then
    log "🔥 Configuring firewall..."
    ufw allow "$PORT/tcp" comment "Lightning Node Tools Webhook"
    success "Firewall rule added for port $PORT"
else
    warn "UFW not found. Please manually configure firewall to allow port $PORT"
fi

# Create environment file template
ENV_FILE="$REPO_PATH/.env"
if [[ ! -f "$ENV_FILE" ]]; then
    log "📄 Creating environment file template..."
    cat > "$ENV_FILE" << EOF
# Environment variables for Lightning Node Tools
# WEBHOOK_SECRET is read from $SECRET_FILE

# Optional: Slack webhook URL for deployment notifications
# SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK

# Optional: Custom service names (space-separated)
# SERVICES="bitcoin-dashboard-api bitcoin-dashboard-collector bitcoin-forwarding-collector"
EOF
    chown "$USER:$GROUP" "$ENV_FILE"
    chmod 644 "$ENV_FILE"
    success "Environment file created"
fi

# Start the service
log "🚀 Starting webhook deployer service..."
systemctl start webhook-deployer

# Wait a moment and check status
sleep 2
if systemctl is-active --quiet webhook-deployer; then
    success "Webhook deployer service is running"
else
    error "Failed to start webhook deployer service"
fi

# Show webhook secret for GitHub configuration
WEBHOOK_SECRET=$(cat "$SECRET_FILE")

echo ""
echo "🎉 Auto-deployment setup completed successfully!"
echo ""
echo "📋 Next steps:"
echo "1. Configure GitHub webhook:"
echo "   - Go to your GitHub repository settings"
echo "   - Navigate to Settings > Webhooks > Add webhook"
echo "   - Set Payload URL: http://YOUR_SERVER_IP:$PORT/webhook"
echo "   - Set Content type: application/json"
echo "   - Set Secret: $WEBHOOK_SECRET"
echo "   - Select 'Just the push event'"
echo "   - Check 'Active'"
echo ""
echo "2. Test the webhook:"
echo "   curl http://localhost:$PORT/health"
echo ""
echo "3. Monitor deployment logs:"
echo "   sudo journalctl -u webhook-deployer -f"
echo "   tail -f /var/log/lightning-deploy.log"
echo ""
echo "4. Service management:"
echo "   sudo systemctl status webhook-deployer"
echo "   sudo systemctl stop webhook-deployer"
echo "   sudo systemctl start webhook-deployer"
echo "   sudo systemctl restart webhook-deployer"
echo ""

# Show service status
echo "📊 Current service status:"
systemctl status webhook-deployer --no-pager -l