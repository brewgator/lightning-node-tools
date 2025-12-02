# Lightning Node Tools

A comprehensive toolkit for Lightning Network node management and monitoring. This repository contains multiple tools designed to help Lightning node operators manage their channels, monitor activity, and optimize performance.

## Tools Overview

### Channel Manager

Advanced Lightning Network channel management and fee optimization tool with comprehensive analytics.

**Features:**

- **Visual Channel Balances**: Interactive display of channel liquidity with progress bars
- **Fee Management**: Set and optimize routing fees for individual channels or bulk operations
- **Earnings Analytics**: Track fee earnings with detailed per-channel breakdowns
- **Performance Monitoring**: Monitor channel activity and routing performance

### Telegram Monitor

Real-time Lightning node monitoring with Telegram notifications for critical events.

**Features:**

- **Real-time Lightning Node Monitoring**: Track channel opens/closes, pending operations, and balance changes
- **Telegram Notifications**: Get instant alerts about your Lightning node activity
- **Smart Balance Tracking**: Adaptive thresholds based on account size with precise change detection
- **Portfolio Focus**: Total balance excludes remote Lightning balances (only counts your actual funds)
- **Forward Monitoring**: Track routing fees and forward activity with detailed 24h summaries
- **Super Detailed Earnings**: Automated forwarding analysis with top earning channels and recent activity
- **Server Reboot Detection**: Get notified when your Lightning node server restarts

## Architecture

The project uses a modular architecture with shared packages for common functionality:

- **`pkg/lnd/`**: Shared Lightning Network API client and data structures
- **`pkg/utils/`**: Common utility functions for formatting and calculations
- **`cmd/channel-manager/`**: Channel management tool implementation
- **`cmd/telegram-monitor/`**: Telegram monitoring tool implementation

## Setup

1. **Clone the repository**:

   ```bash
   git clone <your-repo-url>
   cd lightning-node-tools
   ```

2. **Create your environment file**:

   ```bash
   cp .env.example .env
   ```

