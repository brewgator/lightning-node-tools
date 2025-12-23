package bitcoin

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/brewgator/lightning-node-tools/internal/db"
)

// RealtimeBalanceService provides real-time balance calculations from Bitcoin Core
type RealtimeBalanceService struct {
	client    *Client
	database  *db.Database
	cache     *BalanceCache
	txScanner *TransactionScanner
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
func NewRealtimeBalanceService(client *Client, database *db.Database) *RealtimeBalanceService {
	cache := &BalanceCache{
		entries: make(map[string]*CacheEntry),
		ttl:     45 * time.Second, // 45 second cache TTL for good performance
	}

	return &RealtimeBalanceService{
		client:    client,
		database:  database,
		cache:     cache,
		txScanner: NewTransactionScanner(client),
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

// GetPortfolioHistory generates real-time portfolio history
func (s *RealtimeBalanceService) GetPortfolioHistory(from, to time.Time) ([]PortfolioSnapshot, error) {
	// For now, we'll generate daily snapshots by scanning transaction history
	// This is a simplified implementation - in practice, you might want to optimize this
	log.Printf("📈 Generating portfolio history from %v to %v", from.Format("2006-01-02"), to.Format("2006-01-02"))

	var snapshots []PortfolioSnapshot

	// Generate daily snapshots
	current := from
	for current.Before(to) || current.Equal(to) {
		snapshot, err := s.getPortfolioSnapshotForDate(current)
		if err != nil {
			log.Printf("⚠️  Failed to get snapshot for %v: %v", current.Format("2006-01-02"), err)
			// Continue with empty snapshot for this date
			snapshot = &PortfolioSnapshot{
				Timestamp: current,
			}
		}
		snapshots = append(snapshots, *snapshot)
		current = current.AddDate(0, 0, 1) // Next day
	}

	log.Printf("✅ Generated %d portfolio snapshots", len(snapshots))
	return snapshots, nil
}

// getPortfolioSnapshotForDate calculates portfolio value for a specific date
func (s *RealtimeBalanceService) getPortfolioSnapshotForDate(date time.Time) (*PortfolioSnapshot, error) {
	// This is a simplified approach - we'll use current balances as approximation
	// For true historical accuracy, we'd need to scan transactions up to that date

	addresses, err := s.database.GetOnchainAddresses()
	if err != nil {
		return nil, err
	}

	var trackedTotal int64
	for _, addr := range addresses {
		if !addr.Active {
			continue
		}

		// For simplicity, use current balance
		// TODO: Implement proper historical balance calculation by scanning transactions
		result, err := s.GetAddressBalance(addr.Address)
		if err != nil {
			log.Printf("⚠️  Failed to get balance for %s: %v", addr.Address, err)
			continue
		}
		trackedTotal += result.Balance
	}

	coldTotal, _ := s.getColdStorageTotal()

	return &PortfolioSnapshot{
		Timestamp:        date,
		TrackedAddresses: trackedTotal,
		ColdStorage:      coldTotal,
		TotalPortfolio:   trackedTotal + coldTotal,
		TotalLiquid:      trackedTotal,
	}, nil
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
