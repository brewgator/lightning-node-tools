package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/brewgator/lightning-node-tools/internal/lnd"
)

// openChannelToPeer opens a new channel to a peer
func openChannelToPeer(peerAddress string, channelSize int64, feeRate int64) error {
	// Parse peer address to extract pubkey
	peerPubkey := extractPubkeyFromAddress(peerAddress)

	fmt.Printf("Opening channel to peer: %s\n", peerAddress)
	fmt.Printf("Channel size: %d sats\n", channelSize)
	fmt.Printf("Fee rate: %d sat/vByte\n", feeRate)
	fmt.Println()

	// Step 1: Connect to the peer
	fmt.Printf("Step 1: Connecting to peer...\n")
	err := lnd.ConnectPeer(peerAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to peer: %v", err)
	}
	fmt.Printf("✓ Successfully connected to peer\n")

	// Step 2: Verify connection
	fmt.Printf("Step 2: Verifying connection...\n")
	peers, err := lnd.ListPeers()
	if err != nil {
		return fmt.Errorf("failed to list peers: %v", err)
	}

	connected := false
	for _, peer := range peers {
		if peer.PubKey == peerPubkey {
			connected = true
			break
		}
	}

	if !connected {
		return fmt.Errorf("peer connection verification failed")
	}
	fmt.Printf("✓ Peer connection verified\n")

	// Step 3: Open the channel
	fmt.Printf("Step 3: Opening channel...\n")
	response, err := lnd.OpenChannel(peerPubkey, channelSize, feeRate)
	if err != nil {
		return fmt.Errorf("failed to open channel: %v", err)
	}

	// Step 4: Output the pending transaction ID
	fmt.Printf("✓ Channel opening initiated successfully!\n")
	fmt.Println()
	fmt.Printf("Pending Transaction ID: %s\n", response.FundingTxidStr)
	fmt.Printf("Output Index: %d\n", response.OutputIndex)
	fmt.Println()
	fmt.Printf("The channel will become active once the transaction is confirmed.\n")
	fmt.Printf("You can monitor the transaction status using: lncli pendingchannels\n")

	return nil
}

// extractPubkeyFromAddress extracts the pubkey from a peer address
// Peer addresses can be in format: pubkey@host:port or just pubkey
func extractPubkeyFromAddress(peerAddress string) string {
	if strings.Contains(peerAddress, "@") {
		parts := strings.Split(peerAddress, "@")
		return parts[0]
	}
	return peerAddress
}

// handleOpenChannel handles the open-channel command
func handleOpenChannel() {
	if len(os.Args) < 8 {
		fmt.Println("Error: Missing required arguments for open-channel command")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  channel-manager open-channel --peer <peer_address> --size <channel_size_sats> --fee-rate <sat_per_vbyte>")
		fmt.Println()
		fmt.Println("Arguments:")
		fmt.Println("  --peer       Peer address in format pubkey@host:port or just pubkey")
		fmt.Println("  --size       Channel size in satoshis")
		fmt.Println("  --fee-rate   Fee rate in satoshis per vByte")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  channel-manager open-channel --peer 02a1b2c3d4e5f6...@192.168.1.100:9735 --size 1000000 --fee-rate 10")
		return
	}

	var peerAddress string
	var channelSize int64
	var feeRate int64

	// Parse command line arguments
	for i := 2; i < len(os.Args); i += 2 {
		if i+1 >= len(os.Args) {
			fmt.Printf("Error: Missing value for argument %s\n", os.Args[i])
			return
		}

		switch os.Args[i] {
		case "--peer":
			peerAddress = os.Args[i+1]
		case "--size":
			var err error
			channelSize, err = strconv.ParseInt(os.Args[i+1], 10, 64)
			if err != nil {
				fmt.Printf("Error: Invalid channel size: %s\n", os.Args[i+1])
				return
			}
		case "--fee-rate":
			var err error
			feeRate, err = strconv.ParseInt(os.Args[i+1], 10, 64)
			if err != nil {
				fmt.Printf("Error: Invalid fee rate: %s\n", os.Args[i+1])
				return
			}
		default:
			fmt.Printf("Error: Unknown argument: %s\n", os.Args[i])
			return
		}
	}

	// Validate required arguments
	if peerAddress == "" {
		fmt.Println("Error: --peer argument is required")
		return
	}
	if channelSize <= 0 {
		fmt.Println("Error: --size argument is required and must be positive")
		return
	}
	if feeRate <= 0 {
		fmt.Println("Error: --fee-rate argument is required and must be positive")
		return
	}

	// Validate minimum channel size (LND minimum is typically 20,000 sats)
	if channelSize < 20000 {
		fmt.Println("Error: Channel size must be at least 20,000 satoshis")
		return
	}

	// Open the channel
	err := openChannelToPeer(peerAddress, channelSize, feeRate)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
}
