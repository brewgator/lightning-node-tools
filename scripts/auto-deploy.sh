#!/bin/bash

# auto-deploy.sh - Automated deployment script triggered by GitHub webhooks
# This script pulls the latest code, builds, and restarts all Lightning Node services

set -e

# Configuration
REPO_DIR="/opt/lightning-node-tools"
BACKUP_DIR="/opt/lightning-node-tools-backups"
LOG_FILE="/var/log/lightning-deploy.log"
MAX_BACKUPS=5

# Service names (adjust these to match your systemd service names)
SERVICES=(
    "bitcoin-dashboard-api"
    "bitcoin-dashboard-collector" 
    "bitcoin-forwarding-collector"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "$(date '+%Y-%m-%d %H:%M:%S') $1" | tee -a "$LOG_FILE"
}

# Error handling
handle_error() {
    log "${RED}❌ Deployment failed at: $1${NC}"
    log "${YELLOW}🔄 Starting rollback process...${NC}"
    rollback
    exit 1
}

# Rollback function
rollback() {
    log "${YELLOW}🔄 Rolling back to previous version...${NC}"
    
    if [ -d "$BACKUP_DIR/previous" ]; then
        # Stop services
        for service in "${SERVICES[@]}"; do
            log "${BLUE}⏸️  Stopping $service${NC}"
            sudo systemctl stop "$service" || true
        done
        
        # Restore previous version
        sudo rm -rf "$REPO_DIR.rollback" || true
        sudo mv "$REPO_DIR" "$REPO_DIR.rollback" || true
        sudo mv "$BACKUP_DIR/previous" "$REPO_DIR" || true
        
        # Restart services
        for service in "${SERVICES[@]}"; do
            log "${BLUE}▶️  Starting $service${NC}"
            sudo systemctl start "$service"
            
            # Check service health
            sleep 2
            if ! sudo systemctl is-active --quiet "$service"; then
                log "${RED}❌ Failed to start $service after rollback${NC}"
            else
                log "${GREEN}✅ $service started successfully${NC}"
            fi
        done
        
        log "${GREEN}✅ Rollback completed${NC}"
    else
        log "${RED}❌ No backup available for rollback${NC}"
    fi
}

# Health check function
health_check() {
    local service=$1
    local retries=5
    local delay=3
    
    for i in $(seq 1 $retries); do
        if sudo systemctl is-active --quiet "$service"; then
            log "${GREEN}✅ $service is healthy${NC}"
            return 0
        else
            log "${YELLOW}⏳ Waiting for $service to start (attempt $i/$retries)${NC}"
            sleep $delay
        fi
    done
    
    log "${RED}❌ $service failed health check${NC}"
    return 1
}

# Create backup directory
create_backup_dir() {
    sudo mkdir -p "$BACKUP_DIR"
    sudo chown -R $(whoami):$(whoami) "$BACKUP_DIR"
}

# Cleanup old backups
cleanup_backups() {
    log "${BLUE}🧹 Cleaning up old backups...${NC}"
    
    cd "$BACKUP_DIR"
    # Keep only the most recent backups
    ls -1t | tail -n +$((MAX_BACKUPS + 1)) | xargs -r rm -rf
    
    log "${GREEN}✅ Backup cleanup completed${NC}"
}

# Main deployment function
main() {
    log "${BLUE}🚀 Starting automated deployment...${NC}"
    log "${BLUE}📝 Commit: ${DEPLOY_COMMIT:-unknown}${NC}"
    log "${BLUE}💬 Message: ${DEPLOY_MESSAGE:-unknown}${NC}"
    log "${BLUE}👤 Author: ${DEPLOY_AUTHOR:-unknown}${NC}"
    
    # Trap errors
    trap 'handle_error "line $LINENO"' ERR
    
    # Create necessary directories
    create_backup_dir
    
    # Change to repository directory
    cd "$REPO_DIR" || handle_error "changing to repo directory"
    
    # Create backup of current version
    BACKUP_NAME="backup-$(date +%Y%m%d-%H%M%S)"
    log "${BLUE}💾 Creating backup: $BACKUP_NAME${NC}"
    
    sudo rm -rf "$BACKUP_DIR/previous" || true
    sudo cp -r "$REPO_DIR" "$BACKUP_DIR/$BACKUP_NAME" || handle_error "creating backup"
    sudo ln -sfn "$BACKUP_DIR/$BACKUP_NAME" "$BACKUP_DIR/previous" || handle_error "linking previous backup"
    
    # Fetch latest changes
    log "${BLUE}📥 Fetching latest changes...${NC}"
    git fetch origin || handle_error "git fetch"
    
    # Get current commit for comparison
    OLD_COMMIT=$(git rev-parse HEAD)
    
    # Reset to latest main
    log "${BLUE}🔄 Resetting to origin/main...${NC}"
    git reset --hard origin/main || handle_error "git reset"
    
    NEW_COMMIT=$(git rev-parse HEAD)
    
    # Check if there are actually new changes
    if [ "$OLD_COMMIT" = "$NEW_COMMIT" ]; then
        log "${YELLOW}ℹ️  No new changes, deployment not needed${NC}"
        exit 0
    fi
    
    log "${GREEN}📈 Updated from ${OLD_COMMIT:0:8} to ${NEW_COMMIT:0:8}${NC}"
    
    # Install/update dependencies
    log "${BLUE}📦 Installing dependencies...${NC}"
    go mod download || handle_error "go mod download"
    go mod verify || handle_error "go mod verify"
    
    # Run tests to ensure code quality
    log "${BLUE}🧪 Running tests...${NC}"
    make test || handle_error "running tests"
    
    # Build all components
    log "${BLUE}🔨 Building components...${NC}"
    make build || handle_error "building components"
    
    # Stop services gracefully
    log "${BLUE}⏹️  Stopping services...${NC}"
    for service in "${SERVICES[@]}"; do
        log "${BLUE}⏸️  Stopping $service${NC}"
        sudo systemctl stop "$service" || log "${YELLOW}⚠️  Warning: failed to stop $service${NC}"
    done
    
    # Small delay to ensure services are fully stopped
    sleep 2
    
    # Start services
    log "${BLUE}▶️  Starting services...${NC}"
    for service in "${SERVICES[@]}"; do
        log "${BLUE}🚀 Starting $service${NC}"
        sudo systemctl start "$service" || handle_error "starting $service"
        
        # Health check
        if ! health_check "$service"; then
            handle_error "$service failed health check"
        fi
    done
    
    # Verify all services are running
    log "${BLUE}🔍 Verifying service status...${NC}"
    all_healthy=true
    for service in "${SERVICES[@]}"; do
        if sudo systemctl is-active --quiet "$service"; then
            log "${GREEN}✅ $service is running${NC}"
        else
            log "${RED}❌ $service is not running${NC}"
            all_healthy=false
        fi
    done
    
    if [ "$all_healthy" = false ]; then
        handle_error "some services failed to start properly"
    fi
    
    # Cleanup old backups
    cleanup_backups
    
    # Success!
    DURATION=$(($(date +%s) - $(date +%s -d "1 minute ago")))
    log "${GREEN}🎉 Deployment completed successfully!${NC}"
    log "${GREEN}📊 New commit: ${NEW_COMMIT:0:8}${NC}"
    log "${GREEN}⏱️  Total time: ${DURATION}s${NC}"
    
    # Optional: Send notification (uncomment and configure as needed)
    # curl -X POST -H 'Content-type: application/json' \
    #     --data '{"text":"✅ Lightning Node Tools deployed successfully"}' \
    #     "$SLACK_WEBHOOK_URL"
}

# Run main function
main "$@"