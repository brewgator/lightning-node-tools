package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/brewgator/lightning-node-tools/internal/bitcoin"
	"github.com/brewgator/lightning-node-tools/internal/db"
	"github.com/brewgator/lightning-node-tools/internal/utils"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

const (
	// MaxHistoryDays is the maximum number of days that can be requested for historical data
	MaxHistoryDays = 365
	// BitcoinGenesisDate is the date of the Bitcoin genesis block (January 3, 2009)
	BitcoinGenesisDate = "2009-01-03"
)

type Server struct {
	db             *db.Database
	router         *mux.Router
	balanceService *bitcoin.BalanceService
	mockMode       bool
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
		port          = flag.String("port", "8080", "Port to serve on")
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

	server := &Server{
		db:             database,
		router:         mux.NewRouter(),
		balanceService: balanceService,
		mockMode:       *mockMode,
	}

	server.setupRoutes()

	// Setup CORS
	c := cors.New(cors.Options{
		// TODO: Replace with your actual frontend domain(s) in production.
		AllowedOrigins: []string{"https://your-frontend-domain.com"},
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

	// Offline/Cold storage endpoints (consolidated)
	api.HandleFunc("/offline/accounts", s.handleGetOfflineAccounts).Methods("GET")
	api.HandleFunc("/offline/accounts", s.handleAddOfflineAccount).Methods("POST")
	api.HandleFunc("/offline/accounts/{id:[0-9]+}/balance", s.handleUpdateOfflineAccountBalance).Methods("PUT")
	api.HandleFunc("/offline/accounts/{id:[0-9]+}", s.handleDeleteOfflineAccount).Methods("DELETE")
	api.HandleFunc("/offline/history", s.handleOfflineHistory).Methods("GET")

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
