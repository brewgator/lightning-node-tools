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

	"lightning-node-tools/pkg/db"
	"lightning-node-tools/pkg/lnd"
)

type Config struct {
	DatabasePath     string
	CollectionInterval time.Duration
	LNDClient        *lnd.Client
}

type Collector struct {
	config *Config
	db     *db.Database
}

func main() {
	var (
		configPath = flag.String("config", "configs/dashboard.yaml", "Path to configuration file")
		dbPath     = flag.String("db", "data/portfolio.db", "Path to SQLite database")
		interval   = flag.Duration("interval", 15*time.Minute, "Collection interval")
		oneshot    = flag.Bool("oneshot", false, "Run once and exit (for testing)")
	)
	flag.Parse()

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize database
	database, err := db.NewDatabase(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize LND client (reusing existing client)
	lndClient, err := lnd.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize LND client: %v", err)
	}

	config := &Config{
		DatabasePath:       *dbPath,
		CollectionInterval: *interval,
		LNDClient:          lndClient,
	}

	collector := &Collector{
		config: config,
		db:     database,
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

	// TODO: Collect tracked addresses data (Mempool.space API)
	trackedAddresses := int64(0)

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
	// Get channel balances using existing LND client
	balances, err := c.config.LNDClient.GetChannelBalances()
	if err != nil {
		return 0, 0, err
	}

	return balances.LocalBalance, balances.RemoteBalance, nil
}

func (c *Collector) collectOnchainData() (confirmed, unconfirmed int64, err error) {
	// Get on-chain balances using existing LND client
	balance, err := c.config.LNDClient.GetWalletBalance()
	if err != nil {
		return 0, 0, err
	}

	return balance.ConfirmedBalance, balance.UnconfirmedBalance, nil
}