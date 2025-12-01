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
		dbPath = flag.String("db", "data/portfolio.db", "Path to SQLite database")
		port   = flag.String("port", "8080", "Port to serve on")
		host   = flag.String("host", "127.0.0.1", "Host to serve on")
	)
	flag.Parse()

	// Initialize database
	database, err := db.NewDatabase(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

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
	fmt.Printf("📊 Database: %s\n", *dbPath)

	log.Fatal(http.ListenAndServe(addr, handler))
}

func (s *Server) setupRoutes() {
	// API routes
	api := s.router.PathPrefix("/api").Subrouter()

	// Portfolio endpoints
	api.HandleFunc("/portfolio/current", s.handleCurrentPortfolio).Methods("GET")
	api.HandleFunc("/portfolio/history", s.handlePortfolioHistory).Methods("GET")

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
