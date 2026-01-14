package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/brewgator/lightning-node-tools/internal/db"
	"github.com/brewgator/lightning-node-tools/internal/strike"
)

type Config struct {
	DatabasePath       string
	CollectionInterval time.Duration
	StrikeClient       *strike.Client
	CurrencyFilter     string // Optional: only track specific currency (e.g., "BTC")
}

type BalanceCollector struct {
	config   *Config
	db       *db.Database
	mockMode bool
}

// loadEnv loads environment variables from .env file
func loadEnv(envPath string) error {
	file, err := os.Open(envPath)
	if err != nil {
		// .env file is optional, so don't fail if it doesn't exist
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

		// Only set if not already set in environment
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

func main() {
	// Try to load .env file from project root
	// This allows reading STRIKE_API_KEY from .env
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		// If running from bin/, go up one level to project root
		projectRoot := filepath.Dir(exeDir)
		envPath := filepath.Join(projectRoot, ".env")
		if err := loadEnv(envPath); err != nil {
			log.Printf("Warning: Failed to load .env file: %v", err)
		}
	}

	// Also try loading from current directory (useful for development)
	if err := loadEnv(".env"); err != nil {
		log.Printf("Warning: Failed to load .env from current directory: %v", err)
	}

	var (
		dbPath   = flag.String("db", "data/portfolio.db", "Path to SQLite database")
		interval = flag.Duration("interval", 15*time.Minute, "Collection interval")
		oneshot  = flag.Bool("oneshot", false, "Run once and exit (for testing)")
		mockMode = flag.Bool("mock", false, "Use mock data for testing without Strike API")
		apiKey   = flag.String("api-key", "", "Strike API key (or set STRIKE_API_KEY env var or in .env file)")
		currency = flag.String("currency", "", "Optional: only track specific currency (BTC, USD, etc.)")
	)
	flag.Parse()

	// Priority order: CLI flag > Environment variable > .env file
	// Get API key from environment if not provided via flag
	if *apiKey == "" {
		*apiKey = os.Getenv("STRIKE_API_KEY")
	}

	// In mock mode, we don't need an API key
	if !*mockMode && *apiKey == "" {
		log.Fatal("❌ Strike API key required! Use --api-key flag, STRIKE_API_KEY environment variable, or add to .env file")
	}

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

	// Initialize Strike client
	var strikeClient *strike.Client
	if *mockMode {
		fmt.Println("⚠️  Running in mock mode - using test data")
		strikeClient = nil
	} else {
		strikeClient = strike.NewClient(*apiKey)
		fmt.Println("⚡ Connected to Strike API")
	}

	config := &Config{
		DatabasePath:       *dbPath,
		CollectionInterval: *interval,
		StrikeClient:       strikeClient,
		CurrencyFilter:     *currency,
	}

	collector := &BalanceCollector{
		config:   config,
		db:       database,
		mockMode: *mockMode,
	}

	if *oneshot {
		fmt.Println("Running Strike balance collection once...")
		if err := collector.collectBalances(); err != nil {
			log.Fatalf("Strike balance collection failed: %v", err)
		}
		fmt.Println("Strike balance collection completed successfully")
		return
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start collection loop
	ticker := time.NewTicker(config.CollectionInterval)
	defer ticker.Stop()

	fmt.Printf("Starting Strike balance collection every %v...\n", config.CollectionInterval)

	// Collect initial data
	if err := collector.collectBalances(); err != nil {
		log.Printf("Initial Strike balance collection failed: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := collector.collectBalances(); err != nil {
				log.Printf("Strike balance collection failed: %v", err)
			}
		case <-sigChan:
			fmt.Println("Received shutdown signal, exiting...")
			return
		}
	}
}

func (c *BalanceCollector) collectBalances() error {
	currentTime := time.Now()
	fmt.Printf("[%s] Collecting Strike balances...\n",
		currentTime.Format("2006-01-02 15:04:05"))

	if c.mockMode {
		return c.collectMockBalances()
	}

	if c.config.StrikeClient == nil {
		return fmt.Errorf("Strike client is nil")
	}

	// Get balances from Strike API
	balances, err := c.config.StrikeClient.GetAccountBalance()
	if err != nil {
		return fmt.Errorf("failed to get Strike balances: %w", err)
	}

	if len(balances) == 0 {
		fmt.Printf("⚠️  No balances returned from Strike API\n")
		return nil
	}

	// Insert each currency balance into database
	var insertedCount int
	for _, balance := range balances {
		// Apply currency filter if specified
		if c.config.CurrencyFilter != "" && balance.Currency != c.config.CurrencyFilter {
			continue
		}

		snapshot := &db.StrikeBalanceSnapshot{
			Timestamp: balance.Timestamp,
			Currency:  balance.Currency,
			Available: balance.Available,
			Total:     balance.Total,
			Pending:   balance.Pending,
			Reserved:  balance.Reserved,
		}

		if err := c.db.InsertStrikeBalanceSnapshot(snapshot); err != nil {
			log.Printf("Warning: failed to insert Strike balance for %s: %v", balance.Currency, err)
			continue
		}

		fmt.Printf("  💰 %s: Available=%d, Total=%d\n",
			balance.Currency,
			balance.Available,
			balance.Total)
		insertedCount++
	}

	fmt.Printf("✅ Inserted %d Strike balance snapshots\n", insertedCount)
	return nil
}

func (c *BalanceCollector) collectMockBalances() error {
	// Create mock Strike balance data for testing
	now := time.Now()

	mockBalances := []*db.StrikeBalanceSnapshot{
		{
			Timestamp: now,
			Currency:  "BTC",
			Available: 5000000, // 0.05 BTC in sats
			Total:     5100000, // 0.051 BTC in sats
			Pending:   50000,   // 0.0005 BTC in sats
			Reserved:  50000,   // 0.0005 BTC in sats
		},
		{
			Timestamp: now,
			Currency:  "USD",
			Available: 100000, // $1000.00 in cents
			Total:     110000, // $1100.00 in cents
			Pending:   5000,   // $50.00 in cents
			Reserved:  5000,   // $50.00 in cents
		},
	}

	var insertedCount int
	for _, balance := range mockBalances {
		// Apply currency filter if specified
		if c.config.CurrencyFilter != "" && balance.Currency != c.config.CurrencyFilter {
			continue
		}

		if err := c.db.InsertStrikeBalanceSnapshot(balance); err != nil {
			log.Printf("Warning: failed to insert mock Strike balance for %s: %v", balance.Currency, err)
			continue
		}

		fmt.Printf("  💰 %s: Available=%d, Total=%d (mock)\n",
			balance.Currency,
			balance.Available,
			balance.Total)
		insertedCount++
	}

	fmt.Printf("✅ Inserted %d mock Strike balance snapshots\n", insertedCount)
	return nil
}
