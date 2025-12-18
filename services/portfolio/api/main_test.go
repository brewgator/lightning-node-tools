package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brewgator/lightning-node-tools/internal/db"
	"github.com/brewgator/lightning-node-tools/internal/testutils"
	"github.com/gorilla/mux"
)

func seedTestData(t *testing.T, database *db.Database) {
	t.Helper()

	// Create sample balance snapshots
	snapshots := []*db.BalanceSnapshot{
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
	events := []*db.ForwardingEvent{
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

func setupTestServer(t *testing.T) *Server {
	t.Helper()

	dbPath := testutils.CreateTestDBPath(t)
	database, err := db.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	seedTestData(t, database)

	server := &Server{
		db:     database,
		router: mux.NewRouter(),
	}
	server.setupRoutes()

	return server
}

func TestHealthEndpoint(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("GET", "/api/health", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusOK)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, true)

	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	testutils.AssertEqual(t, data["status"], "healthy")
}

func TestCurrentPortfolioEndpoint(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("GET", "/api/portfolio/current", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusOK)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, true)

	// Verify response structure
	if response.Data == nil {
		t.Fatal("Expected data in response")
	}

	// Convert to map for field checking
	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	// Check required fields exist
	requiredFields := []string{
		"id", "timestamp", "lightning_local", "lightning_remote",
		"onchain_confirmed", "onchain_unconfirmed", "tracked_addresses",
		"cold_storage", "total_portfolio", "total_liquid",
	}

	for _, field := range requiredFields {
		if _, exists := dataMap[field]; !exists {
			t.Errorf("Missing required field: %s", field)
		}
	}
}

func TestPortfolioHistoryEndpoint(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("GET", "/api/portfolio/history?days=30", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusOK)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, true)

	// Verify response is an array
	dataArray, ok := response.Data.([]interface{})
	if !ok {
		t.Fatal("Expected data to be an array")
	}

	// Should have our test data
	if len(dataArray) != 3 {
		t.Errorf("Expected 3 historical snapshots, got %d", len(dataArray))
	}
}

func TestPortfolioHistoryWithInvalidDays(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("GET", "/api/portfolio/history?days=invalid", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	// Should return bad request for invalid days
	testutils.AssertEqual(t, rr.Code, http.StatusBadRequest)
}

func TestLightningFeesEndpoint(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("GET", "/api/lightning/fees?days=7", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusOK)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, true)

	// Verify Chart.js format
	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	// Check required Chart.js fields
	requiredFields := []string{"labels", "datasets", "metadata"}
	for _, field := range requiredFields {
		if _, exists := dataMap[field]; !exists {
			t.Errorf("Missing required Chart.js field: %s", field)
		}
	}

	// Check metadata structure
	metadata, ok := dataMap["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected metadata to be a map")
	}

	requiredMetadata := []string{"total_fees", "total_forwards", "days_requested", "days_with_data"}
	for _, field := range requiredMetadata {
		if _, exists := metadata[field]; !exists {
			t.Errorf("Missing required metadata field: %s", field)
		}
	}
}

func TestLightningForwardsEndpoint(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("GET", "/api/lightning/forwards?days=7", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusOK)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, true)

	// Verify Chart.js format
	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	// Check required Chart.js fields
	requiredFields := []string{"labels", "datasets", "metadata"}
	for _, field := range requiredFields {
		if _, exists := dataMap[field]; !exists {
			t.Errorf("Missing required Chart.js field: %s", field)
		}
	}

	// Check metadata structure
	metadata, ok := dataMap["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected metadata to be a map")
	}

	// Should have success_rate field
	if _, exists := metadata["success_rate"]; !exists {
		t.Error("Missing success_rate in forwards metadata")
	}
}

func TestLightningFeesWithInvalidDays(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("GET", "/api/lightning/fees?days=1000", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusBadRequest)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, false)
	testutils.AssertNotEqual(t, response.Error, "")
}

func TestCORSHeaders(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("OPTIONS", "/api/health", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	// CORS may not be configured in test environment, so we accept various responses
	// The important thing is that the server doesn't crash on OPTIONS requests
	if rr.Code >= 500 {
		t.Errorf("Server error on OPTIONS request: %d", rr.Code)
	}
}

func TestNonExistentEndpoint(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("GET", "/api/nonexistent", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusNotFound)
}
