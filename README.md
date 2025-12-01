# Lightning Node Tools

A comprehensive Bitcoin portfolio tracking and Lightning Network management toolkit. This project provides real-time monitoring, historical data collection, and web-based visualization for Lightning node operators who want complete visibility into their Bitcoin holdings.

## 🎯 Project Vision

Building a unified Bitcoin portfolio dashboard that combines Lightning Network monitoring with multi-source balance tracking. The goal is to replace manual spreadsheet tracking with automated data collection and beautiful visualizations.

**Status:** Phase 2 in progress - Basic dashboard operational with real-time portfolio tracking

## Tools Overview

### Channel Manager

Advanced Lightning Network channel management and fee optimization tool with comprehensive analytics.

**Features:**
- Visual channel balances with interactive liquidity display
- Fee management and optimization with smart suggestions
- Earnings analytics with detailed per-channel breakdowns
- Performance monitoring and routing statistics

### Telegram Monitor

Real-time Lightning node monitoring with Telegram notifications for critical events.

**Features:**
- Real-time Lightning node monitoring with instant alerts
- Smart balance tracking with adaptive thresholds
- Forward monitoring with detailed 24h summaries
- Server reboot detection and notifications

### Portfolio Dashboard 🚀

Comprehensive Bitcoin portfolio tracking dashboard with automated data collection and web visualization.

**Current Features (Phase 1-2 Complete):**
- ✅ **Automated Data Collection**: SQLite-backed service collecting Lightning + on-chain data every 15 minutes
- ✅ **Real-time Dashboard**: Web interface showing total portfolio breakdown across all sources
- ✅ **Lightning Integration**: Reuses existing LND client for seamless channel and wallet data
- ✅ **Historical Storage**: Time-series snapshots with indexed queries for trend analysis
- ✅ **Mock Mode**: Demo functionality for testing without live LND connection
- ✅ **Configuration Management**: YAML-based settings for collection intervals and data sources

**Planned Features (Phase 3+):**
- 📈 Interactive historical charts with Chart.js
- 🏦 Multiple on-chain address tracking via Mempool.space API
- 📊 Monthly portfolio reports with CSV export
- 💾 Cold storage balance management
- 📱 Mobile-optimized progressive web app
- 🔍 Lightning routing analytics and fee optimization

## Architecture

The project uses a modular, service-oriented architecture:

### Core Services
- **`cmd/dashboard-collector/`**: Automated data collection service (15min intervals)
- **`web/api/`**: REST API server with dashboard web interface 
- **`cmd/channel-manager/`**: Interactive channel management CLI tool
- **`cmd/telegram-monitor/`**: Real-time alerting and notifications

### Shared Infrastructure
- **`pkg/lnd/`**: Lightning Network API client (shared across all tools)
- **`pkg/db/`**: SQLite database operations for historical data
- **`pkg/utils/`**: Common utilities for formatting and calculations

### Data Flow
```
LND Node → dashboard-collector → SQLite DB → Web API → Dashboard UI
    ↓
Telegram Alerts (real-time)
Channel Manager (on-demand)
```

## Quick Start

### 🚀 Portfolio Dashboard (Recommended)

```bash
# 1. Clone and build
git clone <your-repo-url>
cd lightning-node-tools
make

# 2. Start with demo data
./bin/dashboard-collector --oneshot --mock
./start-dashboard.sh
# Open http://localhost:8080

# 3. Use with real LND data
./bin/dashboard-collector --oneshot  # Test collection
./bin/dashboard-collector             # Run continuously
```

### ⚡ Channel Management

```bash
# Build and configure
make
cp .env.example .env  # Add Telegram credentials if using alerts

# Channel operations
./bin/channel-manager balance    # Visual liquidity overview
./bin/channel-manager fees       # Current fee settings  
./bin/channel-manager earnings   # Fee earnings analysis

# Fee optimization
./bin/channel-manager suggest-fees     # AI-powered recommendations
./bin/channel-manager fee-optimizer    # Apply optimizations
```

### 📱 Real-time Monitoring

```bash
# Set up Telegram alerts
./bin/telegram-monitor           # Test manually

# Add to cron for continuous monitoring
*/2 * * * * /path/to/lightning-node-tools/bin/telegram-monitor >/dev/null 2>&1
```

## Channel Manager Commands

**Basic Operations:**
```bash
./bin/channel-manager balance     # Visual channel liquidity overview
./bin/channel-manager fees       # Current fee settings
./bin/channel-manager earnings   # Fee earnings summary
./bin/channel-manager earnings -d # Detailed earnings breakdown
```

