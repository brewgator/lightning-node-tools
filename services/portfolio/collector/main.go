package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/brewgator/lightning-node-tools/internal/db"
	"github.com/brewgator/lightning-node-tools/internal/lnd"
)

type Config struct {
	DatabasePath       string
	CollectionInterval time.Duration
	LNDClient          *lnd.Client
	OnchainCollector   *OnchainCollector
}

type Collector struct {
	config   *Config
	db       *db.Database
	mockMode bool
}

func main() {
	var (
		dbPath          = flag.String("db", "data/portfolio.db", "Path to SQLite database")
		interval        = flag.Duration("interval", 15*time.Minute, "Collection interval")
		oneshot         = flag.Bool("oneshot", false, "Run once and exit (for testing)")
		mockMode        = flag.Bool("mock", false, "Use mock data for testing without LND")
		onchainInterval = flag.Duration("onchain-interval", 30*time.Minute, "Onchain balance collection interval")
		mempoolURL      = flag.String("mempool-url", "https://mempool.space/api", "Mempool.space API base URL")
		bitcoinFirst    = flag.Bool("bitcoin-first", true, "Try Bitcoin node first, fallback to Mempool.space")
		skipOnchain     = flag.Bool("skip-onchain", false, "Skip onchain address balance collection")
	)
	flag.Parse()

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize database with mock mode support
	database, err := db.NewDatabaseWithMockMode(*dbPath, *mockMode)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	if *mockMode {
		fmt.Println("📊 Using mock database tables (data will not affect real data)")
	}

	// Initialize LND client (reusing existing client)
	var lndClient *lnd.Client

	if *mockMode {
		fmt.Println("⚠️  Running in mock mode - using test data")
		lndClient = nil // We'll handle mock data in collection
	} else {
		var err error
		lndClient, err = lnd.NewClient()
		if err != nil {
			log.Fatalf("Failed to initialize LND client: %v (try --mock for testing)", err)
		}
	}

	// Initialize onchain collector if not skipped
	var onchainCollector *OnchainCollector
	if !*skipOnchain && !*mockMode {
		onchainConfig := CollectorConfig{
			Database:         database,
			UpdateInterval:   *onchainInterval,
			MempoolBaseURL:   *mempoolURL,
			RetryLimit:       3,
			RetryDelay:       5 * time.Second,
			BitcoinNodeFirst: *bitcoinFirst,
		}
		onchainCollector = NewOnchainCollector(onchainConfig)
		log.Printf("✅ Onchain collector initialized (interval: %v)", *onchainInterval)
	} else if *skipOnchain {
		log.Printf("⏭️  Onchain address collection disabled")
	} else if *mockMode {
		log.Printf("📊 Mock mode: Using mock onchain data")
	}

	config := &Config{
		DatabasePath:       *dbPath,
		CollectionInterval: *interval,
		LNDClient:          lndClient,
		OnchainCollector:   onchainCollector,
	}

	collector := &Collector{
		config:   config,
		db:       database,
		mockMode: *mockMode,
	}

	if *oneshot {
		fmt.Println("Running data collection once...")
		if err := collector.collectData(); err != nil {
			log.Fatalf("Data collection failed: %v", err)
		}
		fmt.Println("Data collection completed successfully")
		return
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start onchain collector if available
	if config.OnchainCollector != nil {
		go config.OnchainCollector.Start()
		log.Printf("🚀 Onchain balance collector started")
	}

	// Start collection loop
	ticker := time.NewTicker(config.CollectionInterval)
	defer ticker.Stop()

	fmt.Printf("Starting data collection every %v...\n", config.CollectionInterval)

	// Collect initial data
	if err := collector.collectData(); err != nil {
		log.Printf("Initial data collection failed: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := collector.collectData(); err != nil {
				log.Printf("Data collection failed: %v", err)
			}
		case <-sigChan:
			fmt.Println("Received shutdown signal, exiting...")

			// Stop onchain collector if running
			if config.OnchainCollector != nil {
				config.OnchainCollector.Stop()
			}

			return
		}
	}
}

