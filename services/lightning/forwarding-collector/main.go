package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/brewgator/lightning-node-tools/internal/db"
	"github.com/brewgator/lightning-node-tools/internal/lnd"
)

type Config struct {
	DatabasePath       string
	CollectionInterval time.Duration
	LNDClient          *lnd.Client
}

type ForwardingCollector struct {
	config        *Config
	db            *db.Database
	mockMode      bool
	lastTimestamp int64 // Track last collected timestamp to avoid duplicates
}

func main() {
	var (
		dbPath   = flag.String("db", "data/portfolio.db", "Path to SQLite database")
		interval = flag.Duration("interval", 5*time.Minute, "Collection interval")
		oneshot  = flag.Bool("oneshot", false, "Run once and exit (for testing)")
		mockMode = flag.Bool("mock", false, "Use mock data for testing without LND")
		catchup  = flag.Bool("catchup", false, "Collect entire forwarding history (one-time operation)")
		days     = flag.Int("days", 30, "Number of days to catch up (only used with --catchup)")
	)
	flag.Parse()

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize database with mock mode support
	database, err := db.NewDatabaseWithMockMode(*dbPath, *mockMode)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	if *mockMode {
		fmt.Println("📊 Using mock database tables (data will not affect real data)")
	}

	// Initialize LND client
	var lndClient *lnd.Client

	if *mockMode {
		fmt.Println("⚠️  Running in mock mode - using test data")
		lndClient = nil
	} else {
		var err error
		lndClient, err = lnd.NewClient()
		if err != nil {
			log.Fatalf("Failed to initialize LND client: %v (try --mock for testing)", err)
		}
	}

	config := &Config{
		DatabasePath:       *dbPath,
		CollectionInterval: *interval,
		LNDClient:          lndClient,
	}

	collector := &ForwardingCollector{
		config:        config,
		db:            database,
		mockMode:      *mockMode,
		lastTimestamp: time.Now().Add(-24 * time.Hour).Unix(), // Start from 24h ago initially
	}

	if *catchup {
		fmt.Printf("Running catch-up collection for the last %d days...\n", *days)
		if err := collector.catchupForwardingEvents(*days); err != nil {
			log.Fatalf("Catch-up collection failed: %v", err)
		}
		fmt.Println("Catch-up collection completed successfully")
		return
	}

	if *oneshot {
		fmt.Println("Running forwarding event collection once...")
		if err := collector.collectForwardingEvents(); err != nil {
			log.Fatalf("Forwarding event collection failed: %v", err)
		}
		fmt.Println("Forwarding event collection completed successfully")
		return
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start collection loop
	ticker := time.NewTicker(config.CollectionInterval)
	defer ticker.Stop()

	fmt.Printf("Starting forwarding event collection every %v...\n", config.CollectionInterval)

	// Collect initial data
	if err := collector.collectForwardingEvents(); err != nil {
		log.Printf("Initial forwarding event collection failed: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := collector.collectForwardingEvents(); err != nil {
				log.Printf("Forwarding event collection failed: %v", err)
			}
		case <-sigChan:
			fmt.Println("Received shutdown signal, exiting...")
			return
		}
	}
}

