package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/brewgator/lightning-node-tools/internal/bitcoin"
	"github.com/brewgator/lightning-node-tools/internal/db"
	"github.com/brewgator/lightning-node-tools/internal/multisig"
)

func main() {
	var (
		dbPath   = flag.String("db", "data/portfolio.db", "Path to SQLite database")
		dryRun   = flag.Bool("dry-run", false, "Show what would be done without making changes")
		walletID = flag.Int64("wallet", 0, "Specific wallet ID to backfill (0 = all wallets)")
	)
	flag.Parse()

	fmt.Println("🔄 Lightning Node Tools - Historical Balance Backfill")
	fmt.Println("   This tool will fetch complete transaction history and backfill")
	fmt.Println("   balance snapshots to show accurate historical data like Sparrow.")
	fmt.Println()

	if *dryRun {
		fmt.Println("⚠️  DRY RUN MODE - No changes will be made")
		fmt.Println()
	}

	// Initialize database
	database, err := db.NewDatabase(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize Bitcoin client
	bitcoinClient, err := bitcoin.NewClient()
	if err != nil {
		log.Fatalf("Failed to connect to Bitcoin node: %v", err)
	}

	// Initialize multisig service
	multisigService := multisig.NewMultisigService(database)

	backfiller := &HistoricalBackfiller{
		db:              database,
		bitcoinClient:   bitcoinClient,
		multisigService: multisigService,
		dryRun:          *dryRun,
	}

	if *walletID > 0 {
		err = backfiller.BackfillWallet(*walletID)
	} else {
		err = backfiller.BackfillAllWallets()
	}

	if err != nil {
		log.Fatalf("Backfill failed: %v", err)
	}

	fmt.Println("✅ Historical backfill completed successfully!")
}

type HistoricalBackfiller struct {
	db              *db.Database
	bitcoinClient   *bitcoin.Client
	multisigService *multisig.MultisigService
	dryRun          bool
}

// BackfillAllWallets backfills historical data for all multisig wallets
func (hb *HistoricalBackfiller) BackfillAllWallets() error {
	wallets, err := hb.multisigService.GetWallets()
	if err != nil {
		return fmt.Errorf("failed to get wallets: %w", err)
	}

	fmt.Printf("📊 Found %d multisig wallets to backfill\n\n", len(wallets))

	for i, wallet := range wallets {
		fmt.Printf("[%d/%d] Processing %s...\n", i+1, len(wallets), wallet.Name)
		err = hb.BackfillWallet(wallet.ID)
		if err != nil {
			log.Printf("Failed to backfill wallet %s: %v", wallet.Name, err)
			continue
		}
		fmt.Printf("✅ Completed %s\n\n", wallet.Name)
	}

	return nil
}

// BackfillWallet backfills historical data for a specific wallet
func (hb *HistoricalBackfiller) BackfillWallet(walletID int64) error {
	wallet, err := hb.multisigService.GetWalletByID(walletID)
	if err != nil {
		return fmt.Errorf("failed to get wallet: %w", err)
	}
	if wallet == nil {
		return fmt.Errorf("wallet not found")
	}

	fmt.Printf("   Wallet: %s [%d/%d]\n", wallet.Name, wallet.RequiredSigners, wallet.TotalSigners)

	// Get all addresses for this wallet
	addresses, err := hb.multisigService.GetWalletAddresses(walletID)
	if err != nil {
		return fmt.Errorf("failed to get wallet addresses: %w", err)
	}

	fmt.Printf("   Addresses: %d\n", len(addresses))

	// Collect all transaction history points for this wallet
	var allHistoryPoints []HistoryPoint

	for i, addr := range addresses {
		fmt.Printf("   [%d/%d] Fetching history for %s...\n", i+1, len(addresses), addr.Address)

		history, err := hb.bitcoinClient.GetAddressBalanceHistory(addr.Address)
		if err != nil {
			log.Printf("   Warning: Failed to get history for %s: %v", addr.Address, err)
			continue
		}

		// Convert to our format and add wallet/address info
		for _, point := range history {
			historyPoint := HistoryPoint{
				WalletID:     walletID,
				WalletName:   wallet.Name,
				Address:      addr.Address,
				AddressIndex: addr.AddressIndex,
				Timestamp:    point.Timestamp,
				Balance:      point.Balance,
				TxID:         point.TxID,
				Amount:       point.Amount,
				Category:     point.Category,
			}
			allHistoryPoints = append(allHistoryPoints, historyPoint)
		}

		fmt.Printf("   └─ Found %d transaction points\n", len(history))
	}

	if len(allHistoryPoints) == 0 {
		fmt.Printf("   No transaction history found\n")
		return nil
	}

	// Sort by timestamp
	sort.Slice(allHistoryPoints, func(i, j int) bool {
		return allHistoryPoints[i].Timestamp.Before(allHistoryPoints[j].Timestamp)
	})

	fmt.Printf("   Total history points: %d\n", len(allHistoryPoints))
	fmt.Printf("   Date range: %s to %s\n",
		allHistoryPoints[0].Timestamp.Format("2006-01-02"),
		allHistoryPoints[len(allHistoryPoints)-1].Timestamp.Format("2006-01-02"))

	// Add addresses to onchain tracking if not already there
	err = hb.ensureAddressTracking(addresses, wallet)
	if err != nil {
		return fmt.Errorf("failed to ensure address tracking: %w", err)
	}

	// Store historical balance snapshots
	err = hb.storeHistoricalBalances(allHistoryPoints, wallet)
	if err != nil {
		return fmt.Errorf("failed to store historical balances: %w", err)
	}

	return nil
}

// HistoryPoint represents a balance change point in time
type HistoryPoint struct {
	WalletID     int64
	WalletName   string
	Address      string
	AddressIndex int
	Timestamp    time.Time
	Balance      int64
	TxID         string
	Amount       int64
	Category     string
}

// ensureAddressTracking makes sure all multisig addresses are being tracked
func (hb *HistoricalBackfiller) ensureAddressTracking(addresses []db.MultisigAddress, wallet *db.MultisigWallet) error {
	fmt.Printf("   Ensuring address tracking...\n")

	for _, addr := range addresses {
		label := fmt.Sprintf("%s [%d/%d] - Index %d", wallet.Name, wallet.RequiredSigners, wallet.TotalSigners, addr.AddressIndex)

		if !hb.dryRun {
			_, err := hb.db.InsertOnchainAddress(addr.Address, label)
			if err != nil && !strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return fmt.Errorf("failed to insert address %s: %w", addr.Address, err)
			}
		}
	}

	fmt.Printf("   ✅ Address tracking ensured\n")
	return nil
}

