# Systemd Service Files

This directory contains systemd service files for Lightning Node Tools.

## Quick Installation

**🚀 Automated Installation (Recommended):**
```bash
# One command to install everything
make install-all

# OR install just services
make install-services-auto
```

**📋 Manual Installation:**

For copy-paste ready service files, use the `.example` files:

### 1. Copy Service Files
```bash
sudo cp deployment/systemd/bitcoin-dashboard-api.service.example /etc/systemd/system/bitcoin-dashboard-api.service
sudo cp deployment/systemd/bitcoin-dashboard-collector.service.example /etc/systemd/system/bitcoin-dashboard-collector.service
sudo cp deployment/systemd/bitcoin-forwarding-collector.service.example /etc/systemd/system/bitcoin-forwarding-collector.service
sudo cp deployment/systemd/webhook-deployer.service.example /etc/systemd/system/webhook-deployer.service
```

### 2. Edit User/Path Settings
Replace `YOUR_USERNAME` with your actual username in each file:
```bash
sudo sed -i 's/YOUR_USERNAME/your_actual_username/g' /etc/systemd/system/bitcoin-dashboard-api.service
sudo sed -i 's/YOUR_USERNAME/your_actual_username/g' /etc/systemd/system/bitcoin-dashboard-collector.service
sudo sed -i 's/YOUR_USERNAME/your_actual_username/g' /etc/systemd/system/bitcoin-forwarding-collector.service
sudo sed -i 's/YOUR_USERNAME/your_actual_username/g' /etc/systemd/system/webhook-deployer.service
```

### 3. Enable and Start Services
```bash
sudo systemctl daemon-reload
sudo systemctl enable bitcoin-dashboard-api bitcoin-dashboard-collector bitcoin-forwarding-collector webhook-deployer
sudo systemctl start bitcoin-dashboard-api bitcoin-dashboard-collector bitcoin-forwarding-collector webhook-deployer
```

### 4. Verify Services
```bash
sudo systemctl status bitcoin-dashboard-api
sudo systemctl status bitcoin-dashboard-collector
sudo systemctl status bitcoin-forwarding-collector
sudo systemctl status webhook-deployer
```

## File Types

- **`.service`** - Template files with placeholder variables (used by install scripts)
- **`.service.example`** - Copy-paste ready files with $HOME variables

## Service Overview

| Service | Port | Purpose |
|---------|------|---------|
| bitcoin-dashboard-api | 8090 | Portfolio REST API |
| bitcoin-dashboard-collector | - | Data collection (15min intervals) |
| bitcoin-forwarding-collector | - | Lightning forwarding events |
| webhook-deployer | 9000 | Auto-deployment webhook |

## Troubleshooting

If services fail to start, check:
1. **Paths**: Ensure `$HOME/lightning-node-tools` exists and has correct permissions
2. **Binaries**: Run `make build` to ensure all binaries are built
3. **Logs**: Use `sudo journalctl -u service-name` to check error logs
4. **Permissions**: Ensure your user can access the working directory

## Security Notes

All services run as your user (not root) with restricted permissions:
- `NoNewPrivileges=true` - Cannot escalate privileges  
- `PrivateTmp=true` - Isolated temporary directory
- `ProtectSystem=strict` - Read-only filesystem protection
- Limited `ReadWritePaths` - Only specified directories are writable