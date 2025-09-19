package lnd

import (
	"encoding/json"
	"fmt"
	"os/exec"
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