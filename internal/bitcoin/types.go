package bitcoin

import "time"

// BlockchainInfo represents blockchain information from getblockchaininfo
type BlockchainInfo struct {
	Chain                string  `json:"chain"`
	Blocks               int64   `json:"blocks"`
	Headers              int64   `json:"headers"`
	BestBlockHash        string  `json:"bestblockhash"`
	Difficulty           float64 `json:"difficulty"`
	MedianTime           int64   `json:"mediantime"`
	VerificationProgress float64 `json:"verificationprogress"`
	InitialBlockDownload bool    `json:"initialblockdownload"`
	ChainWork            string  `json:"chainwork"`
	SizeOnDisk           int64   `json:"size_on_disk"`
	Pruned               bool    `json:"pruned"`
	Warnings             string  `json:"warnings"`
}

// UTXO represents an unspent transaction output
type UTXO struct {
	TxID          string  `json:"txid"`
	Vout          int     `json:"vout"`
	Address       string  `json:"address"`
	Label         string  `json:"label,omitempty"`
	ScriptPubKey  string  `json:"scriptPubKey"`
	Amount        float64 `json:"amount"`
	Confirmations int64   `json:"confirmations"`
	RedeemScript  string  `json:"redeemScript,omitempty"`
	WitnessScript string  `json:"witnessScript,omitempty"`
	Spendable     bool    `json:"spendable"`
	Solvable      bool    `json:"solvable"`
	Safe          bool    `json:"safe"`
}

// AddressTransaction represents a transaction involving an address
type AddressTransaction struct {
	Account         string   `json:"account,omitempty"`
	Address         string   `json:"address"`
	Category        string   `json:"category"` // "send", "receive", "generate", etc.
	Amount          float64  `json:"amount"`
	Label           string   `json:"label,omitempty"`
	Vout            int      `json:"vout"`
	Fee             float64  `json:"fee,omitempty"`
	Confirmations   int64    `json:"confirmations"`
	Blockhash       string   `json:"blockhash,omitempty"`
	Blockindex      int      `json:"blockindex,omitempty"`
	Blocktime       int64    `json:"blocktime,omitempty"`
	TxID            string   `json:"txid"`
	WalletConflicts []string `json:"walletconflicts"`
	Time            int64    `json:"time"`
	TimeReceived    int64    `json:"timereceived"`
	Comment         string   `json:"comment,omitempty"`
	To              string   `json:"to,omitempty"`
	OtherAccount    string   `json:"otheraccount,omitempty"`
	Abandoned       bool     `json:"abandoned,omitempty"`
}

// AddressValidation represents the result of validateaddress
type AddressValidation struct {
	IsValid        bool   `json:"isvalid"`
	Address        string `json:"address,omitempty"`
	ScriptType     string `json:"scripttype,omitempty"`
	IsScript       bool   `json:"isscript,omitempty"`
	IsWitness      bool   `json:"iswitness,omitempty"`
	WitnessVersion int    `json:"witness_version,omitempty"`
	WitnessProgram string `json:"witness_program,omitempty"`
	ErrorMessage   string `json:"error,omitempty"`
}

// AddressInfo represents detailed address information (Bitcoin Core 0.17+)
type AddressInfo struct {
	Address             string   `json:"address"`
	ScriptPubKey        string   `json:"scriptpubkey"`
	IsWitness           bool     `json:"iswitness"`
	WitnessVersion      int      `json:"witness_version,omitempty"`
	WitnessProgram      string   `json:"witness_program,omitempty"`
	IsScript            bool     `json:"isscript"`
	IsChange            bool     `json:"ischange"`
	IsMine              bool     `json:"ismine"`
	IsWatchOnly         bool     `json:"iswatchonly"`
	Solvable            bool     `json:"solvable"`
	Desc                string   `json:"desc,omitempty"`
	Parent              string   `json:"parent_desc,omitempty"`
	Checksum            string   `json:"checksum,omitempty"`
	HDKeyPath           string   `json:"hdkeypath,omitempty"`
	HDSeedID            string   `json:"hdseedid,omitempty"`
	HDMasterFingerprint string   `json:"hdmasterfingerprint,omitempty"`
	Labels              []string `json:"labels"`
}

// BalanceUpdate represents a balance update for an address
type BalanceUpdate struct {
	Address       string    `json:"address"`
	Balance       int64     `json:"balance"`       // Balance in satoshis
	Confirmations int64     `json:"confirmations"` // Number of confirmations
	TxCount       int64     `json:"tx_count"`      // Number of transactions
	LastUpdated   time.Time `json:"last_updated"`
	BlockHeight   int64     `json:"block_height"`
}

// DescriptorInfo represents the result of getdescriptorinfo
type DescriptorInfo struct {
	Descriptor     string `json:"descriptor"`
	Checksum       string `json:"checksum"`
	IsRange        bool   `json:"isrange"`
	IsSolvable     bool   `json:"issolvable"`
	HasPrivateKeys bool   `json:"hasprivatekeys"`
}

// BalanceHistoryPoint represents a point in time balance for step chart visualization
// This is used to create Sparrow-wallet-style balance history charts
type BalanceHistoryPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Balance   int64     `json:"balance"`            // Balance in satoshis at this point in time
	TxID      string    `json:"txid,omitempty"`     // Transaction ID that caused this balance change
	Amount    int64     `json:"amount,omitempty"`   // Amount of this transaction in satoshis
	Category  string    `json:"category,omitempty"` // Transaction category: "receive", "send", "generate"
}
