package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"queuectl/internal/db"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the queue database",
	Long:  `Reset the queue database by deleting all jobs. This will remove the queuectl.db file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("❌ Failed to get home directory: %w", err)
		}

		dbPath := filepath.Join(homeDir, ".queuectl", "queuectl.db")
		
		// Check if database exists
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Println("ℹ️  Database does not exist. Nothing to reset.")
			return nil
		}

		// Close database connection if open
		if err := db.Close(); err != nil {
			// Continue even if close fails
			fmt.Printf("⚠️  Warning: Failed to close database connection: %v\n", err)
		}
		
		// Delete database file
		if err := os.Remove(dbPath); err != nil {
			return fmt.Errorf("❌ Failed to delete database: %w\n\n💡 Make sure no workers are running: queuectl worker stop", err)
		}

		fmt.Printf("✅ Database reset successfully. All jobs have been removed.\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}

