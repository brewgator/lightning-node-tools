package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// showChannelFees displays the fee information for all channels
func showChannelFees() {
	channels, err := getChannels()
	if err != nil {
		log.Fatal("Failed to get channels:", err)
	}

	feeReport, err := getFeeReport()
	if err != nil {
		log.Fatal("Failed to get fee report:", err)
	}

	if len(channels) == 0 {
		fmt.Println("No channels found")
		return
	}

	fmt.Println("\n💰 Channel Fees Overview")
	fmt.Println(strings.Repeat("━", 95))

	// Create a map for quick fee lookup by channel ID
	feeMap := make(map[string]ChannelFeeReport)
	for _, fee := range feeReport.ChannelFees {
		feeMap[fee.ChanID] = fee
	}

	fmt.Printf("%-32s %-20s %-12s %-12s %s\n", "Channel", "Channel ID", "Base Fee", "Fee Rate", "Status")
	fmt.Println(strings.Repeat("─", 95))

	for _, channel := range channels {
		displayChannelFees(channel, feeMap)
	}

	// Summary
	fmt.Println(strings.Repeat("━", 95))
	if feeReport.DayFeeSum != "" || feeReport.WeekFeeSum != "" || feeReport.MonthFeeSum != "" {
		fmt.Printf("📊 Fee Summary:\n")
		if feeReport.DayFeeSum != "" {
			dayFee, _ := strconv.ParseInt(feeReport.DayFeeSum, 10, 64)
			fmt.Printf("   Today: %s", formatSats(dayFee))
		}
		if feeReport.WeekFeeSum != "" {
			weekFee, _ := strconv.ParseInt(feeReport.WeekFeeSum, 10, 64)
			fmt.Printf(" │ Week: %s", formatSats(weekFee))
		}
		if feeReport.MonthFeeSum != "" {
			monthFee, _ := strconv.ParseInt(feeReport.MonthFeeSum, 10, 64)
			fmt.Printf(" │ Month: %s", formatSats(monthFee))
		}
		fmt.Println()
	}
	fmt.Println()
}

// displayChannelFees displays fee information for a single channel
func displayChannelFees(channel Channel, feeMap map[string]ChannelFeeReport) {
	alias := getNodeAlias(channel.RemotePubkey)

	// Truncate alias if too long
	if len(alias) > 29 {
		alias = alias[:26] + "..."
	}

	// Status indicator
	status := "🟢"
	if !channel.Active {
		status = "🔴"
	}

	// Get fee information
	baseFee := "N/A"
	feeRatePPM := "N/A"

	if feeInfo, exists := feeMap[channel.ChanID]; exists {
		if feeInfo.BaseFeeMsat != "" {
			baseFeeMsat, _ := strconv.ParseInt(feeInfo.BaseFeeMsat, 10, 64)
			baseFee = fmt.Sprintf("%d msat", baseFeeMsat)
		}

		// Use FeePerMil if available, otherwise convert FeeRate to PPM
		if feeInfo.FeePerMil != "" {
			feeRatePPM = feeInfo.FeePerMil + " ppm"
		} else if feeInfo.FeeRate > 0 {
			// Convert fee rate to PPM (parts per million)
			ppmValue := feeInfo.FeeRate * 1000000
			feeRatePPM = fmt.Sprintf("%.0f ppm", ppmValue)
		}
	}

	fmt.Printf("%s %-29s %-20s %-12s %-12s %s\n",
		status,
		alias+":",
		channel.ChanID,
		baseFee,
		feeRatePPM,
		getChannelStatus(channel))
}

