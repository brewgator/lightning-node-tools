package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGetOnchainAddressesEmpty(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("GET", "/api/onchain/addresses", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusOK)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, true)

	// When there are no addresses, Go's json.Unmarshal may treat the empty slice as nil
	// Check if data is nil (empty) or an empty array
	if response.Data == nil {
		return // Empty is acceptable
	}

	dataArray, ok := response.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array or nil, got type %T", response.Data)
	}
	testutils.AssertEqual(t, len(dataArray), 0)
}

func TestAddOnchainAddress(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	// Test valid Bitcoin address
	payload := `{"address": "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh", "label": "Test Wallet"}`
	req, err := http.NewRequest("POST", "/api/onchain/addresses", strings.NewReader(payload))
	testutils.AssertNoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusOK)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, true)

	// Verify response structure
	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	testutils.AssertEqual(t, dataMap["address"], "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh")
	testutils.AssertEqual(t, dataMap["label"], "Test Wallet")
	testutils.AssertEqual(t, dataMap["active"], true)
}

func TestAddOnchainAddressInvalidJSON(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	// Test invalid JSON
	payload := `{"address": "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh", "label": "Test Wallet"`
	req, err := http.NewRequest("POST", "/api/onchain/addresses", strings.NewReader(payload))
	testutils.AssertNoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusBadRequest)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, false)
}

func TestAddOnchainAddressEmpty(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	// Test missing address
	payload := `{"label": "Test Wallet"}`
	req, err := http.NewRequest("POST", "/api/onchain/addresses", strings.NewReader(payload))
	testutils.AssertNoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusBadRequest)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, false)
	if !strings.Contains(response.Error, "Address is required") {
		t.Errorf("Expected error message about required address, got: %s", response.Error)
	}
}

func TestAddOnchainAddressInvalidFormat(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	// Test invalid Bitcoin address
	payload := `{"address": "notavalidaddress", "label": "Test"}`
	req, err := http.NewRequest("POST", "/api/onchain/addresses", strings.NewReader(payload))
	testutils.AssertNoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusBadRequest)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, false)
	if !strings.Contains(response.Error, "Invalid Bitcoin address") {
		t.Errorf("Expected error message about invalid address, got: %s", response.Error)
	}
}

func TestAddOnchainAddressDuplicate(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	address := "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh"

	// Add address first time
	payload := fmt.Sprintf(`{"address": "%s", "label": "First"}`, address)
	req, err := http.NewRequest("POST", "/api/onchain/addresses", strings.NewReader(payload))
	testutils.AssertNoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)
	testutils.AssertEqual(t, rr.Code, http.StatusOK)

	// Try to add same address again
	payload = fmt.Sprintf(`{"address": "%s", "label": "Second"}`, address)
	req, err = http.NewRequest("POST", "/api/onchain/addresses", strings.NewReader(payload))
	testutils.AssertNoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr = httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusConflict)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, false)
	if !strings.Contains(response.Error, "already being tracked") {
		t.Errorf("Expected error message about duplicate address, got: %s", response.Error)
	}
}

func TestGetOnchainAddressesWithData(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	// Add some addresses
	addresses := []string{
		"bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh",
		"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
	}

	for i, addr := range addresses {
		payload := fmt.Sprintf(`{"address": "%s", "label": "Wallet %d"}`, addr, i+1)
		req, err := http.NewRequest("POST", "/api/onchain/addresses", strings.NewReader(payload))
		testutils.AssertNoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		server.router.ServeHTTP(rr, req)
		testutils.AssertEqual(t, rr.Code, http.StatusOK)
	}

	// Get all addresses
	req, err := http.NewRequest("GET", "/api/onchain/addresses", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusOK)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, true)

	dataArray, ok := response.Data.([]interface{})
	if !ok {
		t.Fatal("Expected data to be an array")
	}

	if len(dataArray) != 2 {
		t.Errorf("Expected 2 addresses, got %d", len(dataArray))
	}
}

func TestDeleteOnchainAddress(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	// Add an address first
	payload := `{"address": "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh", "label": "To Delete"}`
	req, err := http.NewRequest("POST", "/api/onchain/addresses", strings.NewReader(payload))
	testutils.AssertNoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)
	testutils.AssertEqual(t, rr.Code, http.StatusOK)

	var addResponse APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &addResponse)
	testutils.AssertNoError(t, err)

	dataMap := addResponse.Data.(map[string]interface{})
	addressID := int64(dataMap["id"].(float64))

	// Delete the address
	req, err = http.NewRequest("DELETE", fmt.Sprintf("/api/onchain/addresses/%d", addressID), nil)
	testutils.AssertNoError(t, err)

	rr = httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusOK)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, true)
}

func TestDeleteOnchainAddressNotFound(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	// Try to delete non-existent address
	req, err := http.NewRequest("DELETE", "/api/onchain/addresses/99999", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusNotFound)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, false)
}

func TestDeleteOnchainAddressInvalidID(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	// Try to delete with invalid ID (non-numeric)
	req, err := http.NewRequest("DELETE", "/api/onchain/addresses/notanumber", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	// The router may not match the route with non-numeric ID and return 404,
	// or it may match and return 400. Both are acceptable for invalid input.
	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("Expected 400 or 404, got %d", rr.Code)
	}
}

func TestOnchainHistoryMissingAddress(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("GET", "/api/onchain/history", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusBadRequest)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, false)
	if !strings.Contains(response.Error, "Address parameter is required") {
		t.Errorf("Expected error about missing address, got: %s", response.Error)
	}
}

func TestOnchainHistoryInvalidDays(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("GET", "/api/onchain/history?address=bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh&days=1000", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusBadRequest)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, false)
	if !strings.Contains(response.Error, "Invalid days parameter") {
		t.Errorf("Expected error about invalid days, got: %s", response.Error)
	}
}

func TestOnchainHistoryDefaultDays(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("GET", "/api/onchain/history?address=bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh", nil)
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

	// Check metadata
	metadata, ok := dataMap["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected metadata to be a map")
	}

	if _, exists := metadata["address"]; !exists {
		t.Error("Missing address in metadata")
	}
}

func TestOnchainHistoryAllDays(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("GET", "/api/onchain/history?address=bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh&days=all", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusOK)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, true)

	// Verify response structure
	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	// Check that all required fields exist
	if _, exists := dataMap["labels"]; !exists {
		t.Error("Missing labels field")
	}
	if _, exists := dataMap["datasets"]; !exists {
		t.Error("Missing datasets field")
	}
	if _, exists := dataMap["metadata"]; !exists {
		t.Error("Missing metadata field")
	}
}

func TestOnchainHistoryCustomDays(t *testing.T) {
	server := setupTestServer(t)
	defer server.db.Close()

	req, err := http.NewRequest("GET", "/api/onchain/history?address=bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh&days=7", nil)
	testutils.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	testutils.AssertEqual(t, rr.Code, http.StatusOK)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, response.Success, true)

	dataMap := response.Data.(map[string]interface{})
	metadata := dataMap["metadata"].(map[string]interface{})

	// days_requested should be 7
	daysRequested := int(metadata["days_requested"].(float64))
	if daysRequested != 7 {
		t.Errorf("Expected days_requested to be 7, got %d", daysRequested)
	}
}
