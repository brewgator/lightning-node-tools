package bitcoin

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/brewgator/lightning-node-tools/internal/db"
	"github.com/brewgator/lightning-node-tools/internal/lnd"
)

// RealtimeBalanceService provides real-time balance calculations from Bitcoin Core and LND
type RealtimeBalanceService struct {
	client           *Client
	database         *db.Database
	cache            *BalanceCache
	txScanner        *TransactionScanner
	lndClient        *lnd.Client
	lightningScanner *lnd.LightningHistoryScanner
}

// BalanceCache stores recent balance queries with TTL
type BalanceCache struct {
	entries map[string]*CacheEntry
	mutex   sync.RWMutex
	ttl     time.Duration
}

// CacheEntry represents a cached balance result
type CacheEntry struct {
	Balance   int64
	TxCount   int64
	Timestamp time.Time
	Address   string
}

// AddressBalanceResult contains real-time balance information
type AddressBalanceResult struct {
	Address     string    `json:"address"`
	Balance     int64     `json:"balance"`
	TxCount     int64     `json:"tx_count"`
	LastUpdated time.Time `json:"last_updated"`
	Source      string    `json:"source"` // "cache" or "bitcoin-core"
}

// PortfolioSnapshot represents a real-time portfolio snapshot
type PortfolioSnapshot struct {
	Timestamp          time.Time `json:"timestamp"`
	LightningLocal     int64     `json:"lightning_local"`
	LightningRemote    int64     `json:"lightning_remote"`
	OnchainConfirmed   int64     `json:"onchain_confirmed"`
	OnchainUnconfirmed int64     `json:"onchain_unconfirmed"`
	TrackedAddresses   int64     `json:"tracked_addresses"`
	ColdStorage        int64     `json:"cold_storage"`
	TotalPortfolio     int64     `json:"total_portfolio"`
	TotalLiquid        int64     `json:"total_liquid"`
}

// NewRealtimeBalanceService creates a new real-time balance service
func NewRealtimeBalanceService(client *Client, database *db.Database, lndClient *lnd.Client) *RealtimeBalanceService {
	cache := &BalanceCache{
		entries: make(map[string]*CacheEntry),
		ttl:     45 * time.Second, // 45 second cache TTL for good performance
	}

	var lightningScanner *lnd.LightningHistoryScanner
	if lndClient != nil {
		lightningScanner = lnd.NewLightningHistoryScanner(lndClient)
	}

	return &RealtimeBalanceService{
		client:           client,
		database:         database,
		cache:            cache,
		txScanner:        NewTransactionScanner(client),
		lndClient:        lndClient,
		lightningScanner: lightningScanner,
	}
}

// GetCurrentPortfolio calculates the current portfolio in real-time
func (s *RealtimeBalanceService) GetCurrentPortfolio() (*PortfolioSnapshot, error) {
	log.Println("🔄 Calculating real-time portfolio...")

	// Get tracked addresses and calculate their balances
	trackedTotal, err := s.GetTrackedAddressesBalance()
	if err != nil {
		log.Printf("⚠️  Warning: Failed to get tracked addresses balance: %v", err)
		trackedTotal = 0
	}

	// Get cold storage total from database (this remains manual entry)
	coldTotal, err := s.getColdStorageTotal()
	if err != nil {
		log.Printf("⚠️  Warning: Failed to get cold storage total: %v", err)
		coldTotal = 0
	}

	// Calculate totals
	totalLiquid := trackedTotal // Only tracked addresses are liquid in this new model
	totalPortfolio := totalLiquid + coldTotal

	snapshot := &PortfolioSnapshot{
		Timestamp:          time.Now(),
		LightningLocal:     0, // We'll fill this from LND separately
		LightningRemote:    0,
		OnchainConfirmed:   0, // These are now part of tracked addresses
		OnchainUnconfirmed: 0,
		TrackedAddresses:   trackedTotal,
		ColdStorage:        coldTotal,
		TotalPortfolio:     totalPortfolio,
		TotalLiquid:        totalLiquid,
	}

	log.Printf("✅ Real-time portfolio calculated: %d sats total (%d tracked, %d cold)",
		totalPortfolio, trackedTotal, coldTotal)

	return snapshot, nil
}

