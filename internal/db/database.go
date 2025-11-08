package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

// Init initializes the SQLite database connection and creates the schema
func Init() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	queuectlDir := filepath.Join(homeDir, ".queuectl")
	if err := os.MkdirAll(queuectlDir, 0755); err != nil {
		return fmt.Errorf("failed to create .queuectl directory: %w", err)
	}

	dbPath := filepath.Join(queuectlDir, "queuectl.db")
	
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys, set WAL mode for better concurrency, and set busy timeout
	// busy_timeout sets how long SQLite will wait for a lock (in milliseconds)
	// 5000ms = 5 seconds
	if _, err := db.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL; PRAGMA busy_timeout = 5000;"); err != nil {
		return fmt.Errorf("failed to set database pragmas: %w", err)
	}

	DB = db

	// Create schema
	if err := createSchema(); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// GetDB returns the database connection
func GetDB() *sql.DB {
	return DB
}

