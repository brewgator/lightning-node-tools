package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	// ErrNotFound indicates that a requested resource was not found
	ErrNotFound = errors.New("resource not found")
)

type Database struct {
	conn     *sql.DB
	mockMode bool
}

// NewDatabase creates a new database connection and initializes tables
func NewDatabase(dbPath string) (*Database, error) {
	return NewDatabaseWithMockMode(dbPath, false)
}

// NewDatabaseWithMockMode creates a new database connection with mock mode option
func NewDatabaseWithMockMode(dbPath string, mockMode bool) (*Database, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &Database{
		conn:     conn,
		mockMode: mockMode,
	}

	if err := db.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return db, nil
}

// IsMockMode returns true if the database is running in mock mode
func (db *Database) IsMockMode() bool {
	return db.mockMode
}

// getTableName returns the appropriate table name based on mock mode.
// SECURITY NOTE: baseName must ONLY be hardcoded string literals, never user input.
// The mockMode flag is an internal boolean set at database initialization.
// This function is safe because it only uses validated, internal parameters.
func (db *Database) getTableName(baseName string) string {
	if db.mockMode {
		return baseName + "_mock"
	}
	return baseName
}

// GetTableName returns the appropriate table name based on mock mode (public for testing)
func (db *Database) GetTableName(baseName string) string {
	return db.getTableName(baseName)
}

// Close closes the database connection
func (db *Database) Close() error {
	return db.conn.Close()
}

