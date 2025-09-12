package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BotToken string
	ChatID   string
}

type LightningState struct {
	Channels       int   `json:"channels"`
	PendingOpen    int   `json:"pending_open"`
	PendingClose   int   `json:"pending_close"`
	Invoices       int   `json:"invoices"`
	Forwards       int   `json:"forwards"`
	OnchainBalance int64 `json:"onchain_balance"`
	LocalBalance   int64 `json:"local_balance"`
	RemoteBalance  int64 `json:"remote_balance"`
	TotalBalance   int64 `json:"total_balance"`
}

type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

const (
	BalanceThreshold     = 10000   // 10k sats
	SignificantThreshold = 1000000 // 1M sats
)

var (
	config     Config
	dataDir    string
	stateFile  string
	uptimeFile string
)

func main() {
	// Get executable directory
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("Failed to get executable path:", err)
	}
	exeDir := filepath.Dir(exePath)

	// Try current directory first (for development)
	if _, err := os.Stat(".env"); err == nil {
		exeDir = "."
	}

	dataDir = filepath.Join(exeDir, "data")
	stateFile = filepath.Join(dataDir, "last_state.json")
	uptimeFile = filepath.Join(dataDir, "last_uptime.txt")

	// Load configuration
	if err := loadConfig(filepath.Join(exeDir, ".env")); err != nil {
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

func getCurrentLightningState() (*LightningState, error) {
	state := &LightningState{}

	// Get channel count
	channels, err := runLNCLI("listchannels")
	if err != nil {
		return nil, err
	}

	var channelData map[string]any
	if err := json.Unmarshal(channels, &channelData); err != nil {
		return nil, err
	}

	if channelList, ok := channelData["channels"].([]any); ok {
		state.Channels = len(channelList)
	}

	// Get pending channels
	pendingChannels, err := runLNCLI("pendingchannels")
	if err != nil {
		return nil, err
	}

	var pendingData map[string]any
	if err := json.Unmarshal(pendingChannels, &pendingData); err != nil {
		return nil, err
	}

	if pendingOpen, ok := pendingData["pending_open_channels"].([]any); ok {
		state.PendingOpen = len(pendingOpen)
	}

	// Count all types of pending closes
	pendingCloses := 0
	if pendingClosing, ok := pendingData["pending_closing_channels"].([]any); ok {
		pendingCloses += len(pendingClosing)
	}
	if pendingForceClosing, ok := pendingData["pending_force_closing_channels"].([]any); ok {
		pendingCloses += len(pendingForceClosing)
	}
	if waitingClose, ok := pendingData["waiting_close_channels"].([]any); ok {
		pendingCloses += len(waitingClose)
	}
	state.PendingClose = pendingCloses

	// Get invoice count
	invoices, err := runLNCLI("listinvoices")
	if err != nil {
		return nil, err
	}

	var invoiceData map[string]any
	if err := json.Unmarshal(invoices, &invoiceData); err != nil {
		return nil, err
	}

	if invoiceList, ok := invoiceData["invoices"].([]any); ok {
		state.Invoices = len(invoiceList)
	}

	// Get wallet balance
	walletBalance, err := runLNCLI("walletbalance")
	if err != nil {
		return nil, err
	}

	var walletData map[string]any
	if err := json.Unmarshal(walletBalance, &walletData); err != nil {
		return nil, err
	}

	if totalBalance, ok := walletData["total_balance"].(string); ok {
		if balance, err := strconv.ParseInt(totalBalance, 10, 64); err == nil {
			state.OnchainBalance = balance
		}
	}

	// Get channel balance
	channelBalance, err := runLNCLI("channelbalance")
	if err != nil {
		return nil, err
	}

	var balanceData map[string]any
	if err := json.Unmarshal(channelBalance, &balanceData); err != nil {
		return nil, err
	}

	if localBal, ok := balanceData["local_balance"].(map[string]any); ok {
		if sat, ok := localBal["sat"].(string); ok {
			if balance, err := strconv.ParseInt(sat, 10, 64); err == nil {
				state.LocalBalance = balance
			}
		}
	}

	if remoteBal, ok := balanceData["remote_balance"].(map[string]any); ok {
		if sat, ok := remoteBal["sat"].(string); ok {
			if balance, err := strconv.ParseInt(sat, 10, 64); err == nil {
				state.RemoteBalance = balance
			}
		}
	}

	// Add pending balances
	if pendingLocalBal, ok := balanceData["pending_open_local_balance"].(map[string]any); ok {
		if sat, ok := pendingLocalBal["sat"].(string); ok {
			if balance, err := strconv.ParseInt(sat, 10, 64); err == nil {
				state.LocalBalance += balance
			}
		}
	}

	if pendingRemoteBal, ok := balanceData["pending_open_remote_balance"].(map[string]any); ok {
		if sat, ok := pendingRemoteBal["sat"].(string); ok {
			if balance, err := strconv.ParseInt(sat, 10, 64); err == nil {
				state.RemoteBalance += balance
			}
		}
	}

	// Calculate total
	state.TotalBalance = state.OnchainBalance + state.LocalBalance + state.RemoteBalance

	// Get recent forwarding events (last 10 minutes)
	recentTime := time.Now().Add(-10 * time.Minute).Unix()
	fwdHistory, err := runLNCLI("fwdinghistory", "--start_time", strconv.FormatInt(recentTime, 10))
	if err != nil {
		return nil, err
	}

	var fwdData map[string]any
	if err := json.Unmarshal(fwdHistory, &fwdData); err != nil {
		return nil, err
	}

	if fwdEvents, ok := fwdData["forwarding_events"].([]any); ok {
		state.Forwards = len(fwdEvents)
	}

	return state, nil
}

func runLNCLI(args ...string) ([]byte, error) {
	cmd := exec.Command("lncli", args...)
	return cmd.Output()
}

func saveState(state *LightningState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(stateFile, data, 0644)
}

func loadState() (*LightningState, error) {
	data, err := ioutil.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	var state LightningState
	err = json.Unmarshal(data, &state)
	return &state, err
}

func sendTelegram(message string) {
	telegramMsg := TelegramMessage{
		ChatID:    config.ChatID,
		Text:      message,
		ParseMode: "HTML",
	}

	jsonData, err := json.Marshal(telegramMsg)
	if err != nil {
		log.Printf("Failed to marshal telegram message: %v", err)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.BotToken)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to send telegram message: %v", err)
		return
	}
	defer resp.Body.Close()
}

