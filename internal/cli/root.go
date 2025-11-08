package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"queuectl/internal/db"
)

var rootCmd = &cobra.Command{
	Use:   "queuectl",
	Short: "A CLI-based background job queue system",
	Long:  `QueueCTL is a CLI tool for managing background jobs with worker processes, retries, and Dead Letter Queue.`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Initialize database on startup
	cobra.OnInitialize(initDB)
}

func initDB() {
	if err := db.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
}

// GetRootCmd returns the root command (for testing)
func GetRootCmd() *cobra.Command {
	return rootCmd
}

