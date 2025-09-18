package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Channel struct {
	ChanID        string `json:"chan_id"`
	RemotePubkey  string `json:"remote_pubkey"`
	Capacity      string `json:"capacity"`
	LocalBalance  string `json:"local_balance"`
	RemoteBalance string `json:"remote_balance"`
	Active        bool   `json:"active"`
	Private       bool   `json:"private"`
}

type NodeInfo struct {
	Alias string `json:"alias"`
}

type NodeResponse struct {
	Node NodeInfo `json:"node"`
}

type ChannelResponse struct {
	Channels []Channel `json:"channels"`
}

type ChannelFeeReport struct {
	ChanID       string  `json:"chan_id"`
	ChannelPoint string  `json:"channel_point"`
	BaseFeeMsat  string  `json:"base_fee_msat"`
	FeePerMil    string  `json:"fee_per_mil"`
	FeeRate      float64 `json:"fee_rate"`
}

type FeeReportResponse struct {
	ChannelFees []ChannelFeeReport `json:"channel_fees"`
	DayFeeSum   string             `json:"day_fee_sum"`
	WeekFeeSum  string             `json:"week_fee_sum"`
	MonthFeeSum string             `json:"month_fee_sum"`
}

type ForwardingHistory struct {
	ForwardingEvents []ForwardingEvent `json:"forwarding_events"`
}

type ForwardingEvent struct {
	ChanIdIn  string `json:"chan_id_in"`
	ChanIdOut string `json:"chan_id_out"`
	AmtIn     string `json:"amt_in"`
	AmtOut    string `json:"amt_out"`
	Fee       string `json:"fee"`
	FeeMsat   string `json:"fee_msat"`
	Timestamp string `json:"timestamp"`
}

func main() {
	if len(os.Args) < 2 {
		showHelp()
		return
	}

	command := os.Args[1]

	switch command {
	case "balance", "bal":
		showChannelBalances()
	case "fees":
		showChannelFees()
	case "earnings":
		detailed := false
		if len(os.Args) > 2 && (os.Args[2] == "--detailed" || os.Args[2] == "-d") {
			detailed = true
		}
		showFeeEarnings(detailed)
	case "help", "-h", "--help":
		showHelp()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		showHelp()
	}
}

func showHelp() {
	fmt.Println("Channel Manager - Lightning Network Channel Liquidity Tool")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  channel-manager balance              Show visual channel balances")
	fmt.Println("  channel-manager bal                  Short alias for balance")
	fmt.Println("  channel-manager fees                 Show channel fees information")
	fmt.Println("  channel-manager earnings             Show fee earnings summary")
	fmt.Println("  channel-manager earnings --detailed  Show earnings with per-channel breakdown")
	fmt.Println("  channel-manager earnings -d          Short alias for --detailed")
	fmt.Println("  channel-manager help                 Show this help message")
	fmt.Println("")
}

func showChannelBalances() {
	channels, err := getChannels()
	if err != nil {
		log.Fatal("Failed to get channels:", err)
	}

	if len(channels) == 0 {
		fmt.Println("No channels found")
		return
	}

	fmt.Println("\n🔋 Channel Liquidity Overview")
	fmt.Println(strings.Repeat("━", 80))

	for _, channel := range channels {
		displayChannel(channel)
	}

	// Summary
	totalCapacity := int64(0)
	totalLocal := int64(0)
	totalRemote := int64(0)
	activeChannels := 0

	for _, channel := range channels {
		capacity, _ := strconv.ParseInt(channel.Capacity, 10, 64)
		local, _ := strconv.ParseInt(channel.LocalBalance, 10, 64)
		remote, _ := strconv.ParseInt(channel.RemoteBalance, 10, 64)

		totalCapacity += capacity
		totalLocal += local
		totalRemote += remote

		if channel.Active {
			activeChannels++
		}
	}

	fmt.Println(strings.Repeat("━", 80))
	fmt.Printf("📊 Summary: %d/%d active channels | Total: %s | Local: %s | Remote: %s\n\n",
		activeChannels, len(channels),
		formatSats(totalCapacity),
		formatSats(totalLocal),
		formatSats(totalRemote))
}

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
	fmt.Println(strings.Repeat("━", 75))

	// Create a map for quick fee lookup by channel ID
	feeMap := make(map[string]ChannelFeeReport)
	for _, fee := range feeReport.ChannelFees {
		feeMap[fee.ChanID] = fee
	}

	fmt.Printf("%-32s %-12s %-12s %s\n", "Channel", "Base Fee", "Fee Rate", "Status")
	fmt.Println(strings.Repeat("─", 75))

	for _, channel := range channels {
		displayChannelFees(channel, feeMap)
	}

	// Summary
	fmt.Println(strings.Repeat("━", 75))
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

	fmt.Printf("%s %-29s %-12s %-12s %s\n",
		status,
		alias+":",
		baseFee,
		feeRatePPM,
		getChannelStatus(channel))
}

