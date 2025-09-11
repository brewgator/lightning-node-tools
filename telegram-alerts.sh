#!/bin/bash

# Load environment variables
if [ -f "$(dirname "$0")/.env" ]; then
    source "$(dirname "$0")/.env"
else
    echo "Error: .env file not found. Please copy .env.example to .env and configure your tokens."
    exit 1
fi

# Verify required environment variables are set
if [ -z "$BOT_TOKEN" ] || [ -z "$CHAT_ID" ]; then
    echo "Error: BOT_TOKEN and CHAT_ID must be set in .env file"
    exit 1
fi

# File to store last known state
SCRIPT_DIR="$(dirname "$0")"
STATE_FILE="$SCRIPT_DIR/data/last_state.json"

# Function to send Telegram message
send_telegram() {
    MESSAGE="$1"
    curl -s -X POST "https://api.telegram.org/bot${BOT_TOKEN}/sendMessage" \
        -d chat_id="${CHAT_ID}" \
        -d text="${MESSAGE}" \
        -d parse_mode="HTML" > /dev/null
}

# Function to format satoshi amounts
format_sats() {
    local amount=$1
    if [ "$amount" -ge 100000000 ]; then
        # Show in BTC for amounts >= 1 BTC
        printf "%.8f BTC" $(echo "scale=8; $amount / 100000000" | bc -l)
    elif [ "$amount" -ge 1000000 ]; then
        # Show in millions for amounts >= 1M sats
        printf "%.2fM sats" $(echo "scale=2; $amount / 1000000" | bc -l)
    elif [ "$amount" -ge 1000 ]; then
        # Show in thousands for amounts >= 1K sats
        printf "%.1fK sats" $(echo "scale=1; $amount / 1000" | bc -l)
    else
        printf "%d sats" "$amount"
    fi
}

# Check for server reboot by looking at uptime
CURRENT_UPTIME=$(cat /proc/uptime | cut -d' ' -f1 | cut -d'.' -f1)
UPTIME_FILE="$SCRIPT_DIR/data/last_uptime.txt"

if [ -f "$UPTIME_FILE" ]; then
    PREV_UPTIME=$(cat "$UPTIME_FILE")
    if [ "$CURRENT_UPTIME" -lt "$PREV_UPTIME" ]; then
        send_telegram "Server Rebooted
Uptime: $CURRENT_UPTIME seconds
Previous uptime was: $PREV_UPTIME seconds"
    fi
fi
echo "$CURRENT_UPTIME" > "$UPTIME_FILE"

# Get current Lightning state with default values for null
CURRENT_CHANNELS=$(lncli listchannels | jq '.channels | length // 0')
PENDING_CHANNELS=$(lncli pendingchannels)
CURRENT_PENDING=$(echo "$PENDING_CHANNELS" | jq '(.pending_open_channels // []) | length')
CURRENT_CLOSING=$(echo "$PENDING_CHANNELS" | jq '((.pending_closing_channels // []) + (.pending_force_closing_channels // []) + (.waiting_close_channels // [])) | length')
CURRENT_INVOICES=$(lncli listinvoices | jq '.invoices | length // 0')

# Get wallet balance information
WALLET_BALANCE=$(lncli walletbalance)
CURRENT_ONCHAIN=$(echo "$WALLET_BALANCE" | jq '.total_balance | tonumber // 0')
CURRENT_CONFIRMED=$(echo "$WALLET_BALANCE" | jq '.confirmed_balance | tonumber // 0')

# Get channel balance information
CHANNEL_BALANCE=$(lncli channelbalance)
CURRENT_LOCAL=$(echo "$CHANNEL_BALANCE" | jq '.local_balance.sat | tonumber // 0')
CURRENT_REMOTE=$(echo "$CHANNEL_BALANCE" | jq '.remote_balance.sat | tonumber // 0')
CURRENT_PENDING_LOCAL=$(echo "$CHANNEL_BALANCE" | jq '.pending_open_local_balance.sat | tonumber // 0')
CURRENT_PENDING_REMOTE=$(echo "$CHANNEL_BALANCE" | jq '.pending_open_remote_balance.sat | tonumber // 0')

# Calculate total balances
CURRENT_TOTAL_LOCAL=$((CURRENT_LOCAL + CURRENT_PENDING_LOCAL))
CURRENT_TOTAL_REMOTE=$((CURRENT_REMOTE + CURRENT_PENDING_REMOTE))
CURRENT_TOTAL_LIGHTNING=$((CURRENT_TOTAL_LOCAL + CURRENT_TOTAL_REMOTE))
CURRENT_TOTAL_ALL=$((CURRENT_ONCHAIN + CURRENT_TOTAL_LIGHTNING))

