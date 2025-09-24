package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
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
	// Send detailed forwarding summary if there's recent activity
	if current.Forwards > 0 {
		detailedSummary := getSuperDetailedEarningsForTelegram()
		if detailedSummary != "📊 No forwarding activity in the last 24 hours" && detailedSummary != "📊 No fee earnings in the last 24 hours" {
			sendTelegram(detailedSummary)
			return // Don't send the basic message if we sent detailed
		}
	}
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

	// Use adaptive thresholds based on account size
	threshold := getAdaptiveThreshold(current.TotalBalance)

	if onchainChange != 0 && int64(math.Abs(float64(onchainChange))) >= threshold {
		msg := createBalanceMessage("On-chain", onchainChange, current.OnchainBalance)
		sendTelegram(msg)
	}

	if localChange != 0 && int64(math.Abs(float64(localChange))) >= threshold {
		msg := createBalanceMessage("Lightning Local", localChange, current.LocalBalance)
		sendTelegram(msg)
	}

	if remoteChange != 0 && int64(math.Abs(float64(remoteChange))) >= threshold {
		msg := createBalanceMessage("Lightning Remote", remoteChange, current.RemoteBalance)
		sendTelegram(msg)
	}

	// For total portfolio changes, use a higher threshold or significant threshold
	portfolioThreshold := int64(math.Max(float64(threshold*2), float64(SignificantThreshold)))
	if totalChange != 0 && int64(math.Abs(float64(totalChange))) >= portfolioThreshold {
		msg := createBalanceMessage("Total Portfolio", totalChange, current.TotalBalance)
		msg += fmt.Sprintf("\n\n<b>Breakdown:</b>\nOn-chain: %s (%s)\nLightning local: %s (%s)",
			formatSats(current.OnchainBalance), formatSatsChange(onchainChange),
			formatSats(current.LocalBalance), formatSatsChange(localChange))
		sendTelegram(msg)
	}
}

// getAdaptiveThreshold returns an appropriate threshold based on account size
func getAdaptiveThreshold(totalBalance int64) int64 {
	if totalBalance < 100000 { // Less than 100k sats
		return MinimalBalanceThreshold // 1 sat
	} else if totalBalance < 1000000 { // Less than 1M sats
		return 100 // 100 sats
	} else if totalBalance < 10000000 { // Less than 10M sats
		return BalanceThreshold // 1k sats
	}
	return BalanceThreshold * 5 // 5k sats for large accounts
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
		emoji, changeType, direction, formatSatsChange(amount), formatSats(current))
}

// getSuperDetailedEarningsForTelegram returns a formatted string of super detailed earnings for telegram
func getSuperDetailedEarningsForTelegram() string {
	channels, err := lnd.GetChannels()
	if err != nil {
		return "❌ Failed to get channel data"
	}

	// Get recent forwarding history (last 24 hours)
	now := time.Now()
	dayAgo := now.AddDate(0, 0, -1)
	
	output, err := lnd.RunLNCLI("fwdinghistory", "--start_time", strconv.FormatInt(dayAgo.Unix(), 10), "--end_time", strconv.FormatInt(now.Unix(), 10))
	if err != nil {
		return "📊 No forwarding activity in the last 24 hours"
	}

	var history lnd.ForwardingHistory
	if err := json.Unmarshal(output, &history); err != nil {
		return "❌ Failed to parse forwarding history"
	}

	if len(history.ForwardingEvents) == 0 {
		return "📊 No forwarding activity in the last 24 hours"
	}

	// Calculate earnings and activity
	channelEarnings := make(map[string]int64)
	channelForwards := make(map[string]int)
	totalEarnings := int64(0)
	
	for _, event := range history.ForwardingEvents {
		feeMsat, _ := strconv.ParseInt(event.FeeMsat, 10, 64)
		feeSats := feeMsat / 1000
		channelEarnings[event.ChanIdOut] += feeSats
		channelForwards[event.ChanIdOut]++
		totalEarnings += feeSats
	}

	if totalEarnings == 0 {
		return "📊 No fee earnings in the last 24 hours"
	}

	// Build telegram message
	var message strings.Builder
	message.WriteString("💰 <b>24h Routing Summary</b>\n")
	message.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
	message.WriteString(fmt.Sprintf("Total: %s (%d forwards)\n", formatSats(totalEarnings), len(history.ForwardingEvents)))
	
	// Show top earning channels
	type channelSummary struct {
		alias    string
		earnings int64
		forwards int
	}
	
	var topChannels []channelSummary
	for _, channel := range channels {
		if earnings := channelEarnings[channel.ChanID]; earnings > 0 {
			alias := lnd.GetNodeAlias(channel.RemotePubkey)
			if len(alias) > 15 {
				alias = alias[:12] + "..."
			}
			topChannels = append(topChannels, channelSummary{
				alias:    alias,
				earnings: earnings,
				forwards: channelForwards[channel.ChanID],
			})
		}
	}
	
	// Sort by earnings
	sort.Slice(topChannels, func(i, j int) bool {
		return topChannels[i].earnings > topChannels[j].earnings
	})
	
	// Show top 5 channels
	maxShow := 5
	if len(topChannels) < maxShow {
		maxShow = len(topChannels)
	}
	
	for i := 0; i < maxShow; i++ {
		ch := topChannels[i]
		message.WriteString(fmt.Sprintf("• <b>%s</b>: %s (%d)\n", 
			ch.alias, formatSats(ch.earnings), ch.forwards))
	}
	
	// Show most recent forward details
	if len(history.ForwardingEvents) > 0 {
		recent := history.ForwardingEvents[len(history.ForwardingEvents)-1]
		timestamp, _ := strconv.ParseInt(recent.Timestamp, 10, 64)
		recentTime := time.Unix(timestamp, 0)
		amtSats, _ := strconv.ParseInt(recent.AmtOut, 10, 64)
		feeMsat, _ := strconv.ParseInt(recent.FeeMsat, 10, 64)
		
		message.WriteString(fmt.Sprintf("\n🔄 <b>Latest</b>: %s for %s (fee: %s)\n",
			recentTime.Format("15:04"),
			formatSats(amtSats),
			formatSats(feeMsat/1000)))
	}

	return message.String()
}
