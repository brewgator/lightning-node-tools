package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/brewgator/lightning-node-tools/internal/testutils"
)

func createTestDB(t *testing.T) *Database {
	t.Helper()
	dbPath := testutils.CreateTestDBPath(t)
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return db
}

func seedTestData(t *testing.T, database *Database) {
	t.Helper()

	// Create sample balance snapshots
	snapshots := []*BalanceSnapshot{
		{
			Timestamp:          time.Now().Add(-48 * time.Hour),
			LightningLocal:     1000000,
			LightningRemote:    2000000,
			OnchainConfirmed:   500000,
			OnchainUnconfirmed: 0,
			TrackedAddresses:   100000,
			ColdStorage:        0,
			TotalPortfolio:     3600000,
			TotalLiquid:        3600000,
		},
		{
			Timestamp:          time.Now().Add(-24 * time.Hour),
			LightningLocal:     1100000,
			LightningRemote:    1900000,
			OnchainConfirmed:   600000,
			OnchainUnconfirmed: 50000,
			TrackedAddresses:   150000,
			ColdStorage:        1000000,
			TotalPortfolio:     4800000,
			TotalLiquid:        3800000,
		},
		{
			Timestamp:          time.Now().Add(-1 * time.Hour),
			LightningLocal:     1200000,
			LightningRemote:    1800000,
			OnchainConfirmed:   700000,
			OnchainUnconfirmed: 0,
			TrackedAddresses:   200000,
			ColdStorage:        1000000,
			TotalPortfolio:     4900000,
			TotalLiquid:        3900000,
		},
	}

	for _, snapshot := range snapshots {
		err := database.InsertBalanceSnapshot(snapshot)
		if err != nil {
			t.Fatalf("Failed to insert test balance snapshot: %v", err)
		}
	}

	// Create sample forwarding events
	events := []*ForwardingEvent{
		{
			Timestamp:    time.Now().Add(-24 * time.Hour),
			ChannelInID:  "123456789:1:0",
			ChannelOutID: "987654321:1:0",
			AmountIn:     100000,
			AmountOut:    99800,
			Fee:          200,
		},
		{
			Timestamp:    time.Now().Add(-12 * time.Hour),
			ChannelInID:  "111222333:1:0",
			ChannelOutID: "444555666:1:0",
			AmountIn:     50000,
			AmountOut:    49900,
			Fee:          100,
		},
		{
			Timestamp:    time.Now().Add(-6 * time.Hour),
			ChannelInID:  "123456789:1:0",
			ChannelOutID: "444555666:1:0",
			AmountIn:     25000,
			AmountOut:    24950,
			Fee:          50,
		},
	}

	for _, event := range events {
		err := database.InsertForwardingEvent(event)
		if err != nil {
			t.Fatalf("Failed to insert test forwarding event: %v", err)
		}
	}
}

func TestNewDatabase(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Test that database was created successfully
	if db == nil {
		t.Fatal("Expected database to be created, got nil")
	}
}

func TestInsertAndGetBalanceSnapshot(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Test data
	snapshot := &BalanceSnapshot{
		Timestamp:          time.Now().Truncate(time.Second), // SQLite precision
		LightningLocal:     1000000,
		LightningRemote:    2000000,
		OnchainConfirmed:   500000,
		OnchainUnconfirmed: 100000,
		TrackedAddresses:   50000,
		ColdStorage:        5000000,
		TotalPortfolio:     8650000,
		TotalLiquid:        3650000,
	}

	// Insert snapshot
	err := db.InsertBalanceSnapshot(snapshot)
	testutils.AssertNoError(t, err)

	// Get latest snapshot
	retrieved, err := db.GetLatestBalanceSnapshot()
	testutils.AssertNoError(t, err)

	if retrieved == nil {
		t.Fatal("Expected to retrieve balance snapshot, got nil")
	}

	// Verify data
	testutils.AssertEqual(t, retrieved.LightningLocal, snapshot.LightningLocal)
	testutils.AssertEqual(t, retrieved.LightningRemote, snapshot.LightningRemote)
	testutils.AssertEqual(t, retrieved.OnchainConfirmed, snapshot.OnchainConfirmed)
	testutils.AssertEqual(t, retrieved.TotalPortfolio, snapshot.TotalPortfolio)
}

