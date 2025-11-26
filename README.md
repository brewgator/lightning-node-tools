# Lightning Node Tools

A comprehensive toolkit for Lightning Network node management and monitoring. This repository contains multiple tools designed to help Lightning node operators manage their channels, monitor activity, and optimize performance.

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

### Portfolio Dashboard

Web-based Bitcoin portfolio tracking dashboard with historical data visualization.

**Features:**
- Real-time portfolio balance display aggregating all sources
- Historical balance snapshots with SQLite storage
- Lightning Network and on-chain balance tracking
- Responsive web interface optimized for continuous monitoring

## Architecture

The project uses a modular architecture with shared packages:

- **`pkg/lnd/`**: Shared Lightning Network API client and data structures
- **`pkg/utils/`**: Common utility functions for formatting and calculations
- **`pkg/db/`**: Database operations and historical data management
- **`cmd/channel-manager/`**: Channel management tool implementation
- **`cmd/telegram-monitor/`**: Telegram monitoring tool implementation
- **`cmd/dashboard-collector/`**: Portfolio data collection service
- **`web/api/`**: Web API server and dashboard interface

## Quick Start

1. **Clone and configure**:
   ```bash
   git clone <your-repo-url>
   cd lightning-node-tools
   cp .env.example .env
   # Edit .env with your Telegram bot token and chat ID
   ```

2. **Build tools**:
   ```bash
   make
   ```

3. **Run channel manager**:
   ```bash
   ./bin/channel-manager balance    # View channel liquidity
   ./bin/channel-manager fees      # View current fees
   ./bin/channel-manager earnings  # View fee earnings
   ```

4. **Set up monitoring**:
   ```bash
   ./bin/telegram-monitor          # Test manually
   # Add to cron for automated monitoring:
   # */2 * * * * /path/to/lightning-node-tools/bin/telegram-monitor >/dev/null 2>&1
   ```

5. **Start portfolio dashboard**:
   ```bash
   ./start-dashboard.sh            # Launch web dashboard
   # Open http://localhost:8080 in your browser
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

- `make` or `make build` - Build all tools
- `make clean` - Remove build artifacts
- `make channel-manager` - Build only channel-manager
- `make telegram-monitor` - Build only telegram-monitor
- `make install` - Install tools to GOPATH/bin
- `make help` - Show all available targets

## Troubleshooting

- Ensure your Lightning node is running and `lncli` commands work
- Verify your `.env` file has correct bot token and chat ID
- Check that the `data` directory exists
- Test the Telegram bot by sending a manual message first

## Documentation

- **[ROADMAP.md](ROADMAP.md)** - Planned features and future development
- **[DASHBOARD.md](DASHBOARD.md)** - Portfolio Dashboard setup and usage guide

## Future Development

See [ROADMAP.md](ROADMAP.md) for planned features and future development.