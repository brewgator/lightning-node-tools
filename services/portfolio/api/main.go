package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/brewgator/lightning-node-tools/internal/bitcoin"
	"github.com/brewgator/lightning-node-tools/internal/db"
	"github.com/brewgator/lightning-node-tools/internal/multisig"
	"github.com/brewgator/lightning-node-tools/internal/utils"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

const (
	// MaxHistoryDays is the maximum number of days that can be requested for historical data
	MaxHistoryDays = 365
	// BitcoinGenesisDate is the date of the Bitcoin genesis block (January 3, 2009)
	BitcoinGenesisDate = "2009-01-03"
	// MaxAddressGenerationBatch is the maximum number of addresses that can be generated in a single request
	// This limit prevents database overload and API timeouts for large batch operations
	MaxAddressGenerationBatch = 100
)

type Server struct {
	db              *db.Database
	router          *mux.Router
	balanceService  *bitcoin.BalanceService
	multisigService *multisig.MultisigService
	mockMode        bool
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func getVersion() string {
	// Use absolute path to git and set working directory
	cmd := exec.Command("/usr/bin/git", "rev-parse", "--short", "HEAD")
	if output, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(output))
	}
	return "unknown"
}

func main() {
	var (
		dbPath        = flag.String("db", "data/portfolio.db", "Path to SQLite database")
		port          = flag.String("port", "8090", "Port to serve on")
		host          = flag.String("host", "127.0.0.1", "Host to serve on")
		mockMode      = flag.Bool("mock", false, "Use mock data for testing without real data")
		noBitcoinNode = flag.Bool("no-bitcoin", false, "Disable Bitcoin node integration")
	)
	flag.Parse()

	// Initialize database with mock mode support
	database, err := db.NewDatabaseWithMockMode(*dbPath, *mockMode)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	if *mockMode {
		fmt.Println("📊 API running in mock mode (using mock database tables)")
	}

	var balanceService *bitcoin.BalanceService

	// Initialize Bitcoin balance service if not disabled
	if !*noBitcoinNode && !*mockMode {
		bitcoinClient, err := bitcoin.NewClient()
		if err != nil {
			log.Printf("⚠️  Warning: Failed to connect to Bitcoin node: %v", err)
			log.Printf("💡 Balance updates will be disabled. Ensure bitcoin-cli is available and Bitcoin Core is running.")
		} else {
			fmt.Println("₿ Connected to Bitcoin Core node")
			balanceService = bitcoin.NewBalanceService(bitcoinClient, database, 30*time.Minute)

			// Start balance service in background
			go balanceService.Start()
		}
	}

	// Initialize multisig service
	multisigService := multisig.NewMultisigService(database)

	server := &Server{
		db:              database,
		router:          mux.NewRouter(),
		balanceService:  balanceService,
		multisigService: multisigService,
		mockMode:        *mockMode,
	}

	server.setupRoutes()

	// Setup CORS with environment-based configuration
	// Security Note: Using localhost origins in production exposes the API to CSRF attacks
	// from any application running on the user's machine. Always set ALLOWED_ORIGINS in production.
	allowedOrigins := []string{"http://localhost:8090", "http://127.0.0.1:8090"}

	// Check for production origins from environment variable
	if envOrigins := os.Getenv("ALLOWED_ORIGINS"); envOrigins != "" {
		allowedOrigins = strings.Split(envOrigins, ",")
		log.Printf("🔒 Using CORS origins from environment: %v", allowedOrigins)
	} else {
		// WARNING: Using localhost origins - NOT suitable for production!
		log.Printf("⚠️  WARNING: Using default localhost CORS origins. Set ALLOWED_ORIGINS environment variable for production!")
		log.Printf("⚠️  Example: export ALLOWED_ORIGINS='https://yourdomain.com,https://app.yourdomain.com'")
	}

	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(server.router)

	addr := fmt.Sprintf("%s:%s", *host, *port)
	fmt.Printf("🚀 Portfolio Dashboard API starting on http://%s\n", addr)
	fmt.Printf("📊 Database: %s", *dbPath)
	if *mockMode {
		fmt.Printf(" (mock mode)")
	}
	fmt.Printf("\n")

	// Cleanup function
	defer func() {
		if balanceService != nil {
			balanceService.Stop()
		}
	}()

	log.Fatal(http.ListenAndServe(addr, handler))
}

