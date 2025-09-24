package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// showFeeEarnings displays fee earnings summary with optional detailed breakdown
func showFeeEarnings(detailed bool) {
	// Check for super detailed flag
	superDetailed := false
	for _, arg := range os.Args {
		if arg == "--super-detailed" || arg == "--super" {
			superDetailed = true
			detailed = true // Enable detailed too
			break
		}
	}
	feeReport, err := getFeeReport()
	if err != nil {
		log.Fatal("Failed to get fee report:", err)
	}

	fmt.Println("\n💸 Fee Earnings Summary")
	fmt.Println(strings.Repeat("━", 50))

	// Parse and display earnings
	dayFee := int64(0)
	weekFee := int64(0)
	monthFee := int64(0)

	if feeReport.DayFeeSum != "" {
		dayFee, _ = strconv.ParseInt(feeReport.DayFeeSum, 10, 64)
	}
	if feeReport.WeekFeeSum != "" {
		weekFee, _ = strconv.ParseInt(feeReport.WeekFeeSum, 10, 64)
	}
	if feeReport.MonthFeeSum != "" {
		monthFee, _ = strconv.ParseInt(feeReport.MonthFeeSum, 10, 64)
	}

	// Display earnings with nice formatting
	fmt.Printf("📅 Today:    %15s\n", formatSats(dayFee))
	fmt.Printf("📊 Week:     %15s\n", formatSats(weekFee))
	fmt.Printf("📈 Month:    %15s\n", formatSats(monthFee))

	fmt.Println(strings.Repeat("─", 50))

	// Calculate daily and weekly averages
	if weekFee > 0 {
		avgDaily := weekFee / 7
		fmt.Printf("📉 Daily Avg:%15s (7-day)\n", formatSats(avgDaily))
	}
	if monthFee > 0 {
		avgDaily := monthFee / 30
		fmt.Printf("📉 Daily Avg:%15s (30-day)\n", formatSats(avgDaily))
	}

	// Show total channel count for context
	channels, err := getChannels()
	if err == nil {
		activeChannels := 0
		for _, channel := range channels {
			if channel.Active {
				activeChannels++
			}
		}
		fmt.Printf("⚡ Channels: %15d active\n", activeChannels)
	}

	// Show detailed per-channel breakdown if requested
	if detailed {
		fmt.Println()
		if superDetailed {
			showSuperDetailedEarnings()
		} else {
			showDetailedChannelEarnings()
		}
	}

	fmt.Println()
}

// showDetailedChannelEarnings displays earnings breakdown by channel
func showDetailedChannelEarnings() {
	channels, err := getChannels()
	if err != nil {
		log.Printf("Failed to get channels for detailed view: %v", err)
		return
	}

	// Get forwarding history for the last month
	now := time.Now()
	monthAgo := now.AddDate(0, -1, 0)

	history, err := getForwardingHistory(
		fmt.Sprintf("%d", monthAgo.Unix()),
		fmt.Sprintf("%d", now.Unix()),
	)
	if err != nil {
		log.Printf("Failed to get forwarding history: %v", err)
		return
	}

	fmt.Println("📋 Detailed Channel Earnings (30 days)")
	fmt.Println(strings.Repeat("━", 75))

	// Calculate fees per channel
	channelFees := make(map[string]int64)

	for _, event := range history.ForwardingEvents {
		if event.FeeMsat != "" {
			feeMsat, _ := strconv.ParseInt(event.FeeMsat, 10, 64)
			channelFees[event.ChanIdOut] += feeMsat / 1000 // Convert to sats
		}
	}

	fmt.Printf("%-32s %-15s %s\n", "Channel", "Earnings", "Status")
	fmt.Println(strings.Repeat("─", 75))

	totalEarnings := int64(0)
	for _, channel := range channels {
		alias := getNodeAlias(channel.RemotePubkey)
		if len(alias) > 29 {
			alias = alias[:26] + "..."
		}

		earnings := channelFees[channel.ChanID]
		totalEarnings += earnings

		status := "🟢"
		if !channel.Active {
			status = "🔴"
		}

		fmt.Printf("%s %-29s %-15s %s\n",
			status,
			alias+":",
			formatSats(earnings),
			getChannelStatus(channel))
	}

	fmt.Println(strings.Repeat("─", 75))
	fmt.Printf("%-32s %-15s\n", "Total:", formatSats(totalEarnings))
}

