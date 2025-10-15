package main

// Config holds the bot configuration
type Config struct {
	BotToken string
	ChatID   string
}

// LightningState represents the current state of the Lightning node
type LightningState struct {
	Channels              int   `json:"channels"`
	PendingOpen           int   `json:"pending_open"`
	PendingClose          int   `json:"pending_close"`
	Invoices              int   `json:"invoices"`
	Forwards              int   `json:"forwards"`
	OnchainBalance        int64 `json:"onchain_balance"`
	LocalBalance          int64 `json:"local_balance"`
	RemoteBalance         int64 `json:"remote_balance"`
	TotalBalance          int64 `json:"total_balance"`
	LastForwardTimestamp  int64 `json:"last_forward_timestamp"`
}

// TelegramMessage represents a message to send via Telegram API
type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// Constants for monitoring thresholds
const (
	MinimalBalanceThreshold = 1       // 1 sat - minimum change to report
	BalanceThreshold        = 1000    // 1k sats - reduced from 10k for small accounts
	SignificantThreshold    = 1000000 // 1M sats
)