func (s *Server) setupRoutes() {
	// API routes
	api := s.router.PathPrefix("/api").Subrouter()

	// Portfolio endpoints
	api.HandleFunc("/portfolio/current", s.handleCurrentPortfolio).Methods("GET")
	api.HandleFunc("/portfolio/history", s.handlePortfolioHistory).Methods("GET")

	// Lightning endpoints
	api.HandleFunc("/lightning/fees", s.handleLightningFees).Methods("GET")
	api.HandleFunc("/lightning/forwards", s.handleLightningForwards).Methods("GET")

	// Onchain endpoints
	api.HandleFunc("/onchain/addresses", s.handleGetOnchainAddresses).Methods("GET")
	api.HandleFunc("/onchain/addresses", s.handleAddOnchainAddress).Methods("POST")
	api.HandleFunc("/onchain/addresses/{id:[0-9]+}", s.handleDeleteOnchainAddress).Methods("DELETE")
	api.HandleFunc("/onchain/history", s.handleOnchainHistory).Methods("GET")
	api.HandleFunc("/onchain/tx-history", s.handleTransactionHistory).Methods("GET")

	// Offline/Cold storage endpoints (consolidated)
	api.HandleFunc("/offline/accounts", s.handleGetOfflineAccounts).Methods("GET")
	api.HandleFunc("/offline/accounts", s.handleAddOfflineAccount).Methods("POST")
	api.HandleFunc("/offline/accounts/{id:[0-9]+}/balance", s.handleUpdateOfflineAccountBalance).Methods("PUT")
	api.HandleFunc("/offline/accounts/{id:[0-9]+}", s.handleDeleteOfflineAccount).Methods("DELETE")
	api.HandleFunc("/offline/history", s.handleOfflineHistory).Methods("GET")

	// Multisig wallet endpoints
	api.HandleFunc("/multisig/wallets", s.handleGetMultisigWallets).Methods("GET")
	api.HandleFunc("/multisig/wallets", s.handleImportMultisigWallet).Methods("POST")
	api.HandleFunc("/multisig/wallets/{id:[0-9]+}", s.handleGetMultisigWallet).Methods("GET")
	api.HandleFunc("/multisig/wallets/{id:[0-9]+}", s.handleDeleteMultisigWallet).Methods("DELETE")
	api.HandleFunc("/multisig/wallets/{id:[0-9]+}/addresses", s.handleGetMultisigAddresses).Methods("GET")
	api.HandleFunc("/multisig/wallets/{id:[0-9]+}/addresses/generate", s.handleGenerateMultisigAddresses).Methods("POST")

	// Health check
	api.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Version info
	api.HandleFunc("/version", s.handleVersion).Methods("GET")

	// Static file serving
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("web/static/")))
}

func (s *Server) handleCurrentPortfolio(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.db.GetLatestBalanceSnapshot()
	if err != nil {
		log.Printf("handleCurrentPortfolio: failed to get latest balance snapshot: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to get current portfolio")
		return
	}

	if snapshot == nil {
		s.writeError(w, http.StatusNotFound, "No portfolio data available")
		return
	}

	s.writeJSON(w, APIResponse{Success: true, Data: snapshot})
}

func (s *Server) handlePortfolioHistory(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	daysStr := r.URL.Query().Get("days")
	days := 30 // default
	var from, to time.Time

	if daysStr == "all" {
		// For "all" data, get from the earliest possible date
		to = time.Now()
		genesisDate, _ := time.Parse("2006-01-02", BitcoinGenesisDate)
		from = genesisDate
	} else if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= MaxHistoryDays {
			days = d
		} else {
			s.writeError(w, http.StatusBadRequest, "Invalid days parameter. Must be a number between 1 and "+strconv.Itoa(MaxHistoryDays)+", or 'all'")
			return
		}
		// Calculate time range
		to = time.Now()
		from = to.AddDate(0, 0, -days)
	} else {
		// Default case
		to = time.Now()
		from = to.AddDate(0, 0, -days)
	}

	snapshots, err := s.db.GetBalanceSnapshots(from, to)
	if err != nil {
		log.Printf("handlePortfolioHistory: failed to get balance snapshots: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to get portfolio history")
		return
	}

	s.writeJSON(w, APIResponse{Success: true, Data: snapshots})
}

