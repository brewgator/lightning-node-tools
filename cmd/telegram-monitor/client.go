package main

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/brewgator/lightning-node-tools/pkg/lnd"
)

// getCurrentLightningState retrieves the current state of the Lightning node
func getCurrentLightningState() (*LightningState, error) {
	state := &LightningState{}

	// Get channel count using shared client
	channels, err := lnd.GetChannels()
	if err != nil {
		return nil, err
	}
	state.Channels = len(channels)

	// Get pending channels
	pendingChannels, err := lnd.RunLNCLI("pendingchannels")
	if err != nil {
		return nil, err
	}

	var pendingData map[string]any
	if err := json.Unmarshal(pendingChannels, &pendingData); err != nil {
		return nil, err
	}

	if pendingOpen, ok := pendingData["pending_open_channels"].([]any); ok {
		state.PendingOpen = len(pendingOpen)
	}

	// Count all types of pending closes
	pendingCloses := 0
	if pendingClosing, ok := pendingData["pending_closing_channels"].([]any); ok {
		pendingCloses += len(pendingClosing)
	}
	if pendingForceClosing, ok := pendingData["pending_force_closing_channels"].([]any); ok {
		pendingCloses += len(pendingForceClosing)
	}
	if waitingClose, ok := pendingData["waiting_close_channels"].([]any); ok {
		pendingCloses += len(waitingClose)
	}
	state.PendingClose = pendingCloses

	// Get invoice count
	invoices, err := lnd.RunLNCLI("listinvoices")
	if err != nil {
		return nil, err
	}

	var invoiceData map[string]any
	if err := json.Unmarshal(invoices, &invoiceData); err != nil {
		return nil, err
	}

	if invoiceList, ok := invoiceData["invoices"].([]any); ok {
		state.Invoices = len(invoiceList)
	}

	// Get wallet balance
	walletBalance, err := lnd.RunLNCLI("walletbalance")
	if err != nil {
		return nil, err
	}

	var walletData map[string]any
	if err := json.Unmarshal(walletBalance, &walletData); err != nil {
		return nil, err
	}

	if totalBalance, ok := walletData["total_balance"].(string); ok {
		if balance, err := strconv.ParseInt(totalBalance, 10, 64); err == nil {
			state.OnchainBalance = balance
		}
	}

	// Get channel balance
	channelBalance, err := lnd.RunLNCLI("channelbalance")
	if err != nil {
		return nil, err
	}

	var balanceData map[string]any
	if err := json.Unmarshal(channelBalance, &balanceData); err != nil {
		return nil, err
	}

	if localBal, ok := balanceData["local_balance"].(map[string]any); ok {
		if sat, ok := localBal["sat"].(string); ok {
			if balance, err := strconv.ParseInt(sat, 10, 64); err == nil {
				state.LocalBalance = balance
			}
		}
	}

	if remoteBal, ok := balanceData["remote_balance"].(map[string]any); ok {
		if sat, ok := remoteBal["sat"].(string); ok {
			if balance, err := strconv.ParseInt(sat, 10, 64); err == nil {
				state.RemoteBalance = balance
			}
		}
	}

	// Add pending balances
	if pendingLocalBal, ok := balanceData["pending_open_local_balance"].(map[string]any); ok {
		if sat, ok := pendingLocalBal["sat"].(string); ok {
			if balance, err := strconv.ParseInt(sat, 10, 64); err == nil {
				state.LocalBalance += balance
			}
		}
	}

	if pendingRemoteBal, ok := balanceData["pending_open_remote_balance"].(map[string]any); ok {
		if sat, ok := pendingRemoteBal["sat"].(string); ok {
			if balance, err := strconv.ParseInt(sat, 10, 64); err == nil {
				state.RemoteBalance += balance
			}
		}
	}

	// Calculate total (only on-chain and local Lightning balances - remote balances aren't "mine")
	state.TotalBalance = state.OnchainBalance + state.LocalBalance

	// Get recent forwarding events (last 10 minutes)
	recentTime := time.Now().Add(-10 * time.Minute).Unix()
	fwdHistory, err := lnd.RunLNCLI("fwdinghistory", "--start_time", strconv.FormatInt(recentTime, 10))
	if err != nil {
		return nil, err
	}

	var fwdData map[string]any
	if err := json.Unmarshal(fwdHistory, &fwdData); err != nil {
		return nil, err
	}

	if fwdEvents, ok := fwdData["forwarding_events"].([]any); ok {
		state.Forwards = len(fwdEvents)
		
		// Track the latest forward timestamp for routing fee detection
		var latestTimestamp int64
		for _, event := range fwdEvents {
			if eventMap, ok := event.(map[string]any); ok {
				if timestampStr, ok := eventMap["timestamp"].(string); ok {
					if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
						if timestamp > latestTimestamp {
							latestTimestamp = timestamp
						}
					}
				}
			}
		}
		// Do not update state.LastForwardTimestamp here; it should be updated after event processing.
		// The latestTimestamp can be returned or used elsewhere if needed.
	}

	return state, nil
}