func formatSats(amount int64) string {
	if amount >= 100000000 {
		// Show in BTC for amounts >= 1 BTC
		return fmt.Sprintf("%.8f BTC", float64(amount)/100000000)
	} else if amount >= 1000000 {
		// Show in millions for amounts >= 1M sats
		return fmt.Sprintf("%.2fM sats", float64(amount)/1000000)
	} else if amount >= 1000 {
		// Show in thousands for amounts >= 1K sats
		return fmt.Sprintf("%.1fK sats", float64(amount)/1000)
	}
	return fmt.Sprintf("%d sats", amount)
}

func checkChannelChanges(current, prev *LightningState) {
	if current.Channels > prev.Channels {
		newChannels := current.Channels - prev.Channels
		msg := fmt.Sprintf("Channel Opened\nNew active channels: %d\nTotal active channels: %d", newChannels, current.Channels)
		sendTelegram(msg)
	}

	if current.PendingOpen > prev.PendingOpen {
		newPending := current.PendingOpen - prev.PendingOpen
		msg := fmt.Sprintf("New Channel Opening\nNew pending opens: %d\nTotal pending: %d", newPending, current.PendingOpen)
		sendTelegram(msg)
	}

	if current.Channels < prev.Channels {
		closedChannels := prev.Channels - current.Channels
		msg := fmt.Sprintf("Channel Closed\nChannels closed: %d\nRemaining active: %d", closedChannels, current.Channels)
		sendTelegram(msg)
	}

	if current.PendingClose > prev.PendingClose {
		newClosing := current.PendingClose - prev.PendingClose
		msg := fmt.Sprintf("Channel Closing Initiated\nNew pending closes: %d\nTotal pending closes: %d", newClosing, current.PendingClose)
		sendTelegram(msg)
	}
}