func (c *ForwardingCollector) collectForwardingEvents() error {
	currentTime := time.Now()
	fmt.Printf("[%s] Collecting forwarding events since %s...\n",
		currentTime.Format("2006-01-02 15:04:05"),
		time.Unix(c.lastTimestamp, 0).Format("2006-01-02 15:04:05"))

	if c.mockMode {
		return c.collectMockForwardingEvents()
	}

	if c.config.LNDClient == nil {
		return fmt.Errorf("LND client is nil")
	}

	// Get forwarding history since last collection
	startTime := fmt.Sprintf("%d", c.lastTimestamp)
	endTime := fmt.Sprintf("%d", currentTime.Unix())

	history, err := c.config.LNDClient.GetForwardingHistory(startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to get forwarding history: %w", err)
	}

	if len(history.ForwardingEvents) == 0 {
		fmt.Printf("✅ No new forwarding events since last collection\n")
		c.lastTimestamp = currentTime.Unix()
		return nil
	}

	// Convert and insert forwarding events
	var insertedCount int
	for _, event := range history.ForwardingEvents {
		timestamp, err := strconv.ParseInt(event.Timestamp, 10, 64)
		if err != nil {
			log.Printf("Warning: invalid timestamp in forwarding event: %v", err)
			continue
		}

		// Skip events we've already processed
		if timestamp <= c.lastTimestamp {
			continue
		}

		// For amounts, use the msat values and convert to sat
		// If AmtIn/AmtOut are in msat, use those, otherwise use Fee/FeeMsat
		amountIn, _ := strconv.ParseInt(event.AmtIn, 10, 64)
		amountOut, _ := strconv.ParseInt(event.AmtOut, 10, 64)
		fee, _ := strconv.ParseInt(event.FeeMsat, 10, 64)

		// Convert from msat to sat (amounts are already in msat format)
		amountInSat := amountIn / 1000
		amountOutSat := amountOut / 1000
		feeSat := fee / 1000

		dbEvent := &db.ForwardingEvent{
			Timestamp:    time.Unix(timestamp, 0),
			ChannelInID:  event.ChanIdIn,
			ChannelOutID: event.ChanIdOut,
			AmountIn:     amountInSat,
			AmountOut:    amountOutSat,
			Fee:          feeSat,
		}

		if err := c.db.InsertForwardingEvent(dbEvent); err != nil {
			log.Printf("Warning: failed to insert forwarding event: %v", err)
			continue
		}

		insertedCount++
	}

	c.lastTimestamp = currentTime.Unix()
	fmt.Printf("✅ Inserted %d new forwarding events\n", insertedCount)

	return nil
}

func (c *ForwardingCollector) collectMockForwardingEvents() error {
	// Create mock forwarding events for testing
	now := time.Now()

	mockEvents := []*db.ForwardingEvent{
		{
			Timestamp:    now.Add(-5 * time.Minute),
			ChannelInID:  "123456789:1:0",
			ChannelOutID: "987654321:1:0",
			AmountIn:     100000, // 100k sats
			AmountOut:    99800,  // 99.8k sats
			Fee:          200,    // 200 sats fee
		},
		{
			Timestamp:    now.Add(-3 * time.Minute),
			ChannelInID:  "111222333:1:0",
			ChannelOutID: "444555666:1:0",
			AmountIn:     50000, // 50k sats
			AmountOut:    49900, // 49.9k sats
			Fee:          100,   // 100 sats fee
		},
	}

	var insertedCount int
	for _, event := range mockEvents {
		// Skip if we've already processed this timestamp
		if event.Timestamp.Unix() <= c.lastTimestamp {
			continue
		}

		if err := c.db.InsertForwardingEvent(event); err != nil {
			log.Printf("Warning: failed to insert mock forwarding event: %v", err)
			continue
		}
		insertedCount++
	}

	c.lastTimestamp = now.Unix()
	fmt.Printf("✅ Inserted %d mock forwarding events\n", insertedCount)

	return nil
}

