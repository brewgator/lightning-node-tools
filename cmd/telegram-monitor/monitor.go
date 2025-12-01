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

	"github.com/brewgator/lightning-node-tools/pkg/lnd"
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
// Only reports changes that indicate actual payments sent/received, not routing activity
func checkBalanceChanges(current, prev *LightningState) {
	onchainChange := current.OnchainBalance - prev.OnchainBalance
	localChange := current.LocalBalance - prev.LocalBalance
	remoteChange := current.RemoteBalance - prev.RemoteBalance
	totalChange := current.TotalBalance - prev.TotalBalance

	// Use adaptive thresholds based on account size
	threshold := getAdaptiveThreshold(current.TotalBalance)

	// Only report on-chain changes (these are always real payments/receipts)
	if onchainChange != 0 && int64(math.Abs(float64(onchainChange))) >= threshold {
		msg := createBalanceMessage("On-chain", onchainChange, current.OnchainBalance)
		sendTelegram(msg)
	}

	// Only report Lightning balance changes when they indicate actual payments
	// Check if this is likely a payment vs routing by looking at invoice/forward activity
	isLikelyPayment := isBalanceChangeFromPayment(current, prev, localChange, remoteChange)

	if isLikelyPayment {
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
}

// isBalanceChangeFromPayment determines if a balance change is from an actual payment
// rather than just routing activity. Returns true if it's likely a real payment.
func isBalanceChangeFromPayment(current, prev *LightningState, localChange, remoteChange int64) bool {
	// If invoices increased, it's likely a payment received
	if current.Invoices > prev.Invoices {
		return true
	}

	// If no recent forwarding activity in both current and previous states but balance changed, likely a payment
	if current.Forwards == 0 && prev.Forwards == 0 && (localChange != 0 || remoteChange != 0) {
		return true
	}

	// If there's forwarding activity, we need to be more cautious
	// Only report if the change is significant compared to typical routing amounts
	if current.Forwards > 0 {
		// For routing, changes are usually smaller and temporary
		// If the change is very large, it's more likely a real payment
		threshold := getAdaptiveThreshold(current.TotalBalance)
		absLocalChange := int64(math.Abs(float64(localChange)))
		absRemoteChange := int64(math.Abs(float64(remoteChange)))

		// If the change is much larger than normal routing, treat as payment
		routingThreshold := threshold * 10 // 10x normal threshold for routing situations
		if absLocalChange >= routingThreshold || absRemoteChange >= routingThreshold {
			return true
		}

		// Otherwise, assume it's routing activity
		return false
	}

	return true
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

// checkRoutingFees detects and reports new routing fees earned since last check
func checkRoutingFees(current, prev *LightningState) {
	// Get forwarding history since the last check
	startTime := prev.LastForwardTimestamp
	if startTime == 0 {
		// If this is the first run, only check the last 10 minutes to avoid spam
		startTime = time.Now().Add(-10 * time.Minute).Unix()
	}

	fwdHistory, err := lnd.RunLNCLI("fwdinghistory", "--start_time", strconv.FormatInt(startTime, 10))
	if err != nil {
		log.Printf("Failed to get forwarding history for routing fees: %v", err)
		return
	}

	var fwdData map[string]any
	if err := json.Unmarshal(fwdHistory, &fwdData); err != nil {
		log.Printf("Failed to parse forwarding history for routing fees: %v", err)
		return
	}

	fwdEvents, ok := fwdData["forwarding_events"].([]any)
	if !ok || len(fwdEvents) == 0 {
		return
	}

	// Get channel information for aliases
	channels, err := lnd.GetChannels()
	if err != nil {
		log.Printf("Failed to get channels for routing fees: %v", err)
		return
	}

	// Create channel ID to alias mapping
	channelAliases := make(map[string]string)
	for _, channel := range channels {
		alias := lnd.GetNodeAlias(channel.RemotePubkey)
		if len(alias) > 15 {
			alias = alias[:12] + "..."
		}
		channelAliases[channel.ChanID] = alias
	}

	// Process new forwarding events
	var newForwards []map[string]any
	var latestTimestamp int64

	for _, event := range fwdEvents {
		if eventMap, ok := event.(map[string]any); ok {
			if timestampStr, ok := eventMap["timestamp"].(string); ok {
				if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
					if timestamp > startTime {
						newForwards = append(newForwards, eventMap)
						if timestamp > latestTimestamp {
							latestTimestamp = timestamp
						}
					}
				}
			}
		}
	}

	// Update the latest timestamp in current state
	current.LastForwardTimestamp = latestTimestamp

	// Report each new forwarding event
	for _, event := range newForwards {
		reportRoutingFee(event, channelAliases)
	}
}

// reportRoutingFee sends a telegram message for a single routing fee earned
func reportRoutingFee(event map[string]any, channelAliases map[string]string) {
	// Extract event details with error checking
	feeMsatStr, ok := event["fee_msat"].(string)
	if !ok {
		log.Printf("reportRoutingFee: missing or invalid fee_msat field: %#v", event["fee_msat"])
		return
	}
	amtOutStr, ok := event["amt_out"].(string)
	if !ok {
		log.Printf("reportRoutingFee: missing or invalid amt_out field: %#v", event["amt_out"])
		return
	}
	chanIdIn, ok := event["chan_id_in"].(string)
	if !ok {
		log.Printf("reportRoutingFee: missing or invalid chan_id_in field: %#v", event["chan_id_in"])
		return
	}
	chanIdOut, ok := event["chan_id_out"].(string)
	if !ok {
		log.Printf("reportRoutingFee: missing or invalid chan_id_out field: %#v", event["chan_id_out"])
		return
	}

	feeMsat, err := strconv.ParseInt(feeMsatStr, 10, 64)
	if err != nil {
		log.Printf("reportRoutingFee: failed to parse fee_msat: %v", err)
		return
	}
	amtOut, err := strconv.ParseInt(amtOutStr, 10, 64)
	if err != nil {
		log.Printf("reportRoutingFee: failed to parse amt_out: %v", err)
		return
	}
	feeSats := feeMsat / 1000
	amtSats := amtOut

	// Get channel aliases
	inAlias := channelAliases[chanIdIn]
	outAlias := channelAliases[chanIdOut]

	if inAlias == "" {
		inAlias = "Unknown"
	}
	if outAlias == "" {
		outAlias = "Unknown"
	}

	// Create routing fee message
	msg := fmt.Sprintf("💰 <b>Routing Fee Earned</b>\nEarned: %s\nRouted: %s\nFrom: %s\nTo: %s",
		formatSats(feeSats),
		formatSats(amtSats),
		inAlias,
		outAlias)

	sendTelegram(msg)
}

// checkRoutingActivity monitors and reports routing success when balance changes are likely from routing
func checkRoutingActivity(current, prev *LightningState) {
	localChange := current.LocalBalance - prev.LocalBalance
	remoteChange := current.RemoteBalance - prev.RemoteBalance

	// Only report if there was forwarding activity and balance changes that we filtered out
	if current.Forwards > 0 && (localChange != 0 || remoteChange != 0) {
		// Check if this change was filtered out as routing activity
		isLikelyPayment := isBalanceChangeFromPayment(current, prev, localChange, remoteChange)

		if !isLikelyPayment {
			// This was routing activity - send a brief routing success message
			msg := fmt.Sprintf("🔄 <b>Routing Activity</b>\nForwards: %d\nTemp balance shifts during routing", current.Forwards)
			sendTelegram(msg)
		}
	}
}
