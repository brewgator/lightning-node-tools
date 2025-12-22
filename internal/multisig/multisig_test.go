package multisig

import (
	"encoding/json"
	"testing"

	"github.com/brewgator/lightning-node-tools/internal/db"
	"github.com/brewgator/lightning-node-tools/internal/testutils"
)

// createTestDB creates a test database for multisig tests
func createTestDB(t *testing.T) *db.Database {
	t.Helper()
	dbPath := testutils.CreateTestDBPath(t)
	database, err := db.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return database
}

// getTestWalletConfig returns a valid test wallet configuration
func getTestWalletConfig() *db.WalletConfig {
	return &db.WalletConfig{
		Name:        "Test 2-of-3 Multisig",
		UUID:        "test-uuid-12345",
		AddressType: "P2WSH",
		Network:     "mainnet",
		Quorum: db.WalletQuorum{
			RequiredSigners: 2,
			TotalSigners:    3,
		},
		StartingAddressIndex: 0,
		ExtendedPublicKeys: []db.WalletExtendedPubKey{
			{
				Name:      "Key 1",
				XPub:      "xpub6CUGRUonZSQ4TWtTMmzXdrXDtypWKiKrhko4egpiMZbpiaQL2jkwSB1icqYh2cfDfVxdx4df189oLKnC5fSwqPfgyP3hooxujYzAu3fDVmz",
				BIP32Path: "m/48'/0'/0'/2'",
				XFP:       "abcd1234",
			},
			{
				Name:      "Key 2",
				XPub:      "xpub6D4BDPcP2GT577Vvch3R8wDkScZWzQzMMUm3PWbmWvVJrZwQY4VUNgqFJPMM3No2dFDFGTsxxpG5uJh7n7epu4trkrX7x7DogT5Uv6fcLW5",
				BIP32Path: "m/48'/0'/0'/2'",
				XFP:       "efgh5678",
			},
			{
				Name:      "Key 3",
				XPub:      "xpub6FHa3pjLCk84BayeJxFW2SP4XRrFd1JYnxeLeU8EqN3vDfZmbqBqaGJAyiLjTAwm6ZLRQUMv1ZACTj37sR62cfN7fe5JnJ7dh8zL4fiyLHV",
				BIP32Path: "m/48'/0'/0'/2'",
				XFP:       "ijkl9012",
			},
		},
	}
}

func TestNewMultisigService(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	if service == nil {
		t.Fatal("Expected non-nil service")
	}
	if service.db != database {
		t.Error("Service database not set correctly")
	}
	if service.network == nil {
		t.Error("Service network not initialized")
	}
}

func TestValidateWalletConfig_Valid(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()

	err := service.validateWalletConfig(config)
	if err != nil {
		t.Errorf("Expected valid config to pass validation, got error: %v", err)
	}
}

func TestValidateWalletConfig_MissingName(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()
	config.Name = ""

	err := service.validateWalletConfig(config)
	if err == nil {
		t.Error("Expected error for missing name")
	}
}

func TestValidateWalletConfig_MissingUUID(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()
	config.UUID = ""

	err := service.validateWalletConfig(config)
	if err == nil {
		t.Error("Expected error for missing UUID")
	}
}

func TestValidateWalletConfig_UnsupportedAddressType(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()
	config.AddressType = "P2PKH"

	err := service.validateWalletConfig(config)
	if err == nil {
		t.Error("Expected error for unsupported address type")
	}
}

func TestValidateWalletConfig_SupportedAddressTypes(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)

	supportedTypes := []string{"P2SH", "P2WSH", "P2SH-P2WSH", "P2SH_P2WSH"}
	for _, addrType := range supportedTypes {
		config := getTestWalletConfig()
		config.AddressType = addrType

		err := service.validateWalletConfig(config)
		if err != nil {
			t.Errorf("Expected address type %s to be valid, got error: %v", addrType, err)
		}
	}
}

func TestValidateWalletConfig_InvalidNetwork(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()
	config.Network = "regtest"

	err := service.validateWalletConfig(config)
	if err == nil {
		t.Error("Expected error for unsupported network")
	}
}

func TestValidateWalletConfig_InvalidQuorum(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)

	tests := []struct {
		name            string
		requiredSigners int
		totalSigners    int
	}{
		{"zero required", 0, 3},
		{"zero total", 2, 0},
		{"required exceeds total", 4, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := getTestWalletConfig()
			config.Quorum.RequiredSigners = tt.requiredSigners
			config.Quorum.TotalSigners = tt.totalSigners

			err := service.validateWalletConfig(config)
			if err == nil {
				t.Errorf("Expected error for invalid quorum: %d-of-%d", tt.requiredSigners, tt.totalSigners)
			}
		})
	}
}

