package mempool

import "time"

// AddressStats represents comprehensive address statistics from Mempool.space.
//
// Fields tagged with JSON names (e.g. Address, ChainStats, MempoolStats) are
// populated directly from the Mempool.space API response. The TotalReceived,
// TotalSent, and Balance fields are derived values that are not returned by
// the API and therefore use json:"-" to exclude them from (un)marshalling.
// These calculated fields are expected to be populated by higher-level
// application logic after fetching and decoding the raw statistics, typically
// based on the values in ChainStats and MempoolStats.
type AddressStats struct {
	Address      string `json:"address"`
	ChainStats   Stats  `json:"chain_stats"`
	MempoolStats Stats  `json:"mempool_stats"`
	// TotalReceived is the total amount of value ever received by this address.
	// It is a derived field and is not included in the Mempool.space JSON
	// response; callers should compute and populate it from the underlying
	// statistics (for example, using ChainStats.FundedTxoSum).
	TotalReceived int64 `json:"-"`
	// TotalSent is the total amount of value ever sent from this address.
	// It is a derived field and is not included in the Mempool.space JSON
	// response; callers should compute and populate it from the underlying
	// statistics (for example, using ChainStats.SpentTxoSum).
	TotalSent int64 `json:"-"`
	// Balance is the current balance for this address, typically computed as
	// TotalReceived minus TotalSent or from the underlying statistics. It is
	// not returned by the Mempool.space API and must be populated by the
	// caller after decoding the JSON response.
	Balance int64 `json:"-"`
}

// Stats represents transaction and balance statistics
type Stats struct {
	FundedTxoCount int64 `json:"funded_txo_count"`
	FundedTxoSum   int64 `json:"funded_txo_sum"`
	SpentTxoCount  int64 `json:"spent_txo_count"`
	SpentTxoSum    int64 `json:"spent_txo_sum"`
	TxCount        int64 `json:"tx_count"`
}

// UTXO represents an unspent transaction output from Mempool.space
type UTXO struct {
	TxID   string `json:"txid"`
	Vout   int    `json:"vout"`
	Status Status `json:"status"`
	Value  int64  `json:"value"`
}

// Transaction represents a transaction from Mempool.space
type Transaction struct {
	TxID     string   `json:"txid"`
	Version  int      `json:"version"`
	LockTime int64    `json:"locktime"`
	Vin      []Input  `json:"vin"`
	Vout     []Output `json:"vout"`
	Size     int      `json:"size"`
	Weight   int      `json:"weight"`
	Fee      int64    `json:"fee"`
	Status   Status   `json:"status"`
}

// Input represents a transaction input
type Input struct {
	TxID               string   `json:"txid"`
	Vout               int      `json:"vout"`
	PrevOut            *Output  `json:"prevout,omitempty"`
	ScriptSig          string   `json:"scriptsig"`
	ScriptSigAsm       string   `json:"scriptsig_asm"`
	Witness            []string `json:"witness,omitempty"`
	IsCoinbase         bool     `json:"is_coinbase"`
	Sequence           int64    `json:"sequence"`
	InnerWitnessScript string   `json:"inner_witnessscript_asm,omitempty"`
}

// Output represents a transaction output
type Output struct {
	ScriptPubKey     string `json:"scriptpubkey"`
	ScriptPubKeyAsm  string `json:"scriptpubkey_asm"`
	ScriptPubKeyType string `json:"scriptpubkey_type"`
	ScriptPubKeyAddr string `json:"scriptpubkey_address,omitempty"`
	Value            int64  `json:"value"`
}

// Status represents transaction/block status
type Status struct {
	Confirmed   bool   `json:"confirmed"`
	BlockHeight int64  `json:"block_height,omitempty"`
	BlockHash   string `json:"block_hash,omitempty"`
	BlockTime   int64  `json:"block_time,omitempty"`
}

// ChainTips represents blockchain tip information
type ChainTips struct {
	Height int64 `json:"height"`
	// QueryTime reflects when the API was queried, not when the block was mined.
	// To get the actual block time, fetch block details separately.
	QueryTime time.Time `json:"query_time"`
}

// BalanceUpdate represents a balance update from Mempool.space
type BalanceUpdate struct {
	Address     string    `json:"address"`
	Balance     int64     `json:"balance"`    // Balance in satoshis
	TxCount     int64     `json:"tx_count"`   // Total number of transactions
	UTXOCount   int64     `json:"utxo_count"` // Number of UTXOs
	LastUpdated time.Time `json:"last_updated"`
	BlockHeight int64     `json:"block_height"`
	Source      string    `json:"source"` // "mempool" for this client
}

// RateLimitError represents a rate limit error
type RateLimitError struct {
	RetryAfter time.Duration
	Message    string
}

func (e *RateLimitError) Error() string {
	return e.Message
}
