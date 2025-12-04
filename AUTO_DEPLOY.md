# Auto-Deployment Setup Guide

This guide sets up automatic deployment of Lightning Node Tools using GitHub webhooks. When you push to the `main` branch, your server will automatically pull, build, test, and restart all services.

## 🏗️ **Architecture Overview**

```
GitHub Push → GitHub Webhook → Your Server → Auto-Deploy Script → Service Restart
```

**Components:**
- **Webhook Deployer Service**: Go service that receives GitHub webhooks
- **Auto-Deploy Script**: Bash script that handles deployment logic
- **Systemd Services**: Manage all Lightning Node services
- **Monitoring & Rollback**: Automatic health checks and rollback on failure

## 🚀 **Quick Setup**

### **1. On Your Server**

```bash
# Clone or pull the latest code
cd /opt/lightning-node-tools
git pull origin main

# Run the setup script (requires sudo)
sudo ./scripts/setup-auto-deploy.sh
```

### **2. Configure GitHub Webhook**

1. Go to your GitHub repository settings
2. Navigate to **Settings → Webhooks → Add webhook**
3. Configure:
   - **Payload URL**: `http://YOUR_SERVER_IP:9000/webhook`
   - **Content type**: `application/json`
   - **Secret**: (copy from server output)
   - **Events**: Select "Just the push event"
   - **Active**: ✅ Checked

### **3. Test the Setup**

```bash
# Test all components
sudo ./scripts/test-webhook.sh all

# Test just the webhook
sudo ./scripts/test-webhook.sh webhook
```

## 📋 **Detailed Setup Instructions**

### **Prerequisites**

- Ubuntu/Debian server with systemd
- Go 1.21+ installed
- Git configured
- Sudo privileges
- Port 9000 accessible from internet

### **Step 1: Initial Setup**

```bash
# Ensure you're in the repository directory
cd /opt/lightning-node-tools

# Make scripts executable
chmod +x scripts/*.sh

# Run setup (this will prompt for sudo)
sudo ./scripts/setup-auto-deploy.sh
```

**What this does:**
- ✅ Creates `lightning` user and group
- ✅ Sets up directory permissions
- ✅ Generates secure webhook secret
- ✅ Builds webhook deployer binary
- ✅ Installs systemd service
- ✅ Configures firewall (if UFW available)
- ✅ Sets up log rotation

### **Step 2: Configure Services**

Edit the deployment script to match your service names:

```bash
sudo nano /opt/lightning-node-tools/scripts/auto-deploy.sh

# Update this array to match your systemd service names:
SERVICES=(
    "bitcoin-dashboard-api"
    "bitcoin-dashboard-collector" 
    "bitcoin-forwarding-collector"
)
```

### **Step 3: GitHub Configuration**

1. **Get your webhook secret:**
   ```bash
   sudo cat /opt/lightning-node-tools/secrets/webhook.secret
   ```

2. **Add webhook to GitHub:**
   - Repository Settings → Webhooks → Add webhook
   - Payload URL: `http://YOUR_SERVER_IP:9000/webhook`
   - Content type: `application/json`
   - Secret: (paste from step 1)
   - Select: "Just the push event"
   - Active: ✅

## 🔧 **Service Management**

### **Webhook Deployer Service**

```bash
# Check status
sudo systemctl status webhook-deployer

# View live logs
sudo journalctl -u webhook-deployer -f

# Restart service
sudo systemctl restart webhook-deployer

# Stop/start service
sudo systemctl stop webhook-deployer
sudo systemctl start webhook-deployer
```

### **Deployment Logs**

```bash
# View deployment logs
tail -f /var/log/lightning-deploy.log

# View specific deployment
grep "2024-01-15" /var/log/lightning-deploy.log
```

## 🧪 **Testing & Monitoring**

### **Health Checks**

```bash
# Check webhook service health
curl http://localhost:9000/health

# Check deployment status
curl http://localhost:9000/status

# Run full test suite
sudo ./scripts/test-webhook.sh all
```

### **Test Deployment**

```bash
# Test with fake webhook (won't actually deploy)
sudo ./scripts/test-webhook.sh webhook

# Test security (should fail with wrong signature)
sudo ./scripts/test-webhook.sh security
```

## 🔒 **Security Features**

### **Webhook Security**
- ✅ **HMAC-SHA256 signature verification** - Only authentic GitHub webhooks accepted
- ✅ **Secret key protection** - Stored in protected file (600 permissions)
- ✅ **Branch validation** - Only `main` branch triggers deployment
- ✅ **Service isolation** - Runs as dedicated `lightning` user

