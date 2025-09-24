package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/brewgator/lightning-node-tools/pkg/lnd"
)

// getChannels retrieves all channels from LND using shared client
func getChannels() ([]lnd.Channel, error) {
	return lnd.GetChannels()
}

// getFeeReport retrieves the fee report from LND using shared client
func getFeeReport() (*lnd.FeeReportResponse, error) {
	return lnd.GetFeeReport()
}

// getForwardingHistory retrieves forwarding history for a time range
func getForwardingHistory(startTime, endTime string) (*lnd.ForwardingHistory, error) {
	output, err := lnd.RunLNCLI("fwdinghistory", "--start_time", startTime, "--end_time", endTime)
	if err != nil {
		return nil, err
	}

	var response lnd.ForwardingHistory
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// getNodeAlias retrieves the alias for a given pubkey using shared client
func getNodeAlias(pubkey string) string {
	return lnd.GetNodeAlias(pubkey)
}

// getCurrentChannelPolicy gets the current policy for a channel including time lock delta
func getCurrentChannelPolicy(channelID string) (baseFee, feeRatePpm, timeLockDelta string, err error) {
	// Get our node's public key to determine which policy is ours
	ourPubkey, err := lnd.GetNodePubkey()
	if err != nil {
		// Fall back to defaults if we can't get our pubkey
		return "1000", "1", "40", nil
	}

	// Get detailed channel information
	channelInfo, err := lnd.GetChannelInfo(channelID)
	if err != nil {
		// Fall back to defaults if we can't get channel info
		return "1000", "1", "40", nil
	}

	// Determine which policy is ours based on our pubkey
	var ourPolicy lnd.RoutingPolicy
	if channelInfo.Node1Pub == ourPubkey {
		ourPolicy = channelInfo.Node1Policy
	} else if channelInfo.Node2Pub == ourPubkey {
		ourPolicy = channelInfo.Node2Policy
	} else {
		// Fall back to defaults if we're not found in the channel
		return "1000", "1", "40", nil
	}

	// Convert fee rate from milli-msat to PPM
	// fee_rate_milli_msat is in millionths, so we need to convert to PPM
	feeRateMilliMsat, _ := strconv.ParseInt(ourPolicy.FeeRateMilliMsat, 10, 64)
	feeRatePpmValue := feeRateMilliMsat / 1000 // Convert from millionths to PPM

	return ourPolicy.FeeBaseMsat, strconv.FormatInt(feeRatePpmValue, 10), strconv.FormatUint(uint64(ourPolicy.TimeLockDelta), 10), nil
}

// setChannelFees updates the fee policy for a specific channel
func setChannelFees(channelID, baseFee, ppm string) error {
	// Get current channel policy to preserve unspecified values
	feeReport, err := getFeeReport()
	if err != nil {
		return fmt.Errorf("failed to get fee report: %v", err)
	}

	var currentFeeInfo *lnd.ChannelFeeReport
	for _, feeInfo := range feeReport.ChannelFees {
		if feeInfo.ChanID == channelID {
			currentFeeInfo = &feeInfo
			break
		}
	}

	if currentFeeInfo == nil {
		return fmt.Errorf("channel not found for channel ID: %s", channelID)
	}

	args := []string{"updatechanpolicy"}

	// Use provided base fee or preserve current value
	if baseFee != "" {
		args = append(args, "--base_fee_msat", baseFee)
	} else {
		args = append(args, "--base_fee_msat", currentFeeInfo.BaseFeeMsat)
	}

	// Use provided PPM or preserve current value
	if ppm != "" {
		args = append(args, "--fee_rate_ppm", ppm)
	} else {
		args = append(args, "--fee_rate_ppm", currentFeeInfo.FeePerMil)
	}

	// Get current time lock delta or use default
	_, _, timeLockDelta, _ := getCurrentChannelPolicy(channelID)
	args = append(args, "--time_lock_delta", timeLockDelta)
	args = append(args, "--chan_point", currentFeeInfo.ChannelPoint)

	_, err = lnd.RunLNCLI(args...)
	return err
}
