# Lightning Node Tools Roadmap

## Channel Manager Future Development

### Phase 2: Smart Fee Optimization ✅ **IMPLEMENTED**

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

### Phase 3: Channel Rebalancing (Future)

- Automated liquidity rebalancing between channels
- Intelligent rebalancing suggestions based on channel performance
- Cost-aware rebalancing with fee optimization

Planned commands:

```bash
./bin/channel-manager rebalance --from-channel X --to-channel Y --amount Z
./bin/channel-manager suggest-rebalance      # Analyze and suggest optimal moves
./bin/channel-manager auto-rebalance         # Automated rebalancing based on policies
```

### Phase 4: Advanced Analytics & Intelligence (Future)

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

## Bitcoin Portfolio Dashboard

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