func (c *Collector) collectData() error {
	timestamp := time.Now()

	fmt.Printf("[%s] Collecting portfolio data...\n", timestamp.Format("2006-01-02 15:04:05"))

	// Collect Lightning data
	lightningLocal, lightningRemote, err := c.collectLightningData()
	if err != nil {
		return fmt.Errorf("failed to collect Lightning data: %w", err)
	}

	// Collect on-chain data
	onchainConfirmed, onchainUnconfirmed, err := c.collectOnchainData()
	if err != nil {
		log.Printf("Warning: Failed to collect on-chain data: %v", err)
		// Don't fail completely, just log the warning
	}

	// Collect tracked addresses data
	trackedAddresses, err := c.collectTrackedAddressesData()
	if err != nil {
		log.Printf("Warning: Failed to collect tracked addresses data: %v", err)
		trackedAddresses = 0
	}

	// TODO: Collect cold storage data (from config)
	coldStorage := int64(0)

	// Calculate totals
	totalLiquid := lightningLocal + onchainConfirmed + onchainUnconfirmed + trackedAddresses
	totalPortfolio := totalLiquid + coldStorage

	// Create snapshot
	snapshot := &db.BalanceSnapshot{
		Timestamp:          timestamp,
		LightningLocal:     lightningLocal,
		LightningRemote:    lightningRemote,
		OnchainConfirmed:   onchainConfirmed,
		OnchainUnconfirmed: onchainUnconfirmed,
		TrackedAddresses:   trackedAddresses,
		ColdStorage:        coldStorage,
		TotalPortfolio:     totalPortfolio,
		TotalLiquid:        totalLiquid,
	}

	// Insert into database
	if err := c.db.InsertBalanceSnapshot(snapshot); err != nil {
		return fmt.Errorf("failed to insert balance snapshot: %w", err)
	}

	fmt.Printf("✅ Portfolio snapshot saved: Total=%d sats (Local=%d, Remote=%d, Onchain=%d)\n",
		totalPortfolio, lightningLocal, lightningRemote, onchainConfirmed+onchainUnconfirmed)

	return nil
}

func (c *Collector) collectLightningData() (local, remote int64, err error) {
	if c.mockMode {
		// Return mock Lightning data
		return 5000000, 3000000, nil // 5M local, 3M remote
	}

	// Get channel balances using existing LND client
	if c.config.LNDClient == nil {
		return 0, 0, fmt.Errorf("LND client is nil")
	}
	balances, err := c.config.LNDClient.GetChannelBalances()
	if err != nil {
		return 0, 0, err
	}

	return balances.LocalBalance, balances.RemoteBalance, nil
}

func (c *Collector) collectOnchainData() (confirmed, unconfirmed int64, err error) {
	if c.mockMode {
		// Return mock on-chain data
		return 2000000, 100000, nil // 2M confirmed, 100K unconfirmed
	}

	// Get on-chain balances using existing LND client
	if c.config.LNDClient == nil {
		return 0, 0, fmt.Errorf("LND client is nil")
	}
	balance, err := c.config.LNDClient.GetWalletBalance()
	if err != nil {
		return 0, 0, err
	}

	return balance.ConfirmedBalance, balance.UnconfirmedBalance, nil
}

func (c *Collector) collectTrackedAddressesData() (int64, error) {
	if c.mockMode {
		// Return mock tracked addresses data
		return 1500000, nil // 1.5M sats
	}

	// Get all tracked addresses and their most recent balances
	addresses, err := c.db.GetOnchainAddresses()
	if err != nil {
		return 0, fmt.Errorf("failed to get tracked addresses: %w", err)
	}

	if len(addresses) == 0 {
		return 0, nil
	}

	var totalBalance int64

	// For each address, get the most recent balance
	for _, addr := range addresses {
		if !addr.Active {
			continue
		}

		// Get recent balance history for this address (30 days to ensure we capture the most recent balance)
		balances, err := c.db.GetAddressBalanceHistory(
			addr.Address,
			time.Now().AddDate(0, 0, -30), // Last 30 days
			time.Now(),
		)

		if err != nil {
			log.Printf("Warning: Failed to get balance history for %s: %v", addr.Address, err)
			continue
		}

		if len(balances) > 0 {
			// Use most recent balance
			totalBalance += balances[len(balances)-1].Balance
		}
		// If no balance records exist yet, address contributes 0 to total (expected for new addresses)
	}

	return totalBalance, nil
}
