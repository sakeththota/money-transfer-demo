package balances

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var seedData = map[string]float64{
	"Checking Account": 12458.32,
	"Savings Account":  45231.89,
	"Justine Morris":   1500.00,
	"Raul Ruidiaz":     2200.00,
	"Ian Wu":           875.50,
	"Emma Stockton":    3100.00,
}

func Open(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// WAL mode for concurrent access from API + worker processes
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS accounts (
			name    TEXT PRIMARY KEY,
			balance REAL NOT NULL DEFAULT 0
		)
	`); err != nil {
		return nil, err
	}

	// Seed accounts if they don't exist
	for name, balance := range seedData {
		if _, err := db.Exec(
			"INSERT OR IGNORE INTO accounts (name, balance) VALUES (?, ?)",
			name, balance,
		); err != nil {
			return nil, err
		}
	}

	log.Printf("Balances DB ready at %s", dbPath)
	return db, nil
}