# Get forwarding events from last 10 minutes
RECENT_TIME=$(date -d '10 minutes ago' +%s)
FORWARDING_EVENTS=$(lncli fwdinghistory --start_time $RECENT_TIME | jq '.forwarding_events // []')
RECENT_FORWARDS=$(echo "$FORWARDING_EVENTS" | jq 'length')
RECENT_FEES=$(echo "$FORWARDING_EVENTS" | jq '[.[].fee_msat | tonumber] | add // 0')

# Create data directory if it doesn't exist
mkdir -p "$SCRIPT_DIR/data"

# Initialize state file if it doesn't exist
if [ ! -f "$STATE_FILE" ]; then
    echo "{\"channels\": $CURRENT_CHANNELS, \"pending_open\": $CURRENT_PENDING, \"pending_close\": $CURRENT_CLOSING, \"invoices\": $CURRENT_INVOICES, \"forwards\": $RECENT_FORWARDS, \"onchain_balance\": $CURRENT_ONCHAIN, \"local_balance\": $CURRENT_TOTAL_LOCAL, \"remote_balance\": $CURRENT_TOTAL_REMOTE, \"total_balance\": $CURRENT_TOTAL_ALL}" > "$STATE_FILE"

    send_telegram "Lightning Monitor Started
Active channels: $CURRENT_CHANNELS
Pending opens: $CURRENT_PENDING
Pending closes: $CURRENT_CLOSING
Total invoices: $CURRENT_INVOICES

<b>Balance Summary:</b>
On-chain: $(format_sats $CURRENT_ONCHAIN)
Lightning local: $(format_sats $CURRENT_TOTAL_LOCAL)
Lightning remote: $(format_sats $CURRENT_TOTAL_REMOTE)
Total balance: $(format_sats $CURRENT_TOTAL_ALL)"
    exit 0
fi

# Read previous state with default values
PREV_CHANNELS=$(jq '.channels // 0' "$STATE_FILE")
PREV_PENDING=$(jq '.pending_open // 0' "$STATE_FILE")
PREV_CLOSING=$(jq '.pending_close // 0' "$STATE_FILE")
PREV_INVOICES=$(jq '.invoices // 0' "$STATE_FILE")
PREV_FORWARDS=$(jq '.forwards // 0' "$STATE_FILE")
PREV_ONCHAIN=$(jq '.onchain_balance // 0' "$STATE_FILE")
PREV_LOCAL=$(jq '.local_balance // 0' "$STATE_FILE")
PREV_REMOTE=$(jq '.remote_balance // 0' "$STATE_FILE")
PREV_TOTAL=$(jq '.total_balance // 0' "$STATE_FILE")

# Balance change thresholds (in satoshis)
BALANCE_THRESHOLD=10000  # Only notify for changes > 10k sats
SIGNIFICANT_THRESHOLD=1000000  # Highlight changes > 1M sats

# Check for channel opens (pending -> active)
if [ "$CURRENT_CHANNELS" -gt "$PREV_CHANNELS" ]; then
    NEW_CHANNELS=$((CURRENT_CHANNELS - PREV_CHANNELS))
    send_telegram "Channel Opened
New active channels: $NEW_CHANNELS
Total active channels: $CURRENT_CHANNELS"
fi

# Check for new pending channel opens
if [ "$CURRENT_PENDING" -gt "$PREV_PENDING" ]; then
    NEW_PENDING=$((CURRENT_PENDING - PREV_PENDING))
    send_telegram "New Channel Opening
New pending opens: $NEW_PENDING
Total pending: $CURRENT_PENDING"
fi

# Check for channel closes
if [ "$CURRENT_CHANNELS" -lt "$PREV_CHANNELS" ]; then
    CLOSED_CHANNELS=$((PREV_CHANNELS - CURRENT_CHANNELS))
    send_telegram "Channel Closed
Channels closed: $CLOSED_CHANNELS
Remaining active: $CURRENT_CHANNELS"
fi

# Check for new pending closes
if [ "$CURRENT_CLOSING" -gt "$PREV_CLOSING" ]; then
    NEW_CLOSING=$((CURRENT_CLOSING - PREV_CLOSING))
    send_telegram "Channel Closing Initiated
