package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// showChannelBalances displays visual channel balance overview
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

// displayChannel displays balance information for a single channel
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