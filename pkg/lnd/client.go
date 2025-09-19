package lnd

import (
	"encoding/json"
	"os/exec"
)

// RunLNCLI executes lncli commands and returns the output
func RunLNCLI(args ...string) ([]byte, error) {
	cmd := exec.Command("lncli", args...)
	return cmd.Output()
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