// GetTrackedAddressesBalance calculates total balance of all tracked addresses
func (s *RealtimeBalanceService) GetTrackedAddressesBalance() (int64, error) {
	// Get all active tracked addresses
	addresses, err := s.database.GetOnchainAddresses()
	if err != nil {
		return 0, fmt.Errorf("failed to get tracked addresses: %w", err)
	}

	if len(addresses) == 0 {
		return 0, nil
	}

	var totalBalance int64
	successCount := 0

	// Process addresses concurrently for better performance
	results := make(chan *AddressBalanceResult, len(addresses))
	errors := make(chan error, len(addresses))

	// Launch goroutines for each active address
	activeCount := 0
	for _, addr := range addresses {
		if !addr.Active {
			continue
		}
		activeCount++
		go func(address db.OnchainAddress) {
			result, err := s.GetAddressBalance(address.Address)
			if err != nil {
				errors <- fmt.Errorf("address %s: %w", address.Address, err)
				return
			}
			results <- result
		}(addr)
	}

	// Collect results
	for i := 0; i < activeCount; i++ {
		select {
		case result := <-results:
			totalBalance += result.Balance
			successCount++
			log.Printf("📊 %s: %d sats [%s]",
				truncateAddress(result.Address), result.Balance, result.Source)
		case err := <-errors:
			log.Printf("❌ %v", err)
		case <-time.After(10 * time.Second):
			log.Printf("⏰ Timeout waiting for address balance")
		}
	}

	log.Printf("✅ Processed %d/%d active addresses, total: %d sats",
		successCount, activeCount, totalBalance)

	return totalBalance, nil
}

// GetAddressBalance gets balance for a single address with caching
func (s *RealtimeBalanceService) GetAddressBalance(address string) (*AddressBalanceResult, error) {
	// Check cache first
	if cached := s.cache.Get(address); cached != nil {
		return &AddressBalanceResult{
			Address:     cached.Address,
			Balance:     cached.Balance,
			TxCount:     cached.TxCount,
			LastUpdated: cached.Timestamp,
			Source:      "cache",
		}, nil
	}

	// Query Bitcoin Core directly
	balance, err := s.client.GetAddressBalance(address)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance from Bitcoin Core: %w", err)
	}

	// Get UTXO count as transaction count approximation
	utxos, err := s.client.GetAddressUTXOs(address)
	if err != nil {
		log.Printf("⚠️  Failed to get UTXO count for %s: %v", address, err)
		// Continue with balance only
	}

	txCount := int64(len(utxos))
	timestamp := time.Now()

	// Cache the result
	s.cache.Set(address, &CacheEntry{
		Balance:   balance,
		TxCount:   txCount,
		Timestamp: timestamp,
		Address:   address,
	})

	return &AddressBalanceResult{
		Address:     address,
		Balance:     balance,
		TxCount:     txCount,
		LastUpdated: timestamp,
		Source:      "bitcoin-core",
	}, nil
}

// GetAddressHistory generates real-time transaction history for an address
func (s *RealtimeBalanceService) GetAddressHistory(address string, from, to time.Time) ([]AddressBalanceResult, error) {
	// Import address to ensure we have transaction data
	err := s.client.ImportAddress(address)
	if err != nil {
		log.Printf("⚠️  Import warning for %s: %v", address, err)
	}

	// Use transaction scanner to get historical balance points
	return s.txScanner.GetBalanceHistory(address, from, to)
}

