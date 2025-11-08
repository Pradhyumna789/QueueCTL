package job

import (
	"database/sql"
	"fmt"
	"time"

	"queuectl/internal/db"
)

// Create inserts a new job into the database
func Create(j *Job) error {
	query := `
		INSERT INTO jobs (id, command, state, attempts, max_retries, created_at, updated_at, next_retry_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.GetDB().Exec(
		query,
		j.ID,
		j.Command,
		string(j.State),
		j.Attempts,
		j.MaxRetries,
		j.CreatedAt.Format(time.RFC3339),
		j.UpdatedAt.Format(time.RFC3339),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}
	return nil
}

// GetByID retrieves a job by ID
func GetByID(id string) (*Job, error) {
	query := `
		SELECT id, command, state, attempts, max_retries, created_at, updated_at, next_retry_at
		FROM jobs
		WHERE id = ?`

	var j Job
	var createdAtStr, updatedAtStr string
	var nextRetryAtStr sql.NullString

	err := db.GetDB().QueryRow(query, id).Scan(
		&j.ID,
		&j.Command,
		&j.State,
		&j.Attempts,
		&j.MaxRetries,
		&createdAtStr,
		&updatedAtStr,
		&nextRetryAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	// Parse timestamps
	j.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	j.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	if nextRetryAtStr.Valid {
		nextRetryAt, err := time.Parse(time.RFC3339, nextRetryAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse next_retry_at: %w", err)
		}
		j.NextRetryAt = &nextRetryAt
	}

	return &j, nil
}

// GetNextPendingJob retrieves the next pending job that's ready for processing
// Uses SELECT FOR UPDATE to lock the row
func GetNextPendingJob(tx *sql.Tx) (*Job, error) {
	now := time.Now().Format(time.RFC3339)
	query := `
		SELECT id, command, state, attempts, max_retries, created_at, updated_at, next_retry_at
		FROM jobs
		WHERE state = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE`

	var j Job
	var createdAtStr, updatedAtStr string
	var nextRetryAtStr sql.NullString

	err := tx.QueryRow(query, string(StatePending), now).Scan(
		&j.ID,
		&j.Command,
		&j.State,
		&j.Attempts,
		&j.MaxRetries,
		&createdAtStr,
		&updatedAtStr,
		&nextRetryAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No pending jobs
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get next pending job: %w", err)
	}

	// Parse timestamps
	j.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	j.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	if nextRetryAtStr.Valid {
		nextRetryAt, err := time.Parse(time.RFC3339, nextRetryAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse next_retry_at: %w", err)
		}
		j.NextRetryAt = &nextRetryAt
	}

	return &j, nil
}

// UpdateState updates the job state and other fields
func UpdateState(tx *sql.Tx, id string, state State, attempts int, nextRetryAt *time.Time) error {
	now := time.Now()
	query := `
		UPDATE jobs
		SET state = ?, attempts = ?, updated_at = ?, next_retry_at = ?
		WHERE id = ?`

	var nextRetryAtStr interface{}
	if nextRetryAt != nil {
		nextRetryAtStr = nextRetryAt.Format(time.RFC3339)
	} else {
		nextRetryAtStr = nil
	}

	_, err := tx.Exec(
		query,
		string(state),
		attempts,
		now.Format(time.RFC3339),
		nextRetryAtStr,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update job state: %w", err)
	}
	return nil
}

// ListByState retrieves all jobs with a specific state
func ListByState(state State) ([]*Job, error) {
	query := `
		SELECT id, command, state, attempts, max_retries, created_at, updated_at, next_retry_at
		FROM jobs
		WHERE state = ?
		ORDER BY created_at DESC`

	rows, err := db.GetDB().Query(query, string(state))
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		var j Job
		var createdAtStr, updatedAtStr string
		var nextRetryAtStr sql.NullString

		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}

		// Parse timestamps
		j.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}

		j.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at: %w", err)
		}

		if nextRetryAtStr.Valid {
			nextRetryAt, err := time.Parse(time.RFC3339, nextRetryAtStr.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse next_retry_at: %w", err)
			}
			j.NextRetryAt = &nextRetryAt
		}

		jobs = append(jobs, &j)
	}

	return jobs, nil
}

// GetStats returns counts of jobs by state
func GetStats() (map[State]int, error) {
	query := `
		SELECT state, COUNT(*) as count
		FROM jobs
		GROUP BY state`

	rows, err := db.GetDB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[State]int)
	for rows.Next() {
		var stateStr string
		var count int
		if err := rows.Scan(&stateStr, &count); err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		stats[State(stateStr)] = count
	}

	return stats, nil
}

// RetryDeadJob moves a dead job back to pending state
func RetryDeadJob(id string) error {
	query := `
		UPDATE jobs
		SET state = ?, attempts = 0, next_retry_at = NULL, updated_at = ?
		WHERE id = ? AND state = ?`

	now := time.Now()
	result, err := db.GetDB().Exec(
		query,
		string(StatePending),
		now.Format(time.RFC3339),
		id,
		string(StateDead),
	)
	if err != nil {
		return fmt.Errorf("failed to retry dead job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job not found or not in dead state: %s", id)
	}

	return nil
}