func TestValidateWalletConfig_MismatchedKeyCount(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()
	config.Quorum.TotalSigners = 2 // Only 2 signers but 3 keys provided

	err := service.validateWalletConfig(config)
	if err == nil {
		t.Error("Expected error for mismatched key count")
	}
}

func TestValidateWalletConfig_InvalidXpub(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()
	config.ExtendedPublicKeys[0].XPub = "invalid-xpub"

	err := service.validateWalletConfig(config)
	if err == nil {
		t.Error("Expected error for invalid xpub format")
	}
}

func TestImportWalletConfig_Success(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	wallet, err := service.ImportWalletConfig(configJSON)
	if err != nil {
		t.Fatalf("Failed to import wallet config: %v", err)
	}

	if wallet == nil {
		t.Fatal("Expected non-nil wallet")
	}
	if wallet.ID == 0 {
		t.Error("Expected wallet ID to be set")
	}
	if wallet.Name != config.Name {
		t.Errorf("Expected name %s, got %s", config.Name, wallet.Name)
	}
	if wallet.UUID != config.UUID {
		t.Errorf("Expected UUID %s, got %s", config.UUID, wallet.UUID)
	}
	if len(wallet.ExtendedPublicKeys) != len(config.ExtendedPublicKeys) {
		t.Errorf("Expected %d keys, got %d", len(config.ExtendedPublicKeys), len(wallet.ExtendedPublicKeys))
	}
}

func TestImportWalletConfig_DuplicateUUID(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Import first time
	_, err = service.ImportWalletConfig(configJSON)
	if err != nil {
		t.Fatalf("Failed to import wallet config first time: %v", err)
	}

	// Import second time with same UUID
	_, err = service.ImportWalletConfig(configJSON)
	if err == nil {
		t.Error("Expected error for duplicate UUID")
	}
}

func TestImportWalletConfig_InvalidJSON(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)

	_, err := service.ImportWalletConfig([]byte("invalid json"))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestGenerateAddresses_Success(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	wallet, err := service.ImportWalletConfig(configJSON)
	if err != nil {
		t.Fatalf("Failed to import wallet config: %v", err)
	}

	// Generate 5 addresses
	addresses, err := service.GenerateAddresses(wallet.ID, 5)
	if err != nil {
		t.Fatalf("Failed to generate addresses: %v", err)
	}

	if len(addresses) != 5 {
		t.Errorf("Expected 5 addresses, got %d", len(addresses))
	}

	// Verify addresses are unique
	seen := make(map[string]bool)
	for _, addr := range addresses {
		if seen[addr.Address] {
			t.Errorf("Duplicate address generated: %s", addr.Address)
		}
		seen[addr.Address] = true

		// Verify address fields
		if addr.Address == "" {
			t.Error("Address should not be empty")
		}
		if addr.RedeemScript == "" {
			t.Error("Redeem script should not be empty")
		}
		if addr.WalletID != wallet.ID {
			t.Errorf("Expected wallet ID %d, got %d", wallet.ID, addr.WalletID)
		}
	}

	// Verify sequential indices
	for i, addr := range addresses {
		if addr.AddressIndex != i {
			t.Errorf("Expected address index %d, got %d", i, addr.AddressIndex)
		}
	}
}

func TestGenerateAddresses_Sequential(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	wallet, err := service.ImportWalletConfig(configJSON)
	if err != nil {
		t.Fatalf("Failed to import wallet config: %v", err)
	}

	// Generate 3 addresses first time (indices 0, 1, 2)
	addresses1, err := service.GenerateAddresses(wallet.ID, 3)
	if err != nil {
		t.Fatalf("Failed to generate addresses first time: %v", err)
	}

	if len(addresses1) != 3 {
		t.Errorf("Expected 3 addresses on first call, got %d", len(addresses1))
	}

	// Generate 2 more addresses (indices 3, 4)
	addresses2, err := service.GenerateAddresses(wallet.ID, 2)
	if err != nil {
		t.Fatalf("Failed to generate addresses second time: %v", err)
	}

	if len(addresses2) != 2 {
		t.Errorf("Expected 2 addresses on second call, got %d", len(addresses2))
	}

	// Verify addresses are different (different indices)
	for _, addr1 := range addresses1 {
		for _, addr2 := range addresses2 {
			if addr1.Address == addr2.Address {
				t.Errorf("Found duplicate address across batches: %s", addr1.Address)
			}
		}
	}

	// Verify correct indices
	for i, addr := range addresses1 {
		if addr.AddressIndex != i {
			t.Errorf("Expected address index %d in first batch, got %d", i, addr.AddressIndex)
		}
	}

	for i, addr := range addresses2 {
		if addr.AddressIndex != i+3 {
			t.Errorf("Expected address index %d in second batch, got %d", i+3, addr.AddressIndex)
		}
	}

	// Get all wallet addresses to verify total count
	allAddresses, err := service.GetWalletAddresses(wallet.ID)
	if err != nil {
		t.Fatalf("Failed to get all wallet addresses: %v", err)
	}

	if len(allAddresses) != 5 {
		t.Errorf("Expected 5 total addresses in database, got %d", len(allAddresses))
	}
}

