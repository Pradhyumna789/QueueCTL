package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"queuectl/internal/config"
	"queuectl/internal/worker"
)

var workerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start worker processes",
	Long:  `Start one or more worker processes to process jobs from the queue.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		count, err := cmd.Flags().GetInt("count")
		if err != nil {
			return fmt.Errorf("failed to get count flag: %w", err)
		}

		if count < 1 {
			return fmt.Errorf("worker count must be at least 1")
		}

		if err := worker.StartPool(count); err != nil {
			return fmt.Errorf("failed to start workers: %w", err)
		}

		fmt.Printf("Started %d worker(s)\n", count)

		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Wait for interrupt signal
		<-sigChan
		fmt.Println("\nShutting down workers...")

		if err := worker.StopPool(); err != nil {
			return fmt.Errorf("failed to stop workers: %w", err)
		}

		fmt.Println("Workers stopped")
		return nil
	},
}

var workerStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop running workers",
	Long:  `Stop all running worker processes gracefully.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := worker.StopPool(); err != nil {
			return fmt.Errorf("failed to stop workers: %w", err)
		}

		fmt.Println("Workers stopped")
		return nil
	},
}

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Manage worker processes",
	Long:  `Commands for managing worker processes.`,
}

func init() {
	workerStartCmd.Flags().IntP("count", "c", 1, "Number of workers to start")
	
	// Set default from config
	cfg, err := config.Load()
	if err == nil {
		workerStartCmd.Flags().SetDefault("count", fmt.Sprintf("%d", cfg.WorkerCount))
	}

	workerCmd.AddCommand(workerStartCmd)
	workerCmd.AddCommand(workerStopCmd)
	rootCmd.AddCommand(workerCmd)
}

