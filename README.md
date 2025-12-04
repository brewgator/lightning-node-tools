# Lightning Node Tools

Lightning Network node management toolkit with portfolio tracking, channel management, and auto-deployment.

## 🚀 Quick Start

```bash
# Build tools
make

# Start portfolio dashboard with sample data
./bin/dashboard-collector --oneshot --mock
./scripts/start-dashboard.sh
# Open http://localhost:8080

# Channel management
./bin/channel-manager balance
./bin/channel-manager fees

# Real data (requires LND)
./bin/dashboard-collector --oneshot
```

## 📊 Portfolio Dashboard

Real-time Bitcoin portfolio tracking with Lightning Network and on-chain monitoring.

**Features:**
- ✅ Real-time portfolio overview (Lightning + on-chain + cold storage)
- ✅ Interactive Chart.js visualizations 
- ✅ Historical data collection every 15 minutes
- ✅ Mock mode for testing/demos
- ✅ REST API with web interface

**Usage:**
```bash
# Data collection
./bin/dashboard-collector --oneshot          # One-time collection
./bin/dashboard-collector                    # Continuous collection

# Web API
./bin/dashboard-api                          # Start API server
curl http://localhost:8090/api/portfolio/current

# Test with mock data
./bin/dashboard-collector --oneshot --mock
./bin/dashboard-api --mock --port=8081
```

## ⚡ Channel Manager

Advanced Lightning channel management with smart fee optimization.

**Features:**
- Visual channel balances and liquidity display
- Smart fee optimization with AI-powered suggestions
- Fee earnings analytics and performance monitoring
- Bulk operations for managing multiple channels

**Commands:**
```bash
./bin/channel-manager balance                # Visual liquidity overview
./bin/channel-manager fees                  # Current fee settings
./bin/channel-manager earnings              # Fee earnings analysis

# Fee optimization
./bin/channel-manager suggest-fees          # Analyze optimal fees
./bin/channel-manager fee-optimizer --dry-run  # Preview changes
./bin/channel-manager fee-optimizer         # Apply optimizations
```

## 🤖 Auto-Deployment

GitHub webhook-based auto-deployment system for production servers.

**Setup:**
```bash
# One-command server setup
sudo ./scripts/setup-auto-deploy.sh

# Configure GitHub webhook:
# URL: http://YOUR_SERVER_IP:9000/webhook
# Secret: (displayed by setup script)
```

**Features:**
- ✅ HMAC-SHA256 webhook verification
- ✅ Automatic git pull, test, build, restart
- ✅ Rollback on failure with health checks
- ✅ Systemd service management
- ✅ Comprehensive logging and monitoring

## 🧪 Testing & CI/CD

**Local Testing:**
```bash
make test                    # Run all tests
make test-race              # Race condition detection
make ci-ready               # Full CI validation

# Mock mode testing
make test-mock              # Test with mock data
```

**CI/CD:**
- ✅ GitHub Actions with Go 1.24 & 1.25
- ✅ Automated testing, formatting, security checks
- ✅ Coverage reporting and quality gates
- ✅ Auto-deployment on main branch pushes

## 📱 Telegram Monitor

Real-time Lightning node monitoring with Telegram alerts.

**Features:**
- Balance change notifications with adaptive thresholds
- Channel open/close alerts and forward monitoring
- Server reboot detection and earnings summaries

**Setup:**
```bash
# Configure Telegram bot (see .env.example)
./bin/telegram-monitor

# Add to cron for continuous monitoring
*/2 * * * * /path/to/telegram-monitor >/dev/null 2>&1
```

## 🛠️ Production Deployment

**Systemd Services:**
```bash
# Install services
make install-services

# Deploy updates
make deploy

# Service management
sudo systemctl status bitcoin-dashboard-api
sudo systemctl status bitcoin-dashboard-collector
sudo systemctl status bitcoin-forwarding-collector
```

**API Endpoints:**
```bash
curl "http://localhost:8090/api/health"
curl "http://localhost:8090/api/portfolio/current"
curl "http://localhost:8090/api/lightning/fees?days=7"
curl "http://localhost:8090/api/lightning/forwards?days=7"
```

## 🔧 Configuration

**Environment Setup:**
```bash
cp .env.example .env
# Add Telegram credentials and LND settings
```

**Mock Mode:**
All tools support `--mock` flag for testing without live LND connection.

**Build Targets:**
```bash
make                        # Build all tools
make dashboard             # Build dashboard components
make deploy                # Production deployment
make clean                 # Clean build artifacts
```

## 📋 Architecture

```
lightning-node-tools/
├── cmd/                   # Application binaries
│   ├── channel-manager/   # Channel management tool
│   ├── dashboard-api/     # REST API server
│   ├── dashboard-collector/   # Data collection service
│   ├── forwarding-collector/  # Forwarding events collector
│   ├── telegram-monitor/  # Telegram monitoring
│   └── webhook-deployer/  # Auto-deployment service
├── pkg/                   # Shared packages
│   ├── db/               # Database operations
│   ├── lnd/              # Lightning Network client
│   └── utils/            # Common utilities
├── scripts/              # Automation scripts
├── systemd/              # Service templates
└── web/static/           # Dashboard web interface
```

## 🔒 Security

**Auto-deployment security:**
- HMAC-SHA256 webhook signature verification
- Service isolation with dedicated user
- Automatic rollback on deployment failure
- Restricted systemd permissions

**Best practices:**
- Never commit secrets or API keys
- Use mock mode for testing
- Proper file permissions and service hardening

## 📞 Requirements

- Lightning Network node with `lncli` installed
- Go 1.24+ for building
- SQLite for data storage
- Telegram bot token (for monitoring)
- Systemd for production services

## 🎯 Status

**✅ Complete:**
- Portfolio dashboard with Chart.js visualizations
- Smart fee optimization with AI suggestions  
- Mock mode for isolated testing
- Comprehensive test suite with 80%+ coverage
- GitHub Actions CI/CD with auto-deployment
- Production systemd service deployment

**🔮 Planned:**
- Mempool.space API integration for address tracking
- Monthly portfolio reports with CSV export
- Mobile-responsive PWA
- Advanced Lightning routing analytics