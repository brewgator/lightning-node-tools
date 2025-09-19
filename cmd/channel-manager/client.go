package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/luiscarbonell/lightning-node-tools/pkg/lnd"
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

// setChannelFees updates the fee policy for a specific channel
func setChannelFees(channelID, baseFee, ppm string) error {
	// First get the channel info to find the channel point
	channels, err := getChannels()
	if err != nil {
		return fmt.Errorf("failed to get channels: %v", err)
	}

	var channelPoint string
	for _, channel := range channels {
		if channel.ChanID == channelID {
			// Get channel point from fee report which has the channel point info
			feeReport, err := getFeeReport()
			if err != nil {
				return fmt.Errorf("failed to get fee report: %v", err)
			}
			
			for _, feeInfo := range feeReport.ChannelFees {
				if feeInfo.ChanID == channelID {
					channelPoint = feeInfo.ChannelPoint
					break
				}
			}
			break
		}
	}

	if channelPoint == "" {
		return fmt.Errorf("channel not found or no channel point available for channel ID: %s", channelID)
	}

	args := []string{"updatechanpolicy"}
	
	if baseFee != "" {
		args = append(args, "--base_fee_msat", baseFee)
	}
	
	// Convert PPM to fee rate (PPM = parts per million, so 1 ppm = 0.000001)
	ppmInt, _ := strconv.ParseInt(ppm, 10, 64)
	feeRate := float64(ppmInt) / 1000000.0
	args = append(args, "--fee_rate", fmt.Sprintf("%.6f", feeRate))
	args = append(args, "--time_lock_delta", "40")  // Standard time lock delta
	args = append(args, "--chan_point", channelPoint)

	_, err = lnd.RunLNCLI(args...)
	return err
}