# 📊 Bitcoin Portfolio Dashboard

Real-time Bitcoin portfolio tracking dashboard that combines Lightning Network monitoring with on-chain balance tracking.

## 🚀 Quick Start

```bash
# Build the dashboard
make dashboard

# Start the dashboard (with demo data)
./scripts/start-dashboard.sh

# Open http://localhost:8080 in your browser
```

## 🏗️ Architecture

The dashboard consists of three main components:

### 1. Data Collector (`dashboard-collector`)
- Collects balance snapshots every 15 minutes (configurable)
- Integrates with existing LND client for Lightning data
- Tracks on-chain wallet balances
- Stores historical data in SQLite database

**Usage:**
```bash
# Run once with real LND data
./bin/dashboard-collector --oneshot

# Run once with mock data (for testing)
./bin/dashboard-collector --oneshot --mock

# Run continuously (every 15 minutes)
./bin/dashboard-collector

# Custom interval
./bin/dashboard-collector --interval 30m
```

### 2. Web API (`dashboard-api`)
- REST API serving portfolio data as JSON
- Built with Gorilla Mux and CORS support
- Serves static dashboard files

**Endpoints:**
- `GET /api/health` - Health check
- `GET /api/portfolio/current` - Latest portfolio snapshot
- `GET /api/portfolio/history?days=30` - Historical data
- `GET /` - Dashboard web interface

### 3. Web Dashboard
- Responsive single-page application
- Real-time portfolio balance display
- Dark theme optimized for continuous monitoring
- Auto-refreshes every 5 minutes

## 📱 Dashboard Features

### Current Portfolio View
- **Total Portfolio**: Combined Lightning + on-chain + cold storage
- **Total Liquid**: Active funds (excludes cold storage)
- **Lightning Network**: Local/remote balance split with capacity overview
- **On-chain**: Confirmed, unconfirmed, and tracked address balances

### Portfolio Breakdown
```
💰 Current Portfolio
├── Total Portfolio: 7.1M sats (0.071 BTC)
├── Total Liquid: 7.1M sats
└── Last Updated: 2025-11-26 15:08:46

⚡ Lightning Network  
├── Total Capacity: 8.0M sats
├── Local Balance: 5.0M sats (62.5%)
└── Remote Balance: 3.0M sats

🔗 On-chain
├── Total On-chain: 2.1M sats  
├── Confirmed: 2.0M sats
├── Unconfirmed: 100K sats
└── Tracked Addresses: 0 sats
```

## 🛠️ Configuration

### Database
- **Location**: `data/portfolio.db` (SQLite)
- **Schema**: Time-series balance snapshots with indexing
- **Retention**: Unlimited (manually managed)

### Collection Settings
```yaml
# configs/dashboard.yaml
collection:
  interval_minutes: 15

database:
  path: "./data/portfolio.db"

web:
  host: "0.0.0.0"
  port: 8080
```

## 📈 Data Model

### Balance Snapshot
```json
{
  "id": 1,
  "timestamp": "2025-11-26T15:08:46Z",
  "lightning_local": 5000000,
  "lightning_remote": 3000000, 
  "onchain_confirmed": 2000000,
  "onchain_unconfirmed": 100000,
  "tracked_addresses": 0,
  "cold_storage": 0,
  "total_portfolio": 7100000,
  "total_liquid": 7100000
}
```

## 🔌 Integration

### With Existing Tools
The dashboard reuses the existing Lightning Network client library:
- **Channel Manager**: Same LND integration
- **Telegram Monitor**: Complementary real-time alerts
- **Shared Utils**: Common formatting and utilities

### Production Deployment
```bash
# 1. Set up systemd service for data collection
sudo cp scripts/dashboard-collector.service /etc/systemd/system/
sudo systemctl enable dashboard-collector
sudo systemctl start dashboard-collector

# 2. Set up nginx reverse proxy
sudo cp configs/nginx-dashboard.conf /etc/nginx/sites-available/
sudo ln -s /etc/nginx/sites-available/dashboard /etc/nginx/sites-enabled/
sudo systemctl reload nginx

# 3. Configure firewall (optional)
sudo ufw allow 8080
```

## 🧪 Testing

### Mock Mode
Perfect for development and demonstrations:
```bash
# Generate sample data
./bin/dashboard-collector --oneshot --mock

# Start API with sample data
./bin/dashboard-api

# View dashboard at http://localhost:8080
```

### With Real LND
```bash
# Ensure LND is running and lncli works
lncli getinfo

# Run real data collection
./bin/dashboard-collector --oneshot

# Start full dashboard
./scripts/start-dashboard.sh
```

## 🔮 Future Enhancements

See [ROADMAP.md](ROADMAP.md) for planned features:

- **Mempool.space API Integration**: Track multiple Bitcoin addresses
- **Historical Charts**: Interactive time-series visualizations
- **Monthly Reports**: Automated portfolio summaries
- **Cold Storage Management**: Manual entry and tracking
- **Mobile App**: Native iOS/Android companion
- **Advanced Analytics**: Performance metrics and insights

## 🐛 Troubleshooting

### Common Issues

**Database locked error:**
```bash
# Stop any running collectors
pkill dashboard-collector

# Restart
./bin/dashboard-collector --oneshot
```

**API not responding:**
```bash
# Check if port is in use
lsof -i :8080

# Kill existing processes
pkill dashboard-api
```

**LND connection failed:**
```bash
# Test LND connectivity
lncli getinfo

# Use mock mode for testing
./bin/dashboard-collector --mock
```

## 📊 Dashboard Screenshots

The dashboard provides a clean, dark-themed interface optimized for continuous monitoring:

- **Portfolio Overview**: Real-time balance display with color-coded status
- **Lightning Details**: Channel capacity and liquidity distribution  
- **On-chain Summary**: Wallet balances and transaction status
- **Historical Placeholder**: Ready for chart integration

Perfect for displaying on a dedicated monitor for at-a-glance portfolio insights!
