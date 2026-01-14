package bitcoin

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"
)

// TransactionScanner scans Bitcoin Core for transaction history
type TransactionScanner struct {
	client *Client
}

// TransactionSummary represents aggregated transaction data
type TransactionSummary struct {
	Date             time.Time `json:"date"`
	TotalReceived    int64     `json:"total_received"` // Satoshis
	TotalSent        int64     `json:"total_sent"`     // Satoshis
	NetChange        int64     `json:"net_change"`     // Satoshis
	TransactionCount int64     `json:"transaction_count"`
	EndingBalance    int64     `json:"ending_balance"` // Balance after this day's transactions
}

// NewTransactionScanner creates a new transaction scanner
func NewTransactionScanner(client *Client) *TransactionScanner {
	return &TransactionScanner{
		client: client,
	}
}

// GetBalanceHistory scans transaction history and generates balance snapshots
func (ts *TransactionScanner) GetBalanceHistory(address string, from, to time.Time) ([]AddressBalanceResult, error) {
	log.Printf("📈 Scanning transaction history for %s from %v to %v",
		truncateAddress(address), from.Format("2006-01-02"), to.Format("2006-01-02"))

	// Ensure address is imported for transaction scanning
	err := ts.client.ImportAddress(address)
	if err != nil {
		log.Printf("⚠️  Import warning for %s: %v", address, err)
	}

	// Get all transactions for this address
	transactions, err := ts.GetAddressTransactions(address)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Filter transactions within date range and sort by time
	filteredTxs := ts.filterTransactionsByDate(transactions, from, to)
	sort.Slice(filteredTxs, func(i, j int) bool {
		return filteredTxs[i].Time < filteredTxs[j].Time
	})

	log.Printf("📊 Found %d transactions for %s in date range", len(filteredTxs), truncateAddress(address))

	// Generate daily balance snapshots
	snapshots := ts.generateDailySnapshots(filteredTxs, address, from, to)

	return snapshots, nil
}

// GetAddressTransactions gets all transactions for an address from Bitcoin Core
func (ts *TransactionScanner) GetAddressTransactions(address string) ([]AddressTransaction, error) {
	// Use listtransactions to get all wallet transactions
	// Note: This requires the address to be imported as watch-only
	output, err := RunBitcoinCLI("listtransactions", "*", "10000", "0", "true")
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	var allTxs []AddressTransaction
	if err := json.Unmarshal(output, &allTxs); err != nil {
		return nil, fmt.Errorf("failed to parse transactions: %w", err)
	}

	// Filter for this specific address
	var addressTxs []AddressTransaction
	for _, tx := range allTxs {
		if tx.Address == address {
			addressTxs = append(addressTxs, tx)
		}
	}

	log.Printf("🔍 Found %d transactions for address %s", len(addressTxs), truncateAddress(address))
	return addressTxs, nil
}

// GetTransactionSummary generates daily transaction summaries for an address
func (ts *TransactionScanner) GetTransactionSummary(address string, from, to time.Time) ([]TransactionSummary, error) {
	transactions, err := ts.GetAddressTransactions(address)
	if err != nil {
		return nil, err
	}

	filteredTxs := ts.filterTransactionsByDate(transactions, from, to)

	// Group transactions by date
	dailyTxs := make(map[string][]AddressTransaction)
	for _, tx := range filteredTxs {
		date := time.Unix(tx.Blocktime, 0).Format("2006-01-02")
		dailyTxs[date] = append(dailyTxs[date], tx)
	}

	// Generate summaries
	var summaries []TransactionSummary
	currentBalance := int64(0) // This would need to be calculated from initial state

	// Process each day in chronological order
	current := from
	for current.Before(to) || current.Equal(to) {
		dateKey := current.Format("2006-01-02")
		dayTxs := dailyTxs[dateKey]

		var totalReceived, totalSent int64
		for _, tx := range dayTxs {
			amountSats := int64(tx.Amount * 100000000) // Convert BTC to satoshis
			if tx.Category == "receive" || amountSats > 0 {
				totalReceived += amountSats
				currentBalance += amountSats
			} else {
				totalSent += -amountSats     // Make positive for display
				currentBalance += amountSats // amountSats is negative for sends
			}
		}

		if len(dayTxs) > 0 || totalReceived > 0 || totalSent > 0 {
			summaries = append(summaries, TransactionSummary{
				Date:             current,
				TotalReceived:    totalReceived,
				TotalSent:        totalSent,
				NetChange:        totalReceived - totalSent,
				TransactionCount: int64(len(dayTxs)),
				EndingBalance:    currentBalance,
			})
		}

		current = current.AddDate(0, 0, 1)
	}

	return summaries, nil
}

