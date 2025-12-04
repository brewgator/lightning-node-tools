package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/brewgator/lightning-node-tools/pkg/db"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Server struct {
	db     *db.Database
	router *mux.Router
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func main() {
	var (
		dbPath   = flag.String("db", "data/portfolio.db", "Path to SQLite database")
		port     = flag.String("port", "8080", "Port to serve on")
		host     = flag.String("host", "127.0.0.1", "Host to serve on")
		mockMode = flag.Bool("mock", false, "Use mock data for testing without real data")
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

	server := &Server{
		db:     database,
		router: mux.NewRouter(),
	}

	server.setupRoutes()

	// Setup CORS
	c := cors.New(cors.Options{
		// TODO: Replace with your actual frontend domain(s) in production.
		AllowedOrigins: []string{"https://your-frontend-domain.com"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
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

	// Health check
	api.HandleFunc("/health", s.handleHealth).Methods("GET")

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
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	// Calculate time range
	to := time.Now()
	from := to.AddDate(0, 0, -days)

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
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		} else if daysStr != "" {
			s.writeError(w, http.StatusBadRequest, "Invalid days parameter. Must be a number between 1 and 365")
			return
		}
	}

	// Calculate time range
	to := time.Now()
	from := to.AddDate(0, 0, -days)

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
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		} else if daysStr != "" {
			s.writeError(w, http.StatusBadRequest, "Invalid days parameter. Must be a number between 1 and 365")
			return
		}
	}

	// Calculate time range
	to := time.Now()
	from := to.AddDate(0, 0, -days)

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