func (s *Server) handleLightningFees(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	daysStr := r.URL.Query().Get("days")
	days := 30 // default
	var from, to time.Time

	if daysStr == "all" {
		// For "all" data, get from the earliest possible date
		to = time.Now()
		genesisDate, _ := time.Parse("2006-01-02", BitcoinGenesisDate)
		from = genesisDate
	} else if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= MaxHistoryDays {
			days = d
		} else {
			s.writeError(w, http.StatusBadRequest, "Invalid days parameter. Must be a number between 1 and "+strconv.Itoa(MaxHistoryDays)+", or 'all'")
			return
		}
		// Calculate time range
		to = time.Now()
		from = to.AddDate(0, 0, -days)
	} else {
		// Default case
		to = time.Now()
		from = to.AddDate(0, 0, -days)
	}

	feeData, err := s.db.GetForwardingEventsFees(from, to)
	if err != nil {
		log.Printf("handleLightningFees: failed to get forwarding fees: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to get Lightning fee data")
		return
	}

	// Format data for Chart.js consumption
	chartData := map[string]interface{}{
		"labels": make([]string, 0, len(feeData)),
		"datasets": []map[string]interface{}{
			{
				"label":           "Daily Fees (sats)",
				"data":            make([]int64, 0, len(feeData)),
				"backgroundColor": "rgba(54, 162, 235, 0.2)",
				"borderColor":     "rgba(54, 162, 235, 1)",
				"borderWidth":     1,
			},
		},
		"metadata": map[string]interface{}{
			"total_fees":     int64(0),
			"total_forwards": int64(0),
			"days_requested": days,
			"days_with_data": len(feeData),
		},
	}

	// Calculate totals and populate chart data
	var totalFees, totalForwards int64
	labels := chartData["labels"].([]string)
	data := chartData["datasets"].([]map[string]interface{})[0]["data"].([]int64)

	for _, day := range feeData {
		labels = append(labels, day.Date)
		data = append(data, day.TotalFee)
		totalFees += day.TotalFee
		totalForwards += day.ForwardCount
	}

	// Update the slices in the map
	chartData["labels"] = labels
	chartData["datasets"].([]map[string]interface{})[0]["data"] = data
	chartData["metadata"].(map[string]interface{})["total_fees"] = totalFees
	chartData["metadata"].(map[string]interface{})["total_forwards"] = totalForwards

	s.writeJSON(w, APIResponse{Success: true, Data: chartData})
}

func (s *Server) handleLightningForwards(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	daysStr := r.URL.Query().Get("days")
	days := 30 // default
	var from, to time.Time

	if daysStr == "all" {
		// For "all" data, get from the earliest possible date
		to = time.Now()
		genesisDate, _ := time.Parse("2006-01-02", BitcoinGenesisDate)
		from = genesisDate
	} else if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= MaxHistoryDays {
			days = d
		} else {
			s.writeError(w, http.StatusBadRequest, "Invalid days parameter. Must be a number between 1 and "+strconv.Itoa(MaxHistoryDays)+", or 'all'")
			return
		}
		// Calculate time range
		to = time.Now()
		from = to.AddDate(0, 0, -days)
	} else {
		// Default case
		to = time.Now()
		from = to.AddDate(0, 0, -days)
	}

	forwardData, err := s.db.GetForwardingEventsFees(from, to)
	if err != nil {
		log.Printf("handleLightningForwards: failed to get forwarding data: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to get Lightning forwards data")
		return
	}

	// Format data for Chart.js consumption
	chartData := map[string]interface{}{
		"labels": make([]string, 0, len(forwardData)),
		"datasets": []map[string]interface{}{
			{
				"label":           "Daily Forwards",
				"data":            make([]int64, 0, len(forwardData)),
				"backgroundColor": "rgba(75, 192, 192, 0.2)",
				"borderColor":     "rgba(75, 192, 192, 1)",
				"borderWidth":     1,
			},
		},
		"metadata": map[string]interface{}{
			"total_forwards": int64(0),
			"total_fees":     int64(0),
			"success_rate":   float64(100.0), // Currently no failure data available
			"days_requested": days,
			"days_with_data": len(forwardData),
		},
	}

	// Calculate totals and populate chart data
	var totalForwards, totalFees int64
	labels := chartData["labels"].([]string)
	data := chartData["datasets"].([]map[string]interface{})[0]["data"].([]int64)

	for _, day := range forwardData {
		labels = append(labels, day.Date)
		data = append(data, day.ForwardCount)
		totalForwards += day.ForwardCount
		totalFees += day.TotalFee
	}

	// Update the slices in the map
	chartData["labels"] = labels
	chartData["datasets"].([]map[string]interface{})[0]["data"] = data
	chartData["metadata"].(map[string]interface{})["total_forwards"] = totalForwards
	chartData["metadata"].(map[string]interface{})["total_fees"] = totalFees

	s.writeJSON(w, APIResponse{Success: true, Data: chartData})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now(),
		},
	})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"version": getVersion(),
		},
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	}); err != nil {
		log.Printf("Failed to encode error response (status %d, message %q): %v", status, message, err)
	}
}