// GetPortfolioHistory generates real-time portfolio history based on actual transaction dates
func (s *RealtimeBalanceService) GetPortfolioHistory(from, to time.Time) ([]PortfolioSnapshot, error) {
	log.Printf("📈 Generating Lightning + Bitcoin transaction-based portfolio history from %v to %v", from.Format("2006-01-02"), to.Format("2006-01-02"))

	// Get Lightning transaction history if available
	var lightningHistory []lnd.LightningBalancePoint
	if s.lightningScanner != nil {
		history, err := s.lightningScanner.GetLightningHistory(from, to)
		if err != nil {
			log.Printf("⚠️  Warning: Failed to get Lightning history: %v", err)
		} else {
			lightningHistory = history
			log.Printf("⚡ Found %d Lightning balance points", len(lightningHistory))
		}
	}

	// Get all addresses to analyze
	addresses, err := s.database.GetOnchainAddresses()
	if err != nil {
		return nil, err
	}

	// Collect all transaction dates across all addresses to create snapshots at meaningful points
	transactionDates := make(map[string]bool)

	for _, addr := range addresses {
		if !addr.Active {
			continue
		}

		transactions, err := s.txScanner.GetAddressTransactions(addr.Address)
		if err != nil {
			log.Printf("⚠️  Failed to get transactions for %s: %v", addr.Address, err)
			continue
		}

		// Add dates of transactions within our range
		fromUnix := from.Unix()
		toUnix := to.Unix()
		for _, tx := range transactions {
			if tx.Blocktime >= fromUnix && tx.Blocktime <= toUnix {
				txDate := time.Unix(tx.Blocktime, 0).Format("2006-01-02")
				transactionDates[txDate] = true
			}
		}
	}

	// Create snapshots for transaction dates + start and end dates
	dateSet := make(map[string]time.Time)

	// Always include start and end dates
	dateSet[from.Format("2006-01-02")] = from
	dateSet[to.Format("2006-01-02")] = to

	// Add Bitcoin transaction dates
	for dateStr := range transactionDates {
		if date, err := time.Parse("2006-01-02", dateStr); err == nil {
			dateSet[dateStr] = date
		}
	}

	// Add Lightning transaction dates
	for _, lightningPoint := range lightningHistory {
		dateStr := lightningPoint.Timestamp.Format("2006-01-02")
		dateSet[dateStr] = lightningPoint.Timestamp
	}

	// Convert to sorted slice
	var sortedDates []time.Time
	for _, date := range dateSet {
		sortedDates = append(sortedDates, date)
	}

	// Sort dates chronologically
	for i := 0; i < len(sortedDates)-1; i++ {
		for j := i + 1; j < len(sortedDates); j++ {
			if sortedDates[i].After(sortedDates[j]) {
				sortedDates[i], sortedDates[j] = sortedDates[j], sortedDates[i]
			}
		}
	}

	log.Printf("📊 Creating %d snapshots for Lightning + Bitcoin transaction dates", len(sortedDates))

	// Generate enhanced snapshots using Lightning data as primary source
	var snapshots []PortfolioSnapshot
	for _, date := range sortedDates {
		snapshot := s.getPortfolioSnapshotWithLightningData(date, lightningHistory)
		snapshots = append(snapshots, snapshot)
	}

	log.Printf("✅ Generated %d portfolio snapshots based on transaction history", len(snapshots))
	return snapshots, nil
}

// getPortfolioSnapshotForDate calculates portfolio value for a specific date using historical transactions
func (s *RealtimeBalanceService) getPortfolioSnapshotForDate(date time.Time) (*PortfolioSnapshot, error) {
	addresses, err := s.database.GetOnchainAddresses()
	if err != nil {
		return nil, err
	}

	var trackedTotal int64

	// Calculate historical balance for each address by scanning transactions up to this date
	for _, addr := range addresses {
		if !addr.Active {
			continue
		}

		balance, err := s.getHistoricalBalanceForDate(addr.Address, date)
		if err != nil {
			log.Printf("⚠️  Failed to get historical balance for %s on %v: %v", addr.Address, date.Format("2006-01-02"), err)
			continue
		}
		trackedTotal += balance
	}

	// For cold storage, we only have manual entries, so use current values
	// TODO: Could implement historical cold storage tracking if needed
	coldTotal, _ := s.getColdStorageTotal()

	return &PortfolioSnapshot{
		Timestamp:        date,
		TrackedAddresses: trackedTotal,
		ColdStorage:      coldTotal,
		TotalPortfolio:   trackedTotal + coldTotal,
		TotalLiquid:      trackedTotal,
	}, nil
}

// getHistoricalBalanceForDate calculates an address balance as of a specific date
func (s *RealtimeBalanceService) getHistoricalBalanceForDate(address string, targetDate time.Time) (int64, error) {
	// Get all transactions for this address
	transactions, err := s.txScanner.GetAddressTransactions(address)
	if err != nil {
		return 0, fmt.Errorf("failed to get transactions: %w", err)
	}

	// If no transactions exist, balance is always zero
	if len(transactions) == 0 {
		return 0, nil
	}

	// Find the first transaction date for this address
	var firstTxTime int64 = 0
	for _, tx := range transactions {
		if firstTxTime == 0 || tx.Blocktime < firstTxTime {
			firstTxTime = tx.Blocktime
		}
	}

	// If target date is before first transaction, balance was zero
	targetUnix := targetDate.Unix()
	if targetUnix < firstTxTime {
		return 0, nil
	}

	// Calculate balance by replaying transactions up to the target date
	var balance int64

	for _, tx := range transactions {
		// Skip transactions that happened after our target date
		if tx.Blocktime > targetUnix {
			continue
		}

		// Add the transaction amount to balance
		amountSats := int64(tx.Amount * 100000000)
		balance += amountSats
	}

	return balance, nil
}

