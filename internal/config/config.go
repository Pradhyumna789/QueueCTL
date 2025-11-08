package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	KeyMaxRetries  = "max-retries"
	KeyBackoffBase = "backoff-base"
	KeyWorkerCount = "worker-count"
)

type Config struct {
	MaxRetries  int     `json:"max-retries"`
	BackoffBase float64 `json:"backoff-base"`
	WorkerCount int     `json:"worker-count"`
}

var defaultConfig = Config{
	MaxRetries:  3,
	BackoffBase: 2.0,
	WorkerCount: 1,
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	queuectlDir := filepath.Join(homeDir, ".queuectl")
	if err := os.MkdirAll(queuectlDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .queuectl directory: %w", err)
	}

	return filepath.Join(queuectlDir, "config.json"), nil
}

// Load loads the configuration from file
func Load() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &defaultConfig, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Merge with defaults for missing values
	if config.MaxRetries == 0 {
		config.MaxRetries = defaultConfig.MaxRetries
	}
	if config.BackoffBase == 0 {
		config.BackoffBase = defaultConfig.BackoffBase
	}
	if config.WorkerCount == 0 {
		config.WorkerCount = defaultConfig.WorkerCount
	}

	return &config, nil
}

// Save saves the configuration to file
func Save(config *Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Get returns a configuration value by key
func Get(key string) (string, error) {
	config, err := Load()
	if err != nil {
		return "", err
	}

	switch key {
	case KeyMaxRetries:
		return fmt.Sprintf("%d", config.MaxRetries), nil
	case KeyBackoffBase:
		return fmt.Sprintf("%.2f", config.BackoffBase), nil
	case KeyWorkerCount:
		return fmt.Sprintf("%d", config.WorkerCount), nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// Set sets a configuration value by key
func Set(key, value string) error {
	config, err := Load()
	if err != nil {
		return err
	}

	switch key {
	case KeyMaxRetries:
		var maxRetries int
		if _, err := fmt.Sscanf(value, "%d", &maxRetries); err != nil {
			return fmt.Errorf("invalid value for max-retries: %s", value)
		}
		if maxRetries < 0 {
			return fmt.Errorf("max-retries must be non-negative")
		}
		config.MaxRetries = maxRetries
	case KeyBackoffBase:
		var backoffBase float64
		if _, err := fmt.Sscanf(value, "%f", &backoffBase); err != nil {
			return fmt.Errorf("invalid value for backoff-base: %s", value)
		}
		if backoffBase <= 0 {
			return fmt.Errorf("backoff-base must be positive")
		}
		config.BackoffBase = backoffBase
	case KeyWorkerCount:
		var workerCount int
		if _, err := fmt.Sscanf(value, "%d", &workerCount); err != nil {
			return fmt.Errorf("invalid value for worker-count: %s", value)
		}
		if workerCount < 1 {
			return fmt.Errorf("worker-count must be at least 1")
		}
		config.WorkerCount = workerCount
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return Save(config)
}

