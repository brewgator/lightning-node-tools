package db

import (
	"time"
)

// BalanceSnapshot represents a point-in-time snapshot of portfolio balances
type BalanceSnapshot struct {
	ID        int64     `json:"id" db:"id"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`

	// Lightning balances
	LightningLocal  int64 `json:"lightning_local" db:"lightning_local"`
	LightningRemote int64 `json:"lightning_remote" db:"lightning_remote"`

	// On-chain balances
	OnchainConfirmed   int64 `json:"onchain_confirmed" db:"onchain_confirmed"`
	OnchainUnconfirmed int64 `json:"onchain_unconfirmed" db:"onchain_unconfirmed"`

	// Tracked on-chain addresses
	TrackedAddresses int64 `json:"tracked_addresses" db:"tracked_addresses"`

	// Cold storage (manual entry)
	ColdStorage int64 `json:"cold_storage" db:"cold_storage"`

	// Calculated totals
	TotalPortfolio int64 `json:"total_portfolio" db:"total_portfolio"`
	TotalLiquid    int64 `json:"total_liquid" db:"total_liquid"` // Excludes cold storage
}

// ChannelSnapshot represents channel-specific data over time
type ChannelSnapshot struct {
	ID            int64     `json:"id" db:"id"`
	Timestamp     time.Time `json:"timestamp" db:"timestamp"`
	ChannelID     string    `json:"channel_id" db:"channel_id"`
	Capacity      int64     `json:"capacity" db:"capacity"`
	LocalBalance  int64     `json:"local_balance" db:"local_balance"`
	RemoteBalance int64     `json:"remote_balance" db:"remote_balance"`
	Active        bool      `json:"active" db:"active"`
	PeerAlias     string    `json:"peer_alias" db:"peer_alias"`
	FeePPM        int64     `json:"fee_ppm" db:"fee_ppm"`
	BaseFee       int64     `json:"base_fee" db:"base_fee"`
}

// ForwardingEvent represents a forwarding event for analytics
type ForwardingEvent struct {
	ID           int64     `json:"id" db:"id"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
	ChannelInID  string    `json:"channel_in_id" db:"channel_in_id"`
	ChannelOutID string    `json:"channel_out_id" db:"channel_out_id"`
	AmountIn     int64     `json:"amount_in" db:"amount_in"`
	AmountOut    int64     `json:"amount_out" db:"amount_out"`
	Fee          int64     `json:"fee" db:"fee"`
}

// OnchainAddress represents a tracked Bitcoin address
type OnchainAddress struct {
	ID      int64  `json:"id" db:"id"`
	Address string `json:"address" db:"address"`
	Label   string `json:"label" db:"label"`
	Active  bool   `json:"active" db:"active"`
}

// AddressBalance represents the balance of a tracked address at a point in time
type AddressBalance struct {
	ID        int64     `json:"id" db:"id"`
	AddressID int64     `json:"address_id" db:"address_id"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
	Balance   int64     `json:"balance" db:"balance"`
	TxCount   int64     `json:"tx_count" db:"tx_count"`
}

// ColdStorageEntry represents manually tracked cold storage
type ColdStorageEntry struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Balance     int64     `json:"balance" db:"balance"`
	LastUpdated time.Time `json:"last_updated" db:"last_updated"`
	Notes       string    `json:"notes" db:"notes"`
}

// DailyFeeData represents aggregated fee data for a specific day
type DailyFeeData struct {
	Date         string `json:"date" db:"date"`
	TotalFee     int64  `json:"total_fee" db:"total_fee"`
	ForwardCount int64  `json:"forward_count" db:"forward_count"`
}
