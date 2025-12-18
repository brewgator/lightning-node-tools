package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/brewgator/lightning-node-tools/internal/utils"
)

// Global variables for configuration and file paths
var (
	config     Config
	dataDir    string
	stateFile  string
	uptimeFile string
)

// loadConfig loads configuration from the .env file
func loadConfig(envFile string) error {
	file, err := os.Open(envFile)
	if err != nil {
		return fmt.Errorf(".env file not found. Please copy .env.example to .env and configure your tokens")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

		switch key {
		case "BOT_TOKEN":
			config.BotToken = value
		case "CHAT_ID":
			config.ChatID = value
		}
	}

	if config.BotToken == "" || config.ChatID == "" {
		return fmt.Errorf("BOT_TOKEN and CHAT_ID must be set in .env file")
	}

	return scanner.Err()
}

// saveState saves the Lightning state to disk
func saveState(state *LightningState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(stateFile, data, 0644)
}

// loadState loads the Lightning state from disk
func loadState() (*LightningState, error) {
	data, err := ioutil.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	var state LightningState
	err = json.Unmarshal(data, &state)
	return &state, err
}

// formatSats formats satoshi amounts using shared utility
func formatSats(amount int64) string {
	return utils.FormatSats(amount)
}

// formatSatsChange formats balance changes with better precision for small amounts
func formatSatsChange(amount int64) string {
	if amount == 0 {
		return "0 sats"
	}

	absAmount := amount
	if absAmount < 0 {
		absAmount = -absAmount
	}

	// For very small amounts, always show exact sats
	if absAmount < 1000 {
		return fmt.Sprintf("%d sats", amount)
	}

	// For larger amounts, use the standard formatting
	return utils.FormatSats(amount)
}

// checkServerReboot checks if the server has rebooted and sends notification
func checkServerReboot() error {
	uptime, err := getSystemUptime()
	if err != nil {
		return err
	}

	if _, err := os.Stat(uptimeFile); os.IsNotExist(err) {
		// First run, just save uptime
		return ioutil.WriteFile(uptimeFile, []byte(strconv.Itoa(uptime)), 0644)
	}

	prevUptimeBytes, err := os.ReadFile(uptimeFile)
	if err != nil {
		return err
	}

	prevUptime, err := strconv.Atoi(strings.TrimSpace(string(prevUptimeBytes)))
	if err != nil {
		return err
	}

	if uptime < prevUptime {
		msg := fmt.Sprintf("Server Rebooted\nUptime: %d seconds\nPrevious uptime was: %d seconds", uptime, prevUptime)
		sendTelegram(msg)
	}

	return ioutil.WriteFile(uptimeFile, []byte(strconv.Itoa(uptime)), 0644)
}

// getSystemUptime returns the system uptime in seconds
func getSystemUptime() (int, error) {
	uptimeBytes, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	uptimeStr := strings.Split(strings.TrimSpace(string(uptimeBytes)), " ")[0]
	uptimeFloat, err := strconv.ParseFloat(uptimeStr, 64)
	if err != nil {
		return 0, err
	}

	return int(uptimeFloat), nil
}
