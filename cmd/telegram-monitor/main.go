package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// Get executable directory
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("Failed to get executable path:", err)
	}
	exeDir := filepath.Dir(exePath)

	// Go up two directories from bin/ to get to project root
	projectRoot := filepath.Dir(exeDir)

	dataDir = filepath.Join(projectRoot, "data")
	stateFile = filepath.Join(dataDir, "last_state.json")
	uptimeFile = filepath.Join(dataDir, "last_uptime.txt")

	// Load configuration from project root
	if err := loadConfig(filepath.Join(projectRoot, ".env")); err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Create data directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	// Check for server reboot
	if err := checkServerReboot(); err != nil {
		log.Printf("Error checking server reboot: %v", err)
	}

	// Get current Lightning state
	currentState, err := getCurrentLightningState()
	if err != nil {
		log.Fatal("Failed to get Lightning state:", err)
	}

	// Check if this is the first run
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		// First run - initialize state and send startup message
		if err := saveState(currentState); err != nil {
			log.Fatal("Failed to save initial state:", err)
		}

		msg := fmt.Sprintf("Lightning Monitor Started\nActive channels: %d\nPending opens: %d\nPending closes: %d\nTotal invoices: %d\n\n<b>Balance Summary:</b>\nOn-chain: %s\nLightning local: %s\nLightning remote: %s\nTotal balance: %s",
			currentState.Channels,
			currentState.PendingOpen,
			currentState.PendingClose,
			currentState.Invoices,
			formatSats(currentState.OnchainBalance),
			formatSats(currentState.LocalBalance),
			formatSats(currentState.RemoteBalance),
			formatSats(currentState.TotalBalance))

		sendTelegram(msg)
		return
	}

	// Load previous state
	prevState, err := loadState()
	if err != nil {
		log.Fatal("Failed to load previous state:", err)
	}

	// Check for changes and send notifications
	checkChannelChanges(currentState, prevState)
	checkForwardingActivity(currentState)
	checkInvoiceChanges(currentState, prevState)
	checkBalanceChanges(currentState, prevState)

	// Save current state
	if err := saveState(currentState); err != nil {
		log.Printf("Failed to save state: %v", err)
	}
}