// showSuperDetailedEarnings displays comprehensive forwarding event details
func showSuperDetailedEarnings() {
	channels, err := getChannels()
	if err != nil {
		log.Printf("Failed to get channels for super detailed view: %v", err)
		return
	}

	// Get forwarding history for the last 7 days for recent activity
	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)
	monthAgo := now.AddDate(0, -1, 0)

	recentHistory, err := getForwardingHistory(
		fmt.Sprintf("%d", weekAgo.Unix()),
		fmt.Sprintf("%d", now.Unix()),
	)
	if err != nil {
		log.Printf("Failed to get recent forwarding history: %v", err)
		recentHistory = &ForwardingHistory{ForwardingEvents: []ForwardingEvent{}}
	}

	monthHistory, err := getForwardingHistory(
		fmt.Sprintf("%d", monthAgo.Unix()),
		fmt.Sprintf("%d", now.Unix()),
	)
	if err != nil {
		log.Printf("Failed to get monthly forwarding history: %v", err)
		monthHistory = &ForwardingHistory{ForwardingEvents: []ForwardingEvent{}}
	}

	fmt.Println("🔍 Super Detailed Channel Earnings Analysis")
	fmt.Println(strings.Repeat("━", 100))

	// Create channel lookup map
	channelMap := make(map[string]Channel)
	for _, channel := range channels {
		channelMap[channel.ChanID] = channel
	}

	// Calculate per-channel metrics
	type ChannelEarningsDetail struct {
		Channel          Channel
		Alias            string
		MonthlyEarnings  int64
		WeeklyEarnings   int64
		MonthlyForwards  int
		WeeklyForwards   int
		AvgFeePerForward int64
		LastForwardTime  time.Time
		RecentEvents     []ForwardingEvent
	}

	channelDetails := make(map[string]*ChannelEarningsDetail)

	// Initialize channel details
	for _, channel := range channels {
		alias := getNodeAlias(channel.RemotePubkey)
		if len(alias) > 25 {
			alias = alias[:22] + "..."
		}
		channelDetails[channel.ChanID] = &ChannelEarningsDetail{
			Channel: channel,
			Alias:   alias,
		}
	}

	// Process monthly history
	for _, event := range monthHistory.ForwardingEvents {
		if detail, exists := channelDetails[event.ChanIdOut]; exists {
			feeMsat, _ := strconv.ParseInt(event.FeeMsat, 10, 64)
			detail.MonthlyEarnings += feeMsat / 1000
			detail.MonthlyForwards++

			timestamp, _ := strconv.ParseInt(event.Timestamp, 10, 64)
			eventTime := time.Unix(timestamp, 0)
			if eventTime.After(detail.LastForwardTime) {
				detail.LastForwardTime = eventTime
			}
		}
	}

	// Process recent history and collect recent events
	for _, event := range recentHistory.ForwardingEvents {
		if detail, exists := channelDetails[event.ChanIdOut]; exists {
			feeMsat, _ := strconv.ParseInt(event.FeeMsat, 10, 64)
			detail.WeeklyEarnings += feeMsat / 1000
			detail.WeeklyForwards++
			detail.RecentEvents = append(detail.RecentEvents, event)
		}
	}

	// Calculate averages and sort recent events
	for _, detail := range channelDetails {
		if detail.MonthlyForwards > 0 {
			detail.AvgFeePerForward = detail.MonthlyEarnings / int64(detail.MonthlyForwards)
		}

		// Sort recent events by timestamp (most recent first)
		sort.Slice(detail.RecentEvents, func(i, j int) bool {
			timeI, _ := strconv.ParseInt(detail.RecentEvents[i].Timestamp, 10, 64)
			timeJ, _ := strconv.ParseInt(detail.RecentEvents[j].Timestamp, 10, 64)
			return timeI > timeJ
		})

		// Keep only the 3 most recent events per channel
		if len(detail.RecentEvents) > 3 {
			detail.RecentEvents = detail.RecentEvents[:3]
		}
	}

	// Sort channels by monthly earnings
	var sortedDetails []*ChannelEarningsDetail
	for _, detail := range channelDetails {
		sortedDetails = append(sortedDetails, detail)
	}
	sort.Slice(sortedDetails, func(i, j int) bool {
		return sortedDetails[i].MonthlyEarnings > sortedDetails[j].MonthlyEarnings
	})

	// Display super detailed information
	for _, detail := range sortedDetails {
		if detail.MonthlyEarnings == 0 && detail.WeeklyEarnings == 0 {
			continue // Skip channels with no activity
		}

		status := "🟢"
		if !detail.Channel.Active {
			status = "🔴"
		}

		fmt.Printf("\n%s %s\n", status, detail.Alias)
		fmt.Println(strings.Repeat("─", 80))

		// Earnings summary
		fmt.Printf("💰 Earnings: Month: %s | Week: %s | Avg/Forward: %s\n",
			formatSats(detail.MonthlyEarnings),
			formatSats(detail.WeeklyEarnings),
			formatSats(detail.AvgFeePerForward))

		// Activity summary
		fmt.Printf("📊 Activity: Month: %d forwards | Week: %d forwards",
			detail.MonthlyForwards, detail.WeeklyForwards)

		if !detail.LastForwardTime.IsZero() {
			daysSince := int(time.Since(detail.LastForwardTime).Hours() / 24)
			fmt.Printf(" | Last: %dd ago", daysSince)
		}
		fmt.Println()

		// Channel info
		localBalance, _ := strconv.ParseInt(detail.Channel.LocalBalance, 10, 64)
		capacity, _ := strconv.ParseInt(detail.Channel.Capacity, 10, 64)
		localRatio := float64(localBalance) / float64(capacity) * 100

		fmt.Printf("⚡ Channel: %s capacity | %.1f%% local | %s\n",
			formatSats(capacity), localRatio, getChannelStatus(detail.Channel))

		// Recent forwarding events
		if len(detail.RecentEvents) > 0 {
			fmt.Printf("🔄 Recent Forwards (%d):\n", len(detail.RecentEvents))
			for i, event := range detail.RecentEvents {
				timestamp, _ := strconv.ParseInt(event.Timestamp, 10, 64)
				eventTime := time.Unix(timestamp, 0)

				amtIn, _ := strconv.ParseInt(event.AmtIn, 10, 64)
				amtOut, _ := strconv.ParseInt(event.AmtOut, 10, 64)
				feeMsat, _ := strconv.ParseInt(event.FeeMsat, 10, 64)

				fmt.Printf("   %d. %s: %s → %s (fee: %s) via %s\n",
					i+1,
					eventTime.Format("Jan 02 15:04"),
					formatSats(amtIn),
					formatSats(amtOut),
					formatSats(feeMsat/1000),
					getIncomingChannelAlias(event.ChanIdIn))
			}
		}
	}

	// Summary statistics
	totalMonthly := int64(0)
	totalWeekly := int64(0)
	totalMonthlyForwards := 0
	totalWeeklyForwards := 0
	activeChannels := 0

	for _, detail := range channelDetails {
		if detail.MonthlyEarnings > 0 || detail.WeeklyEarnings > 0 {
			activeChannels++
		}
		totalMonthly += detail.MonthlyEarnings
		totalWeekly += detail.WeeklyEarnings
		totalMonthlyForwards += detail.MonthlyForwards
		totalWeeklyForwards += detail.WeeklyForwards
	}

	fmt.Printf("\n")
	fmt.Println(strings.Repeat("━", 100))
	fmt.Printf("📈 Summary: %d active routing channels\n", activeChannels)
	fmt.Printf("💰 Total: Month: %s (%d forwards) | Week: %s (%d forwards)\n",
		formatSats(totalMonthly), totalMonthlyForwards,
		formatSats(totalWeekly), totalWeeklyForwards)

	if totalMonthlyForwards > 0 {
		avgFeePerForward := totalMonthly / int64(totalMonthlyForwards)
		fmt.Printf("📊 Average fee per forward: %s\n", formatSats(avgFeePerForward))
	}
}