func displayChannel(channel Channel) {
	alias := getNodeAlias(channel.RemotePubkey)
	capacity, _ := strconv.ParseInt(channel.Capacity, 10, 64)
	local, _ := strconv.ParseInt(channel.LocalBalance, 10, 64)
	remote, _ := strconv.ParseInt(channel.RemoteBalance, 10, 64)

	// Truncate alias if too long
	if len(alias) > 22 {
		alias = alias[:19] + "..."
	}

	// Status indicator
	status := "🟢"
	if !channel.Active {
		status = "🔴"
	}

	// Calculate percentage for display
	localPercent := float64(0)
	if capacity > 0 {
		localPercent = (float64(local) / float64(capacity)) * 100
	}

	// Create progress bar visualization: |#####------------------------|
	barWidth := 30
	localWidth := 0
	if capacity > 0 {
		localWidth = int((float64(local) / float64(capacity)) * float64(barWidth))
		if localWidth > barWidth {
			localWidth = barWidth
		}
	}
	remoteWidth := barWidth - localWidth

	// Create the progress bar with # for local balance and - for remote
	localBar := strings.Repeat("#", localWidth)
	remoteBar := strings.Repeat("-", remoteWidth)

	// Format the balances
	localStr := formatSats(local)
	remoteStr := formatSats(remote)

	// Pad alias name for consistent alignment
	const maxNameWidth = 25
	if len(alias) > maxNameWidth {
		alias = alias[:maxNameWidth-3] + "..."
	}

	// Display: "Remote Name: |#####-----------| 1000/90000"
	fmt.Printf("%s %-*s |%s%s| %s/%s\n",
		status,
		maxNameWidth, alias+":",
		localBar,
		remoteBar,
		localStr,
		remoteStr)

	// Show capacity and percentage info on second line
	fmt.Printf("   %*s  Capacity: %s │ Local: %.1f%% │ %s\n",
		maxNameWidth, "",
		formatSats(capacity),
		localPercent,
		getChannelStatus(channel))

	fmt.Println()
}

func getChannelStatus(channel Channel) string {
	status := ""
	if channel.Private {
		status += "Private"
	} else {
		status += "Public"
	}

	if !channel.Active {
		status += " (Inactive)"
	}

	return status
}

func getChannels() ([]Channel, error) {
	output, err := runLNCLI("listchannels")
	if err != nil {
		return nil, err
	}

	var response ChannelResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}

	return response.Channels, nil
}

func getFeeReport() (*FeeReportResponse, error) {
	output, err := runLNCLI("feereport")
	if err != nil {
		return nil, err
	}

	var response FeeReportResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func getForwardingHistory(startTime, endTime string) (*ForwardingHistory, error) {
	output, err := runLNCLI("fwdinghistory", "--start_time", startTime, "--end_time", endTime)
	if err != nil {
		return nil, err
	}

	var response ForwardingHistory
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func getNodeAlias(pubkey string) string {
	output, err := runLNCLI("getnodeinfo", pubkey)
	if err != nil {
		// Return truncated pubkey if we can't get alias
		if len(pubkey) > 12 {
			return pubkey[:12] + "..."
		}
		return pubkey
	}

	var response NodeResponse
	if err := json.Unmarshal(output, &response); err != nil {
		// Return truncated pubkey if parsing fails
		if len(pubkey) > 12 {
			return pubkey[:12] + "..."
		}
		return pubkey
	}

	if response.Node.Alias == "" {
		// Return truncated pubkey if no alias
		if len(pubkey) > 12 {
			return pubkey[:12] + "..."
		}
		return pubkey
	}

	return response.Node.Alias
}

func runLNCLI(args ...string) ([]byte, error) {
	cmd := exec.Command("lncli", args...)
	return cmd.Output()
}

func formatSats(amount int64) string {
	if amount >= 100000000 {
		// Show in BTC for amounts >= 1 BTC
		return fmt.Sprintf("%.3f BTC", float64(amount)/100000000)
	} else if amount >= 1000000 {
		// Show in millions for amounts >= 1M sats
		return fmt.Sprintf("%.1fM", float64(amount)/1000000)
	} else if amount >= 1000 {
		// Show in thousands for amounts >= 1K sats
		return fmt.Sprintf("%.0fK", float64(amount)/1000)
	}
	return fmt.Sprintf("%d", amount)
}
