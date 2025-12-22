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

// ColdStorageBalanceHistory represents balance changes for cold storage accounts over time
type ColdStorageBalanceHistory struct {
	ID              int64     `json:"id" db:"id"`
	AccountID       int64     `json:"account_id" db:"account_id"`
	Timestamp       time.Time `json:"timestamp" db:"timestamp"`
	Balance         int64     `json:"balance" db:"balance"`
	PreviousBalance int64     `json:"previous_balance" db:"previous_balance"`
	IsVerified      bool      `json:"is_verified" db:"is_verified"`
	Notes           string    `json:"notes" db:"notes"`
}

// MultisigWallet represents a multisig wallet configuration
type MultisigWallet struct {
	ID                   int64                    `json:"id" db:"id"`
	Name                 string                   `json:"name" db:"name"`
	UUID                 string                   `json:"uuid" db:"uuid"`
	AddressType          string                   `json:"address_type" db:"address_type"`
	Network              string                   `json:"network" db:"network"`
	RequiredSigners      int                      `json:"required_signers" db:"required_signers"`
	TotalSigners         int                      `json:"total_signers" db:"total_signers"`
	StartingAddressIndex int                      `json:"starting_address_index" db:"starting_address_index"`
	ExtendedPublicKeys   []MultisigExtendedPubKey `json:"extended_public_keys" db:"-"`
	Active               bool                     `json:"active" db:"active"`
	CreatedAt            time.Time                `json:"created_at" db:"created_at"`
	LastScanned          *time.Time               `json:"last_scanned" db:"last_scanned"`
	NextAddressIndex     int                      `json:"next_address_index" db:"next_address_index"`
}

// MultisigExtendedPubKey represents an extended public key in a multisig configuration
type MultisigExtendedPubKey struct {
	ID          int64  `json:"id" db:"id"`
	WalletID    int64  `json:"wallet_id" db:"wallet_id"`
	Name        string `json:"name" db:"name"`
	XPub        string `json:"xpub" db:"xpub"`
	BIP32Path   string `json:"bip32_path" db:"bip32_path"`
	Fingerprint string `json:"fingerprint" db:"fingerprint"`
	KeyIndex    int    `json:"key_index" db:"key_index"`
}

// MultisigAddress represents a derived address from a multisig wallet
type MultisigAddress struct {
	ID           int64     `json:"id" db:"id"`
	WalletID     int64     `json:"wallet_id" db:"wallet_id"`
	Address      string    `json:"address" db:"address"`
	AddressIndex int       `json:"address_index" db:"address_index"`
	ScriptType   string    `json:"script_type" db:"script_type"`
	RedeemScript string    `json:"redeem_script" db:"redeem_script"`
	Active       bool      `json:"active" db:"active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// WalletConfig represents the JSON structure of wallet config files
type WalletConfig struct {
	Name                 string                 `json:"name"`
	UUID                 string                 `json:"uuid"`
	AddressType          string                 `json:"addressType"`
	Network              string                 `json:"network"`
	Quorum               WalletQuorum           `json:"quorum"`
	StartingAddressIndex int                    `json:"startingAddressIndex"`
	ExtendedPublicKeys   []WalletExtendedPubKey `json:"extendedPublicKeys"`
}

// WalletQuorum represents the quorum configuration in wallet config
type WalletQuorum struct {
	RequiredSigners int `json:"requiredSigners"`
	TotalSigners    int `json:"totalSigners"`
}

// WalletExtendedPubKey represents an extended public key in wallet config JSON
type WalletExtendedPubKey struct {
	Name      string `json:"name"`
	XPub      string `json:"xpub"`
	BIP32Path string `json:"bip32Path"`
	XFP       string `json:"xfp"`
}
