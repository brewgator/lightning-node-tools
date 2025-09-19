package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// showFeeEarnings displays fee earnings summary with optional detailed breakdown
func showFeeEarnings(detailed bool) {
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
		showDetailedChannelEarnings()
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