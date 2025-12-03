package main

import (
	"testing"
	"time"

	"github.com/brewgator/lightning-node-tools/pkg/db"
	"github.com/brewgator/lightning-node-tools/pkg/testutils"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		valid    bool
	}{
		{"Valid 5 minutes", 5 * time.Minute, true},
		{"Valid 1 hour", 1 * time.Hour, true},
		{"Too short", 30 * time.Second, false},
		{"Zero duration", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				DatabasePath:       ":memory:",
				CollectionInterval: tt.interval,
				LNDClient:          nil,
			}

			isValid := config.CollectionInterval >= 1*time.Minute
			if isValid != tt.valid {
				t.Errorf("Expected validity %v for interval %v, got %v",
					tt.valid, tt.interval, isValid)
			}
		})
	}
}

func TestDatabaseConnection(t *testing.T) {
	dbPath := testutils.CreateTestDBPath(t)
	database, err := db.NewDatabase(dbPath)
	testutils.AssertNoError(t, err)
	defer database.Close()

	// Test that we can insert a balance snapshot
	snapshot := &db.BalanceSnapshot{
		Timestamp:          time.Now(),
		LightningLocal:     1000000,
		LightningRemote:    2000000,
		OnchainConfirmed:   500000,
		OnchainUnconfirmed: 0,
		TrackedAddresses:   100000,
		ColdStorage:        0,
		TotalPortfolio:     3600000,
		TotalLiquid:        3600000,
	}

	err = database.InsertBalanceSnapshot(snapshot)
	testutils.AssertNoError(t, err)

	// Verify we can retrieve it
	retrieved, err := database.GetLatestBalanceSnapshot()
	testutils.AssertNoError(t, err)

	if retrieved == nil {
		t.Fatal("Expected to retrieve balance snapshot")
	}

	testutils.AssertEqual(t, retrieved.LightningLocal, snapshot.LightningLocal)
	testutils.AssertEqual(t, retrieved.TotalPortfolio, snapshot.TotalPortfolio)
}