func TestGenerateAddresses_NonExistentWallet(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)

	// Try to generate addresses for non-existent wallet
	_, err := service.GenerateAddresses(99999, 5)
	if err == nil {
		t.Error("Expected error for non-existent wallet")
	}
}

func TestGetWallets(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)

	// Initially should have no wallets
	wallets, err := service.GetWallets()
	if err != nil {
		t.Fatalf("Failed to get wallets: %v", err)
	}
	if len(wallets) != 0 {
		t.Errorf("Expected 0 wallets initially, got %d", len(wallets))
	}

	// Import a wallet
	config := getTestWalletConfig()
	configJSON, _ := json.Marshal(config)
	_, err = service.ImportWalletConfig(configJSON)
	if err != nil {
		t.Fatalf("Failed to import wallet: %v", err)
	}

	// Should now have 1 wallet
	wallets, err = service.GetWallets()
	if err != nil {
		t.Fatalf("Failed to get wallets after import: %v", err)
	}
	if len(wallets) != 1 {
		t.Errorf("Expected 1 wallet after import, got %d", len(wallets))
	}
}

func TestGetWalletByID(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()

	configJSON, _ := json.Marshal(config)
	wallet, err := service.ImportWalletConfig(configJSON)
	if err != nil {
		t.Fatalf("Failed to import wallet: %v", err)
	}

	// Get wallet by ID
	retrieved, err := service.GetWalletByID(wallet.ID)
	if err != nil {
		t.Fatalf("Failed to get wallet by ID: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected non-nil wallet")
	}
	if retrieved.ID != wallet.ID {
		t.Errorf("Expected ID %d, got %d", wallet.ID, retrieved.ID)
	}
	if retrieved.Name != wallet.Name {
		t.Errorf("Expected name %s, got %s", wallet.Name, retrieved.Name)
	}
}

func TestDeleteWallet(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()

	configJSON, _ := json.Marshal(config)
	wallet, err := service.ImportWalletConfig(configJSON)
	if err != nil {
		t.Fatalf("Failed to import wallet: %v", err)
	}

	// Delete wallet
	err = service.DeleteWallet(wallet.ID)
	if err != nil {
		t.Fatalf("Failed to delete wallet: %v", err)
	}

	// Wallet should not be returned in active wallets list
	wallets, err := service.GetWallets()
	if err != nil {
		t.Fatalf("Failed to get wallets: %v", err)
	}
	if len(wallets) != 0 {
		t.Errorf("Expected 0 active wallets after deletion, got %d", len(wallets))
	}
}

func TestDeriveAddress_Deterministic(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()

	configJSON, _ := json.Marshal(config)
	wallet, err := service.ImportWalletConfig(configJSON)
	if err != nil {
		t.Fatalf("Failed to import wallet: %v", err)
	}

	// Derive the same address twice
	addr1, err := service.DeriveAddress(wallet, 0)
	if err != nil {
		t.Fatalf("Failed to derive address first time: %v", err)
	}

	addr2, err := service.DeriveAddress(wallet, 0)
	if err != nil {
		t.Fatalf("Failed to derive address second time: %v", err)
	}

	// Addresses should be identical
	if addr1.Address != addr2.Address {
		t.Errorf("Expected deterministic address generation, got different addresses: %s vs %s", addr1.Address, addr2.Address)
	}
	if addr1.RedeemScript != addr2.RedeemScript {
		t.Error("Expected identical redeem scripts")
	}
}

func TestGetWalletAddresses(t *testing.T) {
	database := createTestDB(t)
	defer database.Close()

	service := NewMultisigService(database)
	config := getTestWalletConfig()

	configJSON, _ := json.Marshal(config)
	wallet, err := service.ImportWalletConfig(configJSON)
	if err != nil {
		t.Fatalf("Failed to import wallet: %v", err)
	}

	// Generate some addresses
	_, err = service.GenerateAddresses(wallet.ID, 3)
	if err != nil {
		t.Fatalf("Failed to generate addresses: %v", err)
	}

	// Get wallet addresses
	addresses, err := service.GetWalletAddresses(wallet.ID)
	if err != nil {
		t.Fatalf("Failed to get wallet addresses: %v", err)
	}

	if len(addresses) != 3 {
		t.Errorf("Expected 3 addresses, got %d", len(addresses))
	}
}
