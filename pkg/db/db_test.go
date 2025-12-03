package db

import (
	"testing"
	"time"

	"github.com/brewgator/lightning-node-tools/pkg/testutils"
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