func checkForwardingActivity(current *LightningState) {
	if current.Forwards > 0 {
		// Get fee information
		recentTime := time.Now().Add(-10 * time.Minute).Unix()
		fwdHistory, err := runLNCLI("fwdinghistory", "--start_time", strconv.FormatInt(recentTime, 10))
		if err != nil {
			log.Printf("Failed to get forwarding history: %v", err)
			return
		}

		var fwdData map[string]any
		if err := json.Unmarshal(fwdHistory, &fwdData); err != nil {
			log.Printf("Failed to parse forwarding history: %v", err)
			return
		}

		totalFees := int64(0)
		if fwdEvents, ok := fwdData["forwarding_events"].([]any); ok {
			for _, event := range fwdEvents {
				if eventMap, ok := event.(map[string]any); ok {
					if feeMsat, ok := eventMap["fee_msat"].(string); ok {
						if fee, err := strconv.ParseInt(feeMsat, 10, 64); err == nil {
							totalFees += fee
						}
					}
				}
			}
		}

		msg := fmt.Sprintf("Lightning Forwards\nRecent forwards: %d\nFees earned: %d sats", current.Forwards, totalFees/1000)
		sendTelegram(msg)
	}
}

func checkInvoiceChanges(current, prev *LightningState) {
	if current.Invoices > prev.Invoices {
		newInvoices := current.Invoices - prev.Invoices
		msg := fmt.Sprintf("Invoice Paid\nNew payments received: %d\nTotal invoices: %d", newInvoices, current.Invoices)
		sendTelegram(msg)
	}
}

func checkBalanceChanges(current, prev *LightningState) {
	onchainChange := current.OnchainBalance - prev.OnchainBalance
	localChange := current.LocalBalance - prev.LocalBalance
	remoteChange := current.RemoteBalance - prev.RemoteBalance
	totalChange := current.TotalBalance - prev.TotalBalance

	if onchainChange != 0 && int64(math.Abs(float64(onchainChange))) >= BalanceThreshold {
		msg := createBalanceMessage("On-chain", onchainChange, current.OnchainBalance)
		sendTelegram(msg)
	}

	if localChange != 0 && int64(math.Abs(float64(localChange))) >= BalanceThreshold {
		msg := createBalanceMessage("Lightning Local", localChange, current.LocalBalance)
		sendTelegram(msg)
	}

	if remoteChange != 0 && int64(math.Abs(float64(remoteChange))) >= BalanceThreshold {
		msg := createBalanceMessage("Lightning Remote", remoteChange, current.RemoteBalance)
		sendTelegram(msg)
	}

	if totalChange != 0 && int64(math.Abs(float64(totalChange))) >= SignificantThreshold {
		msg := createBalanceMessage("Total Portfolio", totalChange, current.TotalBalance)
		msg += fmt.Sprintf("\n\n<b>Breakdown:</b>\nOn-chain: %s (%+d)\nLightning: %s (%+d)",
			formatSats(current.OnchainBalance), onchainChange,
			formatSats(current.LocalBalance+current.RemoteBalance), localChange+remoteChange)
		sendTelegram(msg)
	}
}

func createBalanceMessage(changeType string, amount int64, current int64) string {
	var emoji string
	var direction string

	if amount > 0 {
		emoji = "📈"
		direction = "increased"
	} else {
		emoji = "📉"
		direction = "decreased"
		amount = -amount // Make positive for display
	}

	if amount >= SignificantThreshold {
		emoji = "⚠️ " + emoji
	}

	return fmt.Sprintf("%s <b>%s Balance %s</b>\nChange: %s\nCurrent: %s",
		emoji, changeType, direction, formatSats(amount), formatSats(current))
}