func TestGetBalanceSnapshots(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Seed test data
	seedTestData(t, db)

	// Get snapshots for last 72 hours to ensure we get all test data
	from := time.Now().Add(-72 * time.Hour)
	to := time.Now().Add(1 * time.Hour)

	snapshots, err := db.GetBalanceSnapshots(from, to)
	testutils.AssertNoError(t, err)

	if len(snapshots) != 3 {
		t.Errorf("Expected 3 snapshots, got %d", len(snapshots))
	}

	// Verify chronological order
	for i := 1; i < len(snapshots); i++ {
		if snapshots[i].Timestamp.Before(snapshots[i-1].Timestamp) {
			t.Error("Snapshots should be in chronological order")
		}
	}
}

func TestInsertAndGetForwardingEvents(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Test data
	event := &ForwardingEvent{
		Timestamp:    time.Now().Truncate(time.Second),
		ChannelInID:  "123456789:1:0",
		ChannelOutID: "987654321:1:0",
		AmountIn:     100000,
		AmountOut:    99800,
		Fee:          200,
	}

	// Insert event
	err := db.InsertForwardingEvent(event)
	testutils.AssertNoError(t, err)

	// Test fees aggregation
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now().Add(1 * time.Hour)

	feeData, err := db.GetForwardingEventsFees(from, to)
	testutils.AssertNoError(t, err)

	if len(feeData) == 0 {
		t.Error("Expected at least one day of fee data")
	}

	// Check fee data structure
	dayData := feeData[0]
	testutils.AssertEqual(t, dayData.TotalFee, int64(200))
	testutils.AssertEqual(t, dayData.ForwardCount, int64(1))
}

func TestInsertForwardingEventIgnoreDuplicate(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Test data
	event := &ForwardingEvent{
		Timestamp:    time.Now().Truncate(time.Second),
		ChannelInID:  "123456789:1:0",
		ChannelOutID: "987654321:1:0",
		AmountIn:     100000,
		AmountOut:    99800,
		Fee:          200,
	}

	// Insert event twice
	err := db.InsertForwardingEvent(event)
	testutils.AssertNoError(t, err)

	err = db.InsertForwardingEventIgnoreDuplicate(event)
	testutils.AssertNoError(t, err)

	// Verify only one event exists
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now().Add(1 * time.Hour)

	feeData, err := db.GetForwardingEventsFees(from, to)
	testutils.AssertNoError(t, err)

	if len(feeData) > 0 {
		testutils.AssertEqual(t, feeData[0].ForwardCount, int64(1))
	}
}

func TestGetForwardingEventsFees(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Seed test data
	seedTestData(t, db)

	// Get fees for last 48 hours
	from := time.Now().Add(-48 * time.Hour)
	to := time.Now()

	feeData, err := db.GetForwardingEventsFees(from, to)
	testutils.AssertNoError(t, err)

	if len(feeData) == 0 {
		t.Error("Expected fee data, got none")
	}

	// Verify fee data structure and content
	totalFees := int64(0)
	totalForwards := int64(0)

	for _, day := range feeData {
		if day.Date == "" {
			t.Error("Date should not be empty")
		}
		if day.TotalFee < 0 {
			t.Error("Total fee should not be negative")
		}
		if day.ForwardCount < 0 {
			t.Error("Forward count should not be negative")
		}
		totalFees += day.TotalFee
		totalForwards += day.ForwardCount
	}

	// We should have some fees from our test data
	if totalFees == 0 {
		t.Error("Expected some fees from test data")
	}
	if totalForwards == 0 {
		t.Error("Expected some forwards from test data")
	}
}

