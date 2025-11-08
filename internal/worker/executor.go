package worker

import (
	"database/sql"
	"fmt"
	"time"

	"queuectl/internal/config"
	"queuectl/internal/db"
	"queuectl/internal/job"
)

// ExecuteJob executes a job with retry logic and state management
func ExecuteJob(j *job.Job) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Job is already in processing state (set by GetNextJob)
	// No need to update it again

	// Execute the job
	result := job.Execute(j)

	// Start transaction for result update
	tx, err := db.GetDB().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if result.Success {
		// Job succeeded
		if err := job.UpdateState(tx, j.ID, job.StateCompleted, j.Attempts, nil); err != nil {
			return fmt.Errorf("failed to update job to completed: %w", err)
		}
	} else {
		// Job failed - increment attempts
		newAttempts := j.Attempts + 1

		if newAttempts > j.MaxRetries {
			// Move to DLQ
			if err := job.UpdateState(tx, j.ID, job.StateDead, newAttempts, nil); err != nil {
				return fmt.Errorf("failed to update job to dead: %w", err)
			}
		} else {
			// Schedule retry with exponential backoff
			nextRetryAt := job.CalculateNextRetry(newAttempts, cfg.BackoffBase)
			// Set state to failed with next_retry_at - GetNextPendingJob will pick it up when ready
			if err := job.UpdateState(tx, j.ID, job.StateFailed, newAttempts, &nextRetryAt); err != nil {
				return fmt.Errorf("failed to update job for retry: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetNextJob retrieves and locks the next available job
func GetNextJob() (*job.Job, error) {
	// For SQLite, we use an atomic UPDATE to claim the job
	// This prevents race conditions by updating the state atomically
	now := time.Now().Format(time.RFC3339)
	
	// Retry logic for SQLITE_BUSY errors
	maxRetries := 5
	retryDelay := 10 * time.Millisecond
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		tx, err := db.GetDB().Begin()
		if err != nil {
			// Check if it's a SQLITE_BUSY error
			errStr := err.Error()
			if errStr == "database is locked" || 
			   errStr == "database is locked (5)" ||
			   errStr == "SQLITE_BUSY" ||
			   errStr == "database is locked (5) (SQLITE_BUSY)" {
				if attempt < maxRetries-1 {
					time.Sleep(retryDelay)
					retryDelay *= 2 // Exponential backoff
					continue
				}
			}
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
		
		// Step 1: Get the ID of the next job to claim
		var jobID string
		selectQuery := `
			SELECT id FROM jobs
			WHERE (state = ? AND (next_retry_at IS NULL OR next_retry_at <= ?))
			   OR (state = ? AND next_retry_at IS NOT NULL AND next_retry_at <= ?)
			ORDER BY created_at ASC
			LIMIT 1`
		
		err = tx.QueryRow(selectQuery, 
			string(job.StatePending),
			now,
			string(job.StateFailed),
			now,
		).Scan(&jobID)
		
		if err == sql.ErrNoRows {
			tx.Rollback()
			return nil, nil // No jobs available
		}
		if err != nil {
			tx.Rollback()
			// Check if it's a SQLITE_BUSY error
			errStr := err.Error()
			if errStr == "database is locked" || 
			   errStr == "database is locked (5)" ||
			   errStr == "SQLITE_BUSY" ||
			   errStr == "database is locked (5) (SQLITE_BUSY)" {
				if attempt < maxRetries-1 {
					time.Sleep(retryDelay)
					retryDelay *= 2
					continue
				}
			}
			return nil, fmt.Errorf("failed to select next job: %w", err)
		}
		
		// Step 2: Atomically update the job to processing state
		// This prevents other workers from picking it up
		updateQuery := `
			UPDATE jobs
			SET state = ?, updated_at = ?, next_retry_at = NULL
			WHERE id = ? AND (state = ? OR state = ?)`
		
		result, err := tx.Exec(updateQuery,
			string(job.StateProcessing),
			time.Now().Format(time.RFC3339),
			jobID,
			string(job.StatePending),
			string(job.StateFailed),
		)
		if err != nil {
			tx.Rollback()
			// Check if it's a SQLITE_BUSY error
			errStr := err.Error()
			if errStr == "database is locked" || 
			   errStr == "database is locked (5)" ||
			   errStr == "SQLITE_BUSY" ||
			   errStr == "database is locked (5) (SQLITE_BUSY)" {
				if attempt < maxRetries-1 {
					time.Sleep(retryDelay)
					retryDelay *= 2
					continue
				}
			}
			return nil, fmt.Errorf("failed to claim job: %w", err)
		}
		
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to get rows affected: %w", err)
		}
		
		if rowsAffected == 0 {
			// Another worker claimed it between SELECT and UPDATE
			tx.Rollback()
			return nil, nil
		}
		
		// Step 3: Select the job we just claimed
		selectJobQuery := `
			SELECT id, command, state, attempts, max_retries, created_at, updated_at, next_retry_at
			FROM jobs
			WHERE id = ?`
		
		var j job.Job
		var createdAtStr, updatedAtStr string
		var nextRetryAtStr sql.NullString
		
		err = tx.QueryRow(selectJobQuery, jobID).Scan(
			&j.ID,
			&j.Command,
			&j.State,
			&j.Attempts,
			&j.MaxRetries,
			&createdAtStr,
			&updatedAtStr,
			&nextRetryAtStr,
		)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to get claimed job: %w", err)
		}
		
		// Parse timestamps
		j.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}
		
		j.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to parse updated_at: %w", err)
		}
		
		if nextRetryAtStr.Valid {
			nextRetryAt, err := time.Parse(time.RFC3339, nextRetryAtStr.String)
			if err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to parse next_retry_at: %w", err)
			}
			j.NextRetryAt = &nextRetryAt
		}
		
		if err := tx.Commit(); err != nil {
			// Check if it's a SQLITE_BUSY error
			errStr := err.Error()
			if errStr == "database is locked" || 
			   errStr == "database is locked (5)" ||
			   errStr == "SQLITE_BUSY" ||
			   errStr == "database is locked (5) (SQLITE_BUSY)" {
				if attempt < maxRetries-1 {
					time.Sleep(retryDelay)
					retryDelay *= 2
					continue
				}
			}
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}
		
		return &j, nil
	}
	
	return nil, fmt.Errorf("failed to get next job after %d retries: database is locked", maxRetries)
}

