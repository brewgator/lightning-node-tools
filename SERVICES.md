# Lightning Node Tools - Service Overview

This document explains each service in the Lightning Node Tools stack, their purposes, and how they work together.

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                   LIGHTNING NODE TOOLS SERVICES                │
├─────────────────────────────────────────────────────────────────┤
│ PERSISTENT SERVICES (systemd services - always running)        │
├─────────────────┬─────────────────┬─────────────────────────────┤
│ Portfolio API   │ Data Collector  │ Forwarding Collector        │
│ Port: 8090      │ Every 15min     │ Every 5min                  │
│ REST API        │ Portfolio data  │ Lightning forwards          │
└─────────────────┴─────────────────┴─────────────────────────────┘
┌─────────────────────────────────────────────────────────────────┐
│ DEPLOYMENT SERVICE (optional)                                   │
├─────────────────────────────────────────────────────────────────┤
│ Webhook Deployer - Port: 9000                                  │
│ Listens for GitHub pushes, auto-deploys updates                │
└─────────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────────┐
│ MONITORING SERVICE (periodic execution)                        │
├─────────────────────────────────────────────────────────────────┤
│ Telegram Monitor - Every 2 minutes via cron/timer              │
│ Sends alerts for balance changes, channel events, etc.         │
└─────────────────────────────────────────────────────────────────┘
```

## 📊 Service Details

### 1. **Portfolio API** (`bitcoin-dashboard-api.service`)
- **Binary**: `portfolio-api`
- **Type**: Persistent web service
- **Port**: 8090
- **Purpose**: 
  - Serves REST API for portfolio data
  - Provides web interface for viewing balances and charts
  - Handles onchain address management
  - Manages cold storage entries

**Key Features:**
- Real-time portfolio data access
- Historical data charts (Chart.js)
- Lightning fee analytics
- Onchain address tracking
- Cold storage management
- Mock mode for testing

**API Endpoints:**
```
GET  /api/health                    - Health check
GET  /api/portfolio/current         - Current portfolio snapshot
GET  /api/portfolio/history         - Historical portfolio data
GET  /api/lightning/fees            - Lightning fee earnings
GET  /api/lightning/forwards        - Lightning forwarding stats
GET  /api/onchain/addresses         - Tracked onchain addresses
POST /api/onchain/addresses         - Add new address to track
GET  /api/offline/accounts          - Cold storage accounts
```

---

### 2. **Portfolio Collector** (`bitcoin-dashboard-collector.service`)
- **Binary**: `portfolio-collector`
- **Type**: Persistent background daemon
- **Interval**: Every 15 minutes
- **Purpose**: 
  - Collects Lightning channel balances
  - Gathers onchain wallet balances
  - Tracks external address balances (via Mempool.space)
  - Updates cold storage totals
  - Stores historical snapshots in SQLite

**Data Sources:**
- **Lightning**: LND gRPC (channel balances, liquidity)
- **Onchain**: Bitcoin Core RPC (wallet balances)
- **External Addresses**: Mempool.space API (tracked addresses)
- **Cold Storage**: Manual entries (offline wallets)

**Key Features:**
- Automatic data collection every 15 minutes
- Supports mock mode for testing
- Tracks both confirmed and unconfirmed balances
- Historical trend storage
- Resilient error handling (continues on partial failures)

---

### 3. **Forwarding Collector** (`bitcoin-forwarding-collector.service`)
- **Binary**: `forwarding-collector`
- **Type**: Persistent background daemon  
- **Interval**: Every 5 minutes
- **Purpose**:
  - Collects Lightning forwarding events from LND
  - Tracks routing fees earned
  - Monitors channel activity
  - Provides data for fee optimization

**Key Features:**
- Collects forwarding history from LND
- Deduplicates events to avoid double-counting
- Tracks fees earned per channel
- Provides data for routing analytics
- Essential for channel fee optimization

---

### 4. **Webhook Deployer** (`webhook-deployer.service`) - Optional
- **Binary**: `webhook-deployer`
- **Type**: Persistent web service
- **Port**: 9000
- **Purpose**:
  - Listens for GitHub webhook events
  - Automatically deploys updates when you push to main branch
  - Handles authentication with HMAC-SHA256
  - Provides zero-downtime deployments

**Key Features:**
- GitHub webhook integration
- Secure HMAC signature verification
- Automatic git pull and rebuild
- Service restart after deployment
- Rollback capability on failure
- Health check endpoints

**Webhook URL**: `http://your-server:9000/webhook`

