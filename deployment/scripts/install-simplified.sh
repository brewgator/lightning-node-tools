#!/bin/bash

# Simplified Installation Script
# Combines functionality from multiple deployment scripts

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

print_step() {
    echo -e "${BLUE}🔵 $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if systemctl is available
check_systemd() {
    if ! command -v systemctl &> /dev/null; then
        print_warning "systemctl not found. Service installation will be skipped."
        return 1
    fi
    return 0
}

# Install systemd services
install_services() {
    if ! check_systemd; then
        return 0
    fi
    
    print_step "Installing systemd services..."
    
    local username="$(whoami)"
    local working_dir="$PROJECT_ROOT"
    
    # Service definitions
    local services=(
        "bitcoin-dashboard-api:portfolio-api:Portfolio REST API"
        "bitcoin-dashboard-collector:portfolio-collector:Portfolio data collector"
        "bitcoin-forwarding-collector:forwarding-collector:Lightning forwarding collector"
        "webhook-deployer:webhook-deployer:Auto-deployment webhook"
        "lightning-telegram-monitor:telegram-monitor:Telegram monitoring (oneshot)"
    )
    
    # Create systemd user directory
    local user_systemd_dir="$HOME/.config/systemd/user"
    mkdir -p "$user_systemd_dir"
    
    for service_def in "${services[@]}"; do
        IFS=':' read -r service_name binary_name description <<< "$service_def"
        
        local service_file="$user_systemd_dir/${service_name}.service"
        
        print_step "Creating $service_name.service..."
        
        # Create service file
        cat > "$service_file" << EOF
[Unit]
Description=$description
After=network.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=$working_dir
ExecStart=$working_dir/bin/$binary_name
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security settings
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ReadWritePaths=$working_dir/data
ProtectHome=read-only

[Install]
WantedBy=default.target
EOF
        
        # Add specific configurations per service
        case "$service_name" in
            "bitcoin-dashboard-api")
                # Update ExecStart to include port 8090
                sed -i "s|ExecStart=$working_dir/bin/$binary_name|ExecStart=$working_dir/bin/$binary_name --port 8090 --host 0.0.0.0|" "$service_file"
                ;;
            "webhook-deployer")
                # Add port 9000 for webhook deployer
                sed -i "s|ExecStart=$working_dir/bin/$binary_name|ExecStart=$working_dir/bin/$binary_name --port 9000|" "$service_file"
                ;;
        esac
        
        print_success "$service_name.service created"
        
        # Special handling for telegram monitor - also create timer
        if [ "$service_name" = "lightning-telegram-monitor" ]; then
            local timer_file="$user_systemd_dir/${service_name}.timer"
            print_step "Creating $service_name.timer..."
            
            cat > "$timer_file" << EOF
[Unit]
Description=Run Lightning Telegram Monitor every 2 minutes
Requires=lightning-telegram-monitor.service

[Timer]
OnCalendar=*:0/2
Persistent=true

[Install]
WantedBy=timers.target
EOF
            print_success "$service_name.timer created"
        fi
    done
    
    # Reload systemd
    print_step "Reloading systemd..."
    systemctl --user daemon-reload
    
    # Enable services
    for service_def in "${services[@]}"; do
        IFS=':' read -r service_name binary_name description <<< "$service_def"
        
        if [ -f "bin/$binary_name" ]; then
            systemctl --user enable "$service_name"
            print_success "$service_name enabled"
            
            # Enable timer for telegram monitor
            if [ "$service_name" = "lightning-telegram-monitor" ]; then
                systemctl --user enable "$service_name.timer"
                print_success "$service_name.timer enabled"
            fi
        else
            print_warning "$service_name enabled but binary not found (run 'make build' first)"
        fi
    done
    
    print_success "All services installed"
}

# Install cron jobs
install_crontab() {
    if ! command -v crontab &> /dev/null; then
        print_warning "crontab not found. Skipping cron job installation."
        return 0
    fi
    
    print_step "Installing cron jobs..."
    
    local username="$(whoami)"
    local working_dir="$PROJECT_ROOT"
    
    # Create temporary crontab file
    local temp_cron=$(mktemp)
    
    # Get existing crontab (if any) and filter out our jobs
    crontab -l 2>/dev/null | grep -v "lightning-node-tools" > "$temp_cron" || true
    
    # Add our cron jobs
    cat >> "$temp_cron" << EOF

# Lightning Node Tools - Auto-generated cron jobs
# Channel backups (daily 2:00 AM)
0 2 * * * cd $working_dir && lncli exportchanbackup --all --output_file=data/channel-backups-\$(date +\%Y\%m\%d).backup 2>/dev/null || echo "Channel backup failed"

# Fee optimization (weekly Sunday 2:15 AM)
15 2 * * 0 cd $working_dir && ./bin/channel-manager fee-optimizer >/dev/null 2>&1

# Telegram monitoring (every 2 minutes)
*/2 * * * * cd $working_dir && ./bin/telegram-monitor >/dev/null 2>&1

# Log cleanup (daily 3:00 AM)
0 3 * * * find $working_dir/logs -name "*.log" -mtime +30 -delete 2>/dev/null || true

EOF
    
    # Install new crontab
    if crontab "$temp_cron"; then
        print_success "Cron jobs installed"
    else
        print_error "Failed to install cron jobs"
        rm -f "$temp_cron"
        return 1
    fi
    
    rm -f "$temp_cron"
    
    # Show installed jobs
    echo ""
    echo "Installed cron jobs:"
    crontab -l | grep -A 10 "Lightning Node Tools" || true
}

# Start services
start_services() {
    if ! check_systemd; then
        return 0
    fi
    
    print_step "Starting services..."
    
    local services=("bitcoin-dashboard-api" "bitcoin-dashboard-collector" "bitcoin-forwarding-collector")
    
    for service in "${services[@]}"; do
        if [ -f "bin/$(echo $service | sed 's/bitcoin-dashboard-/portfolio-/' | sed 's/bitcoin-//')" ]; then
            if systemctl --user start "$service" 2>/dev/null; then
                print_success "$service started"
            else
                print_warning "$service failed to start (check: systemctl --user status $service)"
            fi
        else
            print_warning "Skipping $service (binary not found)"
        fi
    done
}

# Main installation function
install_all() {
    echo -e "${BLUE}╔═══════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║    Lightning Node Tools Installer    ║${NC}"
    echo -e "${BLUE}╚═══════════════════════════════════════╝${NC}"
    echo ""
    
    # Create directories
    print_step "Creating directories..."
    mkdir -p data logs
    print_success "Directories created"
    
    # Install services
    install_services
    
    # Install crontab
    install_crontab
    
    # Build if binaries don't exist
    if [ ! -f "bin/portfolio-api" ]; then
        print_step "Building binaries..."
        if make build; then
            print_success "Binaries built"
        else
            print_error "Build failed"
            return 1
        fi
    fi
    
    # Start services
    start_services
    
    echo ""
    print_success "Installation completed!"
    echo ""
    echo -e "${YELLOW}🎯 Next steps:${NC}"
    echo "1. Configure environment: cp .env.example .env && nano .env"
    echo "2. Test API: curl http://localhost:8090/api/health"
    echo "3. Check services: systemctl --user status bitcoin-dashboard-api"
    echo "4. View logs: journalctl --user -f -u bitcoin-dashboard-api"
    echo ""
}

# Run installation
install_all