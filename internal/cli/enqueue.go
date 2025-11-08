package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"queuectl/internal/job"
)

var enqueueCmd = &cobra.Command{
	Use:   "enqueue [json]",
	Short: "Add a new job to the queue",
	Long:  `Enqueue a new job by providing a JSON string with job details.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonStr := args[0]

		j, err := job.FromJSON(jsonStr)
		if err != nil {
			return fmt.Errorf("❌ Invalid JSON format: %w\n\n💡 Example: {\"id\":\"job1\",\"command\":\"echo hello\"}", err)
		}

		if err := job.Create(j); err != nil {
			// Check if it's a duplicate ID error
			if err.Error() == fmt.Sprintf("job with ID '%s' already exists", j.ID) {
				existingJob, getErr := job.GetByID(j.ID)
				if getErr == nil && existingJob != nil {
					return fmt.Errorf("❌ Job with ID '%s' already exists (state: %s)\n\n💡 Solutions:\n   • Use a different job ID\n   • Check existing jobs: queuectl list\n   • Clear database: rm ~/.queuectl/queuectl.db", j.ID, existingJob.State)
				}
				return fmt.Errorf("❌ Job with ID '%s' already exists\n\n💡 Use a different job ID or clear the database: rm ~/.queuectl/queuectl.db", j.ID)
			}
			return fmt.Errorf("❌ Failed to enqueue job: %w", err)
		}

		fmt.Printf("✅ Job '%s' enqueued successfully\n", j.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(enqueueCmd)
}

