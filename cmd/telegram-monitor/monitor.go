package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/luiscarbonell/lightning-node-tools/pkg/lnd"
)

// checkChannelChanges monitors and reports channel state changes
func checkChannelChanges(current, prev *LightningState) {
	if current.Channels > prev.Channels {
		newChannels := current.Channels - prev.Channels
		msg := fmt.Sprintf("Channel Opened\nNew active channels: %d\nTotal active channels: %d", newChannels, current.Channels)
		sendTelegram(msg)
	}

	if current.PendingOpen > prev.PendingOpen {
		newPending := current.PendingOpen - prev.PendingOpen
		msg := fmt.Sprintf("New Channel Opening\nNew pending opens: %d\nTotal pending: %d", newPending, current.PendingOpen)
		sendTelegram(msg)
	}

	if current.Channels < prev.Channels {
		closedChannels := prev.Channels - current.Channels
		msg := fmt.Sprintf("Channel Closed\nChannels closed: %d\nRemaining active: %d", closedChannels, current.Channels)
		sendTelegram(msg)
	}

	if current.PendingClose > prev.PendingClose {
		newClosing := current.PendingClose - prev.PendingClose
		msg := fmt.Sprintf("Channel Closing Initiated\nNew pending closes: %d\nTotal pending closes: %d", newClosing, current.PendingClose)
		sendTelegram(msg)
	}
}

// checkForwardingActivity monitors and reports forwarding events
func checkForwardingActivity(current *LightningState) {
	if current.Forwards > 0 {
		// Get fee information
		recentTime := time.Now().Add(-10 * time.Minute).Unix()
		fwdHistory, err := lnd.RunLNCLI("fwdinghistory", "--start_time", strconv.FormatInt(recentTime, 10))
		if err != nil {
			log.Printf("Failed to get forwarding history: %v", err)
			return
		}

		var fwdData map[string]any
		if err := json.Unmarshal(fwdHistory, &fwdData); err != nil {
			log.Printf("Failed to parse forwarding history: %v", err)
			return
		}

		totalFees := int64(0)
		if fwdEvents, ok := fwdData["forwarding_events"].([]any); ok {
			for _, event := range fwdEvents {
				if eventMap, ok := event.(map[string]any); ok {
					if feeMsat, ok := eventMap["fee_msat"].(string); ok {
						if fee, err := strconv.ParseInt(feeMsat, 10, 64); err == nil {
							totalFees += fee
						}
					}
				}
			}
		}

		msg := fmt.Sprintf("Lightning Forwards\nRecent forwards: %d\nFees earned: %d sats", current.Forwards, totalFees/1000)
		sendTelegram(msg)
	}
}

// checkInvoiceChanges monitors and reports invoice changes
func checkInvoiceChanges(current, prev *LightningState) {
	if current.Invoices > prev.Invoices {
		newInvoices := current.Invoices - prev.Invoices
		msg := fmt.Sprintf("Invoice Paid\nNew payments received: %d\nTotal invoices: %d", newInvoices, current.Invoices)
		sendTelegram(msg)
	}
}

// checkBalanceChanges monitors and reports significant balance changes
func checkBalanceChanges(current, prev *LightningState) {
	onchainChange := current.OnchainBalance - prev.OnchainBalance
	localChange := current.LocalBalance - prev.LocalBalance
	remoteChange := current.RemoteBalance - prev.RemoteBalance
	totalChange := current.TotalBalance - prev.TotalBalance

	if onchainChange != 0 && int64(math.Abs(float64(onchainChange))) >= BalanceThreshold {
		msg := createBalanceMessage("On-chain", onchainChange, current.OnchainBalance)
		sendTelegram(msg)
	}

	if localChange != 0 && int64(math.Abs(float64(localChange))) >= BalanceThreshold {
		msg := createBalanceMessage("Lightning Local", localChange, current.LocalBalance)
		sendTelegram(msg)
	}

	if remoteChange != 0 && int64(math.Abs(float64(remoteChange))) >= BalanceThreshold {
		msg := createBalanceMessage("Lightning Remote", remoteChange, current.RemoteBalance)
		sendTelegram(msg)
	}

	if totalChange != 0 && int64(math.Abs(float64(totalChange))) >= SignificantThreshold {
		msg := createBalanceMessage("Total Portfolio", totalChange, current.TotalBalance)
		msg += fmt.Sprintf("\n\n<b>Breakdown:</b>\nOn-chain: %s (%+d)\nLightning: %s (%+d)",
			formatSats(current.OnchainBalance), onchainChange,
			formatSats(current.LocalBalance+current.RemoteBalance), localChange+remoteChange)
		sendTelegram(msg)
	}
}

// createBalanceMessage creates a formatted message for balance changes
func createBalanceMessage(changeType string, amount int64, current int64) string {
	var emoji string
	var direction string

	if amount > 0 {
		emoji = "📈"
		direction = "increased"
	} else {
		emoji = "📉"
		direction = "decreased"
		amount = -amount // Make positive for display
	}

	if amount >= SignificantThreshold {
		emoji = "⚠️ " + emoji
	}

	return fmt.Sprintf("%s <b>%s Balance %s</b>\nChange: %s\nCurrent: %s",
		emoji, changeType, direction, formatSats(amount), formatSats(current))
}