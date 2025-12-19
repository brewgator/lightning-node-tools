package mempool

import "time"

// AddressStats represents comprehensive address statistics from Mempool.space
type AddressStats struct {
	Address                 string `json:"address"`
	ChainStats              Stats  `json:"chain_stats"`
	MempoolStats            Stats  `json:"mempool_stats"`
	TotalReceived          int64  `json:"-"` // Calculated field
	TotalSent              int64  `json:"-"` // Calculated field
	Balance                int64  `json:"-"` // Calculated field
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
	TxID     string  `json:"txid"`
	Version  int     `json:"version"`
	LockTime int64   `json:"locktime"`
	Vin      []Input `json:"vin"`
	Vout     []Output `json:"vout"`
	Size     int     `json:"size"`
	Weight   int     `json:"weight"`
	Fee      int64   `json:"fee"`
	Status   Status  `json:"status"`
}

// Input represents a transaction input
type Input struct {
	TxID           string      `json:"txid"`
	Vout           int         `json:"vout"`
	PrevOut        *Output     `json:"prevout,omitempty"`
	ScriptSig      string      `json:"scriptsig"`
	ScriptSigAsm   string      `json:"scriptsig_asm"`
	Witness        []string    `json:"witness,omitempty"`
	IsCoinbase     bool        `json:"is_coinbase"`
	Sequence       int64       `json:"sequence"`
	InnerWitnessScript string  `json:"inner_witnessscript_asm,omitempty"`
}

// Output represents a transaction output
type Output struct {
	ScriptPubKey    string `json:"scriptpubkey"`
	ScriptPubKeyAsm string `json:"scriptpubkey_asm"`
	ScriptPubKeyType string `json:"scriptpubkey_type"`
	ScriptPubKeyAddr string `json:"scriptpubkey_address,omitempty"`
	Value           int64  `json:"value"`
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
	Height    int64     `json:"height"`
	Timestamp time.Time `json:"timestamp"`
}

// BalanceUpdate represents a balance update from Mempool.space
type BalanceUpdate struct {
	Address       string    `json:"address"`
	Balance       int64     `json:"balance"`        // Balance in satoshis
	TxCount       int64     `json:"tx_count"`       // Total number of transactions
	UTXOCount     int64     `json:"utxo_count"`     // Number of UTXOs
	LastUpdated   time.Time `json:"last_updated"`
	BlockHeight   int64     `json:"block_height"`
	Source        string    `json:"source"`         // "mempool" for this client
}

// RateLimitError represents a rate limit error
type RateLimitError struct {
	RetryAfter time.Duration
	Message    string
}

func (e *RateLimitError) Error() string {
	return e.Message
}
