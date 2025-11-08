package db

import (
	"fmt"
)

// createSchema creates the database tables
func createSchema() error {
	// Create jobs table
	jobsTableSQL := `
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		command TEXT NOT NULL,
		state TEXT NOT NULL DEFAULT 'pending',
		attempts INTEGER NOT NULL DEFAULT 0,
		max_retries INTEGER NOT NULL DEFAULT 3,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		next_retry_at TEXT
	);`

	if _, err := DB.Exec(jobsTableSQL); err != nil {
		return fmt.Errorf("failed to create jobs table: %w", err)
	}

	// Create index on state for faster queries
	indexSQL := `
	CREATE INDEX IF NOT EXISTS idx_jobs_state ON jobs(state);
	CREATE INDEX IF NOT EXISTS idx_jobs_next_retry_at ON jobs(next_retry_at);`

	if _, err := DB.Exec(indexSQL); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