// getPortfolioSnapshotWithLightningData creates a portfolio snapshot prioritizing Lightning wallet data
func (s *RealtimeBalanceService) getPortfolioSnapshotWithLightningData(date time.Time, lightningHistory []lnd.LightningBalancePoint) PortfolioSnapshot {
	// Find the Lightning balance point closest to this date
	var lightningLocal, lightningRemote, onchainConfirmed int64

	// Find the most recent Lightning balance point at or before this date
	var closestPoint *lnd.LightningBalancePoint
	for i := range lightningHistory {
		point := &lightningHistory[i]
		if point.Timestamp.Before(date) || point.Timestamp.Equal(date) {
			if closestPoint == nil || point.Timestamp.After(closestPoint.Timestamp) {
				closestPoint = point
			}
		}
	}

	if closestPoint != nil {
		lightningLocal = closestPoint.LightningLocal
		lightningRemote = closestPoint.LightningRemote
		onchainConfirmed = closestPoint.OnchainBalance
	}

	// Get tracked addresses balance for this date (secondary)
	trackedTotal := int64(0)
	addresses, err := s.database.GetOnchainAddresses()
	if err == nil {
		for _, addr := range addresses {
			if !addr.Active {
				continue
			}
			if balance, err := s.getHistoricalBalanceForDate(addr.Address, date); err == nil {
				trackedTotal += balance
			}
		}
	}

	// Get cold storage total (current values, as these are manual entries)
	coldTotal, _ := s.getColdStorageTotal()

	// Calculate totals with Lightning as primary focus
	totalLiquid := lightningLocal + onchainConfirmed + trackedTotal
	totalPortfolio := totalLiquid + lightningRemote + coldTotal

	return PortfolioSnapshot{
		Timestamp:          date,
		LightningLocal:     lightningLocal,   // Lightning channel local balance
		LightningRemote:    lightningRemote,  // Lightning channel remote balance
		OnchainConfirmed:   onchainConfirmed, // Lightning wallet on-chain balance
		OnchainUnconfirmed: 0,                // Would need to track unconfirmed separately
		TrackedAddresses:   trackedTotal,     // Additional tracked addresses
		ColdStorage:        coldTotal,        // Cold storage (manual entries)
		TotalPortfolio:     totalPortfolio,   // Everything combined
		TotalLiquid:        totalLiquid,      // Spendable (local + on-chain + tracked)
	}
}

// getColdStorageTotal gets total cold storage balance from database
func (s *RealtimeBalanceService) getColdStorageTotal() (int64, error) {
	entries, err := s.database.GetColdStorageEntriesWithWarnings()
	if err != nil {
		return 0, err
	}

	var total int64
	for _, entry := range entries {
		if balance, ok := entry["balance"].(int64); ok {
			total += balance
		}
	}

	return total, nil
}

// Cache methods

// Get retrieves a cached balance if still valid
func (c *BalanceCache) Get(address string) *CacheEntry {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.entries[address]
	if !exists {
		return nil
	}

	// Check if entry has expired
	if time.Since(entry.Timestamp) > c.ttl {
		return nil
	}

	return entry
}

// Set stores a balance in cache
func (c *BalanceCache) Set(address string, entry *CacheEntry) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries[address] = entry

	// Clean up expired entries periodically
	go c.cleanupExpired()
}

// cleanupExpired removes expired cache entries
func (c *BalanceCache) cleanupExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for addr, entry := range c.entries {
		if now.Sub(entry.Timestamp) > c.ttl {
			delete(c.entries, addr)
		}
	}
}

// Helper function to truncate addresses for logging
func truncateAddress(address string) string {
	if len(address) <= 16 {
		return address
	}
	return fmt.Sprintf("%s...%s", address[:8], address[len(address)-8:])
}