// getIncomingChannelAlias gets the alias for the incoming channel
func getIncomingChannelAlias(chanID string) string {
	channels, err := getChannels()
	if err != nil {
		return "Unknown"
	}

	for _, channel := range channels {
		if channel.ChanID == chanID {
			alias := getNodeAlias(channel.RemotePubkey)
			if len(alias) > 15 {
				alias = alias[:12] + "..."
			}
			return alias
		}
	}
	return "Unknown"
}

// getSuperDetailedEarningsForTelegram returns a formatted string of super detailed earnings for telegram
func getSuperDetailedEarningsForTelegram() string {
	channels, err := getChannels()
	if err != nil {
		return "❌ Failed to get channel data"
	}

	// Get recent forwarding history (last 24 hours)
	now := time.Now()
	dayAgo := now.AddDate(0, 0, -1)

	history, err := getForwardingHistory(
		fmt.Sprintf("%d", dayAgo.Unix()),
		fmt.Sprintf("%d", now.Unix()),
	)
	if err != nil || len(history.ForwardingEvents) == 0 {
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
	message.WriteString("💰 24h Routing Summary\n")
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
			alias := getNodeAlias(channel.RemotePubkey)
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
		message.WriteString(fmt.Sprintf("• %s: %s (%d)\n",
			ch.alias, formatSats(ch.earnings), ch.forwards))
	}

	// Show most recent forward details
	if len(history.ForwardingEvents) > 0 {
		recent := history.ForwardingEvents[len(history.ForwardingEvents)-1]
		timestamp, _ := strconv.ParseInt(recent.Timestamp, 10, 64)
		recentTime := time.Unix(timestamp, 0)
		amtMsat, _ := strconv.ParseInt(recent.AmtOut, 10, 64)
		feeMsat, _ := strconv.ParseInt(recent.FeeMsat, 10, 64)

		message.WriteString(fmt.Sprintf("\n🔄 Latest: %s for %s (fee: %s)\n",
			recentTime.Format("15:04"),
			formatSats(amtMsat),
			formatSats(feeMsat/1000)))
	}

	return message.String()
}