// AddOnchainAddressRequest represents the request body for adding a new onchain address
type AddOnchainAddressRequest struct {
	// Address is the required Bitcoin on-chain address or extended public key (xpub) to track.
	// It must be a valid Bitcoin address or xpub string.
	Address string `json:"address"`
	// Label is an optional human-readable description for the address; it may be empty.
	Label string `json:"label"`
}

// handleGetOnchainAddresses handles GET /api/onchain/addresses
func (s *Server) handleGetOnchainAddresses(w http.ResponseWriter, r *http.Request) {
	addresses, err := s.db.GetOnchainAddresses()
	if err != nil {
		log.Printf("handleGetOnchainAddresses: failed to get onchain addresses: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to get tracked addresses")
		return
	}

	s.writeJSON(w, APIResponse{Success: true, Data: addresses})
}

// handleAddOnchainAddress handles POST /api/onchain/addresses
func (s *Server) handleAddOnchainAddress(w http.ResponseWriter, r *http.Request) {
	var req AddOnchainAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	// Validate required fields
	if req.Address == "" {
		s.writeError(w, http.StatusBadRequest, "Address is required")
		return
	}

	// Validate Bitcoin address format
	if !utils.ValidateBitcoinAddress(req.Address) && !utils.ValidateXPub(req.Address) {
		s.writeError(w, http.StatusBadRequest, "Invalid Bitcoin address or xpub format")
		return
	}

	// If Bitcoin balance service is available, use it to import and track the address
	if s.balanceService != nil {
		newAddress, err := s.balanceService.ImportAndTrackAddress(req.Address, req.Label)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				s.writeError(w, http.StatusConflict, "Address is already being tracked")
				return
			}
			log.Printf("handleAddOnchainAddress: failed to import address via balance service: %v", err)
			s.writeError(w, http.StatusInternalServerError, "Failed to add address")
			return
		}

		s.writeJSON(w, APIResponse{
			Success: true,
			Data:    newAddress,
		})
	} else {
		// Fallback to basic database insertion if no balance service
		address, err := s.db.InsertOnchainAddress(req.Address, req.Label)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				s.writeError(w, http.StatusConflict, "Address is already being tracked")
				return
			}
			log.Printf("handleAddOnchainAddress: failed to insert address: %v", err)
			s.writeError(w, http.StatusInternalServerError, "Failed to add address")
			return
		}

		s.writeJSON(w, APIResponse{
			Success: true,
			Data:    address,
		})
	}
}

// handleDeleteOnchainAddress handles DELETE /api/onchain/addresses/:id
func (s *Server) handleDeleteOnchainAddress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		s.writeError(w, http.StatusBadRequest, "Address ID is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid address ID")
		return
	}

	// Check if address exists
	address, err := s.db.GetOnchainAddressByID(id)
	if err != nil {
		log.Printf("handleDeleteOnchainAddress: failed to get address by ID: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to check address")
		return
	}
	if address == nil {
		s.writeError(w, http.StatusNotFound, "Address not found")
		return
	}

	// Delete the address
	err = s.db.DeleteOnchainAddress(id)
	if err != nil {
		log.Printf("handleDeleteOnchainAddress: failed to delete address: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to delete address")
		return
	}

	s.writeJSON(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Address deleted successfully",
			"address": address.Address,
		},
	})
}

