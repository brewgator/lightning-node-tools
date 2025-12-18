# Deployment Guide

## Runtime Overview

This system runs multiple components across different execution contexts:

### Systemd Services (Always Running)
Located in `deployment/systemd/`

- **bitcoin-dashboard-api.service** - Portfolio REST API (port 8090)
- **bitcoin-dashboard-collector.service** - Data collection service (15min intervals)
- **bitcoin-forwarding-collector.service** - Lightning forwarding events collector
- **webhook-deployer.service** - Auto-deployment webhook server (port 9000)

### Cron Jobs (Scheduled Tasks)
Template: `deployment/crontab.example`

- **Daily 2:00 AM** - Channel backups (`lncli exportchanbackup`)
- **Weekly Sun 2:15 AM** - Fee optimization (`tools/channel-manager/`)
- **Every 2 minutes** - Telegram monitoring (`tools/monitoring/`)
- **Daily 3:00 AM** - Log cleanup & backup rotation

### On-Demand Tools
Located in `tools/`

- **channel-manager** - Interactive channel management CLI
- **telegram-monitor** - Manual monitoring checks

## Quick Deployment

### 1. Install Services
```bash
make install-services
systemctl --user daemon-reload
systemctl --user enable --now bitcoin-dashboard-api
systemctl --user enable --now bitcoin-dashboard-collector  
systemctl --user enable --now bitcoin-forwarding-collector
systemctl --user enable --now webhook-deployer
```

### 2. Install Cron Jobs
```bash
cp deployment/crontab.example /tmp/mycron
# Edit paths to match your setup
nano /tmp/mycron
crontab /tmp/mycron
```

### 3. Verify Deployment
```bash
# Check services
systemctl --user status bitcoin-dashboard-api
systemctl --user status webhook-deployer

# Check cron
crontab -l

# Check logs
journalctl --user -u bitcoin-dashboard-api -f
tail -f logs/telegram-monitor.log
```

## Service Dependencies

```
bitcoin-dashboard-collector → bitcoin-dashboard-api
webhook-deployer → (independent)
bitcoin-forwarding-collector → (independent)
```

## Ports Used

- **8090** - Portfolio API
- **9000** - Webhook deployer
- **LND gRPC** - Lightning node connection

## File Locations

- **Binaries**: `./bin/`
- **Data**: `./data/portfolio.db`
- **Logs**: `./logs/`
- **Backups**: `~/backups/`
- **Config**: `./configs/`
- **Secrets**: `./secrets/` (webhook.secret)