# Deployment Guide

## Quick Deploy

```bash
# Automated setup (recommended)
./setup.sh --full

# Legacy method
make install-all
```

## Services

**Systemd Services (always running):**
- `bitcoin-dashboard-api` - Portfolio API (port 8090)
- `bitcoin-dashboard-collector` - Data collection (15min intervals)  
- `bitcoin-forwarding-collector` - Lightning forwarding events
- `webhook-deployer` - Auto-deployment (port 9000)
- `lightning-telegram-monitor` - Telegram alerts (timer: every 2min)

## Manual Steps

### 1. Build
```bash
make build
```

### 2. Install Services  
```bash
# Automated
./deployment/scripts/install-simplified.sh

# Manual
sudo cp deployment/systemd/*.service.example /etc/systemd/system/
# Edit paths in service files
sudo systemctl daemon-reload
sudo systemctl enable bitcoin-dashboard-api bitcoin-dashboard-collector
```

### 3. Configure
```bash
cp .env.example .env
# Edit with your settings
```

### 4. Start Services
```bash
sudo systemctl start bitcoin-dashboard-api
sudo systemctl start bitcoin-dashboard-collector  
sudo systemctl start bitcoin-forwarding-collector
```

## Monitoring

```bash
# Status
sudo systemctl status bitcoin-dashboard-api

# Logs
journalctl -f -u bitcoin-dashboard-api

# Health check
curl http://localhost:8090/api/health
```

## Systemd Service Files

All service files are in `deployment/systemd/`:
- `.service` - Templates with `{{USER}}` placeholders
- `.service.example` - Copy-paste ready versions

Auto-deployment handles template substitution automatically.