// handleOnchainHistory handles GET /api/onchain/history
func (s *Server) handleOnchainHistory(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	address := r.URL.Query().Get("address")
	if address == "" {
		s.writeError(w, http.StatusBadRequest, "Address parameter is required")
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 30 // default
	var from, to time.Time

	if daysStr == "all" {
		// For "all" data, get from the earliest possible date
		to = time.Now()
		genesisDate, _ := time.Parse("2006-01-02", BitcoinGenesisDate)
		from = genesisDate
	} else if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= MaxHistoryDays {
			days = d
		} else {
			s.writeError(w, http.StatusBadRequest, "Invalid days parameter. Must be a number between 1 and "+strconv.Itoa(MaxHistoryDays)+", or 'all'")
			return
		}
		// Calculate time range
		to = time.Now()
		from = to.AddDate(0, 0, -days)
	} else {
		// Default case
		to = time.Now()
		from = to.AddDate(0, 0, -days)
	}

	balances, err := s.db.GetAddressBalanceHistory(address, from, to)
	if err != nil {
		log.Printf("handleOnchainHistory: failed to get balance history for address %s: %v", address, err)
		s.writeError(w, http.StatusInternalServerError, "Failed to get address balance history")
		return
	}

	// Format data for Chart.js consumption
	chartData := map[string]interface{}{
		"labels": make([]string, 0, len(balances)),
		"datasets": []map[string]interface{}{
			{
				"label":           fmt.Sprintf("Balance for %s", address),
				"data":            make([]int64, 0, len(balances)),
				"backgroundColor": "rgba(255, 159, 64, 0.2)",
				"borderColor":     "rgba(255, 159, 64, 1)",
				"borderWidth":     1,
			},
		},
		"metadata": map[string]interface{}{
			"address":        address,
			"days_requested": days,
			"days_with_data": len(balances),
		},
	}

	// Populate chart data
	labels := chartData["labels"].([]string)
	data := chartData["datasets"].([]map[string]interface{})[0]["data"].([]int64)

	for _, balance := range balances {
		labels = append(labels, balance.Timestamp.Format("2006-01-02"))
		data = append(data, balance.Balance)
	}

	// Update the slices in the map
	chartData["labels"] = labels
	chartData["datasets"].([]map[string]interface{})[0]["data"] = data

	s.writeJSON(w, APIResponse{Success: true, Data: chartData})
}

// handleTransactionHistory handles GET /api/onchain/tx-history
// Returns transaction-based balance history like Sparrow wallet (step chart data)
func (s *Server) handleTransactionHistory(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	address := r.URL.Query().Get("address")
	if address == "" {
		s.writeError(w, http.StatusBadRequest, "Address parameter is required")
		return
	}

	// Check if Bitcoin client is available
	if s.balanceService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Bitcoin client not available")
		return
	}

	// We need access to the Bitcoin client - let's create one directly
	bitcoinClient, err := bitcoin.NewClient()
	if err != nil {
		log.Printf("handleTransactionHistory: failed to create Bitcoin client: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Bitcoin client unavailable")
		return
	}

	// Get transaction-based balance history
	history, err := bitcoinClient.GetAddressBalanceHistory(address)
	if err != nil {
		log.Printf("handleTransactionHistory: failed to get transaction history for address %s: %v", address, err)
		s.writeError(w, http.StatusInternalServerError, "Failed to get transaction-based balance history")
		return
	}

	// Format data for step chart visualization
	chartData := map[string]interface{}{
		"labels": make([]string, 0, len(history)),
		"datasets": []map[string]interface{}{
			{
				"label":           fmt.Sprintf("Balance for %s", address),
				"data":            make([]map[string]interface{}, 0, len(history)),
				"backgroundColor": "rgba(240, 136, 62, 0.2)",
				"borderColor":     "rgba(240, 136, 62, 1)",
				"borderWidth":     2,
				"stepped":         true, // Enable step chart
				"fill":            false,
			},
		},
		"metadata": map[string]interface{}{
			"address":       address,
			"data_points":   len(history),
			"data_type":     "transaction_based",
			"visualization": "step_chart",
		},
	}

	// Populate chart data with transaction-based points
	labels, ok := chartData["labels"].([]string)
	if !ok {
		log.Printf("handleTransactionHistory: failed to assert labels as []string")
		s.writeError(w, http.StatusInternalServerError, "Failed to build chart data structure")
		return
	}
	
	datasets, ok := chartData["datasets"].([]map[string]interface{})
	if !ok || len(datasets) == 0 {
		log.Printf("handleTransactionHistory: failed to assert datasets as []map[string]interface{}")
		s.writeError(w, http.StatusInternalServerError, "Failed to build chart data structure")
		return
	}
	
	dataPoints, ok := datasets[0]["data"].([]map[string]interface{})
	if !ok {
		log.Printf("handleTransactionHistory: failed to assert data as []map[string]interface{}")
		s.writeError(w, http.StatusInternalServerError, "Failed to build chart data structure")
		return
	}

	for _, point := range history {
		labels = append(labels, point.Timestamp.Format("2006-01-02T15:04:05Z"))

		dataPoint := map[string]interface{}{
			"x": point.Timestamp.Format("2006-01-02T15:04:05Z"),
			"y": point.Balance,
		}

		// Add transaction metadata if available
		if point.TxID != "" {
			dataPoint["txid"] = point.TxID
			dataPoint["amount"] = point.Amount
			dataPoint["category"] = point.Category
		}

		dataPoints = append(dataPoints, dataPoint)
	}

	// Update the slices in the map
	chartData["labels"] = labels
	datasets, ok = chartData["datasets"].([]map[string]interface{})
	if ok && len(datasets) > 0 {
		datasets[0]["data"] = dataPoints
	}

	s.writeJSON(w, APIResponse{Success: true, Data: chartData})
}