// initTables creates all required tables
func (db *Database) initTables() error {
	queries := []string{
		// Real data tables
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

		// Mock data tables (identical structure with _mock suffix)
		`CREATE TABLE IF NOT EXISTS balance_snapshots_mock (
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

		`CREATE INDEX IF NOT EXISTS idx_balance_snapshots_mock_timestamp ON balance_snapshots_mock(timestamp);`,

		`CREATE TABLE IF NOT EXISTS channel_snapshots_mock (
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

		`CREATE INDEX IF NOT EXISTS idx_channel_snapshots_mock_timestamp ON channel_snapshots_mock(timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_channel_snapshots_mock_channel_id ON channel_snapshots_mock(channel_id);`,

		`CREATE TABLE IF NOT EXISTS forwarding_events_mock (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL,
			channel_in_id TEXT NOT NULL,
			channel_out_id TEXT NOT NULL,
			amount_in INTEGER NOT NULL,
			amount_out INTEGER NOT NULL,
			fee INTEGER NOT NULL
		);`,

		`CREATE INDEX IF NOT EXISTS idx_forwarding_events_mock_timestamp ON forwarding_events_mock(timestamp);`,

		`CREATE TABLE IF NOT EXISTS onchain_addresses_mock (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			address TEXT UNIQUE NOT NULL,
			label TEXT,
			active BOOLEAN NOT NULL DEFAULT 1
		);`,

		`CREATE TABLE IF NOT EXISTS address_balances_mock (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			address_id INTEGER NOT NULL,
			timestamp DATETIME NOT NULL,
			balance INTEGER NOT NULL,
			tx_count INTEGER NOT NULL,
			FOREIGN KEY(address_id) REFERENCES onchain_addresses_mock(id)
		);`,

		`CREATE INDEX IF NOT EXISTS idx_address_balances_mock_timestamp ON address_balances_mock(timestamp);`,

		`CREATE TABLE IF NOT EXISTS cold_storage_entries_mock (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			balance INTEGER NOT NULL,
			last_updated DATETIME NOT NULL,
			notes TEXT
		);`,

		// Cold storage balance history tables
		`CREATE TABLE IF NOT EXISTS cold_storage_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id INTEGER NOT NULL,
			timestamp DATETIME NOT NULL,
			balance INTEGER NOT NULL,
			previous_balance INTEGER NOT NULL DEFAULT 0,
			is_verified BOOLEAN NOT NULL DEFAULT 1,
			notes TEXT,
			FOREIGN KEY(account_id) REFERENCES cold_storage_entries(id) ON DELETE CASCADE
		);`,

		`CREATE INDEX IF NOT EXISTS idx_cold_storage_history_timestamp ON cold_storage_history(timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_cold_storage_history_account_id ON cold_storage_history(account_id);`,

		`CREATE TABLE IF NOT EXISTS cold_storage_history_mock (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id INTEGER NOT NULL,
			timestamp DATETIME NOT NULL,
			balance INTEGER NOT NULL,
			previous_balance INTEGER NOT NULL DEFAULT 0,
			is_verified BOOLEAN NOT NULL DEFAULT 1,
			notes TEXT,
			FOREIGN KEY(account_id) REFERENCES cold_storage_entries_mock(id) ON DELETE CASCADE
		);`,

		`CREATE INDEX IF NOT EXISTS idx_cold_storage_history_mock_timestamp ON cold_storage_history_mock(timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_cold_storage_history_mock_account_id ON cold_storage_history_mock(account_id);`,

		// Multisig wallet tables
		`CREATE TABLE IF NOT EXISTS multisig_wallets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			uuid TEXT UNIQUE NOT NULL,
			address_type TEXT NOT NULL,
			network TEXT NOT NULL,
			required_signers INTEGER NOT NULL,
			total_signers INTEGER NOT NULL,
			starting_address_index INTEGER NOT NULL DEFAULT 0,
			next_address_index INTEGER NOT NULL DEFAULT 0,
			active BOOLEAN NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_scanned DATETIME
		);`,

		`CREATE INDEX IF NOT EXISTS idx_multisig_wallets_uuid ON multisig_wallets(uuid);`,
		`CREATE INDEX IF NOT EXISTS idx_multisig_wallets_active ON multisig_wallets(active);`,

		`CREATE TABLE IF NOT EXISTS multisig_extended_pubkeys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			wallet_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			xpub TEXT NOT NULL,
			bip32_path TEXT NOT NULL,
			fingerprint TEXT NOT NULL,
			key_index INTEGER NOT NULL,
			FOREIGN KEY(wallet_id) REFERENCES multisig_wallets(id) ON DELETE CASCADE
		);`,

		`CREATE INDEX IF NOT EXISTS idx_multisig_extended_pubkeys_wallet_id ON multisig_extended_pubkeys(wallet_id);`,

		`CREATE TABLE IF NOT EXISTS multisig_addresses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			wallet_id INTEGER NOT NULL,
			address TEXT UNIQUE NOT NULL,
			address_index INTEGER NOT NULL,
			script_type TEXT NOT NULL,
			redeem_script TEXT,
			active BOOLEAN NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(wallet_id) REFERENCES multisig_wallets(id) ON DELETE CASCADE
		);`,

		`CREATE INDEX IF NOT EXISTS idx_multisig_addresses_wallet_id ON multisig_addresses(wallet_id);`,
		`CREATE INDEX IF NOT EXISTS idx_multisig_addresses_address ON multisig_addresses(address);`,

		// Mock multisig tables
		`CREATE TABLE IF NOT EXISTS multisig_wallets_mock (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			uuid TEXT UNIQUE NOT NULL,
			address_type TEXT NOT NULL,
			network TEXT NOT NULL,
			required_signers INTEGER NOT NULL,
			total_signers INTEGER NOT NULL,
			starting_address_index INTEGER NOT NULL DEFAULT 0,
			next_address_index INTEGER NOT NULL DEFAULT 0,
			active BOOLEAN NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_scanned DATETIME
		);`,

		`CREATE INDEX IF NOT EXISTS idx_multisig_wallets_mock_uuid ON multisig_wallets_mock(uuid);`,
		`CREATE INDEX IF NOT EXISTS idx_multisig_wallets_mock_active ON multisig_wallets_mock(active);`,

		`CREATE TABLE IF NOT EXISTS multisig_extended_pubkeys_mock (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			wallet_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			xpub TEXT NOT NULL,
			bip32_path TEXT NOT NULL,
			fingerprint TEXT NOT NULL,
			key_index INTEGER NOT NULL,
			FOREIGN KEY(wallet_id) REFERENCES multisig_wallets_mock(id) ON DELETE CASCADE
		);`,

		`CREATE INDEX IF NOT EXISTS idx_multisig_extended_pubkeys_mock_wallet_id ON multisig_extended_pubkeys_mock(wallet_id);`,

		`CREATE TABLE IF NOT EXISTS multisig_addresses_mock (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			wallet_id INTEGER NOT NULL,
			address TEXT UNIQUE NOT NULL,
			address_index INTEGER NOT NULL,
			script_type TEXT NOT NULL,
			redeem_script TEXT,
			active BOOLEAN NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(wallet_id) REFERENCES multisig_wallets_mock(id) ON DELETE CASCADE
		);`,

		`CREATE INDEX IF NOT EXISTS idx_multisig_addresses_mock_wallet_id ON multisig_addresses_mock(wallet_id);`,
		`CREATE INDEX IF NOT EXISTS idx_multisig_addresses_mock_address ON multisig_addresses_mock(address);`,
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// InsertBalanceSnapshot inserts a new balance snapshot.
// If a snapshot with the same timestamp already exists, it will be replaced due to the use of INSERT OR REPLACE.
func (db *Database) InsertBalanceSnapshot(snapshot *BalanceSnapshot) error {
	tableName := db.getTableName("balance_snapshots")
	query := fmt.Sprintf(`
		INSERT OR REPLACE INTO %s
		(timestamp, lightning_local, lightning_remote, onchain_confirmed, onchain_unconfirmed,
		 tracked_addresses, cold_storage, total_portfolio, total_liquid)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tableName)

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
	tableName := db.getTableName("balance_snapshots")
	query := fmt.Sprintf(`
		SELECT id, timestamp, lightning_local, lightning_remote, onchain_confirmed,
		       onchain_unconfirmed, tracked_addresses, cold_storage, total_portfolio, total_liquid
		FROM %s
		WHERE timestamp BETWEEN ? AND ?
		ORDER BY timestamp ASC
	`, tableName)

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
	tableName := db.getTableName("balance_snapshots")
	query := fmt.Sprintf(`
		SELECT id, timestamp, lightning_local, lightning_remote, onchain_confirmed,
		       onchain_unconfirmed, tracked_addresses, cold_storage, total_portfolio, total_liquid
		FROM %s
		ORDER BY timestamp DESC
		LIMIT 1
	`, tableName)

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

// GetForwardingEventsFees retrieves forwarding fee data aggregated by day within a time range
func (db *Database) GetForwardingEventsFees(from, to time.Time) ([]DailyFeeData, error) {
	tableName := db.getTableName("forwarding_events")
	query := fmt.Sprintf(`
		SELECT
			DATE(timestamp) as date,
			SUM(fee) as total_fee,
			COUNT(*) as forward_count
		FROM %s
		WHERE timestamp BETWEEN ? AND ?
		GROUP BY DATE(timestamp)
		ORDER BY date ASC
	`, tableName)

	rows, err := db.conn.Query(query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeData []DailyFeeData
	for rows.Next() {
		var d DailyFeeData
		err := rows.Scan(&d.Date, &d.TotalFee, &d.ForwardCount)
		if err != nil {
			return nil, err
		}
		feeData = append(feeData, d)
	}

	return feeData, rows.Err()
}

// InsertForwardingEvent inserts a new forwarding event
func (db *Database) InsertForwardingEvent(event *ForwardingEvent) error {
	tableName := db.getTableName("forwarding_events")
	query := fmt.Sprintf(`
		INSERT INTO %s
		(timestamp, channel_in_id, channel_out_id, amount_in, amount_out, fee)
		VALUES (?, ?, ?, ?, ?, ?)
	`, tableName)

	_, err := db.conn.Exec(query,
		event.Timestamp,
		event.ChannelInID,
		event.ChannelOutID,
		event.AmountIn,
		event.AmountOut,
		event.Fee,
	)

	return err
}

// InsertForwardingEventIgnoreDuplicate inserts a new forwarding event, ignoring duplicates
func (db *Database) InsertForwardingEventIgnoreDuplicate(event *ForwardingEvent) error {
	tableName := db.getTableName("forwarding_events")

	// Check if event already exists (same timestamp, channel_in_id, channel_out_id)
	checkQuery := fmt.Sprintf(`
		SELECT id FROM %s
		WHERE timestamp = ? AND channel_in_id = ? AND channel_out_id = ?
		LIMIT 1
	`, tableName)

	var existingID int64
	err := db.conn.QueryRow(checkQuery, event.Timestamp, event.ChannelInID, event.ChannelOutID).Scan(&existingID)
	if err == nil {
		// Event already exists, ignore
		return nil
	}

	// Insert the new event
	insertQuery := fmt.Sprintf(`
		INSERT INTO %s
		(timestamp, channel_in_id, channel_out_id, amount_in, amount_out, fee)
		VALUES (?, ?, ?, ?, ?, ?)
	`, tableName)

	_, err = db.conn.Exec(insertQuery,
		event.Timestamp,
		event.ChannelInID,
		event.ChannelOutID,
		event.AmountIn,
		event.AmountOut,
		event.Fee,
	)

	return err
}

// GetOnchainAddresses retrieves all tracked onchain addresses
func (db *Database) GetOnchainAddresses() ([]OnchainAddress, error) {
	tableName := db.getTableName("onchain_addresses")
	query := fmt.Sprintf(`
		SELECT id, address, label, active
		FROM %s
		ORDER BY id ASC
	`, tableName)

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []OnchainAddress
	for rows.Next() {
		var addr OnchainAddress
		err := rows.Scan(&addr.ID, &addr.Address, &addr.Label, &addr.Active)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, addr)
	}

	return addresses, rows.Err()
}

// GetOnchainAddressByID retrieves a specific onchain address by ID
func (db *Database) GetOnchainAddressByID(id int64) (*OnchainAddress, error) {
	tableName := db.getTableName("onchain_addresses")
	query := fmt.Sprintf(`
		SELECT id, address, label, active
		FROM %s
		WHERE id = ?
	`, tableName)

	var addr OnchainAddress
	err := db.conn.QueryRow(query, id).Scan(&addr.ID, &addr.Address, &addr.Label, &addr.Active)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &addr, nil
}

// InsertOnchainAddress adds a new tracked onchain address
func (db *Database) InsertOnchainAddress(address, label string) (*OnchainAddress, error) {
	tableName := db.getTableName("onchain_addresses")
	query := fmt.Sprintf(`
		INSERT INTO %s (address, label, active)
		VALUES (?, ?, 1)
	`, tableName)

	result, err := db.conn.Exec(query, address, label)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &OnchainAddress{
		ID:      id,
		Address: address,
		Label:   label,
		Active:  true,
	}, nil
}

// DeleteOnchainAddress removes a tracked onchain address
func (db *Database) DeleteOnchainAddress(id int64) error {
	tableName := db.getTableName("onchain_addresses")
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, tableName)

	result, err := db.conn.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetAddressBalanceHistory retrieves balance history for a specific address
func (db *Database) GetAddressBalanceHistory(address string, from, to time.Time) ([]AddressBalance, error) {
	tableName := db.getTableName("address_balances")
	addrTableName := db.getTableName("onchain_addresses")

	query := fmt.Sprintf(`
		SELECT ab.id, ab.address_id, ab.timestamp, ab.balance, ab.tx_count
		FROM %s ab
		JOIN %s oa ON ab.address_id = oa.id
		WHERE oa.address = ? AND ab.timestamp BETWEEN ? AND ?
		ORDER BY ab.timestamp ASC
	`, tableName, addrTableName)

	rows, err := db.conn.Query(query, address, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []AddressBalance
	for rows.Next() {
		var balance AddressBalance
		err := rows.Scan(
			&balance.ID, &balance.AddressID, &balance.Timestamp,
			&balance.Balance, &balance.TxCount,
		)
		if err != nil {
			return nil, err
		}
		balances = append(balances, balance)
	}

	return balances, rows.Err()
}

// InsertAddressBalance adds a new balance record for an address
func (db *Database) InsertAddressBalance(balance *AddressBalance) error {
	tableName := db.getTableName("address_balances")
	query := fmt.Sprintf(`
		INSERT OR REPLACE INTO %s
		(address_id, timestamp, balance, tx_count)
		VALUES (?, ?, ?, ?)
	`, tableName)

	_, err := db.conn.Exec(query,
		balance.AddressID,
		balance.Timestamp,
		balance.Balance,
		balance.TxCount,
	)

	return err
}

// GetColdStorageEntries retrieves all cold storage entries
func (db *Database) GetColdStorageEntries() ([]ColdStorageEntry, error) {
	tableName := db.getTableName("cold_storage_entries")
	query := fmt.Sprintf(`
		SELECT id, name, balance, last_updated, notes
		FROM %s
		ORDER BY id ASC
	`, tableName)

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ColdStorageEntry
	for rows.Next() {
		var entry ColdStorageEntry
		err := rows.Scan(&entry.ID, &entry.Name, &entry.Balance, &entry.LastUpdated, &entry.Notes)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// GetColdStorageEntryByID retrieves a specific cold storage entry by ID
func (db *Database) GetColdStorageEntryByID(id int64) (*ColdStorageEntry, error) {
	tableName := db.getTableName("cold_storage_entries")
	query := fmt.Sprintf(`
		SELECT id, name, balance, last_updated, notes
		FROM %s
		WHERE id = ?
	`, tableName)

	var entry ColdStorageEntry
	err := db.conn.QueryRow(query, id).Scan(&entry.ID, &entry.Name, &entry.Balance, &entry.LastUpdated, &entry.Notes)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// InsertColdStorageEntry adds a new cold storage entry
func (db *Database) InsertColdStorageEntry(name string, balance int64, notes string) (*ColdStorageEntry, error) {
	tableName := db.getTableName("cold_storage_entries")
	query := fmt.Sprintf(`
		INSERT INTO %s (name, balance, last_updated, notes)
		VALUES (?, ?, ?, ?)
	`, tableName)

	now := time.Now()
	result, err := db.conn.Exec(query, name, balance, now, notes)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &ColdStorageEntry{
		ID:          id,
		Name:        name,
		Balance:     balance,
		LastUpdated: now,
		Notes:       notes,
	}, nil
}

// UpdateColdStorageEntry updates an existing cold storage entry and records balance history
func (db *Database) UpdateColdStorageEntry(id int64, name string, balance int64, notes string) (*ColdStorageEntry, error) {
	// Get current entry to track previous balance
	current, err := db.GetColdStorageEntryByID(id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, sql.ErrNoRows
	}

	tableName := db.getTableName("cold_storage_entries")
	query := fmt.Sprintf(`
		UPDATE %s
		SET name = ?, balance = ?, last_updated = ?, notes = ?
		WHERE id = ?
	`, tableName)

	now := time.Now()
	result, err := db.conn.Exec(query, name, balance, now, notes, id)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if rowsAffected == 0 {
		return nil, sql.ErrNoRows
	}

	// Record balance history if balance changed
	if current.Balance != balance {
		historyEntry := &ColdStorageBalanceHistory{
			AccountID:       id,
			Timestamp:       now,
			Balance:         balance,
			PreviousBalance: current.Balance,
			IsVerified:      true, // Assume verified when manually updated
			Notes:           notes,
		}

		if err := db.InsertColdStorageHistory(historyEntry); err != nil {
			// Log error but don't fail the update
			log.Printf("Warning: failed to record balance history: %v\n", err)
		}
	}

	return &ColdStorageEntry{
		ID:          id,
		Name:        name,
		Balance:     balance,
		LastUpdated: now,
		Notes:       notes,
	}, nil
}

// DeleteColdStorageEntry removes a cold storage entry
func (db *Database) DeleteColdStorageEntry(id int64) error {
	tableName := db.getTableName("cold_storage_entries")
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, tableName)

	result, err := db.conn.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// InsertColdStorageHistory records a balance change in cold storage history
func (db *Database) InsertColdStorageHistory(history *ColdStorageBalanceHistory) error {
	tableName := db.getTableName("cold_storage_history")
	query := fmt.Sprintf(`
		INSERT INTO %s (account_id, timestamp, balance, previous_balance, is_verified, notes)
		VALUES (?, ?, ?, ?, ?, ?)
	`, tableName)

	_, err := db.conn.Exec(query,
		history.AccountID,
		history.Timestamp,
		history.Balance,
		history.PreviousBalance,
		history.IsVerified,
		history.Notes,
	)

	return err
}

// GetColdStorageHistory retrieves balance history for a specific account
func (db *Database) GetColdStorageHistory(accountID int64, from, to time.Time) ([]ColdStorageBalanceHistory, error) {
	tableName := db.getTableName("cold_storage_history")
	query := fmt.Sprintf(`
		SELECT id, account_id, timestamp, balance, previous_balance, is_verified, notes
		FROM %s
		WHERE account_id = ? AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp ASC
	`, tableName)

	rows, err := db.conn.Query(query, accountID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []ColdStorageBalanceHistory
	for rows.Next() {
		var entry ColdStorageBalanceHistory
		err := rows.Scan(
			&entry.ID, &entry.AccountID, &entry.Timestamp,
			&entry.Balance, &entry.PreviousBalance, &entry.IsVerified, &entry.Notes,
		)
		if err != nil {
			return nil, err
		}
		history = append(history, entry)
	}

	return history, rows.Err()
}

// GetColdStorageEntriesWithWarnings retrieves all cold storage entries with warning status
func (db *Database) GetColdStorageEntriesWithWarnings() ([]map[string]interface{}, error) {
	tableName := db.getTableName("cold_storage_entries")
	query := fmt.Sprintf(`
		SELECT id, name, balance, last_updated, notes,
		       (julianday('now') - julianday(last_updated)) as days_since_update
		FROM %s
		ORDER BY id ASC
	`, tableName)

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []map[string]interface{}
	for rows.Next() {
		var id int64
		var name, notes string
		var balance int64
		var lastUpdated time.Time
		var daysSinceUpdate float64

		err := rows.Scan(&id, &name, &balance, &lastUpdated, &notes, &daysSinceUpdate)
		if err != nil {
			return nil, err
		}

		entry := map[string]interface{}{
			"id":                int64(id),
			"name":              name,
			"balance":           balance,
			"last_updated":      lastUpdated,
			"notes":             notes,
			"days_since_update": int(daysSinceUpdate),
			"needs_warning":     daysSinceUpdate > 90,
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// MultisigWallet database methods

// InsertMultisigWallet creates a new multisig wallet and its extended public keys
func (db *Database) InsertMultisigWallet(wallet *MultisigWallet) (*MultisigWallet, error) {
	walletTableName := db.getTableName("multisig_wallets")
	keyTableName := db.getTableName("multisig_extended_pubkeys")

	// Start transaction
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert wallet
	walletQuery := fmt.Sprintf(`
		INSERT INTO %s (name, uuid, address_type, network, required_signers, total_signers,
		                starting_address_index, next_address_index, active, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, walletTableName)

	now := time.Now()
	result, err := tx.Exec(walletQuery,
		wallet.Name, wallet.UUID, wallet.AddressType, wallet.Network,
		wallet.RequiredSigners, wallet.TotalSigners, wallet.StartingAddressIndex,
		wallet.NextAddressIndex, wallet.Active, now,
	)
	if err != nil {
		// Check for constraint violations (e.g., duplicate UUID)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, fmt.Errorf("wallet with this UUID already exists: %w", err)
		}
		return nil, fmt.Errorf("failed to insert wallet: %w", err)
	}

	walletID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet ID: %w", err)
	}

	// Insert extended public keys
	keyQuery := fmt.Sprintf(`
		INSERT INTO %s (wallet_id, name, xpub, bip32_path, fingerprint, key_index)
		VALUES (?, ?, ?, ?, ?, ?)
	`, keyTableName)

	for i, key := range wallet.ExtendedPublicKeys {
		_, err = tx.Exec(keyQuery,
			walletID, key.Name, key.XPub, key.BIP32Path, key.Fingerprint, i,
		)
		if err != nil {
			// Check for constraint violations (e.g., duplicate UUID)
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return nil, fmt.Errorf("extended public key %d: duplicate entry detected: %w", i, err)
			}
			return nil, fmt.Errorf("failed to insert extended public key %d: %w", i, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	wallet.ID = walletID
	wallet.CreatedAt = now
	return wallet, nil
}

// GetMultisigWallets retrieves all multisig wallets with their extended public keys
func (db *Database) GetMultisigWallets() ([]MultisigWallet, error) {
	walletTableName := db.getTableName("multisig_wallets")
	keyTableName := db.getTableName("multisig_extended_pubkeys")

	// Get all wallets
	walletQuery := fmt.Sprintf(`
		SELECT id, name, uuid, address_type, network, required_signers, total_signers,
		       starting_address_index, next_address_index, active, created_at, last_scanned
		FROM %s
		WHERE active = 1
		ORDER BY created_at DESC
	`, walletTableName)

	rows, err := db.conn.Query(walletQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []MultisigWallet
	for rows.Next() {
		var wallet MultisigWallet
		var lastScanned sql.NullTime
		err := rows.Scan(
			&wallet.ID, &wallet.Name, &wallet.UUID, &wallet.AddressType, &wallet.Network,
			&wallet.RequiredSigners, &wallet.TotalSigners, &wallet.StartingAddressIndex,
			&wallet.NextAddressIndex, &wallet.Active, &wallet.CreatedAt, &lastScanned,
		)
		if err != nil {
			return nil, err
		}

		if lastScanned.Valid {
			wallet.LastScanned = &lastScanned.Time
		}

		// Get extended public keys for this wallet
		keyQuery := fmt.Sprintf(`
			SELECT id, wallet_id, name, xpub, bip32_path, fingerprint, key_index
			FROM %s
			WHERE wallet_id = ?
			ORDER BY key_index ASC
		`, keyTableName)

		keyRows, err := db.conn.Query(keyQuery, wallet.ID)
		if err != nil {
			return nil, err
		}

		var keys []MultisigExtendedPubKey
		for keyRows.Next() {
			var key MultisigExtendedPubKey
			err := keyRows.Scan(
				&key.ID, &key.WalletID, &key.Name, &key.XPub, &key.BIP32Path,
				&key.Fingerprint, &key.KeyIndex,
			)
			if err != nil {
				keyRows.Close()
				return nil, err
			}
			keys = append(keys, key)
		}
		keyRows.Close()

		wallet.ExtendedPublicKeys = keys
		wallets = append(wallets, wallet)
	}

	return wallets, rows.Err()
}

// GetMultisigWalletByID retrieves a specific multisig wallet by ID
func (db *Database) GetMultisigWalletByID(id int64) (*MultisigWallet, error) {
	walletTableName := db.getTableName("multisig_wallets")
	keyTableName := db.getTableName("multisig_extended_pubkeys")

	// Get wallet
	walletQuery := fmt.Sprintf(`
		SELECT id, name, uuid, address_type, network, required_signers, total_signers,
		       starting_address_index, next_address_index, active, created_at, last_scanned
		FROM %s
		WHERE id = ?
	`, walletTableName)

	var wallet MultisigWallet
	var lastScanned sql.NullTime
	err := db.conn.QueryRow(walletQuery, id).Scan(
		&wallet.ID, &wallet.Name, &wallet.UUID, &wallet.AddressType, &wallet.Network,
		&wallet.RequiredSigners, &wallet.TotalSigners, &wallet.StartingAddressIndex,
		&wallet.NextAddressIndex, &wallet.Active, &wallet.CreatedAt, &lastScanned,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastScanned.Valid {
		wallet.LastScanned = &lastScanned.Time
	}

	// Get extended public keys
	keyQuery := fmt.Sprintf(`
		SELECT id, wallet_id, name, xpub, bip32_path, fingerprint, key_index
		FROM %s
		WHERE wallet_id = ?
		ORDER BY key_index ASC
	`, keyTableName)

	keyRows, err := db.conn.Query(keyQuery, wallet.ID)
	if err != nil {
		return nil, err
	}
	defer keyRows.Close()

	var keys []MultisigExtendedPubKey
	for keyRows.Next() {
		var key MultisigExtendedPubKey
		err := keyRows.Scan(
			&key.ID, &key.WalletID, &key.Name, &key.XPub, &key.BIP32Path,
			&key.Fingerprint, &key.KeyIndex,
		)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	wallet.ExtendedPublicKeys = keys
	return &wallet, nil
}

// GetMultisigWalletByUUID retrieves a specific multisig wallet by UUID
func (db *Database) GetMultisigWalletByUUID(uuid string) (*MultisigWallet, error) {
	walletTableName := db.getTableName("multisig_wallets")

	query := fmt.Sprintf(`
		SELECT id FROM %s WHERE uuid = ?
	`, walletTableName)

	var id int64
	err := db.conn.QueryRow(query, uuid).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return db.GetMultisigWalletByID(id)
}

// DeleteMultisigWallet marks a multisig wallet as inactive and removes associated addresses from tracking
func (db *Database) DeleteMultisigWallet(id int64) error {
	// First get the wallet to construct the label pattern
	wallet, err := db.GetMultisigWalletByID(id)
	if err != nil {
		return fmt.Errorf("failed to get wallet for deletion: %w", err)
	}
	if wallet == nil {
		return sql.ErrNoRows
	}

	// Mark wallet as inactive
	walletTableName := db.getTableName("multisig_wallets")
	query := fmt.Sprintf(`UPDATE %s SET active = 0 WHERE id = ?`, walletTableName)

	result, err := db.conn.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Remove associated addresses from onchain tracking
	onchainTableName := db.getTableName("onchain_addresses")
	labelPattern := fmt.Sprintf("%s [%d/%d] -%%", wallet.Name, wallet.RequiredSigners, wallet.TotalSigners)
	deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE label LIKE ?`, onchainTableName)

	_, err = db.conn.Exec(deleteQuery, labelPattern)
	if err != nil {
		// Log but don't fail - the wallet deletion should succeed even if onchain cleanup fails
		fmt.Printf("Warning: failed to remove multisig addresses from onchain tracking: %v\n", err)
	}

	return nil
}

// InsertMultisigAddress creates a new derived address from a multisig wallet
func (db *Database) InsertMultisigAddress(address *MultisigAddress) (*MultisigAddress, error) {
	tableName := db.getTableName("multisig_addresses")
	query := fmt.Sprintf(`
		INSERT INTO %s (wallet_id, address, address_index, script_type, redeem_script, active, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, tableName)

	now := time.Now()
	result, err := db.conn.Exec(query,
		address.WalletID, address.Address, address.AddressIndex, address.ScriptType,
		address.RedeemScript, address.Active, now,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	address.ID = id
	address.CreatedAt = now
	return address, nil
}

// GetMultisigAddressesByWalletID retrieves all addresses for a specific multisig wallet
func (db *Database) GetMultisigAddressesByWalletID(walletID int64) ([]MultisigAddress, error) {
	tableName := db.getTableName("multisig_addresses")
	query := fmt.Sprintf(`
		SELECT id, wallet_id, address, address_index, script_type, redeem_script, active, created_at
		FROM %s
		WHERE wallet_id = ? AND active = 1
		ORDER BY address_index ASC
	`, tableName)

	rows, err := db.conn.Query(query, walletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []MultisigAddress
	for rows.Next() {
		var addr MultisigAddress
		err := rows.Scan(
			&addr.ID, &addr.WalletID, &addr.Address, &addr.AddressIndex,
			&addr.ScriptType, &addr.RedeemScript, &addr.Active, &addr.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, addr)
	}

	return addresses, rows.Err()
}

// UpdateMultisigWalletLastScanned updates the last scanned timestamp for a wallet
func (db *Database) UpdateMultisigWalletLastScanned(id int64, timestamp time.Time) error {
	tableName := db.getTableName("multisig_wallets")
	query := fmt.Sprintf(`UPDATE %s SET last_scanned = ? WHERE id = ?`, tableName)

	_, err := db.conn.Exec(query, timestamp, id)
	return err
}

// UpdateMultisigWalletNextAddressIndex updates the next address index for a wallet
func (db *Database) UpdateMultisigWalletNextAddressIndex(id int64, nextIndex int) error {
	tableName := db.getTableName("multisig_wallets")
	query := fmt.Sprintf(`UPDATE %s SET next_address_index = ? WHERE id = ?`, tableName)

	_, err := db.conn.Exec(query, nextIndex, id)
	return err
}

// InsertAddressBalanceHistory inserts a balance record for an address at a specific timestamp
func (db *Database) InsertAddressBalanceHistory(address string, balance int64, timestamp time.Time) error {
	// First get the address ID
	addressTableName := db.getTableName("onchain_addresses")
	var addressID int64
	err := db.conn.QueryRow(fmt.Sprintf(`SELECT id FROM %s WHERE address = ?`, addressTableName), address).Scan(&addressID)
	if err != nil {
		return fmt.Errorf("address not found in tracking: %w", err)
	}

	// Insert the balance history
	balanceTableName := db.getTableName("address_balances")
	query := fmt.Sprintf(`
		INSERT OR REPLACE INTO %s (address_id, timestamp, balance, tx_count)
		VALUES (?, ?, ?, 0)
	`, balanceTableName)

	_, err = db.conn.Exec(query, addressID, timestamp, balance)
	return err
}