// handleSetFees handles the set-fees command
func handleSetFees() {
	if len(os.Args) < 6 {
		fmt.Println("Usage: channel-manager set-fees --channel-id <ID> --base-fee <msat> --ppm <rate>")
		fmt.Println("   or: channel-manager set-fees --channel-id <ID> --ppm <rate>")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  channel-manager set-fees --channel-id 12345 --base-fee 1000 --ppm 1")
		fmt.Println("  channel-manager set-fees --channel-id 12345 --ppm 2")
		return
	}

	var channelID, baseFee, ppm string

	// Parse arguments
	for i := 2; i < len(os.Args); i += 2 {
		if i+1 >= len(os.Args) {
			fmt.Printf("Error: Missing value for %s\n", os.Args[i])
			return
		}

		switch os.Args[i] {
		case "--channel-id":
			channelID = os.Args[i+1]
		case "--base-fee":
			baseFee = os.Args[i+1]
		case "--ppm":
			ppm = os.Args[i+1]
		default:
			fmt.Printf("Unknown flag: %s\n", os.Args[i])
			return
		}
	}

	if channelID == "" || ppm == "" {
		fmt.Println("Error: --channel-id and --ppm are required")
		return
	}

	// Validate arguments
	if _, err := strconv.ParseInt(channelID, 10, 64); err != nil {
		fmt.Printf("Error: Invalid channel ID: %s\n", channelID)
		return
	}

	if _, err := strconv.ParseInt(ppm, 10, 64); err != nil {
		fmt.Printf("Error: Invalid PPM rate: %s\n", ppm)
		return
	}

	if baseFee != "" {
		if _, err := strconv.ParseInt(baseFee, 10, 64); err != nil {
			fmt.Printf("Error: Invalid base fee: %s\n", baseFee)
			return
		}
	}

	err := setChannelFees(channelID, baseFee, ppm)
	if err != nil {
		fmt.Printf("Error setting fees: %v\n", err)
		return
	}

	fmt.Printf("✅ Successfully updated fees for channel %s\n", channelID)
	if baseFee != "" {
		fmt.Printf("   Base fee: %s msat\n", baseFee)
	}
	fmt.Printf("   Fee rate: %s ppm\n", ppm)
}

// handleBulkSetFees handles the bulk-set-fees command
func handleBulkSetFees() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: channel-manager bulk-set-fees --ppm <rate> [--base-fee <msat>]")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  channel-manager bulk-set-fees --ppm 1")
		fmt.Println("  channel-manager bulk-set-fees --ppm 2 --base-fee 1000")
		return
	}

	var baseFee, ppm string

	// Parse arguments
	for i := 2; i < len(os.Args); i += 2 {
		if i+1 >= len(os.Args) {
			fmt.Printf("Error: Missing value for %s\n", os.Args[i])
			return
		}

		switch os.Args[i] {
		case "--base-fee":
			baseFee = os.Args[i+1]
		case "--ppm":
			ppm = os.Args[i+1]
		default:
			fmt.Printf("Unknown flag: %s\n", os.Args[i])
			return
		}
	}

	if ppm == "" {
		fmt.Println("Error: --ppm is required")
		return
	}

	// Validate arguments
	if _, err := strconv.ParseInt(ppm, 10, 64); err != nil {
		fmt.Printf("Error: Invalid PPM rate: %s\n", ppm)
		return
	}

	if baseFee != "" {
		if _, err := strconv.ParseInt(baseFee, 10, 64); err != nil {
			fmt.Printf("Error: Invalid base fee: %s\n", baseFee)
			return
		}
	}

	// Get all channels
	channels, err := getChannels()
	if err != nil {
		fmt.Printf("Error getting channels: %v\n", err)
		return
	}

	if len(channels) == 0 {
		fmt.Println("No channels found")
		return
	}

	fmt.Printf("🔄 Setting fees for %d channels...\n", len(channels))

	successCount := 0
	for _, channel := range channels {
		if !channel.Active {
			fmt.Printf("⏭️  Skipping inactive channel: %s\n", getNodeAlias(channel.RemotePubkey))
			continue
		}

		err := setChannelFees(channel.ChanID, baseFee, ppm)
		if err != nil {
			fmt.Printf("❌ Failed to set fees for %s: %v\n", getNodeAlias(channel.RemotePubkey), err)
			continue
		}

		fmt.Printf("✅ %s\n", getNodeAlias(channel.RemotePubkey))
		successCount++
	}

	fmt.Printf("\n📊 Summary: Successfully updated %d/%d channels\n", successCount, len(channels))
	if baseFee != "" {
		fmt.Printf("   Base fee: %s msat\n", baseFee)
	}
	fmt.Printf("   Fee rate: %s ppm\n", ppm)
}