func TestMockModeIsolation(t *testing.T) {
	// Create regular database
	dbPath := testutils.CreateTestDBPath(t)
	regularDB, err := NewDatabase(dbPath)
	testutils.AssertNoError(t, err)
	defer regularDB.Close()

	// Create mock mode database with same path
	mockDB, err := NewDatabaseWithMockMode(dbPath, true)
	testutils.AssertNoError(t, err)
	defer mockDB.Close()

	// Verify mock mode status
	testutils.AssertEqual(t, regularDB.IsMockMode(), false)
	testutils.AssertEqual(t, mockDB.IsMockMode(), true)

	// Add data to regular database
	regularSnapshot := &BalanceSnapshot{
		Timestamp:      time.Now().Truncate(time.Second),
		LightningLocal: 1000000,
		TotalPortfolio: 1000000,
		TotalLiquid:    1000000,
	}
	err = regularDB.InsertBalanceSnapshot(regularSnapshot)
	testutils.AssertNoError(t, err)

	// Add data to mock database
	mockSnapshot := &BalanceSnapshot{
		Timestamp:      time.Now().Truncate(time.Second),
		LightningLocal: 2000000,
		TotalPortfolio: 2000000,
		TotalLiquid:    2000000,
	}
	err = mockDB.InsertBalanceSnapshot(mockSnapshot)
	testutils.AssertNoError(t, err)

	// Verify isolation: regular DB should only see regular data
	regularData, err := regularDB.GetLatestBalanceSnapshot()
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, regularData.LightningLocal, int64(1000000))

	// Verify isolation: mock DB should only see mock data
	mockData, err := mockDB.GetLatestBalanceSnapshot()
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, mockData.LightningLocal, int64(2000000))
}

func TestMockModeForwardingEvents(t *testing.T) {
	// Create databases with same path but different modes
	dbPath := testutils.CreateTestDBPath(t)
	regularDB, err := NewDatabase(dbPath)
	testutils.AssertNoError(t, err)
	defer regularDB.Close()

	mockDB, err := NewDatabaseWithMockMode(dbPath, true)
	testutils.AssertNoError(t, err)
	defer mockDB.Close()

	// Add forwarding event to regular database
	regularEvent := &ForwardingEvent{
		Timestamp:    time.Now().Truncate(time.Second),
		ChannelInID:  "regular:1:0",
		ChannelOutID: "regular:2:0",
		AmountIn:     100000,
		AmountOut:    99800,
		Fee:          200,
	}
	err = regularDB.InsertForwardingEvent(regularEvent)
	testutils.AssertNoError(t, err)

	// Add forwarding event to mock database
	mockEvent := &ForwardingEvent{
		Timestamp:    time.Now().Truncate(time.Second),
		ChannelInID:  "mock:1:0",
		ChannelOutID: "mock:2:0",
		AmountIn:     50000,
		AmountOut:    49900,
		Fee:          100,
	}
	err = mockDB.InsertForwardingEvent(mockEvent)
	testutils.AssertNoError(t, err)

	// Verify isolation for fee data
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(1 * time.Hour)

	regularFees, err := regularDB.GetForwardingEventsFees(from, to)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, len(regularFees) > 0, true)
	testutils.AssertEqual(t, regularFees[0].TotalFee, int64(200))

	mockFees, err := mockDB.GetForwardingEventsFees(from, to)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, len(mockFees) > 0, true)
	testutils.AssertEqual(t, mockFees[0].TotalFee, int64(100))
}

func TestTableNameGeneration(t *testing.T) {
	// Test regular mode
	dbPath := testutils.CreateTestDBPath(t)
	regularDB, err := NewDatabase(dbPath)
	testutils.AssertNoError(t, err)
	defer regularDB.Close()

	testutils.AssertEqual(t, regularDB.GetTableName("balance_snapshots"), "balance_snapshots")
	testutils.AssertEqual(t, regularDB.GetTableName("forwarding_events"), "forwarding_events")

	// Test mock mode
	mockDB, err := NewDatabaseWithMockMode(dbPath, true)
	testutils.AssertNoError(t, err)
	defer mockDB.Close()

	testutils.AssertEqual(t, mockDB.GetTableName("balance_snapshots"), "balance_snapshots_mock")
	testutils.AssertEqual(t, mockDB.GetTableName("forwarding_events"), "forwarding_events_mock")
}

