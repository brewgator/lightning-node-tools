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

## 🚀 Quick Deployment (Recommended)

### One-Command Installation
```bash
# Complete installation: builds + services + crontab + start
make install-all
```

### Step-by-Step Installation
```bash
# 1. Install systemd services automatically
make install-services-auto

# 2. Install cron jobs automatically  
make install-crontab

# 3. Build and deploy
make build
sudo systemctl start bitcoin-dashboard-api bitcoin-dashboard-collector bitcoin-forwarding-collector webhook-deployer
```

## 🔧 Manual Installation (Advanced)

### 1. Install Services Manually
```bash
# Copy ready-to-use service files
sudo cp deployment/systemd/bitcoin-dashboard-api.service.example /etc/systemd/system/bitcoin-dashboard-api.service
sudo cp deployment/systemd/bitcoin-dashboard-collector.service.example /etc/systemd/system/bitcoin-dashboard-collector.service
sudo cp deployment/systemd/bitcoin-forwarding-collector.service.example /etc/systemd/system/bitcoin-forwarding-collector.service
sudo cp deployment/systemd/webhook-deployer.service.example /etc/systemd/system/webhook-deployer.service

# Replace YOUR_USERNAME with your actual username
sudo sed -i 's/YOUR_USERNAME/your_username/g' /etc/systemd/system/bitcoin-dashboard-api.service
sudo sed -i 's/YOUR_USERNAME/your_username/g' /etc/systemd/system/bitcoin-dashboard-collector.service
sudo sed -i 's/YOUR_USERNAME/your_username/g' /etc/systemd/system/bitcoin-forwarding-collector.service
sudo sed -i 's/YOUR_USERNAME/your_username/g' /etc/systemd/system/webhook-deployer.service

# Enable services
sudo systemctl daemon-reload
sudo systemctl enable bitcoin-dashboard-api bitcoin-dashboard-collector bitcoin-forwarding-collector webhook-deployer
```

### 2. Install Cron Jobs Manually
```bash
# Copy and edit template
cp deployment/crontab.example /tmp/mycron
nano /tmp/mycron  # Edit if needed (should work as-is with $HOME variables)
crontab /tmp/mycron
```

### 3. Verify Deployment
```bash
# Check services (note: these are SYSTEM services, not user services)
sudo systemctl status bitcoin-dashboard-api
sudo systemctl status bitcoin-dashboard-collector
sudo systemctl status bitcoin-forwarding-collector
sudo systemctl status webhook-deployer

# Check cron jobs
crontab -l

# Check API endpoints
curl http://localhost:8090/api/health    # Dashboard API
curl http://localhost:9000/health        # Webhook deployer

# Check logs
sudo journalctl -u bitcoin-dashboard-api -f
tail -f logs/telegram-monitor.log
```

## 📁 Directory Structure

After deployment, your structure should be:

```
$HOME/lightning-node-tools/
├── bin/                     # Built binaries
├── services/               # Service source code
├── tools/                  # CLI tools source code
├── internal/              # Shared packages
├── deployment/            # Infrastructure files
│   ├── systemd/          # Service templates & examples
│   ├── scripts/          # Installation & deployment scripts
│   ├── configs/          # Configuration files
│   ├── crontab.example   # Cron job template
│   └── DEPLOYMENT.md     # This file
├── data/                  # Database files
├── logs/                  # Application logs
└── secrets/              # Webhook secrets (create manually)
```

## 🔍 Service Dependencies

```
bitcoin-dashboard-collector → bitcoin-dashboard-api (starts after API)
webhook-deployer → (independent)
bitcoin-forwarding-collector → (independent)
```

## 🌐 Ports Used

- **8090** - Portfolio API
- **9000** - Webhook deployer
- **LND gRPC** - Lightning node connection (default: localhost:10009)

## 📂 File Locations

- **Binaries**: `./bin/`
- **Service source**: `./services/`
- **CLI tools**: `./tools/`
- **Data**: `./data/portfolio.db`
- **Logs**: `./logs/`
- **Backups**: `~/backups/`
- **Config**: `./deployment/configs/`
- **Secrets**: `./secrets/` (create: `mkdir -p secrets`)

## 🚨 Troubleshooting

### Services Won't Start
```bash
# Check service file syntax
sudo systemctl status bitcoin-dashboard-api

# Check if binaries exist
ls -la bin/

# Check permissions
sudo journalctl -u bitcoin-dashboard-api --since "5 minutes ago"
```

### Webhook Errors
```bash
# Check webhook service configuration
sudo systemctl cat webhook-deployer | grep script

# Verify script path exists
ls -la deployment/scripts/auto-deploy.sh

# Check webhook secret
ls -la secrets/webhook.secret
```

### Cron Jobs Not Running
```bash
# Check cron is running
sudo systemctl status cron

# Check cron logs
sudo tail -f /var/log/syslog | grep CRON

# Verify paths in crontab
crontab -l
```

## 🔄 Updates and Maintenance

### Updating Code
```bash
git pull origin main
make build
sudo systemctl restart bitcoin-dashboard-api bitcoin-dashboard-collector bitcoin-forwarding-collector
```

### Re-installing Services (after reorganization)
```bash
make install-services-auto
sudo systemctl daemon-reload
sudo systemctl restart bitcoin-dashboard-api bitcoin-dashboard-collector bitcoin-forwarding-collector webhook-deployer
```

### Backup Important Data
```bash
# Database backup
cp data/portfolio.db ~/backups/portfolio-$(date +%Y%m%d).db

# Configuration backup
tar -czf ~/backups/lightning-config-$(date +%Y%m%d).tar.gz deployment/configs/ .env secrets/
```

## 📊 Monitoring

### Health Checks
```bash
# API health
curl http://localhost:8090/api/health

# Webhook health  
curl http://localhost:9000/health

# Service status
sudo systemctl is-active bitcoin-dashboard-api bitcoin-dashboard-collector bitcoin-forwarding-collector webhook-deployer
```

### Log Monitoring
```bash
# Real-time logs
sudo journalctl -f -u bitcoin-dashboard-api -u bitcoin-dashboard-collector -u bitcoin-forwarding-collector -u webhook-deployer

# Application logs
tail -f logs/telegram-monitor.log
tail -f logs/fee-optimizer-$(date +%Y%m%d).log
```

## 🔒 Security Notes

- All services run as your user (not root) with restricted permissions
- systemd security features: `NoNewPrivileges`, `PrivateTmp`, `ProtectSystem=strict`
- Webhook uses HMAC-SHA256 signature verification
- Secrets stored in `./secrets/` (not in git)
- Log files rotated automatically via cron jobs