3. **Configure your Telegram bot**:
   - Message [@BotFather](https://t.me/botfather) on Telegram to create a new bot
   - Copy the bot token to your `.env` file
   - Get your chat ID by messaging your bot and visiting: `https://api.telegram.org/bot<YourBOTToken>/getUpdates`
   - Update the `.env` file with your actual values:

     ```
     BOT_TOKEN="your-actual-bot-token"
     CHAT_ID="your-actual-chat-id"
     ```

4. **Make the script executable**:

   ```bash
   chmod +x scripts/telegram-alerts.sh
   ```

5. **Create the data directory**:

   ```bash
   mkdir -p data
   ```

## Usage

### Build All Tools

**Simple one-command build:**
```bash
make
```

**Or use the traditional Go commands:**
```bash
go build -o bin/telegram-monitor ./cmd/telegram-monitor
go build -o bin/channel-manager ./cmd/channel-manager
```

**Available make targets:**
- `make` or `make build` - Build all tools
- `make clean` - Remove build artifacts
- `make channel-manager` - Build only channel-manager
- `make telegram-monitor` - Build only telegram-monitor
- `make install` - Install tools to GOPATH/bin
- `make help` - Show all available targets

### Telegram Monitor

1. **Build the Go program**:

   ```bash
   make telegram-monitor
   # or: go build -o bin/telegram-monitor ./cmd/telegram-monitor
   ```

2. **Run manually to test**:

   ```bash
   ./bin/telegram-monitor
   ```

3. **Set up automated monitoring with cron** (runs every 2 minutes):

   ```bash
   crontab -e
   ```

   Add this line (replace `/path/to/lightning-node-tools` with the actual path):

   ```crontab
   */2 * * * * /path/to/lightning-node-tools/bin/telegram-monitor >/dev/null 2>&1
   ```

### Channel Manager

The Channel Manager provides comprehensive Lightning Network channel analysis and monitoring capabilities.

#### Available Commands

**1. Show visual channel balances:**

```bash
./bin/channel-manager balance
# or short alias:
./bin/channel-manager bal
```

**2. Show channel fees information:**

```bash
./bin/channel-manager fees
```

**3. Show fee earnings summary:**

```bash
./bin/channel-manager earnings
```

**4. Show detailed earnings breakdown:**

```bash
./bin/channel-manager earnings --detailed
# or short alias:
./bin/channel-manager earnings -d
```

**5. Show super detailed earnings with forwarding event analysis:**

```bash
./bin/channel-manager earnings --super-detailed
# or short alias:
./bin/channel-manager earnings --super
```

**6. Set fees for a specific channel:**

```bash
./bin/channel-manager set-fees --channel-id 12345 --ppm 1 --base-fee 1000
# or just set PPM (preserves existing base fee and time lock delta):
./bin/channel-manager set-fees --channel-id 12345 --ppm 2
# or just set base fee (preserves existing PPM and time lock delta):
./bin/channel-manager set-fees --channel-id 12345 --base-fee 1500
```

*Note: The tool intelligently preserves existing channel policy values for any parameters not explicitly specified.*

**6. Set fees for all channels:**

```bash
./bin/channel-manager bulk-set-fees --ppm 1
# or with base fee:
./bin/channel-manager bulk-set-fees --ppm 2 --base-fee 1000
```

*Note: Like set-fees, bulk operations preserve existing values for unspecified parameters on each channel.*

**7. Analyze and suggest optimal fee adjustments:**

```bash
./bin/channel-manager suggest-fees
```

**8. Automatically optimize fees based on channel performance:**

```bash
# Preview changes without applying them
./bin/channel-manager fee-optimizer --dry-run

# Apply optimizations automatically
./bin/channel-manager fee-optimizer
```

#### Smart Fee Optimization

The Channel Manager includes intelligent fee optimization with automated suggestions based on channel performance, liquidity distribution, and routing activity.

#### Example Outputs

**Balance Overview:**

```text
🔋 Channel Liquidity Overview
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🟢 ACINQ:                     |#######-----------------------| 250K/750K
                               Capacity: 1.0M │ Local: 25.0% │ Public

🟢 Bitrefill:                 |##----------------------------| 50K/950K
                               Capacity: 1.0M │ Local: 5.0% │ Public

🔴 Offline Node:              |------------------------------| 0/500K
                               Capacity: 500K │ Local: 0.0% │ Private (Inactive)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Summary: 2/3 active channels | Total: 2.5M | Local: 300K | Remote: 2.2M
```

**Fees Overview:**

```text
💰 Channel Fees Overview
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Channel                          Channel ID           Base Fee     Fee Rate     Status
───────────────────────────────────────────────────────────────────────────────────────────────────
🟢 ACINQ:                        123456789012345678   1000 msat    1 ppm        Public
🟢 LN Big:                       234567890123456789   1000 msat    1 ppm        Public
🟢 Bitrefill:                    345678901234567890   1000 msat    1 ppm        Public
🟢 WalletOfSatoshi.com:          456789012345678901   1000 msat    1 ppm        Public
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Fee Summary:
   Today: 0 │ Week: 27 │ Month: 27
```

**Earnings Summary:**

```text
💸 Fee Earnings Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📅 Today:                  0
📊 Week:                  27
📈 Month:                 27
──────────────────────────────────────────────────
📉 Daily Avg:              3 (7-day)
📉 Daily Avg:              0 (30-day)
⚡ Channels:               6 active
```

**Detailed Earnings Breakdown:**

```text
💸 Fee Earnings Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📅 Today:                  0
📊 Week:                  27
📈 Month:                 27
──────────────────────────────────────────────────
📉 Daily Avg:              3 (7-day)
⚡ Channels:               6 active

📋 Detailed Channel Earnings (30 days)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Channel                          Earnings        Status
───────────────────────────────────────────────────────────────────────────
🟢 LN Big:                       21              Public
🟢 ACINQ:                        1               Public
🟢 Bitrefill:                    0               Public
🟢 WalletOfSatoshi.com:          0               Public
───────────────────────────────────────────────────────────────────────────
Total:                           22
```

**Smart Fee Optimization Output:**

```text
🔍 Analyzing channels for fee optimization opportunities...

💡 Fee Optimization Suggestions:
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
🔴 ACINQ                      │ ↗  50 ↗ 120 ppm │ 1.0M   │  85.2% │  2d │ high-cap-outbound
   └─ High-capacity outbound channel - competitive fees to attract large payments (reduced for recent activity)
🟡 Bitrefill                  │ ↘ 200 ↘  80 ppm │ 1.0M   │  15.3% │  5d │ high-cap-inbound
🟡 WalletOfSatoshi            │ → 100 → 100 ppm │ 500K   │  45.0% │ 12d │ balanced
🟢 LowCap Node                │ ↗ 150 ↗ 500 ppm │ 200K   │  60.0% │ 45d │ low-liquidity
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
📊 Summary: 1 high priority, 2 medium priority changes suggested

💡 Commands:
   ./bin/channel-manager fee-optimizer --dry-run    # Preview changes
   ./bin/channel-manager fee-optimizer             # Apply optimizations

🔑 Legend:
   🔴 High priority  🟡 Medium priority  🟢 Low priority
   ↗ Increase fees  ↘ Decrease fees  → No change
   Categories: high-cap-outbound, balanced, high-cap-inbound, low-liquidity
```

**Fee Optimizer Dry Run:**

```text
🧪 Running fee optimizer in dry-run mode (no changes will be applied)...

📊 Found 3 channels that would benefit from fee optimization:

🔧 Would update ACINQ: 50 → 120 ppm (high priority)
🔧 Would update Bitrefill: 200 → 80 ppm (medium priority)
🔧 Would update LowCap Node: 150 → 500 ppm (medium priority)

🧪 Dry run complete: 3 channels would be updated
💡 Run without --dry-run to apply changes
```

See [ROADMAP.md](ROADMAP.md) for planned features and future development.

## What Gets Monitored

- **Channel Events**: Opens, closes, pending operations
- **Balance Changes**: On-chain and Lightning balances with adaptive thresholds based on account size
- **Payment Activity**: New invoices and forwards
- **Server Status**: Reboot detection
- **Routing Fees**: Recent forwarding activity and fees earned

## Configuration

The telegram monitor uses adaptive balance change thresholds:
- **Very small accounts** (<100k sats): 1 sat minimum change detection
- **Small accounts** (<1M sats): 100 sats threshold
- **Medium accounts** (<10M sats): 1k sats threshold
- **Large accounts** (10M+ sats): 5k sats threshold

## Requirements

- Lightning Network node with `lncli` installed and configured
- Go 1.19+ for building the tools
- Telegram bot token and chat ID (for telegram-monitor)

### Legacy Bash Script Requirements

- `jq` for JSON parsing
- `bc` for mathematical calculations
- `curl` for Telegram API calls

##### Phase 2: Smart Fee Optimization ✅ **IMPLEMENTED**

- ✅ **Intelligent fee analysis based on channel performance and liquidity distribution**
- ✅ **Automated fee optimization with dry-run capability**
- ✅ **Multi-path routing optimization for large payments (500K+ sats)**
- ✅ **Performance-based fee adjustments using forwarding history**

**Features:**
- **Smart Channel Categorization**: Analyzes channels based on capacity, liquidity ratio, and activity
- **Revenue Optimization**: Balances competitive fees with earnings maximization
- **Large Payment Support**: Ensures multiple channels can route 500K+ sat payments
- **Activity-Based Adjustments**: Rewards active channels with competitive fees

**Available commands:**

```bash
./bin/channel-manager suggest-fees           # Analyze and suggest optimal fees
./bin/channel-manager fee-optimizer --dry-run # Preview fee optimizations
./bin/channel-manager fee-optimizer          # Apply automatic optimizations
```

##### Phase 3: Channel Rebalancing (Future)

- Automated liquidity rebalancing between channels
- Intelligent rebalancing suggestions based on channel performance
- Cost-aware rebalancing with fee optimization

Planned commands:

```bash
./bin/channel-manager rebalance --from-channel X --to-channel Y --amount Z
./bin/channel-manager suggest-rebalance      # Analyze and suggest optimal moves
./bin/channel-manager auto-rebalance         # Automated rebalancing based on policies
```

##### Phase 4: Advanced Analytics & Intelligence (Future)

- Deep channel performance analysis and health scoring
- Peer recommendations based on network flow analysis
- Historical trend analysis and predictive insights

Planned commands:

```bash
./bin/channel-manager analyze --channel X     # Performance metrics and insights
./bin/channel-manager health-check           # Identify problematic channels
./bin/channel-manager recommend-peers        # Suggest profitable channel partners
./bin/channel-manager forecast              # Predict future routing performance
```

These features will build upon the existing foundation to provide a comprehensive Lightning Network management solution comparable to tools like rebalance-lnd, charge-lnd, and lndmanage, while maintaining the clean, intuitive interface and Go-based performance advantages.

## Future Work: Bitcoin Portfolio Dashboard

A comprehensive Bitcoin portfolio tracking dashboard that combines Lightning Network monitoring with multi-source balance tracking is planned as a major expansion of this repository.

### Vision

**Problem**: Currently tracking Bitcoin portfolio manually across multiple sources (Lightning node, onchain addresses, cold storage) takes significant time and provides no historical visibility or optimization insights.

**Solution**: Automated dashboard that runs continuously, collects data at regular intervals, and provides actionable analytics for Lightning channel management and portfolio optimization.

### Planned Features

#### Data Collection & Storage
- **SQLite database** with historical balance snapshots
- **Lightning node integration** via existing LND client library
- **Onchain monitoring** via Mempool.space API for address tracking
- **Cold storage tracking** via manual entry support
- **Time-series data** collected every 15-30 minutes for trend analysis

#### Web Dashboard
- **FastAPI-powered** web interface with responsive design
- **Real-time portfolio balance** display aggregating all sources
- **Historical charts** showing balance trends (7d, 30d, 90d, all-time views)
- **Portfolio allocation** visualization (Lightning vs onchain vs cold storage)
- **Designed for continuous display** on a monitor for at-a-glance insights

#### Lightning Network Analytics
- **Channel health scoring** with performance metrics
- **Routing statistics** and forward success/failure tracking
- **Fee optimization insights** based on earnings and activity
- **Channel recommendations** for liquidity management
- **Top earning channels** analysis with detailed breakdowns

#### Monthly Portfolio Tracking
- **Automated monthly snapshots** capturing portfolio state
- **Month-over-month comparison** reports
- **Portfolio allocation trends** over time
- **CSV export functionality** for external analysis
- **Detailed breakdowns** of changes by source

#### Production Deployment
- **Systemd service** for data collection daemon
- **Nginx reverse proxy** configuration for web access
- **Docker Compose** support for containerized deployment
- **Mobile-responsive design** for on-the-go monitoring
- **Basic authentication** for secure access

### Technical Architecture

The dashboard will extend the existing repository structure:

```
lightning-node-tools/
├── cmd/
│   ├── channel-manager/       # Existing CLI tool
│   ├── telegram-monitor/      # Existing CLI tool
│   └── dashboard-collector/   # New: Data collection service
├── pkg/
│   ├── lnd/                   # Existing: LND client (reused)
│   ├── utils/                 # Existing: Common utilities
│   └── db/                    # New: Database operations
├── web/
│   ├── api/                   # New: FastAPI endpoints
│   ├── static/                # New: Dashboard UI assets
│   └── templates/             # New: HTML templates
└── configs/
    └── dashboard.yaml         # New: Dashboard configuration
```

### Why This Approach

**Builds on Existing Foundation**: Leverages the mature LND client library and monitoring code already developed for channel-manager and telegram-monitor.

**Comprehensive View**: Combines Lightning operations with full portfolio tracking for complete Bitcoin wealth management.

**Practical Application**: Solves a real problem - manual portfolio tracking is time-consuming and error-prone.

**Learning Goals**: Provides hands-on experience with:
- Time-series database design
- Web API development with FastAPI
- Frontend visualization
- Production deployment practices
- Multi-source data aggregation

**Open Source Value**: Creates a unified tool for Lightning operators who want both channel management and portfolio tracking in one place.

### Real-World Use Cases

1. **Portfolio Overview**: "How's my total Bitcoin portfolio doing right now?"
2. **Lightning Optimization**: "Which channels are earning? Which are underperforming?"
3. **Trend Analysis**: "Is my Lightning balance growing or shrinking over time?"
4. **Allocation Management**: "Am I over/under-exposed in Lightning vs cold storage?"
5. **Monthly Reporting**: "What changed in my portfolio this month?"

This dashboard represents the next evolution of this toolkit - moving from discrete command-line tools to a comprehensive, always-on portfolio management system.

### Bash Script (Legacy)

You can still use the original bash script:

1. **Run manually to test**:

   ```bash
   ./scripts/telegram-alerts.sh
   ```

2. **Set up with cron**:

   ```bash
   */2 * * * * /path/to/lightning-node-tools/scripts/telegram-alerts.sh >/dev/null 2>&1
   ```

## What Gets Monitored

- **Channel Events**: Opens, closes, pending operations
- **Balance Changes**: On-chain and Lightning balances with adaptive thresholds based on account size
- **Payment Activity**: New invoices and forwards
- **Server Status**: Reboot detection
- **Routing Fees**: Recent forwarding activity and fees earned

## Configuration

The telegram monitor uses adaptive balance change thresholds based on account size:

- **Very small accounts** (<100k sats): 1 sat minimum change detection
- **Small accounts** (<1M sats): 100 sats threshold
- **Medium accounts** (<10M sats): 1k sats threshold
- **Large accounts** (10M+ sats): 5k sats threshold
- **Portfolio changes**: Uses higher thresholds (2x individual thresholds or 1M sats minimum)

**Key improvements:**

- Total portfolio only includes on-chain and local Lightning balances (excludes remote balances)
- Precise balance change formatting shows exact satoshis for small amounts
- Eliminates false "identical balance" notifications for small accounts

## Requirements

- Lightning Network node with `lncli` installed and configured
- Go 1.19+ for building the tools
- Telegram bot token and chat ID (for telegram-monitor)

### Legacy Bash Script Requirements

- `jq` for JSON parsing
- `bc` for mathematical calculations
- `curl` for Telegram API calls

## Project Structure

```text
lightning-node-tools/
├── cmd/
│   ├── channel-manager/          # Channel management tool
│   │   ├── main.go              # Main entry point and command routing
│   │   ├── types.go             # Tool-specific data structures
│   │   ├── client.go            # LND client wrapper
│   │   ├── fees.go              # Fee management functionality
│   │   ├── earnings.go          # Earnings analysis
│   │   ├── balance.go           # Balance display
│   │   └── utils.go             # Tool-specific utilities
│   └── telegram-monitor/         # Telegram monitoring tool
│       ├── main.go              # Main entry point
│       ├── types.go             # Tool-specific data structures
│       ├── client.go            # LND client wrapper
│       ├── monitor.go           # Monitoring logic
│       ├── telegram.go          # Telegram API integration
│       └── utils.go             # Tool-specific utilities
├── pkg/
│   ├── lnd/                     # Shared Lightning Network functionality
│   │   ├── client.go            # LND API client functions
│   │   └── types.go             # Common LND data structures
│   └── utils/                   # Shared utility functions
│       └── format.go            # Satoshi formatting utilities
├── bin/                         # Compiled binaries (after building)
│   ├── channel-manager
│   └── telegram-monitor
├── data/                        # Runtime data storage
│   ├── last_state.json          # Previous state for comparison
│   └── last_uptime.txt          # Server uptime tracking
├── .env                         # Configuration (not tracked by git)
├── .env.example                 # Configuration template
└── scripts/telegram-alerts.sh           # Legacy bash monitoring script
```

## Troubleshooting

- Ensure your Lightning node is running and `lncli` commands work
- Verify your `.env` file has correct bot token and chat ID
- Check that the `data` directory exists
- Test the Telegram bot by sending a manual message first
