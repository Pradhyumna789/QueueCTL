package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"queuectl/internal/job"
	"queuectl/internal/worker"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show summary of all job states and active workers",
	Long:  `Display a summary of job counts by state and the number of active workers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		stats, err := job.GetStats()
		if err != nil {
			return fmt.Errorf("failed to get job stats: %w", err)
		}

		fmt.Println("Job Queue Status")
		fmt.Println("================")
		fmt.Printf("Pending:   %d\n", stats[job.StatePending])
		fmt.Printf("Processing: %d\n", stats[job.StateProcessing])
		fmt.Printf("Completed: %d\n", stats[job.StateCompleted])
		fmt.Printf("Failed:    %d\n", stats[job.StateFailed])
		fmt.Printf("Dead (DLQ): %d\n", stats[job.StateDead])
		fmt.Println()

		pool := worker.GetPool()
		if pool != nil && pool.IsRunning() {
			fmt.Printf("Active Workers: %d\n", pool.GetWorkerCount())
		} else {
			fmt.Println("Active Workers: 0")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

