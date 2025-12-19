#!/bin/bash

# Lightning Node Tools - Unified Setup Script
# This script provides a single command to set up everything

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
cd "$SCRIPT_DIR"

print_header() {
    echo -e "${BLUE}╔══════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║        Lightning Node Tools         ║${NC}"
    echo -e "${BLUE}║          Setup Script v1.0          ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════╝${NC}"
    echo ""
}

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

check_dependencies() {
    print_step "Checking dependencies..."
    
    # Check for Go
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.20+ and try again."
        exit 1
    fi
    
    local go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | cut -c 3-)
    local major_version=$(echo $go_version | cut -d. -f1)
    local minor_version=$(echo $go_version | cut -d. -f2)
    
    if [ "$major_version" -lt 1 ] || ([ "$major_version" -eq 1 ] && [ "$minor_version" -lt 20 ]); then
        print_error "Go version $go_version detected. Please install Go 1.20+ and try again."
        exit 1
    fi
    
    print_success "Go $go_version detected"
    
    # Check for systemctl (systemd)
    if ! command -v systemctl &> /dev/null; then
        print_warning "systemctl not found. Service installation will be skipped."
        SKIP_SERVICES=true
    fi
    
    # Check for crontab
    if ! command -v crontab &> /dev/null; then
        print_warning "crontab not found. Cron job installation will be skipped."
        SKIP_CRON=true
    fi
}

show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "OPTIONS:"
    echo "  --quick          Quick setup (build + demo with mock data)"
    echo "  --full           Full setup (build + services + cron)"
    echo "  --build-only     Build binaries only"
    echo "  --services-only  Install/update services only"
    echo "  --demo           Run demo with mock data"
    echo "  --help           Show this help message"
    echo ""
    echo "EXAMPLES:"
    echo "  $0 --quick      # Quick demo setup"
    echo "  $0 --full       # Production installation"
    echo "  $0 --demo       # Run demo with existing binaries"
    echo ""
}

build_binaries() {
    print_step "Building binaries..."
    
    if ! make build; then
        print_error "Build failed"
        exit 1
    fi
    
    print_success "All binaries built successfully"
    
    echo ""
    echo "Built binaries:"
    ls -la bin/ | grep -E "(portfolio|channel|telegram|forwarding|webhook)" | while read line; do
        echo "  📦 $(echo $line | awk '{print $9}')"
    done
}

install_services() {
    if [ "$SKIP_SERVICES" = true ]; then
        print_warning "Skipping service installation (systemctl not available)"
        return
    fi
    
    print_step "Installing systemd services..."
    
    # Use the new simplified installer
    if [ -x "deployment/scripts/install-simplified.sh" ]; then
        if deployment/scripts/install-simplified.sh; then
            print_success "Services installed successfully"
        else
            print_error "Service installation failed"
            exit 1
        fi
    else
        print_warning "Simplified installer not found, trying legacy method"
        if [ -x "deployment/scripts/install-services-auto.sh" ]; then
            deployment/scripts/install-services-auto.sh
            print_success "Services installed (legacy method)"
        else
            print_error "No service installer found"
            exit 1
        fi
    fi
}

install_cron_jobs() {
    if [ "$SKIP_CRON" = true ]; then
        print_warning "Skipping cron job installation (crontab not available)"
        return
    fi
    
    print_step "Installing cron jobs..."
    
    if [ -x "deployment/scripts/install-crontab-auto.sh" ]; then
        if deployment/scripts/install-crontab-auto.sh; then
            print_success "Cron jobs installed successfully"
        else
            print_error "Cron job installation failed"
            exit 1
        fi
    else
        print_warning "Auto crontab installer not found"
        print_warning "Please manually configure crontab using deployment/crontab.example"
    fi
}

create_directories() {
    print_step "Creating data directories..."
    
    mkdir -p data
    mkdir -p logs
    
    print_success "Directories created"
}

run_demo() {
    print_step "Running demo with mock data..."
    
    # Ensure binaries exist
    if [ ! -f "bin/portfolio-collector" ] || [ ! -f "bin/portfolio-api" ]; then
        print_error "Binaries not found. Run with --build-only first."
        exit 1
    fi
    
    # Test data collection
    echo "Testing data collection..."
    if ./bin/portfolio-collector --oneshot --mock; then
        print_success "Data collection test successful"
    else
        print_error "Data collection test failed"
        exit 1
    fi
    
    echo ""
    print_success "Demo setup complete!"
    echo ""
    echo -e "${YELLOW}🚀 To start the demo:${NC}"
    echo "   ./deployment/scripts/start-dashboard.sh"
    echo ""
    echo -e "${YELLOW}🌐 API will be available at:${NC}"
    echo "   http://localhost:8090"
    echo ""
    echo -e "${YELLOW}📋 API Endpoints:${NC}"
    echo "   http://localhost:8090/api/health"
    echo "   http://localhost:8090/api/portfolio/current"
    echo ""
}

start_services() {
    if [ "$SKIP_SERVICES" = true ]; then
        print_warning "Cannot start services (systemctl not available)"
        return
    fi
    
    # Check if systemctl is available
    if ! command -v systemctl &> /dev/null; then
        print_warning "systemctl not available - services cannot be started automatically"
        return
    fi
    
    print_step "Starting services..."
    
    # Enable and start services
    local services=("bitcoin-dashboard-api" "bitcoin-dashboard-collector" "bitcoin-forwarding-collector")
    
    for service in "${services[@]}"; do
        if systemctl --user is-enabled "$service" &>/dev/null; then
            systemctl --user start "$service" || true
            if systemctl --user is-active "$service" &>/dev/null; then
                print_success "$service started"
            else
                print_warning "$service failed to start (check logs: journalctl --user -u $service)"
            fi
        else
            print_warning "$service not enabled"
        fi
    done
}