func TestInsertAndGetOnchainAddress(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert a new onchain address
	address, err := db.InsertOnchainAddress("bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh", "Test Wallet")
	testutils.AssertNoError(t, err)

	if address == nil {
		t.Fatal("Expected address to be created, got nil")
	}

	// Verify returned address data
	testutils.AssertEqual(t, address.Address, "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh")
	testutils.AssertEqual(t, address.Label, "Test Wallet")
	testutils.AssertEqual(t, address.Active, true)
	if address.ID == 0 {
		t.Error("Expected ID to be set")
	}

	// Retrieve the address by ID
	retrieved, err := db.GetOnchainAddressByID(address.ID)
	testutils.AssertNoError(t, err)

	if retrieved == nil {
		t.Fatal("Expected to retrieve address, got nil")
	}

	// Verify retrieved data matches
	testutils.AssertEqual(t, retrieved.ID, address.ID)
	testutils.AssertEqual(t, retrieved.Address, address.Address)
	testutils.AssertEqual(t, retrieved.Label, address.Label)
	testutils.AssertEqual(t, retrieved.Active, address.Active)
}

func TestInsertDuplicateOnchainAddress(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	address := "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"

	// Insert address first time
	_, err := db.InsertOnchainAddress(address, "First")
	testutils.AssertNoError(t, err)

	// Try to insert same address again - should fail due to UNIQUE constraint
	_, err = db.InsertOnchainAddress(address, "Second")
	if err == nil {
		t.Error("Expected error when inserting duplicate address, got nil")
	}
}

func TestGetOnchainAddressByIDNotFound(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Try to get non-existent address
	address, err := db.GetOnchainAddressByID(99999)
	testutils.AssertNoError(t, err)

	if address != nil {
		t.Error("Expected nil for non-existent address, got address")
	}
}

func TestGetOnchainAddresses(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert multiple addresses
	addresses := []struct {
		addr  string
		label string
	}{
		{"bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh", "Wallet 1"},
		{"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", "Genesis"},
		{"3J98t1WpEZ73CNmYviecrnyiWrnqRhWNLy", "P2SH Address"},
	}

	for _, a := range addresses {
		_, err := db.InsertOnchainAddress(a.addr, a.label)
		testutils.AssertNoError(t, err)
	}

	// Get all addresses
	retrieved, err := db.GetOnchainAddresses()
	testutils.AssertNoError(t, err)

	if len(retrieved) != 3 {
		t.Errorf("Expected 3 addresses, got %d", len(retrieved))
	}

	// Verify ordering (should be by ID ASC)
	for i := 1; i < len(retrieved); i++ {
		if retrieved[i].ID <= retrieved[i-1].ID {
			t.Error("Addresses should be ordered by ID ASC")
		}
	}

	// Verify all addresses are active
	for _, addr := range retrieved {
		if !addr.Active {
			t.Error("All newly inserted addresses should be active")
		}
	}
}

func TestDeleteOnchainAddress(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert an address
	address, err := db.InsertOnchainAddress("bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh", "To Delete")
	testutils.AssertNoError(t, err)

	// Delete the address
	err = db.DeleteOnchainAddress(address.ID)
	testutils.AssertNoError(t, err)

	// Verify it's deleted
	retrieved, err := db.GetOnchainAddressByID(address.ID)
	testutils.AssertNoError(t, err)
	if retrieved != nil {
		t.Error("Address should be deleted")
	}
}

func TestDeleteNonExistentOnchainAddress(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Try to delete non-existent address
	err := db.DeleteOnchainAddress(99999)
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows when deleting non-existent address, got %v", err)
	}
}

