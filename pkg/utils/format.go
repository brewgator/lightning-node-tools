package utils

import "fmt"

// FormatSats formats satoshi amounts in a human-readable way
// Uses the more precise format suitable for both tools
func FormatSats(amount int64) string {
	if amount >= 100000000 {
		// Show in BTC for amounts >= 1 BTC
		return fmt.Sprintf("%.8f BTC", float64(amount)/100000000)
	} else if amount >= 1000000 {
		// Show in millions for amounts >= 1M sats
		return fmt.Sprintf("%.2fM sats", float64(amount)/1000000)
	} else if amount >= 1000 {
		// Show in thousands for amounts >= 1K sats
		return fmt.Sprintf("%.1fK sats", float64(amount)/1000)
	}
	return fmt.Sprintf("%d sats", amount)
}

// FormatSatsCompact formats satoshi amounts in a compact way for tables
// Used by channel-manager for cleaner table display
func FormatSatsCompact(amount int64) string {
	if amount >= 100000000 {
		// Show in BTC for amounts >= 1 BTC
		return fmt.Sprintf("%.3f BTC", float64(amount)/100000000)
	} else if amount >= 1000000 {
		// Show in millions for amounts >= 1M sats
		return fmt.Sprintf("%.1fM", float64(amount)/1000000)
	} else if amount >= 1000 {
		// Show in thousands for amounts >= 1K sats
		return fmt.Sprintf("%.0fK", float64(amount)/1000)
	}
	return fmt.Sprintf("%d", amount)
}