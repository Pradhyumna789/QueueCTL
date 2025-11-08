package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"queuectl/internal/config"
)

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Long:  `Get the value of a configuration key.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		value, err := config.Get(key)
		if err != nil {
			return fmt.Errorf("❌ Unknown config key: '%s'\n\n💡 Valid keys: max-retries, backoff-base, worker-count", key)
		}

		fmt.Println(value)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Long:  `Set the value of a configuration key.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		if err := config.Set(key, value); err != nil {
			// Check if it's an unknown key error
			if err.Error() == fmt.Sprintf("unknown config key: %s", key) {
				return fmt.Errorf("❌ Unknown config key: '%s'\n\n💡 Valid keys: max-retries, backoff-base, worker-count", key)
			}
			return fmt.Errorf("❌ Failed to set config: %w", err)
		}

		fmt.Printf("Configuration '%s' set to '%s'\n", key, value)
		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Commands for managing configuration settings.`,
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}

