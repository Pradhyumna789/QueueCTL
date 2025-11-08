package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"queuectl/internal/job"
)

var dlqListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all jobs in the Dead Letter Queue",
	Long:  `Display all jobs that have been moved to the Dead Letter Queue (permanently failed).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jobs, err := job.ListByState(job.StateDead)
		if err != nil {
			return fmt.Errorf("failed to list DLQ jobs: %w", err)
		}

		if len(jobs) == 0 {
			fmt.Println("ℹ️  No jobs in Dead Letter Queue")
			return nil
		}

		// Output as JSON
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(jobs); err != nil {
			return fmt.Errorf("failed to encode jobs: %w", err)
		}

		return nil
	},
}

var dlqRetryCmd = &cobra.Command{
	Use:   "retry [job-id]",
	Short: "Retry a job from the Dead Letter Queue",
	Long:  `Move a job from the Dead Letter Queue back to pending state for retry.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobID := args[0]

		if err := job.RetryDeadJob(jobID); err != nil {
			return fmt.Errorf("❌ Failed to retry job: %w\n\n💡 Make sure the job ID exists in DLQ: queuectl dlq list", err)
		}

		fmt.Printf("✅ Job '%s' moved back to pending state\n", jobID)
		return nil
	},
}

var dlqCmd = &cobra.Command{
	Use:   "dlq",
	Short: "Manage Dead Letter Queue",
	Long:  `Commands for managing the Dead Letter Queue (permanently failed jobs).`,
}

func init() {
	dlqCmd.AddCommand(dlqListCmd)
	dlqCmd.AddCommand(dlqRetryCmd)
	rootCmd.AddCommand(dlqCmd)
}

