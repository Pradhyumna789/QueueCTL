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
			return fmt.Errorf("failed to parse job: %w", err)
		}

		if err := job.Create(j); err != nil {
			return fmt.Errorf("failed to enqueue job: %w", err)
		}

		fmt.Printf("Job '%s' enqueued successfully\n", j.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(enqueueCmd)
}

