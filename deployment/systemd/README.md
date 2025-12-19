# Systemd Service Files

Service files for Lightning Node Tools.

## Quick Install

```bash
# Automated (recommended)
./setup.sh --full

# Legacy
make install-all
```

## Manual Install

### Copy service files:
```bash
sudo cp *.service.example /etc/systemd/system/
# Remove .example suffix from filenames
```

### Edit paths:
```bash
# Replace YOUR_USERNAME with actual username in each service file
sudo nano /etc/systemd/system/bitcoin-dashboard-api.service
```

### Enable services:
```bash
sudo systemctl daemon-reload
sudo systemctl enable bitcoin-dashboard-api bitcoin-dashboard-collector
sudo systemctl start bitcoin-dashboard-api bitcoin-dashboard-collector
```

## Services

| Service | Purpose | Port |
|---------|---------|------|
| `bitcoin-dashboard-api` | Portfolio REST API | 8090 |
| `bitcoin-dashboard-collector` | Data collection (15min) | - |
| `bitcoin-forwarding-collector` | Lightning forwarding events | - |
| `webhook-deployer` | Auto-deployment | 9000 |
| `lightning-telegram-monitor` | Telegram alerts (timer) | - |

## File Types

- `.service` - Templates with `{{USER}}` placeholders (for scripts)
- `.service.example` - Copy-paste ready with `$HOME` paths
- `.timer` - Systemd timers for periodic execution

## Troubleshooting

```bash
# Check status
sudo systemctl status bitcoin-dashboard-api

# View logs  
journalctl -f -u bitcoin-dashboard-api

# Restart
sudo systemctl restart bitcoin-dashboard-api
```

Common issues:
- **Path errors**: Ensure `$HOME/lightning-node-tools` exists
- **Missing binaries**: Run `make build` first
- **Permissions**: Services run as your user, not root