New pending closes: $NEW_CLOSING
Total pending closes: $CURRENT_CLOSING"
fi

# Check for new forwards
if [ "$RECENT_FORWARDS" -gt 0 ]; then
    send_telegram "Lightning Forwards
Recent forwards: $RECENT_FORWARDS
Fees earned: $((RECENT_FEES / 1000)) sats"
fi

# Check for paid invoices
if [ "$CURRENT_INVOICES" -gt "$PREV_INVOICES" ]; then
    NEW_INVOICES=$((CURRENT_INVOICES - PREV_INVOICES))
    send_telegram "Invoice Paid
New payments received: $NEW_INVOICES
Total invoices: $CURRENT_INVOICES"
fi

# Check for balance changes
ONCHAIN_CHANGE=$((CURRENT_ONCHAIN - PREV_ONCHAIN))
LOCAL_CHANGE=$((CURRENT_TOTAL_LOCAL - PREV_LOCAL))
REMOTE_CHANGE=$((CURRENT_TOTAL_REMOTE - PREV_REMOTE))
TOTAL_CHANGE=$((CURRENT_TOTAL_ALL - PREV_TOTAL))

# Function to create balance change message
create_balance_message() {
    local change_type="$1"
    local amount="$2"
    local current="$3"
    local emoji=""
    local direction=""

    if [ "$amount" -gt 0 ]; then
        emoji="📈"
        direction="increased"
    else
        emoji="📉"
        direction="decreased"
        amount=$((amount * -1))  # Make positive for display
    fi

    if [ "$amount" -ge "$SIGNIFICANT_THRESHOLD" ]; then
        emoji="⚠️ $emoji"
    fi

    echo "$emoji <b>$change_type Balance $direction</b>
Change: $(format_sats $amount)
Current: $(format_sats $current)"
}

# Check for significant on-chain balance changes
if [ "$ONCHAIN_CHANGE" -ne 0 ] && [ "${ONCHAIN_CHANGE#-}" -ge "$BALANCE_THRESHOLD" ]; then
    send_telegram "$(create_balance_message "On-chain" "$ONCHAIN_CHANGE" "$CURRENT_ONCHAIN")"
fi

# Check for significant local balance changes
if [ "$LOCAL_CHANGE" -ne 0 ] && [ "${LOCAL_CHANGE#-}" -ge "$BALANCE_THRESHOLD" ]; then
    send_telegram "$(create_balance_message "Lightning Local" "$LOCAL_CHANGE" "$CURRENT_TOTAL_LOCAL")"
fi

# Check for significant remote balance changes
if [ "$REMOTE_CHANGE" -ne 0 ] && [ "${REMOTE_CHANGE#-}" -ge "$BALANCE_THRESHOLD" ]; then
    send_telegram "$(create_balance_message "Lightning Remote" "$REMOTE_CHANGE" "$CURRENT_TOTAL_REMOTE")"
fi

# Check for significant total balance changes (overall portfolio)
if [ "$TOTAL_CHANGE" -ne 0 ] && [ "${TOTAL_CHANGE#-}" -ge "$BALANCE_THRESHOLD" ]; then
    # Only send total balance notification if it's a significant change
    # and not already covered by the individual balance notifications
    if [ "${TOTAL_CHANGE#-}" -ge "$SIGNIFICANT_THRESHOLD" ]; then
        send_telegram "$(create_balance_message "Total Portfolio" "$TOTAL_CHANGE" "$CURRENT_TOTAL_ALL")

<b>Breakdown:</b>
On-chain: $(format_sats $CURRENT_ONCHAIN) ($(printf "%+d" $ONCHAIN_CHANGE))
Lightning: $(format_sats $CURRENT_TOTAL_LIGHTNING) ($(printf "%+d" $((LOCAL_CHANGE + REMOTE_CHANGE))))"
    fi
fi

# Update state file
echo "{\"channels\": $CURRENT_CHANNELS, \"pending_open\": $CURRENT_PENDING, \"pending_close\": $CURRENT_CLOSING, \"invoices\": $CURRENT_INVOICES, \"forwards\": $RECENT_FORWARDS, \"onchain_balance\": $CURRENT_ONCHAIN, \"local_balance\": $CURRENT_TOTAL_LOCAL, \"remote_balance\": $CURRENT_TOTAL_REMOTE, \"total_balance\": $CURRENT_TOTAL_ALL}" > "$STATE_FILE"