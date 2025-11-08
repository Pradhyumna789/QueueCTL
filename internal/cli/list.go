package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"queuectl/internal/job"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs by state",
	Long:  `List all jobs filtered by state. If no state is specified, lists all jobs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		stateFlag, err := cmd.Flags().GetString("state")
		if err != nil {
			return fmt.Errorf("failed to get state flag: %w", err)
		}

		var jobs []*job.Job
		if stateFlag != "" {
			// Validate state
			validState := job.State(stateFlag)
			if validState != job.StatePending &&
				validState != job.StateProcessing &&
				validState != job.StateCompleted &&
				validState != job.StateFailed &&
				validState != job.StateDead {
				return fmt.Errorf("❌ Invalid state: '%s'\n\n💡 Valid states: pending, processing, completed, failed, dead", stateFlag)
			}

			jobs, err = job.ListByState(validState)
			if err != nil {
				return fmt.Errorf("failed to list jobs: %w", err)
			}
		} else {
			// List all states
			allStates := []job.State{
				job.StatePending,
				job.StateProcessing,
				job.StateCompleted,
				job.StateFailed,
				job.StateDead,
			}

			for _, state := range allStates {
				stateJobs, err := job.ListByState(state)
				if err != nil {
					return fmt.Errorf("failed to list jobs for state %s: %w", state, err)
				}
				jobs = append(jobs, stateJobs...)
			}
		}

		// Output as JSON (always output array, even if empty)
		if jobs == nil {
			jobs = []*job.Job{}
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(jobs); err != nil {
			return fmt.Errorf("❌ Failed to encode jobs: %w", err)
		}

		return nil
	},
}

func init() {
	listCmd.Flags().StringP("state", "s", "", "Filter jobs by state (pending, processing, completed, failed, dead)")
	rootCmd.AddCommand(listCmd)
}

