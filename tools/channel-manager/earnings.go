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

	// Use shared helper for monthly and weekly earnings
	monthlySummaries, totalMonthly, totalMonthlyForwards := calculateChannelEarnings(channels, monthHistory.ForwardingEvents)
	weeklySummaries, totalWeekly, totalWeeklyForwards := calculateChannelEarnings(channels, recentHistory.ForwardingEvents)

	// Map summaries for quick lookup
	summaryMap := make(map[string]*channelEarningsSummary)
	for i := range monthlySummaries {
		summaryMap[monthlySummaries[i].ChanID] = &monthlySummaries[i]
	}
	for i := range weeklySummaries {
		if s, ok := summaryMap[weeklySummaries[i].ChanID]; ok {
			s.Forwards = weeklySummaries[i].Forwards
			s.Earnings = monthlySummaries[i].Earnings
		}
	}

	// Display super detailed information
	for _, detail := range monthlySummaries {
		if detail.Earnings == 0 && weeklySummaries[0].Earnings == 0 {
			continue // Skip channels with no activity
		}
		var channel Channel
		for _, ch := range channels {
			if ch.ChanID == detail.ChanID {
				channel = ch
				break
			}
		}
		status := "🟢"
		if !channel.Active {
			status = "🔴"
		}
		fmt.Printf("\n%s %s\n", status, detail.Alias)
		fmt.Println(strings.Repeat("─", 80))
		avgFeePerForward := int64(0)
		if detail.Forwards > 0 {
			avgFeePerForward = detail.Earnings / int64(detail.Forwards)
		}
		fmt.Printf("💰 Earnings: Month: %s | Week: %s | Avg/Forward: %s\n",
			formatSats(detail.Earnings),
			formatSats(weeklySummaries[0].Earnings),
			formatSats(avgFeePerForward))
		fmt.Printf("📊 Activity: Month: %d forwards | Week: %d forwards\n",
			detail.Forwards, weeklySummaries[0].Forwards)
		localBalance, _ := strconv.ParseInt(channel.LocalBalance, 10, 64)
		capacity, _ := strconv.ParseInt(channel.Capacity, 10, 64)
		localRatio := float64(localBalance) / float64(capacity) * 100
		fmt.Printf("⚡ Channel: %s capacity | %.1f%% local | %s\n",
			formatSats(capacity), localRatio, getChannelStatus(channel))
	}

	fmt.Printf("\n")
	fmt.Println(strings.Repeat("━", 100))
	fmt.Printf("📈 Summary: %d active routing channels\n", len(monthlySummaries))
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

// channelEarningsSummary calculates per-channel earnings and forwards for a given history and channel list
type channelEarningsSummary struct {
	ChanID   string
	Alias    string
	Earnings int64
	Forwards int
}

func calculateChannelEarnings(channels []Channel, events []ForwardingEvent) ([]channelEarningsSummary, int64, int) {
	channelEarnings := make(map[string]int64)
	channelForwards := make(map[string]int)
	totalEarnings := int64(0)
	totalForwards := 0

	for _, event := range events {
		feeMsat, _ := strconv.ParseInt(event.FeeMsat, 10, 64)
		feeSats := feeMsat / 1000
		channelEarnings[event.ChanIdOut] += feeSats
		channelForwards[event.ChanIdOut]++
		totalEarnings += feeSats
		totalForwards++
	}

	var summaries []channelEarningsSummary
	for _, channel := range channels {
		earnings := channelEarnings[channel.ChanID]
		forwards := channelForwards[channel.ChanID]
		if earnings > 0 || forwards > 0 {
			alias := getNodeAlias(channel.RemotePubkey)
			if len(alias) > 15 {
				alias = alias[:12] + "..."
			}
			summaries = append(summaries, channelEarningsSummary{
				ChanID:   channel.ChanID,
				Alias:    alias,
				Earnings: earnings,
				Forwards: forwards,
			})
		}
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Earnings > summaries[j].Earnings
	})
	return summaries, totalEarnings, totalForwards
}

// getSuperDetailedEarningsForTelegram returns a formatted string of super detailed earnings for telegram
func getSuperDetailedEarningsForTelegram() string {
	channels, err := getChannels()
	if err != nil {
		return "❌ Failed to get channel data"
	}

	now := time.Now()
	dayAgo := now.AddDate(0, 0, -1)
	history, err := getForwardingHistory(
		fmt.Sprintf("%d", dayAgo.Unix()),
		fmt.Sprintf("%d", now.Unix()),
	)
	if err != nil || len(history.ForwardingEvents) == 0 {
		return "📊 No forwarding activity in the last 24 hours"
	}

	summaries, totalEarnings, totalForwards := calculateChannelEarnings(channels, history.ForwardingEvents)
	if totalEarnings == 0 {
		return "📊 No fee earnings in the last 24 hours"
	}

	var message strings.Builder
	message.WriteString("💰 24h Routing Summary\n")
	message.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
	message.WriteString(fmt.Sprintf("Total: %s (%d forwards)\n", formatSats(totalEarnings), totalForwards))

	// Show top 5 channels
	maxShow := 5
	if len(summaries) < maxShow {
		maxShow = len(summaries)
	}
	for i := 0; i < maxShow; i++ {
		ch := summaries[i]
		message.WriteString(fmt.Sprintf("• %s: %s (%d)\n",
			ch.Alias, formatSats(ch.Earnings), ch.Forwards))
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