// filterTransactionsByDate filters transactions within the specified date range
func (ts *TransactionScanner) filterTransactionsByDate(transactions []AddressTransaction, from, to time.Time) []AddressTransaction {
	var filtered []AddressTransaction

	fromUnix := from.Unix()
	toUnix := to.Unix()

	for _, tx := range transactions {
		if tx.Blocktime >= fromUnix && tx.Blocktime <= toUnix {
			filtered = append(filtered, tx)
		}
	}

	return filtered
}

// generateDailySnapshots creates daily balance snapshots from transaction data
func (ts *TransactionScanner) generateDailySnapshots(transactions []AddressTransaction, address string, from, to time.Time) []AddressBalanceResult {
	var snapshots []AddressBalanceResult

	// Get current balance as our ending point
	currentBalance := int64(0)
	if len(transactions) > 0 {
		// Try to get the actual current balance from Bitcoin Core
		balance, err := ts.client.GetAddressBalance(address)
		if err != nil {
			log.Printf("⚠️  Could not get current balance for %s: %v", address, err)
		} else {
			currentBalance = balance
		}
	}

	// Group transactions by date
	dailyTxs := make(map[string][]AddressTransaction)
	for _, tx := range transactions {
		date := time.Unix(tx.Blocktime, 0).Format("2006-01-02")
		dailyTxs[date] = append(dailyTxs[date], tx)
	}

	// Work backwards from current balance to calculate historical balances
	runningBalance := currentBalance

	// Generate snapshots for each day (work backwards for accuracy)
	current := to
	for current.After(from) || current.Equal(from) {
		dateKey := current.Format("2006-01-02")
		dayTxs := dailyTxs[dateKey]

		// Calculate net change for this day
		var netChange int64
		for _, tx := range dayTxs {
			amountSats := int64(tx.Amount * 100000000)
			netChange += amountSats
		}

		// Create snapshot for this day
		snapshot := AddressBalanceResult{
			Address:     address,
			Balance:     runningBalance,
			TxCount:     int64(len(dayTxs)),
			LastUpdated: current,
			Source:      "transaction-scan",
		}

		snapshots = append([]AddressBalanceResult{snapshot}, snapshots...) // Prepend to maintain chronological order

		// Move to previous day and adjust balance
		runningBalance -= netChange
		current = current.AddDate(0, 0, -1)
	}

	return snapshots
}

// GetAddressStatistics calculates comprehensive statistics for an address
func (ts *TransactionScanner) GetAddressStatistics(address string, from, to time.Time) (*AddressStatistics, error) {
	transactions, err := ts.GetAddressTransactions(address)
	if err != nil {
		return nil, err
	}

	filteredTxs := ts.filterTransactionsByDate(transactions, from, to)

	var totalReceived, totalSent int64
	var firstTx, lastTx *AddressTransaction

	for i, tx := range filteredTxs {
		amountSats := int64(tx.Amount * 100000000)

		if tx.Category == "receive" || amountSats > 0 {
			totalReceived += amountSats
		} else {
			totalSent += -amountSats
		}

		if firstTx == nil || tx.Blocktime < firstTx.Blocktime {
			firstTx = &filteredTxs[i]
		}
		if lastTx == nil || tx.Blocktime > lastTx.Blocktime {
			lastTx = &filteredTxs[i]
		}
	}

	// Get current balance
	currentBalance, err := ts.client.GetAddressBalance(address)
	if err != nil {
		log.Printf("⚠️  Could not get current balance: %v", err)
	}

	stats := &AddressStatistics{
		Address:          address,
		TotalReceived:    totalReceived,
		TotalSent:        totalSent,
		CurrentBalance:   currentBalance,
		TransactionCount: int64(len(filteredTxs)),
		DateRange: DateRange{
			From: from,
			To:   to,
		},
	}

	if firstTx != nil {
		stats.FirstTransaction = time.Unix(firstTx.Blocktime, 0)
	}
	if lastTx != nil {
		stats.LastTransaction = time.Unix(lastTx.Blocktime, 0)
	}

	return stats, nil
}

// AddressStatistics contains comprehensive address statistics
type AddressStatistics struct {
	Address          string    `json:"address"`
	TotalReceived    int64     `json:"total_received"`    // Total satoshis received
	TotalSent        int64     `json:"total_sent"`        // Total satoshis sent
	CurrentBalance   int64     `json:"current_balance"`   // Current balance
	TransactionCount int64     `json:"transaction_count"` // Number of transactions
	FirstTransaction time.Time `json:"first_transaction,omitempty"`
	LastTransaction  time.Time `json:"last_transaction,omitempty"`
	DateRange        DateRange `json:"date_range"`
}

// DateRange represents a time range
type DateRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}
