package bitcoin

import (
	"context"
	"log"
	"time"

	"github.com/brewgator/lightning-node-tools/internal/db"
)

// BalanceService handles periodic balance updates from Bitcoin Core
type BalanceService struct {
	client   *Client
	database *db.Database
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewBalanceService creates a new balance service
func NewBalanceService(client *Client, database *db.Database, interval time.Duration) *BalanceService {
	ctx, cancel := context.WithCancel(context.Background())
	return &BalanceService{
		client:   client,
		database: database,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins the periodic balance update process
func (s *BalanceService) Start() {
	log.Println("Starting Bitcoin balance service...")

	// Run initial update immediately
	s.updateAllBalances()

	// Start periodic updates
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updateAllBalances()
		case <-s.ctx.Done():
			log.Println("Bitcoin balance service stopped")
			return
		}
	}
}

// Stop stops the balance service
func (s *BalanceService) Stop() {
	log.Println("Stopping Bitcoin balance service...")
	s.cancel()
}

// updateAllBalances updates balances for all tracked addresses
func (s *BalanceService) updateAllBalances() {
	log.Println("Updating Bitcoin address balances...")

	// Get all tracked addresses
	addresses, err := s.database.GetOnchainAddresses()
	if err != nil {
		log.Printf("Failed to get tracked addresses: %v", err)
		return
	}

	if len(addresses) == 0 {
		log.Println("No addresses to update")
		return
	}

	successCount := 0
	for _, address := range addresses {
		if !address.Active {
			continue // Skip inactive addresses
		}

		balance, err := s.updateAddressBalance(address)
		if err != nil {
			log.Printf("Failed to update balance for %s: %v", address.Address, err)
			continue
		}

		log.Printf("Updated balance for %s: %d satoshis", address.Address, balance)
		successCount++
	}

	log.Printf("Successfully updated %d/%d addresses", successCount, len(addresses))
}

// updateAddressBalance updates the balance for a specific address
func (s *BalanceService) updateAddressBalance(address db.OnchainAddress) (int64, error) {
	// Get current balance from Bitcoin Core
	balance, err := s.client.GetAddressBalance(address.Address)
	if err != nil {
		return 0, err
	}

	// Get UTXOs to count transactions
	utxos, err := s.client.GetAddressUTXOs(address.Address)
	if err != nil {
		log.Printf("Warning: Failed to get UTXOs for %s: %v", address.Address, err)
		// Continue with balance update even if UTXO fetch fails
	}

	txCount := int64(len(utxos))

	// Get blockchain info for current block height
	_, err = s.client.GetBlockchainInfo()
	if err != nil {
		log.Printf("Warning: Failed to get blockchain info: %v", err)
		// Continue with balance update
	}

	// Insert new balance record
	balanceRecord := &db.AddressBalance{
		AddressID: address.ID,
		Timestamp: time.Now(),
		Balance:   balance,
		TxCount:   txCount,
	}

	err = s.database.InsertAddressBalance(balanceRecord)
	if err != nil {
		return 0, err
	}

	log.Printf("Inserted balance record for %s: %d satoshis (%d txs)",
		address.Address, balance, txCount)

	return balance, nil
}

// UpdateSingleAddress manually updates a single address balance
func (s *BalanceService) UpdateSingleAddress(addressID int64) error {
	address, err := s.database.GetOnchainAddressByID(addressID)
	if err != nil {
		return err
	}
	if address == nil {
		return db.ErrNotFound
	}

	_, err = s.updateAddressBalance(*address)
	return err
}

// GetAddressCurrentBalance gets the most recent balance for an address
func (s *BalanceService) GetAddressCurrentBalance(address string) (int64, error) {
	// Try to get from database first (most recent record)
	balances, err := s.database.GetAddressBalanceHistory(address, time.Now().AddDate(0, 0, -7), time.Now())
	if err == nil && len(balances) > 0 {
		// Return the most recent balance
		return balances[len(balances)-1].Balance, nil
	}

	// Fallback to querying Bitcoin Core directly
	return s.client.GetAddressBalance(address)
}

// ImportAndTrackAddress imports an address and starts tracking it
func (s *BalanceService) ImportAndTrackAddress(address, label string) error {
	// First validate the address
	validation, err := s.client.ValidateAddress(address)
	if err != nil {
		return err
	}
	if !validation.IsValid {
		return ErrInvalidAddress
	}

	// Import address to Bitcoin Core as watch-only
	err = s.client.ImportAddress(address)
	if err != nil {
		log.Printf("Warning: Failed to import address %s: %v", address, err)
		// Continue anyway - address might already be imported
	}

	// Add to database
	dbAddress, err := s.database.InsertOnchainAddress(address, label)
	if err != nil {
		return err
	}

	// Update balance immediately
	_, err = s.updateAddressBalance(*dbAddress)
	if err != nil {
		log.Printf("Warning: Failed to update initial balance for %s: %v", address, err)
		// Don't fail the import due to balance update failure
	}

	return nil
}