---

### 5. **Telegram Monitor** (`lightning-telegram-monitor.service`) - Periodic
- **Binary**: `telegram-monitor`
- **Type**: Oneshot (run periodically via cron/systemd timer)
- **Interval**: Every 2 minutes
- **Purpose**:
  - Monitors Lightning node for changes
  - Sends Telegram notifications for important events
  - Tracks balance changes, channel events, forwards
  - Server reboot detection

**Monitored Events:**
- **Balance Changes**: Significant balance movements
- **Channel Events**: Opens, closes, force closes
- **Forwarding Activity**: New routing events
- **System Events**: Server reboots, service starts
- **Invoices**: New invoice creation
- **Fee Changes**: Routing fee adjustments

**Alert Examples:**
```
🟢 Channel opened: 1,000,000 sats to Lightning_Peer
💰 Balance increased: +50,000 sats (routing fees)
🔄 32 forwards processed, earned 1,234 sats
⚠️  Server reboot detected at 14:30
```

## 🔧 Service Dependencies

```
                    ┌─────────────────┐
                    │   Database      │
                    │  (portfolio.db) │
                    └─────────────────┘
                             ▲
                             │
        ┌─────────────────────┼─────────────────────┐
        │                    │                     │
        ▼                    ▼                     ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│ Portfolio API   │ │ Portfolio       │ │ Forwarding      │
│ (Reads data)    │ │ Collector       │ │ Collector       │
│                 │ │ (Writes data)   │ │ (Writes data)   │
└─────────────────┘ └─────────────────┘ └─────────────────┘

┌─────────────────┐         ┌─────────────────┐
│ Telegram        │         │ Webhook         │
│ Monitor         │         │ Deployer        │
│ (Reads data)    │         │ (Independent)   │
└─────────────────┘         └─────────────────┘
```

## 🚀 Running Services

### All Services (Recommended)
```bash
./setup.sh --full
```

### Individual Services
```bash
# Start specific services
systemctl --user start bitcoin-dashboard-api
systemctl --user start bitcoin-dashboard-collector
systemctl --user start bitcoin-forwarding-collector
systemctl --user start webhook-deployer

# For telegram monitor (via timer)
systemctl --user enable lightning-telegram-monitor.timer
systemctl --user start lightning-telegram-monitor.timer

# Check status
systemctl --user status bitcoin-dashboard-api
journalctl --user -f -u bitcoin-dashboard-collector
```

### Manual Testing
```bash
# Test data collection
./bin/portfolio-collector --oneshot --mock

# Test API
./bin/portfolio-api --mock --port 8091 &
curl http://localhost:8091/api/health

# Test telegram monitor
./bin/telegram-monitor

# Test webhook deployer
./bin/webhook-deployer --port 9000 &
curl http://localhost:9000/health
```

## 📋 Resource Usage

| Service | CPU (idle) | Memory | Disk I/O | Network |
|---------|------------|--------|-----------|---------|
| Portfolio API | ~0.1% | ~10MB | Minimal | HTTP requests |
| Portfolio Collector | ~0.5% | ~15MB | SQLite writes | LND gRPC, RPC calls |
| Forwarding Collector | ~0.3% | ~8MB | SQLite writes | LND gRPC |
| Webhook Deployer | ~0.1% | ~5MB | Minimal | GitHub webhooks |
| Telegram Monitor | ~0.1% | ~6MB | SQLite reads | Telegram API |

## 🔍 Monitoring & Logs

### Service Status
```bash
# Check all Lightning Node Tools services
systemctl --user list-units --type=service | grep -E "(bitcoin|lightning|webhook)"

# View logs
journalctl --user -u bitcoin-dashboard-api -f
journalctl --user -u bitcoin-dashboard-collector --since "1 hour ago"
```

### Health Checks
```bash
# API health
curl http://localhost:8090/api/health

# Webhook deployer health  
curl http://localhost:9000/health

# Database status
sqlite3 data/portfolio.db ".tables"
```

## 🛠️ Configuration

Each service can be configured via:

1. **Environment variables** (`.env` file)
2. **Command line flags**
3. **Configuration files** (some services)

Key configuration options:
- Database path
- Collection intervals  
- API ports
- Mock mode
- Telegram credentials
- Webhook secrets

See `.env.example` for configuration options.