func (c *ForwardingCollector) catchupForwardingEvents(days int) error {
	if c.mockMode {
		fmt.Println("⚠️  Mock mode - catch-up will create synthetic historical data")
		return c.catchupMockForwardingEvents(days)
	}

	if c.config.LNDClient == nil {
		return fmt.Errorf("LND client is nil")
	}

	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	fmt.Printf("📅 Collecting forwarding history from %s to %s (%d days)\n",
		startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"),
		days)

	// Process in chunks to avoid overwhelming LND API
	chunkDays := 7 // Process 1 week at a time
	totalInserted := 0

	for currentStart := startTime; currentStart.Before(endTime); {
		currentEnd := currentStart.AddDate(0, 0, chunkDays)
		if currentEnd.After(endTime) {
			currentEnd = endTime
		}

		fmt.Printf("🔍 Processing chunk: %s to %s\n",
			currentStart.Format("2006-01-02"),
			currentEnd.Format("2006-01-02"))

		startTimeStr := fmt.Sprintf("%d", currentStart.Unix())
		endTimeStr := fmt.Sprintf("%d", currentEnd.Unix())

		history, err := c.config.LNDClient.GetForwardingHistory(startTimeStr, endTimeStr)
		if err != nil {
			log.Printf("Warning: failed to get forwarding history for chunk %s-%s: %v",
				currentStart.Format("2006-01-02"), currentEnd.Format("2006-01-02"), err)
			currentStart = currentEnd
			continue
		}

		chunkInserted := 0
		for _, event := range history.ForwardingEvents {
			timestamp, err := strconv.ParseInt(event.Timestamp, 10, 64)
			if err != nil {
				log.Printf("Warning: invalid timestamp in forwarding event: %v", err)
				continue
			}

			amountIn, _ := strconv.ParseInt(event.AmtIn, 10, 64)
			amountOut, _ := strconv.ParseInt(event.AmtOut, 10, 64)
			fee, _ := strconv.ParseInt(event.FeeMsat, 10, 64)

			// Convert from msat to sat
			amountInSat := amountIn / 1000
			amountOutSat := amountOut / 1000
			feeSat := fee / 1000

			dbEvent := &db.ForwardingEvent{
				Timestamp:    time.Unix(timestamp, 0),
				ChannelInID:  event.ChanIdIn,
				ChannelOutID: event.ChanIdOut,
				AmountIn:     amountInSat,
				AmountOut:    amountOutSat,
				Fee:          feeSat,
			}

			// Use INSERT OR IGNORE to handle potential duplicates
			if err := c.db.InsertForwardingEventIgnoreDuplicate(dbEvent); err != nil {
				log.Printf("Warning: failed to insert forwarding event: %v", err)
				continue
			}

			chunkInserted++
		}

		totalInserted += chunkInserted
		fmt.Printf("✅ Chunk complete: %d events inserted (%d total so far)\n", chunkInserted, totalInserted)

		currentStart = currentEnd

		// Small delay between chunks to be nice to LND
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("🎉 Catch-up complete: %d total forwarding events processed\n", totalInserted)

	return nil
}

func (c *ForwardingCollector) catchupMockForwardingEvents(days int) error {
	fmt.Printf("🎭 Creating mock forwarding history for %d days\n", days)

	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	// Create mock events distributed across the time range
	eventsPerDay := 10
	totalEvents := days * eventsPerDay

	var totalInserted int

	for i := 0; i < totalEvents; i++ {
		// Distribute events across the time range
		dayOffset := float64(i) / float64(eventsPerDay)
		eventTime := startTime.Add(time.Duration(dayOffset * 24 * float64(time.Hour)))

		// Skip future events
		if eventTime.After(endTime) {
			continue
		}

		mockEvent := &db.ForwardingEvent{
			Timestamp:    eventTime,
			ChannelInID:  fmt.Sprintf("%d:1:0", 123456789+int64(i%3)),
			ChannelOutID: fmt.Sprintf("%d:1:0", 987654321+int64(i%5)),
			AmountIn:     int64(50000 + (i*1000)%100000), // Varying amounts
			AmountOut:    0,                              // Will be calculated
			Fee:          int64(100 + i%500),             // Varying fees
		}

		// Calculate amount out based on fee
		mockEvent.AmountOut = mockEvent.AmountIn - mockEvent.Fee

		if err := c.db.InsertForwardingEventIgnoreDuplicate(mockEvent); err != nil {
			log.Printf("Warning: failed to insert mock forwarding event: %v", err)
			continue
		}

		totalInserted++
	}

	fmt.Printf("✅ Mock catch-up complete: %d events created\n", totalInserted)
	return nil
}
