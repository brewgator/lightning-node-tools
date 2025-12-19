# Lightning Node Tools

Bitcoin/Lightning portfolio tracking, channel management, and monitoring toolkit.

## Quick Start

```bash
# Demo with mock data
./setup.sh --quick

# Production setup  
./setup.sh --full
```

Open http://localhost:8090 for the dashboard.

## What It Does

**Portfolio Tracking**
- Real-time Lightning + onchain + cold storage balances
- Historical charts and trends
- REST API and web interface

**Channel Management** 
- Visual channel liquidity display
- Smart fee optimization
- Bulk fee operations

**Monitoring**
- Telegram alerts for balance/channel changes
- Forwarding analytics
- Auto-deployment via webhooks

## Services

- **Portfolio API** (port 8090) - Web dashboard and REST API
- **Portfolio Collector** - Data collection every 15min
- **Forwarding Collector** - Lightning routing analytics  
- **Telegram Monitor** - Real-time notifications
- **Webhook Deployer** (port 9000) - Auto-deployment

## Configuration

```bash
cp .env.example .env
# Add your Telegram bot token
```

## Usage

```bash
# Channel management
./bin/channel-manager balance
./bin/channel-manager fees

# Manual data collection
./bin/portfolio-collector --oneshot

# API endpoints
curl http://localhost:8090/api/health
curl http://localhost:8090/api/portfolio/current
```

## Requirements

- Go 1.20+
- Lightning node with lncli
- Bitcoin Core (optional)
- Telegram bot (for alerts)

## Documentation

- [Service Details](SERVICES.md)
- [Deployment Guide](deployment/DEPLOYMENT.md)