# Lightning Node Tools

A toolkit for Lightning Network node management and monitoring. This repository contains the first tool with more utilities planned.

## Current Tool: Telegram Monitoring

Real-time Lightning node monitoring with Telegram notifications.

## Features

- **Real-time Lightning Node Monitoring**: Track channel opens/closes, pending operations, and balance changes
- **Telegram Notifications**: Get instant alerts about your Lightning node activity
- **Balance Tracking**: Monitor on-chain and Lightning channel balances with configurable thresholds
- **Forward Monitoring**: Track routing fees and forward activity
- **Server Reboot Detection**: Get notified when your Lightning node server restarts

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
   chmod +x telegram-alerts.sh
   ```

5. **Create the data directory**:

   ```bash
   mkdir -p data
   ```

## Available Tools

### 1. Telegram Monitor
Real-time Lightning node monitoring with Telegram notifications.

### 2. Channel Manager
Visual channel liquidity management and analysis tool.

## Usage

### Build All Tools

```bash
go build -o bin/telegram-monitor ./cmd/telegram-monitor
go build -o bin/channel-manager ./cmd/channel-manager
```

### Telegram Monitor

1. **Build the Go program**:

   ```bash
   go build -o bin/telegram-monitor ./cmd/telegram-monitor
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

**Show visual channel balances:**
```bash
./bin/channel-manager balance
# or short alias:
./bin/channel-manager bal
```

**Example Output:**
```
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

### Bash Script (Legacy)

You can still use the original bash script:

1. **Run manually to test**:

   ```bash
   ./telegram-alerts.sh
   ```

2. **Set up with cron**:

   ```bash
   */2 * * * * /path/to/lightning-node-tools/telegram-alerts.sh >/dev/null 2>&1
   ```

## What Gets Monitored

- **Channel Events**: Opens, closes, pending operations
- **Balance Changes**: On-chain and Lightning balances (configurable thresholds)
- **Payment Activity**: New invoices and forwards
- **Server Status**: Reboot detection
- **Routing Fees**: Recent forwarding activity and fees earned

## Configuration

The script includes several configurable thresholds:

- `BALANCE_THRESHOLD=10000`: Minimum balance change to trigger notification (10k sats)
- `SIGNIFICANT_THRESHOLD=1000000`: Threshold for highlighting significant changes (1M sats)

## Requirements

- Lightning Network node with `lncli` installed and configured
- Go 1.19+ (for the Go program)
- Telegram bot token and chat ID

### Legacy Bash Script Requirements

- `jq` for JSON parsing
- `bc` for mathematical calculations
- `curl` for Telegram API calls

## File Structure

### Telegram Monitoring Tool

- `cmd/telegram-monitor/main.go`: Go program source code
- `bin/telegram-monitor`: Compiled Go binary (after building)
- `telegram-alerts.sh`: Legacy bash monitoring script
- `.env`: Your private configuration (not tracked by git)
- `.env.example`: Template configuration file
- `data/last_state.json`: Stores previous state for comparison
- `data/last_uptime.txt`: Tracks server uptime for reboot detection

## Troubleshooting

- Ensure your Lightning node is running and `lncli` commands work
- Verify your `.env` file has correct bot token and chat ID
- Check that the `data` directory exists
- Test the Telegram bot by sending a manual message first
