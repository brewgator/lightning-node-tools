package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
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

func main() {
	if len(os.Args) < 2 {
		showHelp()
		return
	}

	command := os.Args[1]

	switch command {
	case "balance", "bal":
		showChannelBalances()
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
	fmt.Println("  channel-manager balance    Show visual channel balances")
	fmt.Println("  channel-manager bal        Short alias for balance")
	fmt.Println("  channel-manager help       Show this help message")
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

	// Create the visual representation like: 10000 -------- Remote Peer Name ---- 90000
	localStr := formatSats(local)
	remoteStr := formatSats(remote)
	
	// Calculate dash lengths to center the peer name
	totalDashes := 50 - len(localStr) - len(remoteStr) - len(alias) - 2 // -2 for spaces
	if totalDashes < 6 {
		totalDashes = 6
	}
	leftDashes := totalDashes / 2
	rightDashes := totalDashes - leftDashes

	fmt.Printf("%s %s %s %s %s %s\n",
		status,
		localStr,
		strings.Repeat("─", leftDashes),
		alias,
		strings.Repeat("─", rightDashes),
		remoteStr)

	// Show capacity and percentage info on second line
	fmt.Printf("   Capacity: %s │ Local: %.1f%% │ %s\n",
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