// OfflineAccountRequest represents the request body for offline account operations
type OfflineAccountRequest struct {
	Name     string `json:"name"`
	Balance  int64  `json:"balance"`
	Notes    string `json:"notes"`
	Verified bool   `json:"verified"`
}

// handleGetOfflineAccounts handles GET /api/offline/accounts
func (s *Server) handleGetOfflineAccounts(w http.ResponseWriter, r *http.Request) {
	entries, err := s.db.GetColdStorageEntriesWithWarnings()
	if err != nil {
		log.Printf("handleGetOfflineAccounts: failed to get offline accounts: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to get offline accounts")
		return
	}

	s.writeJSON(w, APIResponse{Success: true, Data: entries})
}

// handleAddOfflineAccount handles POST /api/offline/accounts
func (s *Server) handleAddOfflineAccount(w http.ResponseWriter, r *http.Request) {
	var req OfflineAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	// Validate required fields
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "Name is required")
		return
	}

	if req.Balance < 0 {
		s.writeError(w, http.StatusBadRequest, "Balance cannot be negative")
		return
	}

	if !req.Verified {
		s.writeError(w, http.StatusBadRequest, "You must verify the balance before adding")
		return
	}

	// Add the offline account to database
	entry, err := s.db.InsertColdStorageEntry(req.Name, req.Balance, req.Notes)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			s.writeError(w, http.StatusConflict, "An account with this name already exists")
			return
		}
		log.Printf("handleAddOfflineAccount: failed to insert entry: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to add offline account")
		return
	}

	s.writeJSON(w, APIResponse{
		Success: true,
		Data:    entry,
	})
}

// handleUpdateOfflineAccountBalance handles PUT /api/offline/accounts/{id}/balance
func (s *Server) handleUpdateOfflineAccountBalance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		s.writeError(w, http.StatusBadRequest, "Entry ID is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid entry ID")
		return
	}

	var req OfflineAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	// Validate required fields
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "Name is required")
		return
	}

	if req.Balance < 0 {
		s.writeError(w, http.StatusBadRequest, "Balance cannot be negative")
		return
	}

	if !req.Verified {
		s.writeError(w, http.StatusBadRequest, "You must verify the balance before updating")
		return
	}

	// Check if entry exists
	existingEntry, err := s.db.GetColdStorageEntryByID(id)
	if err != nil {
		log.Printf("handleUpdateOfflineAccountBalance: failed to get entry by ID: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to check offline account")
		return
	}
	if existingEntry == nil {
		s.writeError(w, http.StatusNotFound, "Offline account not found")
		return
	}

	// Update the entry
	updatedEntry, err := s.db.UpdateColdStorageEntry(id, req.Name, req.Balance, req.Notes)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			s.writeError(w, http.StatusConflict, "An account with this name already exists")
			return
		}
		log.Printf("handleUpdateOfflineAccountBalance: failed to update entry: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to update offline account")
		return
	}

	s.writeJSON(w, APIResponse{
		Success: true,
		Data:    updatedEntry,
	})
}

