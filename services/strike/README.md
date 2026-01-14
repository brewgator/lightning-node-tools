# Strike Balance Tracking

This service tracks your Strike account balance over time by periodically polling the Strike API and storing snapshots in the database.

## Components

### 1. Strike API Client (`internal/strike/client.go`)
- Connects to Strike API
- Fetches current account balances for all currencies
- Converts amounts to smallest units (satoshis for BTC, cents for fiat)

### 2. Balance Collector Service (`services/strike/balance-collector/`)
- Runs periodically (default: every 15 minutes)
- Fetches balance from Strike API
- Stores snapshots in database
- Supports mock mode for testing

### 3. API Endpoints (`services/portfolio/api/main.go`)
- `GET /api/strike/balance/current?currency=BTC` - Current balance
- `GET /api/strike/balance/history?currency=BTC&days=30` - Historical balance (Chart.js format)

## Setup

### 1. Get Strike API Key

1. Log in to your Strike account
2. Go to Settings → Developer → API Keys
3. Create a new API key with `partner.balances.read` scope
4. Save the API key securely

### 2. Configure API Key

There are three ways to provide your Strike API key (in priority order):

**Option 1: .env File (Recommended)**
```bash
# Edit your .env file in the project root (copy from .env.example if needed)
echo "STRIKE_API_KEY=your_api_key_here" >> .env
```

The collector automatically loads from:
- `/opt/lightning-node-tools/.env` (when installed as a service)
- Project root `.env` (when running from bin/)
- Current directory `.env` (for development)

**Option 2: Environment Variable**
```bash
export STRIKE_API_KEY="your_api_key_here"
./bin/strike-balance-collector --oneshot
```

**Option 3: Command-Line Flag**
```bash
./bin/strike-balance-collector --api-key="your_api_key_here" --oneshot
```

**Priority:** CLI flag > Environment variable > .env file

### 3. Build the Service

```bash
make strike-balance-collector
```

### 4. Test in Mock Mode

```bash
./bin/strike-balance-collector --mock --oneshot
```

This will create mock data without calling the Strike API.

### 5. Test with Real API

```bash
./bin/strike-balance-collector --oneshot --api-key="your_key"
```

This will fetch real data once and exit.

### 6. Run as Service

```bash
# Run continuously (collects every 15 minutes by default)
./bin/strike-balance-collector --api-key="your_key"

# Custom interval
./bin/strike-balance-collector --api-key="your_key" --interval=5m

# Filter to only track BTC
./bin/strike-balance-collector --api-key="your_key" --currency=BTC
```

## Systemd Service Installation

1. Copy the example service file:
```bash
sudo cp deployment/systemd/strike-balance-collector.service.example \
        /etc/systemd/system/strike-balance-collector.service
```

2. Edit the service file and add your API key:
```bash
sudo nano /etc/systemd/system/strike-balance-collector.service
# Replace: Environment="STRIKE_API_KEY=your_strike_api_key_here"
```

3. Enable and start the service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable strike-balance-collector
sudo systemctl start strike-balance-collector
```

4. Check status:
```bash
sudo systemctl status strike-balance-collector
sudo journalctl -u strike-balance-collector -f
```

## API Usage

### Get Current Balance

```bash
curl http://localhost:8090/api/strike/balance/current?currency=BTC | jq
```

Response:
```json
{
  "success": true,
  "data": {
    "id": 123,
    "timestamp": "2026-01-14T10:30:00Z",
    "currency": "BTC",
    "available": 5000000,
    "total": 5100000,
    "pending": 50000,
    "reserved": 50000
  }
}
```

### Get Balance History

```bash
curl "http://localhost:8090/api/strike/balance/history?currency=BTC&days=7" | jq
```

Response (Chart.js format):
```json
{
  "success": true,
  "data": {
    "labels": ["2026-01-07 10:00", "2026-01-07 10:15", ...],
    "datasets": [
      {
        "label": "Available Balance (BTC)",
        "data": [5000000, 5050000, ...],
        "borderColor": "rgba(255, 159, 64, 1)",
        ...
      },
      {
        "label": "Total Balance (BTC)",
        "data": [5100000, 5150000, ...],
        "borderColor": "rgba(54, 162, 235, 1)",
        ...
      }
    ],
    "metadata": {
      "currency": "BTC",
      "days_requested": 7,
      "days_with_data": 48
    }
  }
}
```

## Database Schema

```sql
CREATE TABLE strike_balance_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    currency TEXT NOT NULL,
    available INTEGER NOT NULL,  -- Satoshis for BTC, cents for fiat
    total INTEGER NOT NULL,
    pending INTEGER NOT NULL,
    reserved INTEGER NOT NULL
);
```

## Command-Line Flags

- `--db` - Database path (default: `data/portfolio.db`)
- `--interval` - Collection interval (default: `15m`)
- `--oneshot` - Run once and exit (for testing)
- `--mock` - Use mock data (no API calls)
- `--api-key` - Strike API key (or use `STRIKE_API_KEY` env var)
- `--currency` - Only track specific currency (e.g., `BTC`)

## Troubleshooting

### Check if service is running
```bash
systemctl status strike-balance-collector
```

### View logs
```bash
journalctl -u strike-balance-collector -f
```

### Test API connection
```bash
./bin/strike-balance-collector --oneshot --api-key="your_key"
```

### Query database directly
```bash
sqlite3 data/portfolio.db "SELECT * FROM strike_balance_snapshots ORDER BY timestamp DESC LIMIT 5;"
```

## Security Notes

- Store your Strike API key securely
- Never commit the API key to version control
- Use environment variables or systemd service configuration
- The API key only needs `partner.balances.read` scope (read-only)
- Consider using a dedicated API key for this service

## Next Steps

To display Strike balance on the dashboard:
1. Add a new chart component in the frontend
2. Fetch data from `/api/strike/balance/history?currency=BTC&days=30`
3. Render using Chart.js (same format as other charts)
