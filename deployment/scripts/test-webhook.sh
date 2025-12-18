#!/bin/bash

# test-webhook.sh - Test script for webhook deployment system

set -e

# Configuration - Auto-detect paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
WEBHOOK_URL="http://localhost:9000"
SECRET_FILE="$REPO_DIR/secrets/webhook.secret"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Test health endpoint
test_health() {
    log "🔍 Testing health endpoint..."
    
    if response=$(curl -s "$WEBHOOK_URL/health"); then
        echo "$response" | jq . > /dev/null 2>&1
        if [[ $? -eq 0 ]]; then
            success "Health endpoint is working"
            echo "$response" | jq .
        else
            error "Health endpoint returned invalid JSON"
            echo "$response"
        fi
    else
        error "Health endpoint is not responding"
        return 1
    fi
}

# Test status endpoint
test_status() {
    log "📊 Testing status endpoint..."
    
    if response=$(curl -s "$WEBHOOK_URL/status"); then
        echo "$response" | jq . > /dev/null 2>&1
        if [[ $? -eq 0 ]]; then
            success "Status endpoint is working"
            echo "$response" | jq .
        else
            error "Status endpoint returned invalid JSON"
            echo "$response"
        fi
    else
        error "Status endpoint is not responding"
        return 1
    fi
}

# Test webhook with fake payload
test_webhook_fake() {
    log "🧪 Testing webhook endpoint with fake payload..."
    
    # Read webhook secret
    if [[ ! -f "$SECRET_FILE" ]]; then
        error "Webhook secret file not found: $SECRET_FILE"
        return 1
    fi
    
    SECRET=$(cat "$SECRET_FILE")
    
    # Create fake GitHub payload
    PAYLOAD='{
        "ref": "refs/heads/main",
        "repository": {
            "name": "lightning-node-tools",
            "full_name": "brewgator/lightning-node-tools"
        },
        "head_commit": {
            "id": "test123456789",
            "message": "Test deployment",
            "author": {
                "name": "Test User",
                "email": "test@example.com"
            }
        }
    }'
    
    # Calculate HMAC signature
    SIGNATURE="sha256=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | cut -d' ' -f2)"
    
    log "🔐 Using signature: $SIGNATURE"
    
    # Send webhook
    response=$(curl -s -w "%{http_code}" \
        -H "Content-Type: application/json" \
        -H "X-Hub-Signature-256: $SIGNATURE" \
        -d "$PAYLOAD" \
        "$WEBHOOK_URL/webhook")
    
    # Extract HTTP status code and body
    http_code="${response: -3}"
    body="${response%???}"
    
    if [[ "$http_code" == "200" ]]; then
        success "Webhook endpoint accepted the test payload"
        log "Response: $body"
    else
        error "Webhook endpoint returned HTTP $http_code"
        log "Response: $body"
        return 1
    fi
}

# Test with wrong signature
test_webhook_wrong_signature() {
    log "🚫 Testing webhook with wrong signature..."
    
    PAYLOAD='{"ref": "refs/heads/main"}'
    WRONG_SIGNATURE="sha256=wrongsignature"
    
    response=$(curl -s -w "%{http_code}" \
        -H "Content-Type: application/json" \
        -H "X-Hub-Signature-256: $WRONG_SIGNATURE" \
        -d "$PAYLOAD" \
        "$WEBHOOK_URL/webhook")
    
    http_code="${response: -3}"
    
    if [[ "$http_code" == "401" ]]; then
        success "Webhook correctly rejected wrong signature"
    else
        error "Webhook should have rejected wrong signature (got HTTP $http_code)"
        return 1
    fi
}

# Check service status
check_services() {
    log "🔍 Checking service status..."
    
    services=("webhook-deployer" "bitcoin-dashboard-api" "bitcoin-dashboard-collector" "bitcoin-forwarding-collector")
    
    for service in "${services[@]}"; do
        if systemctl is-active --quiet "$service" 2>/dev/null; then
            success "$service is running"
        else
            error "$service is not running"
        fi
    done
}

# Check logs
check_logs() {
    log "📄 Checking recent logs..."
    
    echo "--- Webhook deployer logs (last 10 lines) ---"
    sudo journalctl -u webhook-deployer -n 10 --no-pager
    
    echo ""
    echo "--- Deployment logs (last 10 lines) ---"
    if [[ -f "/var/log/lightning-deploy.log" ]]; then
        tail -n 10 "/var/log/lightning-deploy.log"
    else
        log "No deployment logs found yet"
    fi
}

# Run all tests
run_all_tests() {
    log "🧪 Running all webhook tests..."
    echo ""
    
    test_health || return 1
    echo ""
    
    test_status || return 1
    echo ""
    
    test_webhook_wrong_signature || return 1
    echo ""
    
    # Only run fake webhook test if requested
    if [[ "$1" == "--include-fake" ]]; then
        test_webhook_fake || return 1
        echo ""
    fi
    
    check_services
    echo ""
    
    check_logs
    echo ""
    
    success "All tests completed!"
}

# Main function
case "$1" in
    "health")
        test_health
        ;;
    "status")
        test_status
        ;;
    "webhook")
        test_webhook_fake
        ;;
    "security")
        test_webhook_wrong_signature
        ;;
    "services")
        check_services
        ;;
    "logs")
        check_logs
        ;;
    "all")
        run_all_tests "$2"
        ;;
    *)
        echo "Usage: $0 {health|status|webhook|security|services|logs|all}"
        echo ""
        echo "Commands:"
        echo "  health     - Test health endpoint"
        echo "  status     - Test status endpoint"
        echo "  webhook    - Test webhook with fake payload"
        echo "  security   - Test webhook security (wrong signature)"
        echo "  services   - Check service status"
        echo "  logs       - Show recent logs"
        echo "  all        - Run all tests (use 'all --include-fake' to test webhook)"
        exit 1
        ;;
esac