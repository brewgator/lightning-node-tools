#!/bin/bash

# Lightning Node Tools - Historical Multisig Backfill Script
# This script will backfill transaction-based balance history for all multisig wallets

echo "🔄 Historical Multisig Balance Backfill"
echo "======================================="
echo

# Check if binary exists
if [ ! -f "bin/historical-backfill" ]; then
    echo "❌ Building historical-backfill utility..."
    go build -o bin/historical-backfill ./tools/historical-backfill
    if [ $? -ne 0 ]; then
        echo "❌ Failed to build utility"
        exit 1
    fi
    echo "✅ Built successfully"
    echo
fi

echo "1️⃣  First, let's do a DRY RUN to see what would be done:"
echo "   ./bin/historical-backfill --dry-run"
echo
echo "2️⃣  If that looks good, run the actual backfill:"
echo "   ./bin/historical-backfill"
echo
echo "3️⃣  Or backfill a specific wallet:"
echo "   ./bin/historical-backfill --wallet=1"
echo
echo "📊 Your current multisig wallets:"
sqlite3 data/portfolio.db "SELECT id, name, required_signers||'/'||total_signers as quorum FROM multisig_wallets WHERE active=1;"
echo
echo "🏃 Ready to run! Execute one of the commands above."