// handleDeleteOfflineAccount handles DELETE /api/offline/accounts/{id}
func (s *Server) handleDeleteOfflineAccount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		s.writeError(w, http.StatusBadRequest, "Entry ID is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid entry ID")
		return
	}

	// Check if offline account exists
	entry, err := s.db.GetColdStorageEntryByID(id)
	if err != nil {
		log.Printf("handleDeleteOfflineAccount: failed to get offline account by ID: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to check offline account")
		return
	}
	if entry == nil {
		s.writeError(w, http.StatusNotFound, "Offline account not found")
		return
	}

	// Delete the offline account
	err = s.db.DeleteColdStorageEntry(id)
	if err != nil {
		log.Printf("handleDeleteOfflineAccount: failed to delete offline account: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to delete offline account")
		return
	}

	s.writeJSON(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Offline account deleted successfully",
			"name":    entry.Name,
		},
	})
}

// handleOfflineHistory handles GET /api/offline/history
func (s *Server) handleOfflineHistory(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	accountIDStr := r.URL.Query().Get("account")
	if accountIDStr == "" {
		s.writeError(w, http.StatusBadRequest, "Account ID parameter is required")
		return
	}

	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid account ID")
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 30 // default
	var from, to time.Time

	if daysStr == "all" {
		// For "all" data, get from the earliest possible date
		to = time.Now()
		genesisDate, _ := time.Parse("2006-01-02", BitcoinGenesisDate)
		from = genesisDate
	} else if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= MaxHistoryDays {
			days = d
		} else {
			s.writeError(w, http.StatusBadRequest, "Invalid days parameter. Must be a number between 1 and "+strconv.Itoa(MaxHistoryDays)+", or 'all'")
			return
		}
		// Calculate time range
		to = time.Now()
		from = to.AddDate(0, 0, -days)
	} else {
		// Default case
		to = time.Now()
		from = to.AddDate(0, 0, -days)
	}

	history, err := s.db.GetColdStorageHistory(accountID, from, to)
	if err != nil {
		log.Printf("handleOfflineHistory: failed to get history for account %d: %v", accountID, err)
		s.writeError(w, http.StatusInternalServerError, "Failed to get account balance history")
		return
	}

	// Format data for Chart.js consumption
	chartData := map[string]interface{}{
		"labels": make([]string, 0, len(history)),
		"datasets": []map[string]interface{}{
			{
				"label":           fmt.Sprintf("Balance History"),
				"data":            make([]int64, 0, len(history)),
				"backgroundColor": "rgba(255, 159, 64, 0.2)",
				"borderColor":     "rgba(255, 159, 64, 1)",
				"borderWidth":     2,
				"fill":            false,
				"tension":         0.3,
			},
		},
		"metadata": map[string]interface{}{
			"account_id":     accountID,
			"days_requested": days,
			"days_with_data": len(history),
		},
	}

	// Populate chart data
	labels := chartData["labels"].([]string)
	data := chartData["datasets"].([]map[string]interface{})[0]["data"].([]int64)

	for _, entry := range history {
		labels = append(labels, entry.Timestamp.Format("2006-01-02"))
		data = append(data, entry.Balance)
	}

	// Update the slices in the map
	chartData["labels"] = labels
	chartData["datasets"].([]map[string]interface{})[0]["data"] = data

	s.writeJSON(w, APIResponse{Success: true, Data: chartData})
}

// Multisig wallet handlers

// handleGetMultisigWallets handles GET /api/multisig/wallets
func (s *Server) handleGetMultisigWallets(w http.ResponseWriter, r *http.Request) {
	wallets, err := s.multisigService.GetWallets()
	if err != nil {
		log.Printf("handleGetMultisigWallets: failed to get wallets: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to get multisig wallets")
		return
	}

	s.writeJSON(w, APIResponse{Success: true, Data: wallets})
}

