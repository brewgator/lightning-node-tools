package lnd

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

// LightningHistoryScanner scans Lightning and on-chain transactions for historical data
type LightningHistoryScanner struct {
	client *Client
}

// LightningBalancePoint represents a balance at a specific point in time
type LightningBalancePoint struct {
	Timestamp         time.Time `json:"timestamp"`
	OnchainBalance    int64     `json:"onchain_balance"`    // Confirmed on-chain balance
	LightningLocal    int64     `json:"lightning_local"`    // Local Lightning balance
	LightningRemote   int64     `json:"lightning_remote"`   // Remote Lightning balance
	TransactionType   string    `json:"transaction_type"`   // "onchain", "lightning_in", "lightning_out"
	TransactionAmount int64     `json:"transaction_amount"` // Amount of this transaction
}

// NewLightningHistoryScanner creates a new Lightning history scanner
func NewLightningHistoryScanner(client *Client) *LightningHistoryScanner {
	return &LightningHistoryScanner{
		client: client,
	}
}

// GetLightningHistory generates historical Lightning + on-chain balance progression
func (s *LightningHistoryScanner) GetLightningHistory(from, to time.Time) ([]LightningBalancePoint, error) {
	log.Printf("⚡ Scanning Lightning transaction history from %v to %v",
		from.Format("2006-01-02"), to.Format("2006-01-02"))

	// Get current balances as our endpoint
	currentOnchain := int64(0)
	currentLocal := int64(0)
	currentRemote := int64(0)

	// Get current wallet balance
	if walletBalance, err := s.client.GetWalletBalance(); err == nil {
		currentOnchain = walletBalance.ConfirmedBalance
	} else {
		log.Printf("⚠️  Warning: Could not get wallet balance: %v", err)
	}

	// Get current channel balances
	if channelBalance, err := s.client.GetChannelBalances(); err == nil {
		currentLocal = channelBalance.LocalBalance
		currentRemote = channelBalance.RemoteBalance
	} else {
		log.Printf("⚠️  Warning: Could not get channel balance: %v", err)
	}

	// Collect all transactions within date range
	var allEvents []transactionEvent
	fromUnix := from.Unix()
	toUnix := to.Unix()

	// Get on-chain transactions
	onchainTxs, err := s.client.GetTransactions()
	if err != nil {
		log.Printf("⚠️  Warning: Could not get on-chain transactions: %v", err)
	} else {
		log.Printf("📊 Found %d total on-chain transactions", len(onchainTxs))
		for _, tx := range onchainTxs {
			if timestamp, err := parseTimestamp(tx.TimeStamp); err == nil {
				if timestamp >= fromUnix && timestamp <= toUnix {
					amount, _ := strconv.ParseInt(tx.Amount, 10, 64)
					allEvents = append(allEvents, transactionEvent{
						timestamp: timestamp,
						txType:    "onchain",
						amount:    amount,
						txHash:    tx.TxHash,
					})
					log.Printf("📈 On-chain tx: %s, amount: %d, date: %s",
						tx.TxHash[:8], amount, time.Unix(timestamp, 0).Format("2006-01-02"))
				}
			} else {
				log.Printf("⚠️  Could not parse timestamp for tx %s: %s", tx.TxHash, tx.TimeStamp)
			}
		}
	}

	// Get Lightning invoices (received)
	invoices, err := s.client.GetInvoices()
	if err != nil {
		log.Printf("⚠️  Warning: Could not get invoices: %v", err)
	} else {
		log.Printf("📊 Found %d total invoices", len(invoices))
		settledCount := 0
		for _, invoice := range invoices {
			if invoice.Settled && invoice.State == "SETTLED" {
				settledCount++
				if timestamp, err := parseTimestamp(invoice.SettleDate); err == nil {
					if timestamp >= fromUnix && timestamp <= toUnix {
						amount, _ := strconv.ParseInt(invoice.AmtPaidSat, 10, 64)
						allEvents = append(allEvents, transactionEvent{
							timestamp: timestamp,
							txType:    "lightning_in",
							amount:    amount,
							txHash:    invoice.RHash,
						})
						log.Printf("⚡ Lightning receive: %d sats, date: %s",
							amount, time.Unix(timestamp, 0).Format("2006-01-02"))
					}
				} else {
					log.Printf("⚠️  Could not parse settle date for invoice: %s", invoice.SettleDate)
				}
			}
		}
		log.Printf("📊 %d settled invoices out of %d total", settledCount, len(invoices))
	}

	// Get Lightning payments (sent)
	payments, err := s.client.GetPayments()
	if err != nil {
		log.Printf("⚠️  Warning: Could not get payments: %v", err)
	} else {
		log.Printf("📊 Found %d total payments", len(payments))
		succeededCount := 0
		for _, payment := range payments {
			if payment.Status == "SUCCEEDED" {
				succeededCount++
				if timestamp, err := parseTimestamp(payment.CreationDate); err == nil {
					if timestamp >= fromUnix && timestamp <= toUnix {
						amount, _ := strconv.ParseInt(payment.ValueSat, 10, 64)
						allEvents = append(allEvents, transactionEvent{
							timestamp: timestamp,
							txType:    "lightning_out",
							amount:    -amount, // Negative for outgoing
							txHash:    payment.PaymentHash,
						})
						log.Printf("⚡ Lightning send: -%d sats, date: %s",
							amount, time.Unix(timestamp, 0).Format("2006-01-02"))
					}
				} else {
					log.Printf("⚠️  Could not parse creation date for payment: %s", payment.CreationDate)
				}
			}
		}
		log.Printf("📊 %d succeeded payments out of %d total", succeededCount, len(payments))
	}

	// Sort events by timestamp (oldest first)
	for i := 0; i < len(allEvents)-1; i++ {
		for j := i + 1; j < len(allEvents); j++ {
			if allEvents[i].timestamp > allEvents[j].timestamp {
				allEvents[i], allEvents[j] = allEvents[j], allEvents[i]
			}
		}
	}

	log.Printf("📊 Found %d Lightning transactions in date range", len(allEvents))

	// Build balance progression by working backwards from current balance
	var balancePoints []LightningBalancePoint

	// Start with current balances
	onchainBalance := currentOnchain
	lightningLocal := currentLocal
	lightningRemote := currentRemote

	// Add current state
	balancePoints = append(balancePoints, LightningBalancePoint{
		Timestamp:       to,
		OnchainBalance:  onchainBalance,
		LightningLocal:  lightningLocal,
		LightningRemote: lightningRemote,
		TransactionType: "current",
	})

	// Work backwards through transactions
	for i := len(allEvents) - 1; i >= 0; i-- {
		event := allEvents[i]
		eventTime := time.Unix(event.timestamp, 0)

		// Apply reverse transaction to get balance before this transaction
		switch event.txType {
		case "onchain":
			onchainBalance -= event.amount // Reverse on-chain transaction
		case "lightning_in":
			lightningLocal -= event.amount // Reverse incoming Lightning
		case "lightning_out":
			lightningLocal -= event.amount // Reverse outgoing Lightning (amount is already negative)
		}

		point := LightningBalancePoint{
			Timestamp:         eventTime,
			OnchainBalance:    onchainBalance,
			LightningLocal:    lightningLocal,
			LightningRemote:   lightningRemote,
			TransactionType:   event.txType,
			TransactionAmount: event.amount,
		}
		balancePoints = append([]LightningBalancePoint{point}, balancePoints...)
		log.Printf("📊 Added balance point: %s, on-chain: %d, local: %d, remote: %d",
			eventTime.Format("2006-01-02"), onchainBalance, lightningLocal, lightningRemote)
	}

	// If we have no transactions, create a simple progression using current balance
	if len(allEvents) == 0 {
		log.Println("📊 No Lightning transactions found, creating simple balance progression")

		// Create balance points showing current balance maintained over time
		balancePoints = []LightningBalancePoint{
			{
				Timestamp:       from,
				OnchainBalance:  currentOnchain,
				LightningLocal:  currentLocal,
				LightningRemote: currentRemote,
				TransactionType: "start",
			},
			{
				Timestamp:       to,
				OnchainBalance:  currentOnchain,
				LightningLocal:  currentLocal,
				LightningRemote: currentRemote,
				TransactionType: "current",
			},
		}
	} else {
		// Add starting point if no transactions at start date
		if allEvents[0].timestamp > fromUnix {
			balancePoints = append([]LightningBalancePoint{{
				Timestamp:       from,
				OnchainBalance:  onchainBalance,
				LightningLocal:  lightningLocal,
				LightningRemote: lightningRemote,
				TransactionType: "start",
			}}, balancePoints...)
		}
	}

	log.Printf("✅ Generated %d Lightning balance points", len(balancePoints))
	for i, point := range balancePoints {
		log.Printf("  Point %d: %s - on-chain: %d, local: %d, remote: %d",
			i+1, point.Timestamp.Format("2006-01-02"),
			point.OnchainBalance, point.LightningLocal, point.LightningRemote)
	}
	return balancePoints, nil
}

// transactionEvent represents a transaction event for sorting
type transactionEvent struct {
	timestamp int64
	txType    string // "onchain", "lightning_in", "lightning_out"
	amount    int64
	txHash    string
}

// parseTimestamp parses various timestamp formats from LND
func parseTimestamp(timestampStr string) (int64, error) {
	// Try parsing as Unix timestamp
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		return timestamp, nil
	}

	// Try parsing as RFC3339 format
	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.Unix(), nil
	}

	// Try parsing other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.999999999Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t.Unix(), nil
		}
	}

	return 0, fmt.Errorf("could not parse timestamp: %s", timestampStr)
}
