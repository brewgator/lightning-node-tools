package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brewgator/lightning-node-tools/internal/bitcoin"
	"github.com/brewgator/lightning-node-tools/internal/db"
	"github.com/brewgator/lightning-node-tools/internal/mempool"
)

// OnchainCollector handles automatic balance collection for tracked addresses
type OnchainCollector struct {
	database      *db.Database
	bitcoinClient *bitcoin.Client
	mempoolClient *mempool.Client
	interval      time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
	retryLimit    int
	retryDelay    time.Duration
	requestDelay  time.Duration
}

// CollectorConfig represents configuration for the onchain collector
type CollectorConfig struct {
	Database         *db.Database
	UpdateInterval   time.Duration
	MempoolBaseURL   string
	RetryLimit       int
	RetryDelay       time.Duration
	RequestDelay     time.Duration // Delay between address update requests
	BitcoinNodeFirst bool
}

// NewOnchainCollector creates a new onchain balance collector
func NewOnchainCollector(config CollectorConfig) *OnchainCollector {
	ctx, cancel := context.WithCancel(context.Background())

	// Set defaults
	if config.UpdateInterval == 0 {
		config.UpdateInterval = 30 * time.Minute
	}
	if config.RetryLimit == 0 {
		config.RetryLimit = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}
	if config.RequestDelay == 0 {
		config.RequestDelay = 1 * time.Second
	}

	collector := &OnchainCollector{
		database:     config.Database,
		interval:     config.UpdateInterval,
		ctx:          ctx,
		cancel:       cancel,
		retryLimit:   config.RetryLimit,
		retryDelay:   config.RetryDelay,
		requestDelay: config.RequestDelay,
	}

	// Try to initialize Bitcoin client
	if config.BitcoinNodeFirst {
		bitcoinClient, err := bitcoin.NewClient()
		if err != nil {
			log.Printf("⚠️  Bitcoin node not available: %v", err)
			log.Printf("💡 Will use Mempool.space API as primary source")
		} else {
			collector.bitcoinClient = bitcoinClient
			log.Printf("₿ Connected to Bitcoin Core node")
		}
	}

	// Initialize Mempool.space client
	collector.mempoolClient = mempool.NewClient(config.MempoolBaseURL)
	log.Printf("🌐 Mempool.space client initialized")

	return collector
}

// Start begins the periodic balance collection process
func (c *OnchainCollector) Start() {
	log.Printf("🚀 Starting onchain balance collector (interval: %v)", c.interval)

	// Run initial collection immediately
	c.collectAllBalances()

	// Start periodic collection
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.collectAllBalances()
		case <-c.ctx.Done():
			log.Println("📊 Onchain balance collector stopped")
			return
		}
	}
}

// Stop stops the balance collector
func (c *OnchainCollector) Stop() {
	log.Println("⏹️  Stopping onchain balance collector...")
	c.cancel()
}

// collectAllBalances collects balances for all tracked addresses
func (c *OnchainCollector) collectAllBalances() {
	log.Println("📊 Collecting balances for tracked addresses...")

	// Get all tracked addresses
	addresses, err := c.database.GetOnchainAddresses()
	if err != nil {
		log.Printf("❌ Failed to get tracked addresses: %v", err)
		return
	}

	if len(addresses) == 0 {
		log.Println("📭 No addresses to update")
		return
	}

	successCount := 0
	bitcoinNodeCount := 0
	mempoolCount := 0

	for _, address := range addresses {
		if !address.Active {
			log.Printf("⏭️  Skipping inactive address %s", truncateAddress(address.Address))
			continue
		}

		balance, txCount, source, err := c.collectAddressBalance(address)
		if err != nil {
			log.Printf("❌ Failed to collect balance for %s: %v",
				truncateAddress(address.Address), err)
			continue
		}

		// Store balance record
		balanceRecord := &db.AddressBalance{
			AddressID: address.ID,
			Timestamp: time.Now(),
			Balance:   balance,
			TxCount:   txCount,
		}

		err = c.database.InsertAddressBalance(balanceRecord)
		if err != nil {
			log.Printf("❌ Failed to store balance for %s: %v",
				truncateAddress(address.Address), err)
			continue
		}

		// Log balance change detection
		c.logBalanceChange(address, balance, source)

		successCount++
		switch source {
		case "bitcoin-node":
			bitcoinNodeCount++
		case "mempool":
			mempoolCount++
		}

		// Add delay between requests to be respectful (configurable via RequestDelay)
		time.Sleep(c.requestDelay)
	}

	log.Printf("✅ Successfully collected %d/%d addresses (%d Bitcoin node, %d Mempool.space)",
		successCount, len(addresses), bitcoinNodeCount, mempoolCount)
}

