package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	conn *sql.DB
}

// NewDatabase creates a new database connection and initializes tables
func NewDatabase(dbPath string) (*Database, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &Database{conn: conn}
	if err := db.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *Database) Close() error {
	return db.conn.Close()
}

// initTables creates all required tables
func (db *Database) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS balance_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL,
			lightning_local INTEGER NOT NULL DEFAULT 0,
			lightning_remote INTEGER NOT NULL DEFAULT 0,
			onchain_confirmed INTEGER NOT NULL DEFAULT 0,
			onchain_unconfirmed INTEGER NOT NULL DEFAULT 0,
			tracked_addresses INTEGER NOT NULL DEFAULT 0,
			cold_storage INTEGER NOT NULL DEFAULT 0,
			total_portfolio INTEGER NOT NULL DEFAULT 0,
			total_liquid INTEGER NOT NULL DEFAULT 0,
			UNIQUE(timestamp)
		);`,

		`CREATE INDEX IF NOT EXISTS idx_balance_snapshots_timestamp ON balance_snapshots(timestamp);`,

		`CREATE TABLE IF NOT EXISTS channel_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL,
			channel_id TEXT NOT NULL,
			capacity INTEGER NOT NULL,
			local_balance INTEGER NOT NULL,
			remote_balance INTEGER NOT NULL,
			active BOOLEAN NOT NULL,
			peer_alias TEXT,
			fee_ppm INTEGER,
			base_fee INTEGER
		);`,

		`CREATE INDEX IF NOT EXISTS idx_channel_snapshots_timestamp ON channel_snapshots(timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_channel_snapshots_channel_id ON channel_snapshots(channel_id);`,

		`CREATE TABLE IF NOT EXISTS forwarding_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL,
			channel_in_id TEXT NOT NULL,
			channel_out_id TEXT NOT NULL,
			amount_in INTEGER NOT NULL,
			amount_out INTEGER NOT NULL,
			fee INTEGER NOT NULL
		);`,

		`CREATE INDEX IF NOT EXISTS idx_forwarding_events_timestamp ON forwarding_events(timestamp);`,

		`CREATE TABLE IF NOT EXISTS onchain_addresses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			address TEXT UNIQUE NOT NULL,
			label TEXT,
			active BOOLEAN NOT NULL DEFAULT 1
		);`,

		`CREATE TABLE IF NOT EXISTS address_balances (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			address_id INTEGER NOT NULL,
			timestamp DATETIME NOT NULL,
			balance INTEGER NOT NULL,
			tx_count INTEGER NOT NULL,
			FOREIGN KEY(address_id) REFERENCES onchain_addresses(id)
		);`,

		`CREATE INDEX IF NOT EXISTS idx_address_balances_timestamp ON address_balances(timestamp);`,

		`CREATE TABLE IF NOT EXISTS cold_storage_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			balance INTEGER NOT NULL,
			last_updated DATETIME NOT NULL,
			notes TEXT
		);`,
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// InsertBalanceSnapshot inserts a new balance snapshot
func (db *Database) InsertBalanceSnapshot(snapshot *BalanceSnapshot) error {
	query := `
		INSERT OR REPLACE INTO balance_snapshots
		(timestamp, lightning_local, lightning_remote, onchain_confirmed, onchain_unconfirmed,
		 tracked_addresses, cold_storage, total_portfolio, total_liquid)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.conn.Exec(query,
		snapshot.Timestamp,
		snapshot.LightningLocal,
		snapshot.LightningRemote,
		snapshot.OnchainConfirmed,
		snapshot.OnchainUnconfirmed,
		snapshot.TrackedAddresses,
		snapshot.ColdStorage,
		snapshot.TotalPortfolio,
		snapshot.TotalLiquid,
	)

	return err
}

// GetBalanceSnapshots retrieves balance snapshots within a time range
func (db *Database) GetBalanceSnapshots(from, to time.Time) ([]BalanceSnapshot, error) {
	query := `
		SELECT id, timestamp, lightning_local, lightning_remote, onchain_confirmed,
		       onchain_unconfirmed, tracked_addresses, cold_storage, total_portfolio, total_liquid
		FROM balance_snapshots
		WHERE timestamp BETWEEN ? AND ?
		ORDER BY timestamp ASC
	`

	rows, err := db.conn.Query(query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []BalanceSnapshot
	for rows.Next() {
		var s BalanceSnapshot
		err := rows.Scan(
			&s.ID, &s.Timestamp, &s.LightningLocal, &s.LightningRemote,
			&s.OnchainConfirmed, &s.OnchainUnconfirmed, &s.TrackedAddresses,
			&s.ColdStorage, &s.TotalPortfolio, &s.TotalLiquid,
		)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}

	return snapshots, rows.Err()
}

// GetLatestBalanceSnapshot retrieves the most recent balance snapshot
func (db *Database) GetLatestBalanceSnapshot() (*BalanceSnapshot, error) {
	query := `
		SELECT id, timestamp, lightning_local, lightning_remote, onchain_confirmed,
		       onchain_unconfirmed, tracked_addresses, cold_storage, total_portfolio, total_liquid
		FROM balance_snapshots
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var s BalanceSnapshot
	err := db.conn.QueryRow(query).Scan(
		&s.ID, &s.Timestamp, &s.LightningLocal, &s.LightningRemote,
		&s.OnchainConfirmed, &s.OnchainUnconfirmed, &s.TrackedAddresses,
		&s.ColdStorage, &s.TotalPortfolio, &s.TotalLiquid,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &s, nil
}

