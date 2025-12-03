# Mock Mode Documentation

The Lightning Node Tools now support a comprehensive mock mode that isolates mock data from your real Lightning node data using separate database tables.

## Overview

Mock mode allows you to:
- Test all functionality without connecting to a real Lightning node
- Demonstrate the dashboard with sample data
- Develop and test new features safely
- Run parallel instances with different data sets

## How It Works

When mock mode is enabled:
- All data is stored in separate tables with `_mock` suffix
- `balance_snapshots_mock` instead of `balance_snapshots`
- `forwarding_events_mock` instead of `forwarding_events`
- Mock and real data are completely isolated
- Same database file, different tables

## Usage

### Dashboard Collector (Mock Portfolio Data)
```bash
# Generate mock portfolio snapshot
./bin/dashboard-collector --oneshot --mock

# Run continuous mock collection
./bin/dashboard-collector --mock --interval=1m
```

### Forwarding Collector (Mock Fee Data)
```bash
# Generate mock forwarding events
./bin/forwarding-collector --oneshot --mock

# Generate mock historical data
./bin/forwarding-collector --catchup --days=30 --mock

# Run continuous mock collection  
./bin/forwarding-collector --mock --interval=5m
```

### Dashboard API (Serve Mock Data)
```bash
# Start API serving mock data
./bin/dashboard-api --mock --port=8081

# Start both real and mock APIs (different ports)
./bin/dashboard-api --port=8080 &          # Real data on :8080
./bin/dashboard-api --mock --port=8081 &   # Mock data on :8081
```

## Example Workflow

1. **Generate Mock Data**:
```bash
# Create mock portfolio and forwarding data
./bin/dashboard-collector --oneshot --mock
./bin/forwarding-collector --catchup --days=7 --mock
```

2. **Start Mock Dashboard**:
```bash
# Start API serving mock data
./bin/dashboard-api --mock --port=8081
```

3. **View Dashboard**:
```bash
# Open browser to view mock dashboard
open http://localhost:8081
```

## Data Isolation Verification

You can verify data isolation by checking API responses:

```bash
# Real data API (if available)
curl http://localhost:8080/api/portfolio/current

# Mock data API  
curl http://localhost:8081/api/portfolio/current
```

The responses will show different timestamps and data, confirming isolation.

## Database Tables

### Regular Tables:
- `balance_snapshots`
- `forwarding_events` 
- `channel_snapshots`
- `onchain_addresses`
- `address_balances`
- `cold_storage_entries`

### Mock Tables:
- `balance_snapshots_mock`
- `forwarding_events_mock`
- `channel_snapshots_mock` 
- `onchain_addresses_mock`
- `address_balances_mock`
- `cold_storage_entries_mock`

## Benefits

1. **Safe Testing**: No risk of corrupting real Lightning node data
2. **Parallel Development**: Run real and mock systems simultaneously  
3. **Demonstrations**: Show features without requiring live Lightning infrastructure
4. **Development**: Test new features with controlled data sets
5. **CI/CD**: Run tests with mock data in automated environments

## Notes

- Mock mode uses the same database file but different tables
- All CLI tools support the `--mock` flag
- Mock data does not interact with real Lightning node APIs
- Test suites verify complete data isolation between modes