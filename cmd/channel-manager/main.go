package main

import (
	"fmt"
	"os"
)

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
	case "set-fees":
		handleSetFees()
	case "bulk-set-fees":
		handleBulkSetFees()
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
	fmt.Println("  View Commands:")
	fmt.Println("    channel-manager balance              Show visual channel balances")
	fmt.Println("    channel-manager bal                  Short alias for balance")
	fmt.Println("    channel-manager fees                 Show channel fees information")
	fmt.Println("    channel-manager earnings             Show fee earnings summary")
	fmt.Println("    channel-manager earnings --detailed  Show earnings with per-channel breakdown")
	fmt.Println("    channel-manager earnings -d          Short alias for --detailed")
	fmt.Println("")
	fmt.Println("  Fee Management Commands:")
	fmt.Println("    channel-manager set-fees --channel-id <ID> --ppm <rate> [--base-fee <msat>]")
	fmt.Println("                                         Set fees for specific channel")
	fmt.Println("    channel-manager bulk-set-fees --ppm <rate> [--base-fee <msat>]")
	fmt.Println("                                         Set fees for all active channels")
	fmt.Println("")
	fmt.Println("  Examples:")
	fmt.Println("    channel-manager set-fees --channel-id 12345 --ppm 1 --base-fee 1000")
	fmt.Println("    channel-manager bulk-set-fees --ppm 2")
	fmt.Println("")
	fmt.Println("  Help:")
	fmt.Println("    channel-manager help                 Show this help message")
	fmt.Println("")
}
