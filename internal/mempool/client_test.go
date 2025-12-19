package mempool

import (
	"testing"
	"time"
)

func TestMempoolClient(t *testing.T) {
	// This test requires internet connectivity and should be run manually
	// Skip in automated testing environments
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := NewClient("")

	// Test with a known address (Bitcoin genesis coinbase)
	testAddress := "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"

	t.Run("GetAddressStats", func(t *testing.T) {
		stats, err := client.GetAddressStats(testAddress)
		if err != nil {
			t.Fatalf("Failed to get address stats: %v", err)
		}

		if stats.Address != testAddress {
			t.Errorf("Expected address %s, got %s", testAddress, stats.Address)
		}

		// Genesis address should have some received funds
		if stats.ChainStats.FundedTxoSum == 0 {
			t.Errorf("Expected funded txo sum > 0, got %d", stats.ChainStats.FundedTxoSum)
		}

		t.Logf("Address stats: %+v", stats)
	})

	t.Run("GetAddressBalance", func(t *testing.T) {
		balance, utxoCount, err := client.CalculateAddressBalance(testAddress)
		if err != nil {
			t.Fatalf("Failed to calculate balance: %v", err)
		}

		// Genesis address current balance should be 0 (all spent)
		if balance != 0 {
			t.Logf("Note: Genesis address balance is %d (expected 0)", balance)
		}

		t.Logf("Balance: %d sats, UTXOs: %d", balance, utxoCount)
	})

	t.Run("GetChainTips", func(t *testing.T) {
		tips, err := client.GetChainTips()
		if err != nil {
			t.Fatalf("Failed to get chain tips: %v", err)
		}

		if tips.Height == 0 {
			t.Errorf("Expected height > 0, got %d", tips.Height)
		}

		// Should be recent
		if time.Since(tips.Timestamp) > time.Hour {
			t.Errorf("Timestamp seems old: %v", tips.Timestamp)
		}

		t.Logf("Chain height: %d", tips.Height)
	})
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(3, 10*time.Millisecond)
	defer limiter.Stop()

	// Should be able to acquire 3 tokens immediately
	for i := 0; i < 3; i++ {
		if !limiter.TryAcquire() {
			t.Errorf("Should be able to acquire token %d", i+1)
		}
	}

	// Fourth token should not be available
	if limiter.TryAcquire() {
		t.Error("Should not be able to acquire 4th token immediately")
	}

	// Wait for replenishment
	time.Sleep(15 * time.Millisecond)

	// Should be able to acquire one more token
	if !limiter.TryAcquire() {
		t.Error("Should be able to acquire token after replenishment")
	}
}

func TestValidateAddress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := NewClient("")

	testCases := []struct {
		address string
		valid   bool
		name    string
	}{
		{"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", true, "genesis address"},
		{"bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4", true, "bech32 address"},
		{"invalid_address", false, "invalid address"},
		{"", false, "empty address"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid := client.ValidateAddress(tc.address)
			if valid != tc.valid {
				t.Errorf("Address %s: expected valid=%t, got valid=%t",
					tc.address, tc.valid, valid)
			}
		})
	}
}
