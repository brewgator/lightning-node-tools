package main

import (
	"github.com/brewgator/lightning-node-tools/pkg/utils"
)

// formatSats formats satoshi amounts using shared utility (compact format for tables)
func formatSats(amount int64) string {
	return utils.FormatSatsCompact(amount)
}

// getChannelStatus returns the status string for a channel
func getChannelStatus(channel Channel) string {
	status := ""
	if channel.Private {
		status += "Private"
	} else {
		status += "Public"
	}

	if !channel.Active {
		status += " (Inactive)"
	}

	return status
}
