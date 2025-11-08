package job

import (
	"fmt"
	"os/exec"
	"time"
)

// ExecuteResult represents the result of executing a job
type ExecuteResult struct {
	Success bool
	Error   error
}

// Execute executes a job command and returns the result
func Execute(j *Job) ExecuteResult {
	cmd := exec.Command("sh", "-c", j.Command)
	
	if err := cmd.Run(); err != nil {
		return ExecuteResult{
			Success: false,
			Error:   fmt.Errorf("command failed: %w", err),
		}
	}

	return ExecuteResult{
		Success: true,
		Error:   nil,
	}
}

// CalculateNextRetry calculates the next retry time using exponential backoff
func CalculateNextRetry(attempts int, backoffBase float64) time.Time {
	// delay = base^attempts seconds
	delaySeconds := int64(1)
	for i := 0; i < attempts; i++ {
		delaySeconds = int64(float64(delaySeconds) * backoffBase)
	}
	return time.Now().Add(time.Duration(delaySeconds) * time.Second)
}

