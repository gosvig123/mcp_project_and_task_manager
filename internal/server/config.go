package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// ServerConfig holds configuration for the task manager server
type ServerConfig struct {
	AutoEvaluation AutoEvaluationConfig `json:"auto_evaluation"`
	TasksDir       string               `json:"tasks_dir"`
	LogLevel       string               `json:"log_level"`
}

// LoadServerConfig loads configuration from environment variables and config file
func LoadServerConfig() (ServerConfig, error) {
	config := ServerConfig{
		AutoEvaluation: DefaultAutoEvaluationConfig(),
		LogLevel:       "info",
	}

	// Load from environment variables
	config.loadFromEnv()

	// Try to load from config file
	if err := config.loadFromFile(); err != nil {
		// Config file is optional, just log the error
		fmt.Printf("Config file not found or invalid, using defaults: %v\n", err)
	}

	return config, nil
}

// loadFromEnv loads configuration from environment variables
func (c *ServerConfig) loadFromEnv() {
	// Tasks directory
	if tasksDir := os.Getenv("TASKS_DIR"); tasksDir != "" {
		c.TasksDir = tasksDir
	}

	// Log level
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		c.LogLevel = logLevel
	}

	// Auto-evaluation settings
	if enabled := os.Getenv("AUTO_EVAL_ENABLED"); enabled != "" {
		if val, err := strconv.ParseBool(enabled); err == nil {
			c.AutoEvaluation.Enabled = val
		}
	}

	if timeout := os.Getenv("AUTO_EVAL_CACHE_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			c.AutoEvaluation.CacheTimeout = duration
		}
	}

	if maxConcurrent := os.Getenv("AUTO_EVAL_MAX_CONCURRENT"); maxConcurrent != "" {
		if val, err := strconv.Atoi(maxConcurrent); err == nil {
			c.AutoEvaluation.MaxConcurrent = val
		}
	}

	if skipReadOnly := os.Getenv("AUTO_EVAL_SKIP_READ_ONLY"); skipReadOnly != "" {
		if val, err := strconv.ParseBool(skipReadOnly); err == nil {
			c.AutoEvaluation.SkipReadOnlyTools = val
		}
	}

	if verbose := os.Getenv("AUTO_EVAL_VERBOSE"); verbose != "" {
		if val, err := strconv.ParseBool(verbose); err == nil {
			c.AutoEvaluation.VerboseLogging = val
		}
	}
}

// loadFromFile loads configuration from a JSON config file
func (c *ServerConfig) loadFromFile() error {
	configPaths := []string{
		"config.json",
		"task-manager-config.json",
		filepath.Join(os.Getenv("HOME"), ".task-manager-config.json"),
	}

	for _, path := range configPaths {
		if data, err := os.ReadFile(path); err == nil {
			var fileConfig ServerConfig
			if err := json.Unmarshal(data, &fileConfig); err == nil {
				// Merge file config with current config
				c.mergeConfig(fileConfig)
				return nil
			}
		}
	}

	return fmt.Errorf("no valid config file found")
}

// mergeConfig merges another config into this one (file config takes precedence)
func (c *ServerConfig) mergeConfig(other ServerConfig) {
	if other.TasksDir != "" {
		c.TasksDir = other.TasksDir
	}
	if other.LogLevel != "" {
		c.LogLevel = other.LogLevel
	}

	// Merge auto-evaluation config
	if other.AutoEvaluation.CacheTimeout != 0 {
		c.AutoEvaluation.CacheTimeout = other.AutoEvaluation.CacheTimeout
	}
	if other.AutoEvaluation.MaxConcurrent != 0 {
		c.AutoEvaluation.MaxConcurrent = other.AutoEvaluation.MaxConcurrent
	}
	// Note: boolean fields are merged as-is since false is a valid value
	c.AutoEvaluation.Enabled = other.AutoEvaluation.Enabled
	c.AutoEvaluation.SkipReadOnlyTools = other.AutoEvaluation.SkipReadOnlyTools
	c.AutoEvaluation.VerboseLogging = other.AutoEvaluation.VerboseLogging
}

// SaveConfigTemplate saves a template configuration file
func SaveConfigTemplate(path string) error {
	config := ServerConfig{
		AutoEvaluation: DefaultAutoEvaluationConfig(),
		TasksDir:       "./tasks",
		LogLevel:       "info",
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigSummary returns a summary of current configuration
func (c *ServerConfig) GetConfigSummary() map[string]interface{} {
	return map[string]interface{}{
		"tasks_dir":  c.TasksDir,
		"log_level":  c.LogLevel,
		"auto_evaluation": map[string]interface{}{
			"enabled":             c.AutoEvaluation.Enabled,
			"cache_timeout":       c.AutoEvaluation.CacheTimeout.String(),
			"max_concurrent":      c.AutoEvaluation.MaxConcurrent,
			"skip_read_only_tools": c.AutoEvaluation.SkipReadOnlyTools,
			"verbose_logging":     c.AutoEvaluation.VerboseLogging,
		},
	}
}
