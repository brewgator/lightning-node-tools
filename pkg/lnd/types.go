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

// ChannelBalance represents channel balance information
type ChannelBalance struct {
	LocalBalance  int64 `json:"local_balance,string"`
	RemoteBalance int64 `json:"remote_balance,string"`
}

// WalletBalance represents wallet balance information
type WalletBalance struct {
	ConfirmedBalance   int64 `json:"confirmed_balance,string"`
	UnconfirmedBalance int64 `json:"unconfirmed_balance,string"`
	TotalBalance       int64 `json:"total_balance,string"`
}

// Client represents an LND client
type Client struct {
	// This is a simple wrapper around lncli commands
	// In a production setup, you might use gRPC instead
}