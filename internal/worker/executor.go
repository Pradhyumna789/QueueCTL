package worker

import (
	"fmt"

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

	// Start transaction for atomic state update
	tx, err := db.GetDB().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update job to processing state
	if err := job.UpdateState(tx, j.ID, job.StateProcessing, j.Attempts, nil); err != nil {
		return fmt.Errorf("failed to update job to processing: %w", err)
	}

	// Commit the state change
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Execute the job
	result := job.Execute(j)

	// Start new transaction for result update
	tx, err = db.GetDB().Begin()
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
	tx, err := db.GetDB().Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get next pending job (this also handles failed jobs ready for retry)
	j, err := job.GetNextPendingJob(tx)
	if err != nil {
		return nil, err
	}

	if j == nil {
		tx.Rollback()
		return nil, nil // No jobs available
	}

	// Commit the transaction to release the lock
	// The job is already locked and updated to pending if it was failed
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return j, nil
}