func TestInsertAndGetAddressBalance(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert an address first
	address, err := db.InsertOnchainAddress("bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh", "Test")
	testutils.AssertNoError(t, err)

	// Insert balance record
	balance := &AddressBalance{
		AddressID: address.ID,
		Timestamp: time.Now().Truncate(time.Second),
		Balance:   1000000,
		TxCount:   5,
	}

	err = db.InsertAddressBalance(balance)
	testutils.AssertNoError(t, err)

	// Verify we can retrieve it
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(1 * time.Hour)

	balances, err := db.GetAddressBalanceHistory(address.Address, from, to)
	testutils.AssertNoError(t, err)

	if len(balances) != 1 {
		t.Fatalf("Expected 1 balance record, got %d", len(balances))
	}

	testutils.AssertEqual(t, balances[0].AddressID, address.ID)
	testutils.AssertEqual(t, balances[0].Balance, int64(1000000))
	testutils.AssertEqual(t, balances[0].TxCount, int64(5))
}

func TestInsertAddressBalanceMultiple(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert an address
	address, err := db.InsertOnchainAddress("bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh", "Test")
	testutils.AssertNoError(t, err)

	timestamp := time.Now().Truncate(time.Second)

	// Insert first balance
	balance1 := &AddressBalance{
		AddressID: address.ID,
		Timestamp: timestamp,
		Balance:   1000000,
		TxCount:   5,
	}
	err = db.InsertAddressBalance(balance1)
	testutils.AssertNoError(t, err)

	// Insert another balance with different timestamp
	balance2 := &AddressBalance{
		AddressID: address.ID,
		Timestamp: timestamp.Add(1 * time.Hour),
		Balance:   2000000,
		TxCount:   10,
	}
	err = db.InsertAddressBalance(balance2)
	testutils.AssertNoError(t, err)

	// Verify both records exist
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(2 * time.Hour)

	balances, err := db.GetAddressBalanceHistory(address.Address, from, to)
	testutils.AssertNoError(t, err)

	if len(balances) != 2 {
		t.Fatalf("Expected 2 balance records, got %d", len(balances))
	}
}

func TestGetAddressBalanceHistoryTimeRange(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert an address
	address, err := db.InsertOnchainAddress("bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh", "Test")
	testutils.AssertNoError(t, err)

	// Insert balance records at different times
	now := time.Now()
	timestamps := []time.Time{
		now.Add(-48 * time.Hour),
		now.Add(-24 * time.Hour),
		now.Add(-12 * time.Hour),
		now.Add(-1 * time.Hour),
	}

	for i, ts := range timestamps {
		balance := &AddressBalance{
			AddressID: address.ID,
			Timestamp: ts.Truncate(time.Second),
			Balance:   int64((i + 1) * 100000),
			TxCount:   int64(i + 1),
		}
		err = db.InsertAddressBalance(balance)
		testutils.AssertNoError(t, err)
	}

	// Query for last 24 hours only
	from := now.Add(-25 * time.Hour)
	to := now.Add(1 * time.Hour)

	balances, err := db.GetAddressBalanceHistory(address.Address, from, to)
	testutils.AssertNoError(t, err)

	// Should only get the last 3 records (within 25 hours)
	if len(balances) != 3 {
		t.Errorf("Expected 3 balance records in time range, got %d", len(balances))
	}

	// Verify ordering (should be ASC by timestamp)
	for i := 1; i < len(balances); i++ {
		if balances[i].Timestamp.Before(balances[i-1].Timestamp) {
			t.Error("Balance history should be ordered by timestamp ASC")
		}
	}
}

func TestGetAddressBalanceHistoryEmptyResult(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Query for non-existent address
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	balances, err := db.GetAddressBalanceHistory("nonexistent", from, to)
	testutils.AssertNoError(t, err)

	if len(balances) != 0 {
		t.Errorf("Expected 0 balance records for non-existent address, got %d", len(balances))
	}
}