// storeHistoricalBalances creates balance snapshots based on transaction history
func (hb *HistoricalBackfiller) storeHistoricalBalances(points []HistoryPoint, wallet *db.MultisigWallet) error {
	fmt.Printf("   Storing historical balance snapshots...\n")

	if hb.dryRun {
		fmt.Printf("   [DRY RUN] Would store %d balance snapshots\n", len(points))
		return nil
	}

	// Group points by address and day for individual address tracking
	addressBalances := make(map[string]map[string]int64) // address -> day -> balance

	for _, point := range points {
		day := point.Timestamp.Format("2006-01-02")
		if addressBalances[point.Address] == nil {
			addressBalances[point.Address] = make(map[string]int64)
		}
		// Use the latest balance for each day (points are sorted by time)
		addressBalances[point.Address][day] = point.Balance
	}

	stored := 0
	for address, dailyBalances := range addressBalances {
		fmt.Printf("   Processing address %s...\n", address)

		for day, balance := range dailyBalances {
			timestamp, _ := time.Parse("2006-01-02", day)

			// Store individual address balance history
			err := hb.db.InsertAddressBalanceHistory(address, balance, timestamp)
			if err != nil && !strings.Contains(err.Error(), "UNIQUE constraint failed") {
				log.Printf("   Warning: Failed to store balance for %s on %s: %v", address, day, err)
				continue
			}
			stored++
		}
	}

	fmt.Printf("   ✅ Stored %d address balance snapshots across %d addresses\n", stored, len(addressBalances))
	return nil
}
