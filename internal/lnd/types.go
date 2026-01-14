package lnd

// Channel represents a Lightning Network channel
type Channel struct {
	ChanID        string `json:"chan_id"`
	RemotePubkey  string `json:"remote_pubkey"`
	Capacity      string `json:"capacity"`
	LocalBalance  string `json:"local_balance"`
	RemoteBalance string `json:"remote_balance"`
	Active        bool   `json:"active"`
	Private       bool   `json:"private"`
}

// NodeInfo represents basic node information
type NodeInfo struct {
	Alias string `json:"alias"`
}

// NodeResponse represents the response from getnodeinfo
type NodeResponse struct {
	Node NodeInfo `json:"node"`
}

// ChannelResponse represents the response from listchannels
type ChannelResponse struct {
	Channels []Channel `json:"channels"`
}

// ChannelFeeReport represents fee information for a channel
type ChannelFeeReport struct {
	ChanID       string  `json:"chan_id"`
	ChannelPoint string  `json:"channel_point"`
	BaseFeeMsat  string  `json:"base_fee_msat"`
	FeePerMil    string  `json:"fee_per_mil"`
	FeeRate      float64 `json:"fee_rate"`
}

// FeeReportResponse represents the response from feereport
type FeeReportResponse struct {
	ChannelFees []ChannelFeeReport `json:"channel_fees"`
	DayFeeSum   string             `json:"day_fee_sum"`
	WeekFeeSum  string             `json:"week_fee_sum"`
	MonthFeeSum string             `json:"month_fee_sum"`
}

// ForwardingHistory represents the response from fwdinghistory
type ForwardingHistory struct {
	ForwardingEvents []ForwardingEvent `json:"forwarding_events"`
}

// ForwardingEvent represents a single forwarding event
type ForwardingEvent struct {
	ChanIdIn  string `json:"chan_id_in"`
	ChanIdOut string `json:"chan_id_out"`
	AmtIn     string `json:"amt_in"`
	AmtOut    string `json:"amt_out"`
	Fee       string `json:"fee"`
	FeeMsat   string `json:"fee_msat"`
	Timestamp string `json:"timestamp"`
}

// RoutingPolicy represents the routing policy for a channel
type RoutingPolicy struct {
	TimeLockDelta    uint32 `json:"time_lock_delta"`
	MinHtlc          string `json:"min_htlc"`
	FeeBaseMsat      string `json:"fee_base_msat"`
	FeeRateMilliMsat string `json:"fee_rate_milli_msat"`
	Disabled         bool   `json:"disabled"`
	MaxHtlcMsat      string `json:"max_htlc_msat"`
}

// ChannelEdge represents a channel edge from getchaninfo
type ChannelEdge struct {
	ChannelId   string        `json:"channel_id"`
	ChanPoint   string        `json:"chan_point"`
	Node1Pub    string        `json:"node1_pub"`
	Node2Pub    string        `json:"node2_pub"`
	Node1Policy RoutingPolicy `json:"node1_policy"`
	Node2Policy RoutingPolicy `json:"node2_policy"`
}

// Peer represents a connected peer
type Peer struct {
	PubKey    string `json:"pub_key"`
	Address   string `json:"address"`
	BytesSent string `json:"bytes_sent"`
	BytesRecv string `json:"bytes_recv"`
}

// PeerResponse represents the response from listpeers
type PeerResponse struct {
	Peers []Peer `json:"peers"`
}

// OpenChannelResponse represents the response from openchannel
type OpenChannelResponse struct {
	FundingTxidBytes string `json:"funding_txid_bytes"`
	FundingTxidStr   string `json:"funding_txid_str"`
	OutputIndex      uint32 `json:"output_index"`
}

// Amount represents an amount with sat and msat values
type Amount struct {
	Sat  string `json:"sat"`
	Msat string `json:"msat"`
}

// ChannelBalance represents channel balance information
type ChannelBalance struct {
	Balance            string `json:"balance"`
	PendingOpenBalance string `json:"pending_open_balance"`
	LocalBalance       Amount `json:"local_balance"`
	RemoteBalance      Amount `json:"remote_balance"`
	UnsettledLocal     Amount `json:"unsettled_local_balance"`
	UnsettledRemote    Amount `json:"unsettled_remote_balance"`
	PendingOpenLocal   Amount `json:"pending_open_local_balance"`
	PendingOpenRemote  Amount `json:"pending_open_remote_balance"`
}

// WalletBalance represents wallet balance information
type WalletBalance struct {
	TotalBalance              string `json:"total_balance"`
	ConfirmedBalance          string `json:"confirmed_balance"`
	UnconfirmedBalance        string `json:"unconfirmed_balance"`
	LockedBalance             string `json:"locked_balance"`
	ReservedBalanceAnchorChan string `json:"reserved_balance_anchor_chan"`
}

// Client represents an LND client
type Client struct {
	// This is a simple wrapper around lncli commands
	// In a production setup, you might use gRPC instead
}

// OnchainTransaction represents an on-chain transaction from LND
type OnchainTransaction struct {
	TxHash        string   `json:"tx_hash"`
	Amount        string   `json:"amount"`
	NumConfirms   int64    `json:"num_confirmations"`
	BlockHash     string   `json:"block_hash"`
	BlockHeight   int64    `json:"block_height"`
	TimeStamp     string   `json:"time_stamp"`
	TotalFees     string   `json:"total_fees"`
	DestAddresses []string `json:"dest_addresses"`
	Label         string   `json:"label"`
}

// OnchainTransactionResponse represents the response from listontransactions
type OnchainTransactionResponse struct {
	Transactions []OnchainTransaction `json:"transactions"`
}

// Invoice represents a Lightning invoice
type Invoice struct {
	RHash          string `json:"r_hash"`
	PaymentRequest string `json:"payment_request"`
	AddIndex       string `json:"add_index"`
	PaymentAddr    string `json:"payment_addr"`
	Settled        bool   `json:"settled"`
	SettleDate     string `json:"settle_date"`
	CreationDate   string `json:"creation_date"`
	Value          string `json:"value"`
	ValueMsat      string `json:"value_msat"`
	AmtPaidSat     string `json:"amt_paid_sat"`
	AmtPaidMsat    string `json:"amt_paid_msat"`
	State          string `json:"state"`
	Memo           string `json:"memo"`
}

// InvoiceResponse represents the response from listinvoices
type InvoiceResponse struct {
	Invoices []Invoice `json:"invoices"`
}

// Payment represents a Lightning payment
type Payment struct {
	PaymentHash     string `json:"payment_hash"`
	PaymentPreimage string `json:"payment_preimage"`
	ValueSat        string `json:"value_sat"`
	ValueMsat       string `json:"value_msat"`
	CreationDate    string `json:"creation_date"`
	FeeSat          string `json:"fee_sat"`
	FeeMsat         string `json:"fee_msat"`
	PaymentRequest  string `json:"payment_request"`
	Status          string `json:"status"`
}

// PaymentResponse represents the response from listpayments
type PaymentResponse struct {
	Payments []Payment `json:"payments"`
}