// collectAddressBalance collects balance for a single address with fallback logic
func (c *OnchainCollector) collectAddressBalance(address db.OnchainAddress) (int64, int64, string, error) {
	var lastErr error

	// Try Bitcoin node first if available
	if c.bitcoinClient != nil {
		balance, txCount, err := c.collectFromBitcoinNode(address.Address)
		if err == nil {
			return balance, txCount, "bitcoin-node", nil
		}

		log.Printf("⚠️  Bitcoin node failed for %s: %v, falling back to Mempool.space",
			truncateAddress(address.Address), err)
		lastErr = err
	}

	// Fallback to Mempool.space API with retry logic
	for attempt := 1; attempt <= c.retryLimit; attempt++ {
		balance, txCount, err := c.collectFromMempool(address.Address)
		if err == nil {
			return balance, txCount, "mempool", nil
		}

		lastErr = err

		if attempt < c.retryLimit {
			log.Printf("⚠️  Mempool.space attempt %d failed for %s: %v, retrying in %v",
				attempt, truncateAddress(address.Address), err, c.retryDelay)

			select {
			case <-time.After(c.retryDelay):
				continue
			case <-c.ctx.Done():
				return 0, 0, "", c.ctx.Err()
			}
		}
	}

	return 0, 0, "", fmt.Errorf("all sources failed, last error: %w", lastErr)
}

// collectFromBitcoinNode collects balance using local Bitcoin node
func (c *OnchainCollector) collectFromBitcoinNode(address string) (int64, int64, error) {
	balance, err := c.bitcoinClient.GetAddressBalance(address)
	if err != nil {
		return 0, 0, err
	}

	// Get UTXOs for transaction count
	utxos, err := c.bitcoinClient.GetAddressUTXOs(address)
	if err != nil {
		log.Printf("⚠️  Failed to get UTXOs for %s: %v", truncateAddress(address), err)
		// Continue with balance only
		return balance, 0, nil
	}

	return balance, int64(len(utxos)), nil
}

// collectFromMempool collects balance using Mempool.space API
func (c *OnchainCollector) collectFromMempool(address string) (int64, int64, error) {
	balance, utxoCount, err := c.mempoolClient.CalculateAddressBalance(address)
	if err != nil {
		return 0, 0, err
	}

	// Get address stats for transaction count
	stats, err := c.mempoolClient.GetAddressStats(address)
	if err != nil {
		log.Printf("⚠️  Failed to get stats for %s: %v", truncateAddress(address), err)
		// Continue with balance and UTXO count only
		return balance, utxoCount, nil
	}

	totalTxCount := stats.ChainStats.TxCount + stats.MempoolStats.TxCount
	return balance, totalTxCount, nil
}

// logBalanceChange logs balance changes for monitoring
func (c *OnchainCollector) logBalanceChange(address db.OnchainAddress, newBalance int64, source string) {
	// Get previous balance (query for balances before the current update)
	// Use a timestamp slightly in the past to avoid including the just-inserted record
	endTime := time.Now().Add(-2 * time.Second)
	balances, err := c.database.GetAddressBalanceHistory(
		address.Address,
		endTime.AddDate(0, 0, -1), // Last 24 hours
		endTime,
	)

	if err != nil || len(balances) == 0 {
		log.Printf("📊 New balance for %s: %d sats [%s] (no previous data)",
			truncateAddress(address.Address), newBalance, source)
		return
	}

	// Get the most recent balance from history
	previousBalance := balances[len(balances)-1].Balance

	if newBalance != previousBalance {
		change := newBalance - previousBalance
		changeStr := "📈"
		if change < 0 {
			changeStr = "📉"
		}

		log.Printf("%s Balance change for %s: %d → %d sats (Δ%+d) [%s]",
			changeStr, truncateAddress(address.Address),
			previousBalance, newBalance, change, source)
	} else {
		log.Printf("📊 Balance unchanged for %s: %d sats [%s]",
			truncateAddress(address.Address), newBalance, source)
	}
}

// CollectSingleAddress manually collects balance for a specific address
func (c *OnchainCollector) CollectSingleAddress(addressID int64) error {
	address, err := c.database.GetOnchainAddressByID(addressID)
	if err != nil {
		return err
	}
	if address == nil {
		return db.ErrNotFound
	}

	balance, txCount, source, err := c.collectAddressBalance(*address)
	if err != nil {
		return err
	}

	// Store balance record
	balanceRecord := &db.AddressBalance{
		AddressID: address.ID,
		Timestamp: time.Now(),
		Balance:   balance,
		TxCount:   txCount,
	}

	err = c.database.InsertAddressBalance(balanceRecord)
	if err != nil {
		return err
	}

	// Log the update
	c.logBalanceChange(*address, balance, source)

	return nil
}

// GetCollectorStats returns statistics about the collector
func (c *OnchainCollector) GetCollectorStats() map[string]interface{} {
	return map[string]interface{}{
		"interval_minutes":     c.interval.Minutes(),
		"bitcoin_available":    c.bitcoinClient != nil,
		"mempool_available":    c.mempoolClient != nil,
		"retry_limit":          c.retryLimit,
		"retry_delay_seconds":  c.retryDelay.Seconds(),
		"request_delay_seconds": c.requestDelay.Seconds(),
	}
}

// truncateAddress truncates Bitcoin address for logging
func truncateAddress(address string) string {
	if len(address) <= 16 {
		return address
	}
	return fmt.Sprintf("%s...%s", address[:8], address[len(address)-8:])
}
