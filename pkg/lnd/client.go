package lnd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// RunLNCLI executes lncli commands and returns the output
func RunLNCLI(args ...string) ([]byte, error) {
	cmd := exec.Command("lncli", args...)
	output, err := cmd.Output()
	if err != nil {
		// If there's an error, try to get stderr for more details
		if exitError, ok := err.(*exec.ExitError); ok {
			// Include stderr in the error message
			return nil, fmt.Errorf("lncli command failed: %v, stderr: %s", err, string(exitError.Stderr))
		}
		return nil, fmt.Errorf("lncli command failed: %v", err)
	}
	return output, nil
}

// NewClient creates a new LND client
func NewClient() (*Client, error) {
	// Test LND connectivity
	_, err := RunLNCLI("getinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LND: %w", err)
	}
	return &Client{}, nil
}

// GetChannels retrieves all channels from LND
func GetChannels() ([]Channel, error) {
	output, err := RunLNCLI("listchannels")
	if err != nil {
		return nil, err
	}

	var response ChannelResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}

	return response.Channels, nil
}

// ParsedChannelBalance represents parsed channel balances as int64
type ParsedChannelBalance struct {
	LocalBalance  int64
	RemoteBalance int64
}

// ParsedWalletBalance represents parsed wallet balance as int64
type ParsedWalletBalance struct {
	ConfirmedBalance   int64
	UnconfirmedBalance int64
}

// parseBalanceString parses a balance string value to int64
func parseBalanceString(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// GetChannelBalances retrieves the total channel balances
func (c *Client) GetChannelBalances() (*ParsedChannelBalance, error) {
	output, err := RunLNCLI("channelbalance")
	if err != nil {
		return nil, err
	}

	var balance ChannelBalance
	if err := json.Unmarshal(output, &balance); err != nil {
		return nil, err
	}

	// Parse string values to int64
	localBalance, err := parseBalanceString(balance.LocalBalance.Sat)
	if err != nil {
		return nil, fmt.Errorf("failed to parse local balance: %w", err)
	}

	remoteBalance, err := parseBalanceString(balance.RemoteBalance.Sat)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote balance: %w", err)
	}

	return &ParsedChannelBalance{
		LocalBalance:  localBalance,
		RemoteBalance: remoteBalance,
	}, nil
}

// GetWalletBalance retrieves the wallet balance
func (c *Client) GetWalletBalance() (*ParsedWalletBalance, error) {
	output, err := RunLNCLI("walletbalance")
	if err != nil {
		return nil, err
	}

	var balance WalletBalance
	if err := json.Unmarshal(output, &balance); err != nil {
		return nil, err
	}

	// Parse string values to int64
	confirmedBalance, err := parseBalanceString(balance.ConfirmedBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to parse confirmed balance: %w", err)
	}

	unconfirmedBalance, err := parseBalanceString(balance.UnconfirmedBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to parse unconfirmed balance: %w", err)
	}

	return &ParsedWalletBalance{
		ConfirmedBalance:   confirmedBalance,
		UnconfirmedBalance: unconfirmedBalance,
	}, nil
}

// GetFeeReport retrieves the fee report from LND
func GetFeeReport() (*FeeReportResponse, error) {
	output, err := RunLNCLI("feereport")
	if err != nil {
		return nil, err
	}

	var response FeeReportResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetNodeAlias retrieves the alias for a given pubkey
func GetNodeAlias(pubkey string) string {
	output, err := RunLNCLI("getnodeinfo", pubkey)
	if err != nil {
		// Return truncated pubkey if we can't get alias
		if len(pubkey) > 12 {
			return pubkey[:12] + "..."
		}
		return pubkey
	}

	var response NodeResponse
	if err := json.Unmarshal(output, &response); err != nil {
		// Return truncated pubkey if parsing fails
		if len(pubkey) > 12 {
			return pubkey[:12] + "..."
		}
		return pubkey
	}

	if response.Node.Alias == "" {
		// Return truncated pubkey if no alias
		if len(pubkey) > 12 {
			return pubkey[:12] + "..."
		}
		return pubkey
	}

	return response.Node.Alias
}

// GetNodePubkey retrieves our node's public key
func GetNodePubkey() (string, error) {
	output, err := RunLNCLI("getinfo")
	if err != nil {
		return "", err
	}

	var response struct {
		IdentityPubkey string `json:"identity_pubkey"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return "", err
	}

	return response.IdentityPubkey, nil
}

// GetChannelInfo retrieves detailed channel information
func GetChannelInfo(chanID string) (*ChannelEdge, error) {
	output, err := RunLNCLI("getchaninfo", chanID)
	if err != nil {
		return nil, err
	}

	var response ChannelEdge
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// ConnectPeer connects to a Lightning Network peer
func ConnectPeer(peerAddress string) error {
	_, err := RunLNCLI("connect", peerAddress)
	return err
}

// ListPeers retrieves all connected peers
func ListPeers() ([]Peer, error) {
	output, err := RunLNCLI("listpeers")
	if err != nil {
		return nil, err
	}

	var response PeerResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}

	return response.Peers, nil
}

// OpenChannel opens a channel to a peer
func OpenChannel(peerPubkey string, localAmt int64, satPerVbyte int64) (*OpenChannelResponse, error) {
	args := []string{
		"openchannel",
		"--node_key", peerPubkey,
		"--local_amt", fmt.Sprintf("%d", localAmt),
		"--sat_per_vbyte", fmt.Sprintf("%d", satPerVbyte),
	}

	output, err := RunLNCLI(args...)
	if err != nil {
		return nil, err
	}

	var response OpenChannelResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

