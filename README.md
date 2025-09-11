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

5. **Create the monitor directory**:
   ```bash
   mkdir -p ~/lightning-monitor
   ```

## Usage

### Manual Execution

Run the script manually to test:
```bash
./telegram-alerts.sh
```

### Automated Monitoring with Cron

To set up automated monitoring that runs every 2 minutes:

1. **Edit your crontab**:
   ```bash
   crontab -e
   ```

2. **Add the following line** (replace `/path/to/lightning-node-tools` with the actual path):
   ```
   */2 * * * * /path/to/lightning-node-tools/telegram-alerts.sh >/dev/null 2>&1
   ```

3. **Save and exit**. The monitoring will now run automatically every 2 minutes.

### Example Crontab Entry

```bash
# Lightning Node Monitoring - runs every 2 minutes
*/2 * * * * /home/bitcoin/lightning-node-tools/telegram-alerts.sh >/dev/null 2>&1
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
- `jq` for JSON parsing
- `bc` for mathematical calculations
- `curl` for Telegram API calls
- Telegram bot token and chat ID

## File Structure

### Telegram Monitoring Tool
- `telegram-alerts.sh`: Main monitoring script
- `.env`: Your private configuration (not tracked by git)
- `.env.example`: Template configuration file
- `~/lightning-monitor/last_state.json`: Stores previous state for comparison
- `~/lightning-monitor/last_uptime.txt`: Tracks server uptime for reboot detection

## Troubleshooting

- Ensure your Lightning node is running and `lncli` commands work
- Verify your `.env` file has correct bot token and chat ID
- Check that the `~/lightning-monitor` directory exists
- Test the Telegram bot by sending a manual message first