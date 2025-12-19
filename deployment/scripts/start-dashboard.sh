#!/bin/bash

# Bitcoin Portfolio Dashboard Startup Script

set -e

echo "🚀 Starting Bitcoin Portfolio Dashboard..."

# Check if binaries exist
if [ ! -f "bin/portfolio-collector" ] || [ ! -f "bin/portfolio-api" ]; then
    echo "📦 Building portfolio components..."
    make portfolio
fi

# Create data directory if it doesn't exist
mkdir -p data

echo ""
echo "📊 Testing data collection..."
./bin/portfolio-collector --oneshot --mock

if [ $? -eq 0 ]; then
    echo "✅ Data collection test successful!"
    echo "💡 Note: Using mock data for demo. Remove --mock to use real LND data."
else
    echo "❌ Data collection failed."
    exit 1
fi

echo ""
echo "🌐 Starting web API on http://localhost:8090"
echo "🔄 Data collection will run every 15 minutes"
echo ""
echo "Press Ctrl+C to stop..."

# Start the API server (this will also serve static files)
./bin/portfolio-api
