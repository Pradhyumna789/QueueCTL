package job

import (
	"encoding/json"
	"fmt"
	"time"
)

// State represents the state of a job
type State string

const (
	StatePending   State = "pending"
	StateProcessing State = "processing"
	StateCompleted  State = "completed"
	StateFailed     State = "failed"
	StateDead       State = "dead"
)

// Job represents a background job
type Job struct {
	ID          string    `json:"id"`
	Command     string    `json:"command"`
	State       State     `json:"state"`
	Attempts    int       `json:"attempts"`
	MaxRetries  int       `json:"max_retries"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	NextRetryAt *time.Time `json:"next_retry_at,omitempty"`
}

// Validate validates a job
func (j *Job) Validate() error {
	if j.ID == "" {
		return fmt.Errorf("job ID is required")
	}
	if j.Command == "" {
		return fmt.Errorf("job command is required")
	}
	if j.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative")
	}
	return nil
}

// FromJSON creates a Job from JSON string
func FromJSON(jsonStr string) (*Job, error) {
	var j Job
	if err := json.Unmarshal([]byte(jsonStr), &j); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Set defaults
	if j.State == "" {
		j.State = StatePending
	}
	if j.MaxRetries == 0 {
		j.MaxRetries = 3
	}
	now := time.Now()
	if j.CreatedAt.IsZero() {
		j.CreatedAt = now
	}
	if j.UpdatedAt.IsZero() {
		j.UpdatedAt = now
	}

	if err := j.Validate(); err != nil {
		return nil, err
	}

	return &j, nil
}

// ToJSON converts a Job to JSON string
func (j *Job) ToJSON() (string, error) {
	data, err := json.Marshal(j)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job: %w", err)
	}
	return string(data), nil
}