**Fee Management:**
```bash
# Set fees for specific channel
./bin/channel-manager set-fees --channel-id 12345 --ppm 1 --base-fee 1000

# Set fees for all channels
./bin/channel-manager bulk-set-fees --ppm 1

# Smart fee optimization
./bin/channel-manager suggest-fees           # Analyze and suggest optimal fees
./bin/channel-manager fee-optimizer --dry-run # Preview optimizations
./bin/channel-manager fee-optimizer          # Apply optimizations
```

## Smart Fee Optimization

The Channel Manager includes intelligent fee optimization with automated suggestions based on channel performance, liquidity distribution, and routing activity.

## Configuration

The telegram monitor uses adaptive balance change thresholds:
- **Very small accounts** (<100k sats): 1 sat minimum change detection
- **Small accounts** (<1M sats): 100 sats threshold
- **Medium accounts** (<10M sats): 1k sats threshold
- **Large accounts** (10M+ sats): 5k sats threshold

## What Gets Monitored

- **Channel Events**: Opens, closes, pending operations
- **Balance Changes**: On-chain and Lightning balances with adaptive thresholds
- **Payment Activity**: New invoices and forwards
- **Server Status**: Reboot detection
- **Routing Fees**: Recent forwarding activity and fees earned

## Requirements

- Lightning Network node with `lncli` installed and configured
- Go 1.19+ for building the tools
- Telegram bot token and chat ID (for telegram-monitor)

## Project Structure

```text
lightning-node-tools/
├── cmd/
│   ├── channel-manager/          # Channel management tool
│   └── telegram-monitor/         # Telegram monitoring tool
├── pkg/
│   ├── lnd/                     # Shared Lightning Network functionality
│   └── utils/                   # Shared utility functions
├── bin/                         # Compiled binaries (after building)
├── data/                        # Runtime data storage
├── .env                         # Configuration (not tracked by git)
└── telegram-alerts.sh           # Legacy bash monitoring script
```

## Build Targets

**Primary:**
- `make` or `make build` - Build all tools
- `make dashboard` - Build dashboard components (collector + API)
- `make clean` - Remove build artifacts
- `make help` - Show all available targets

**Individual Tools:**
- `make channel-manager` - Build channel management tool
- `make telegram-monitor` - Build monitoring tool
- `make dashboard-collector` - Build data collection service
- `make dashboard-api` - Build web API server
- `make install` - Install tools to GOPATH/bin

## Troubleshooting

- Ensure your Lightning node is running and `lncli` commands work
- Verify your `.env` file has correct bot token and chat ID
- Check that the `data` directory exists
- Test the Telegram bot by sending a manual message first

## 📋 Project Status & Roadmap

**Current Phase:** Phase 2 - Basic Dashboard (🟡 In Progress)

| Phase | Status | Key Features |
|-------|--------|-------------|
| **Phase 1: Data Foundation** | ✅ Complete | SQLite schema, automated collection, LND integration |
| **Phase 2: Basic Dashboard** | 🟡 In Progress | Web API, real-time UI, portfolio breakdown |
| **Phase 3: Portfolio Integration** | ⏳ Planned | Multi-address tracking, cold storage management |
| **Phase 4: Monthly Tracking** | ⏳ Planned | Historical charts, monthly reports, CSV export |
| **Phase 5: Lightning Analytics** | ⏳ Planned | Channel health scoring, routing optimization |
| **Phase 6: Mobile & Polish** | ⏳ Planned | PWA, mobile responsiveness, production deployment |

**Next Milestones:**
- [ ] Complete historical chart integration (Chart.js)
- [ ] Add Mempool.space API for address tracking
- [ ] Implement monthly report generation

## Documentation

- **[Detailed Roadmap](https://github.com/user/obsidian-vault/.../Bitcoin%20Portfolio%20Dashboard%20Roadmap.md)** - Complete 6-phase development plan
- **[DASHBOARD.md](DASHBOARD.md)** - Portfolio dashboard setup and usage guide  
- **[ROADMAP.md](ROADMAP.md)** - High-level planned features

## 🧪 Testing & Development

**Mock Mode:** Test all features without a live Lightning node
```bash
./bin/dashboard-collector --mock --oneshot  # Generate sample data
./start-dashboard.sh                        # View dashboard
```

**Real Data:** Connect to your Lightning node
```bash
# Ensure LND is running and lncli works
lncli getinfo

# Start data collection
./bin/dashboard-collector --oneshot
```