### **Deployment Security**
- ✅ **Automatic testing** - Tests run before deployment
- ✅ **Backup creation** - Previous version backed up before deployment
- ✅ **Rollback capability** - Automatic rollback on failure
- ✅ **Health checks** - Services verified after restart

### **System Security**
- ✅ **Systemd hardening** - Service runs with restricted permissions
- ✅ **File permissions** - Proper ownership and permissions
- ✅ **Firewall integration** - Only webhook port exposed

## 🔄 **Deployment Process**

When a push to `main` happens:

1. **Webhook received** - GitHub sends webhook to your server
2. **Authentication** - HMAC signature verified
3. **Branch check** - Only `main` branch proceeds
4. **Backup created** - Current version backed up
5. **Code updated** - `git pull` latest changes
6. **Dependencies** - `go mod download` and verify
7. **Testing** - `make test` ensures quality
8. **Building** - `make build` compiles binaries
9. **Service restart** - Services stopped, updated, started
10. **Health checks** - Verify services are healthy
11. **Cleanup** - Old backups removed

**If anything fails: Automatic rollback to previous version**

## 📊 **Monitoring & Alerts**

### **Service Monitoring**

```bash
# Check all Lightning services
sudo ./scripts/test-webhook.sh services

# Individual service status
sudo systemctl status bitcoin-dashboard-api
sudo systemctl status bitcoin-dashboard-collector
sudo systemctl status bitcoin-forwarding-collector
```

### **Log Monitoring**

```bash
# Webhook deployer logs
sudo journalctl -u webhook-deployer -f

# Deployment logs  
tail -f /var/log/lightning-deploy.log

# Service logs
sudo journalctl -u bitcoin-dashboard-api -f
```

### **Optional: Slack Notifications**

Add to `/opt/lightning-node-tools/.env`:
```bash
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK
```

## 🚨 **Troubleshooting**

### **Webhook Not Receiving**

```bash
# Check service status
sudo systemctl status webhook-deployer

# Check if port is open
sudo netstat -tlnp | grep 9000

# Check firewall
sudo ufw status

# Test from GitHub webhook settings (recent deliveries)
```

### **Deployment Failing**

```bash
# Check deployment logs
tail -n 50 /var/log/lightning-deploy.log

# Check service logs
sudo journalctl -u webhook-deployer -n 20

# Test deployment script manually
cd /opt/lightning-node-tools
sudo -u lightning ./scripts/auto-deploy.sh
```

### **Services Not Starting**

```bash
# Check individual service status
sudo systemctl status bitcoin-dashboard-api

# Check service logs
sudo journalctl -u bitcoin-dashboard-api -n 20

# Test manual start
sudo systemctl start bitcoin-dashboard-api
```

### **Rollback Issues**

```bash
# Check available backups
ls -la /opt/lightning-node-tools-backups/

# Manual rollback
cd /opt/lightning-node-tools
sudo ./scripts/auto-deploy.sh
# (it will detect failure and rollback automatically)
```

## 🔧 **Advanced Configuration**

### **Custom Service Names**

Edit `/opt/lightning-node-tools/scripts/auto-deploy.sh`:
```bash
SERVICES=(
    "your-custom-dashboard-api"
    "your-custom-collector"
    "your-custom-forwarding"
)
```

### **Custom Deployment Logic**

The auto-deploy script is fully customizable. You can add:
- Database migrations
- Cache clearing
- Custom health checks
- Integration tests
- Custom notifications

### **Multiple Environments**

You can set up different webhook endpoints for different environments:
- Production: port 9000
- Staging: port 9001
- Development: port 9002

## 📞 **Support**

If you encounter issues:

1. **Check logs** first: `sudo journalctl -u webhook-deployer -f`
2. **Run tests**: `sudo ./scripts/test-webhook.sh all`
3. **Verify GitHub webhook** is configured correctly
4. **Check firewall** allows port 9000
5. **Verify services** are properly named in the deploy script

## 🎉 **Success!**

Once set up, every push to `main` will automatically:
- ✅ Deploy within seconds
- ✅ Run tests before deployment
- ✅ Backup previous version  
- ✅ Restart all services
- ✅ Verify health
- ✅ Rollback on failure

Your Lightning Node Tools are now enterprise-grade with automatic deployment! ⚡🚀