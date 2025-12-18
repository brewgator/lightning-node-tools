package main

import (
	"testing"
	"time"

	"github.com/brewgator/lightning-node-tools/internal/db"
	"github.com/brewgator/lightning-node-tools/internal/testutils"
)

func TestForwardingConfig(t *testing.T) {
	config := &Config{
		DatabasePath:       ":memory:",
		CollectionInterval: 5 * time.Minute,
		LNDClient:          nil,
	}

	if config.CollectionInterval != 5*time.Minute {
		t.Errorf("Expected interval 5m, got %v", config.CollectionInterval)
	}
}

func TestForwardingEventInsertion(t *testing.T) {
	dbPath := testutils.CreateTestDBPath(t)
	database, err := db.NewDatabase(dbPath)
	testutils.AssertNoError(t, err)
	defer database.Close()

	// Test inserting a forwarding event
	event := &db.ForwardingEvent{
		Timestamp:    time.Now(),
		ChannelInID:  "123456789:1:0",
		ChannelOutID: "987654321:1:0",
		AmountIn:     100000,
		AmountOut:    99800,
		Fee:          200,
	}

	err = database.InsertForwardingEvent(event)
	testutils.AssertNoError(t, err)

	// Verify we can retrieve fee data
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(1 * time.Hour)

	feeData, err := database.GetForwardingEventsFees(from, to)
	testutils.AssertNoError(t, err)

	if len(feeData) == 0 {
		t.Error("Expected fee data after inserting forwarding event")
	}

	// Verify fee data
	dayData := feeData[0]
	testutils.AssertEqual(t, dayData.TotalFee, int64(200))
	testutils.AssertEqual(t, dayData.ForwardCount, int64(1))
}

func TestForwardingEventDuplicateHandling(t *testing.T) {
	dbPath := testutils.CreateTestDBPath(t)
	database, err := db.NewDatabase(dbPath)
	testutils.AssertNoError(t, err)
	defer database.Close()

	// Test data
	event := &db.ForwardingEvent{
		Timestamp:    time.Now().Truncate(time.Second),
		ChannelInID:  "123456789:1:0",
		ChannelOutID: "987654321:1:0",
		AmountIn:     100000,
		AmountOut:    99800,
		Fee:          200,
	}

	// Insert event twice
	err = database.InsertForwardingEvent(event)
	testutils.AssertNoError(t, err)

	err = database.InsertForwardingEventIgnoreDuplicate(event)
	testutils.AssertNoError(t, err)

	// Verify only one event exists
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(1 * time.Hour)

	feeData, err := database.GetForwardingEventsFees(from, to)
	testutils.AssertNoError(t, err)

	if len(feeData) > 0 {
		testutils.AssertEqual(t, feeData[0].ForwardCount, int64(1))
	}
}