show_status() {
    print_step "System status..."
    
    echo ""
    echo "📊 Portfolio Data:"
    if [ -f "data/portfolio.db" ]; then
        echo "   ✅ Database exists"
        local size=$(du -h data/portfolio.db | cut -f1)
        echo "   📈 Database size: $size"
    else
        echo "   📝 No database yet (run demo or collector)"
    fi
    
    echo ""
    echo "🔧 Services:"
    if ! command -v systemctl &> /dev/null; then
        echo "   ⚠️  systemctl not available (using cron jobs instead)"
    elif [ "$SKIP_SERVICES" != true ]; then
        local services=("bitcoin-dashboard-api" "bitcoin-dashboard-collector" "bitcoin-forwarding-collector")
        for service in "${services[@]}"; do
            if systemctl --user is-enabled "$service" &>/dev/null; then
                if systemctl --user is-active "$service" &>/dev/null; then
                    echo "   ✅ $service (running)"
                else
                    echo "   🔴 $service (installed but not running)"
                fi
            else
                echo "   📝 $service (not installed)"
            fi
        done
    else
        echo "   ⚠️  systemctl not available"
    fi
    
    echo ""
    echo "⏰ Cron Jobs:"
    if [ "$SKIP_CRON" != true ]; then
        if crontab -l 2>/dev/null | grep -q "lightning-node-tools"; then
            echo "   ✅ Cron jobs installed"
            local count=$(crontab -l 2>/dev/null | grep "lightning-node-tools" | wc -l)
            echo "   📋 $count jobs configured"
        else
            echo "   📝 No cron jobs installed"
        fi
    else
        echo "   ⚠️  crontab not available"
    fi
    
    echo ""
}

print_next_steps() {
    echo -e "${BLUE}🎯 Next Steps:${NC}"
    echo ""
    
    if [ "$SETUP_TYPE" = "demo" ] || [ "$SETUP_TYPE" = "quick" ]; then
        echo "1. Start the demo:"
        echo "   ./deployment/scripts/start-dashboard.sh"
        echo ""
        echo "2. Open in browser:"
        echo "   http://localhost:8090"
        echo ""
        echo "3. Try the API:"
        echo "   curl http://localhost:8090/api/health"
        echo ""
        echo "4. For production setup, run:"
        echo "   ./setup.sh --full"
    else
        echo "1. Configure your environment:"
        echo "   cp .env.example .env"
        echo "   # Edit .env with your settings"
        echo ""
        echo "2. Test with real data (requires LND):"
        echo "   ./bin/portfolio-collector --oneshot"
        echo ""
        if command -v systemctl &> /dev/null; then
            echo "3. Monitor services:"
            echo "   systemctl --user status bitcoin-dashboard-api"
            echo "   journalctl --user -f -u bitcoin-dashboard-api"
            echo ""
            echo "4. View data:"
            echo "   curl http://localhost:8090/api/portfolio/current"
        else
            echo "3. Start services manually (macOS):"
            echo "   ./bin/portfolio-api &"
            echo "   ./bin/portfolio-collector &"
            echo ""
            echo "4. View data:"
            echo "   curl http://localhost:8090/api/portfolio/current"
            echo ""
            echo "5. Cron jobs are handling periodic tasks"
            echo "   (monitoring, collection, backups)"
        fi
    fi
    echo ""
}

# Main script logic
main() {
    print_header
    
    # Parse arguments
    SETUP_TYPE=""
    BUILD_ONLY=false
    SERVICES_ONLY=false
    DEMO_ONLY=false
    SKIP_SERVICES=false
    SKIP_CRON=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --quick)
                SETUP_TYPE="quick"
                shift
                ;;
            --full)
                SETUP_TYPE="full"
                shift
                ;;
            --build-only)
                BUILD_ONLY=true
                shift
                ;;
            --services-only)
                SERVICES_ONLY=true
                shift
                ;;
            --demo)
                DEMO_ONLY=true
                SETUP_TYPE="demo"
                shift
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # Show usage if no arguments
    if [ -z "$SETUP_TYPE" ] && [ "$BUILD_ONLY" = false ] && [ "$SERVICES_ONLY" = false ] && [ "$DEMO_ONLY" = false ]; then
        show_usage
        exit 0
    fi
    
    # Check dependencies
    check_dependencies
    
    # Execute based on options
    if [ "$BUILD_ONLY" = true ]; then
        create_directories
        build_binaries
        print_success "Build completed successfully!"
        
    elif [ "$SERVICES_ONLY" = true ]; then
        install_services
        install_cron_jobs
        start_services
        print_success "Services setup completed!"
        
    elif [ "$DEMO_ONLY" = true ]; then
        create_directories
        run_demo
        
    elif [ "$SETUP_TYPE" = "quick" ]; then
        create_directories
        build_binaries
        run_demo
        
    elif [ "$SETUP_TYPE" = "full" ]; then
        create_directories
        build_binaries
        install_services
        start_services
        print_success "Full setup completed successfully!"
        
    fi
    
    # Show final status
    show_status
    print_next_steps
    
    print_success "Setup script completed!"
}

# Run main function with all arguments
main "$@"