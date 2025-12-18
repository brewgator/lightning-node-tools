# Lightning Node Tools

Lightning Network node management toolkit with portfolio tracking, channel management, and auto-deployment.

## 🏗️ Architecture

```
lightning-node-tools/
├── services/                    # Long-running services (systemd)
│   ├── portfolio/              
│   │   ├── api/               # REST API server (port 8090)
│   │   └── collector/         # Data collection service (15min intervals)
│   ├── lightning/             
│   │   └── forwarding-collector/  # Forwarding events collector
│   └── deployment/            
│       └── webhook-deployer/  # Auto-deploy webhook (port 9000)
│
├── tools/                      # CLI utilities & cron jobs
│   ├── channel-manager/       # Lightning channel management
│   └── monitoring/            # Telegram monitoring (cron every 2min)
│
├── internal/                   # Shared internal packages
│   ├── db/                    # Database operations
│   ├── lnd/                   # Lightning client
│   └── utils/                 # Common utilities
│
├── deployment/                 # Infrastructure & deployment
│   ├── systemd/               # Service files
│   ├── scripts/               # Automation scripts  
│   ├── configs/               # Configuration files
│   ├── crontab.example        # Cron job template
│   └── DEPLOYMENT.md          # Detailed deployment guide
│
└── web/                        # Web assets
    ├── static/
    └── templates/
```

## 🚦 Runtime Overview

### Always Running (Systemd Services)
- **bitcoin-dashboard-api** → Portfolio REST API (port 8090)
- **bitcoin-dashboard-collector** → Data collection every 15 minutes  
- **bitcoin-forwarding-collector** → Lightning forwarding events
- **webhook-deployer** → Auto-deployment server (port 9000)

### Scheduled Tasks (Cron Jobs)
- **Daily 2:00 AM** → Channel backups (`lncli exportchanbackup`)
- **Weekly Sun 2:15 AM** → Fee optimization (`tools/channel-manager/`)
- **Every 2 minutes** → Telegram monitoring (`tools/monitoring/`)
- **Daily 3:00 AM** → Log cleanup & backup rotation

### On-Demand Tools
- **tools/channel-manager/** → Interactive channel management CLI
- **tools/monitoring/** → Manual monitoring checks

## 🚀 Quick Start

```bash
# Build all tools
make build

# Start portfolio dashboard with sample data
./bin/dashboard-collector --oneshot --mock
./deployment/scripts/start-dashboard.sh
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

**Services Running:**
- `bitcoin-dashboard-api.service` → REST API server
- `bitcoin-dashboard-collector.service` → Continuous data collection

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

**Runtime:**
- **Cron Job:** Weekly fee optimization (Sundays 2:15 AM)
- **On-Demand:** Interactive CLI tool

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

## 📱 Telegram Monitor

Real-time Lightning node monitoring with Telegram alerts.

**Features:**
- Balance change notifications with adaptive thresholds
- Channel open/close alerts and forward monitoring
- Server reboot detection and earnings summaries

**Runtime:**
- **Cron Job:** Every 2 minutes monitoring (`*/2 * * * *`)
- **On-Demand:** Manual checks

**Setup:**
```bash
# Configure Telegram bot (see .env.example)
./bin/telegram-monitor

# Add to cron for continuous monitoring (already included in deployment/crontab.example)
```

## 🤖 Auto-Deployment

GitHub webhook-based auto-deployment system for production servers.

**Service Running:**
- `webhook-deployer.service` → Webhook server on port 9000

### Quick Setup

**1. Server Setup:**
```bash
# One-command installation
sudo ./deployment/scripts/setup-auto-deploy.sh
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
sudo ./deployment/scripts/test-webhook.sh all
```

### Features
- ✅ **HMAC-SHA256 verification** - Only authentic GitHub webhooks accepted
- ✅ **Automatic deployment** - Pull, test, build, restart on main branch push
- ✅ **Rollback protection** - Automatic rollback on failure
- ✅ **Health monitoring** - Service verification and status endpoints
- ✅ **Security hardening** - Dedicated user, restricted permissions

## 🛠️ Production Deployment

### Quick Deploy
```bash
# Install services
make install-services

# Install cron jobs  
cp deployment/crontab.example /tmp/mycron
# Edit paths to match your setup
nano /tmp/mycron
crontab /tmp/mycron

# Deploy updates
make deploy
```

### Service Management
```bash
# Check all services
systemctl --user list-units --type=service | grep -E "(bitcoin|webhook)"

# Individual service status
systemctl --user status bitcoin-dashboard-api
systemctl --user status bitcoin-dashboard-collector  
systemctl --user status bitcoin-forwarding-collector
systemctl --user status webhook-deployer

# View logs
journalctl --user -u bitcoin-dashboard-api -f
```

### API Endpoints
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

## 🧪 Testing & CI/CD

### Local Development
```bash
make ci-ready               # Full CI validation
make test                   # Run all tests
make test-race             # Race condition detection
make fmt                   # Format code
```

### Deployment Files
- **deployment/systemd/** → Service templates
- **deployment/crontab.example** → Cron job template  
- **deployment/DEPLOYMENT.md** → Complete deployment guide

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

---

📋 **See [deployment/DEPLOYMENT.md](deployment/DEPLOYMENT.md) for detailed setup instructions**
📋 **See [deployment/crontab.example](deployment/crontab.example) for cron configuration**