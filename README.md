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

### Quick Setup

**1. Server Setup:**
```bash
# One-command installation
sudo ./scripts/setup-auto-deploy.sh
```

**2. GitHub Webhook:**
- Go to repo **Settings → Webhooks → Add webhook**
- **Payload URL**: `http://YOUR_SERVER_IP:9000/webhook`  
- **Content type**: `application/json`
- **Secret**: (copy from setup script output)
- **Events**: "Just the push event"

**3. Test Deployment:**
```bash
# Health check
curl http://YOUR_SERVER_IP:9000/health

# Full test suite
sudo ./scripts/test-webhook.sh all
```

### Features
- ✅ **HMAC-SHA256 verification** - Only authentic GitHub webhooks accepted
- ✅ **Automatic deployment** - Pull, test, build, restart on main branch push
- ✅ **Rollback protection** - Automatic rollback on failure
- ✅ **Health monitoring** - Service verification and status endpoints
- ✅ **Security hardening** - Dedicated user, restricted permissions

### Deployment Process
When you push to `main`:
1. **GitHub webhook** → Your server receives HMAC-verified request
2. **Backup created** → Current version saved for rollback
3. **Code updated** → `git pull` latest changes  
4. **Testing** → `make test` ensures quality
5. **Building** → `make build` compiles binaries
6. **Service restart** → All Lightning services restarted
7. **Health checks** → Verify services are healthy
8. **Rollback** → Automatic rollback if any step fails

### Service Management
```bash
# Check webhook service
sudo systemctl status webhook-deployer
sudo journalctl -u webhook-deployer -f

# Monitor deployments
tail -f /var/log/lightning-deploy.log

# Test webhook manually
sudo ./scripts/test-webhook.sh webhook
```

## 🧪 Testing & CI/CD

### Local Development
**Before pushing code:**
```bash
make ci-ready               # Full CI validation
make test                   # Run all tests
make test-race             # Race condition detection
make fmt                   # Format code

# Mock mode testing
make test-mock             # Test with mock data
```

### GitHub Actions Workflows
**Available workflows:**
- **`test.yml`** ⚡ - Basic CI (recommended) - Single Go version, fast
- **`simple-ci.yml`** 🔧 - Multi-version testing - Go 1.24 & 1.25 matrix
- **`ci.yml`** 🚀 - Full pipeline - Advanced linting and security
- **`deploy.yml`** 🚀 - Auto-deployment coordination with webhooks

**Quality Gates:**
- ✅ Code formatting (gofmt)
- ✅ Code quality (go vet)  
- ✅ Unit tests with race detection
- ✅ Build verification
- ✅ Coverage reporting with Codecov
- ✅ Multi-version Go compatibility (1.24 & 1.25)

**Auto-deployment:**
- ✅ Tests must pass before deployment
- ✅ Automatic rollback on failure
- ✅ Health verification after deployment

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
- Advanced Lightning routing analytics# Testing updated Tailscale auth key