// handleImportMultisigWallet handles POST /api/multisig/wallets
func (s *Server) handleImportMultisigWallet(w http.ResponseWriter, r *http.Request) {
	// Read the uploaded JSON file
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	// Import the wallet config
	wallet, err := s.multisigService.ImportWalletConfig(body)
	if err != nil {
		log.Printf("handleImportMultisigWallet: failed to import wallet: %v", err)
		if strings.Contains(err.Error(), "already exists") {
			s.writeError(w, http.StatusConflict, err.Error())
			return
		}
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to import wallet: %v", err))
		return
	}

	// Generate initial addresses (first 5 addresses)
	addresses, err := s.multisigService.GenerateAddresses(wallet.ID, 5)
	if err != nil {
		log.Printf("handleImportMultisigWallet: failed to generate initial addresses: %v", err)
		// Don't fail the import, but include a warning in the response
		response := map[string]interface{}{
			"wallet":    wallet,
			"addresses": []db.MultisigAddress{},
			"warning":   "Initial address generation failed. You may need to generate addresses manually.",
		}
		s.writeJSON(w, APIResponse{
			Success: true,
			Data:    response,
		})
		return
	}

	response := map[string]interface{}{
		"wallet":    wallet,
		"addresses": addresses,
	}

	s.writeJSON(w, APIResponse{
		Success: true,
		Data:    response,
	})
}

// handleGetMultisigWallet handles GET /api/multisig/wallets/{id}
func (s *Server) handleGetMultisigWallet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		s.writeError(w, http.StatusBadRequest, "Wallet ID is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid wallet ID")
		return
	}

	wallet, err := s.multisigService.GetWalletByID(id)
	if err != nil {
		log.Printf("handleGetMultisigWallet: failed to get wallet: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to get wallet")
		return
	}
	if wallet == nil {
		s.writeError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	s.writeJSON(w, APIResponse{Success: true, Data: wallet})
}

// handleDeleteMultisigWallet handles DELETE /api/multisig/wallets/{id}
func (s *Server) handleDeleteMultisigWallet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		s.writeError(w, http.StatusBadRequest, "Wallet ID is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid wallet ID")
		return
	}

	// Check if wallet exists
	wallet, err := s.multisigService.GetWalletByID(id)
	if err != nil {
		log.Printf("handleDeleteMultisigWallet: failed to get wallet: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to check wallet")
		return
	}
	if wallet == nil {
		s.writeError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	// Delete the wallet
	err = s.multisigService.DeleteWallet(id)
	if err != nil {
		log.Printf("handleDeleteMultisigWallet: failed to delete wallet: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to delete wallet")
		return
	}

	s.writeJSON(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Wallet deleted successfully",
			"name":    html.EscapeString(wallet.Name),
		},
	})
}

// handleGetMultisigAddresses handles GET /api/multisig/wallets/{id}/addresses
func (s *Server) handleGetMultisigAddresses(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		s.writeError(w, http.StatusBadRequest, "Wallet ID is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid wallet ID")
		return
	}

	addresses, err := s.multisigService.GetWalletAddresses(id)
	if err != nil {
		log.Printf("handleGetMultisigAddresses: failed to get addresses: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to get addresses")
		return
	}

	s.writeJSON(w, APIResponse{Success: true, Data: addresses})
}

// GenerateAddressesRequest represents the request body for generating addresses
type GenerateAddressesRequest struct {
	Count int `json:"count"`
}

// handleGenerateMultisigAddresses handles POST /api/multisig/wallets/{id}/addresses/generate
func (s *Server) handleGenerateMultisigAddresses(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		s.writeError(w, http.StatusBadRequest, "Wallet ID is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid wallet ID")
		return
	}

	var req GenerateAddressesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	// Validate count
	if req.Count <= 0 || req.Count > MaxAddressGenerationBatch {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Count must be between 1 and %d", MaxAddressGenerationBatch))
		return
	}

	// Validate wallet exists first
	wallet, err := s.multisigService.GetWalletByID(id)
	if err != nil {
		log.Printf("handleGenerateMultisigAddresses: failed to get wallet: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Failed to check wallet")
		return
	}
	if wallet == nil {
		s.writeError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	// Generate addresses
	addresses, err := s.multisigService.GenerateAddresses(id, req.Count)
	if err != nil {
		log.Printf("handleGenerateMultisigAddresses: failed to generate addresses: %v", err)
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, "Wallet not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "Failed to generate addresses")
		return
	}

	s.writeJSON(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"addresses":       addresses,
			"count":           len(addresses),
			"requested_count": req.Count,
		},
	})
}
