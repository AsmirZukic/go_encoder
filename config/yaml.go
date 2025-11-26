package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadConfigFile loads configuration from a YAML file
func LoadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// FindConfigFile searches for config file in standard locations
// Returns empty string if not found (non-fatal)
func FindConfigFile() string {
	locations := []string{
		"./encoder.yaml",
		"./encoder.yml",
		filepath.Join(os.Getenv("HOME"), ".encoder", "config.yaml"),
		filepath.Join(os.Getenv("HOME"), ".encoder", "config.yml"),
		"/etc/encoder/config.yaml",
		"/etc/encoder/config.yml",
	}

	for _, path := range locations {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// SaveConfigFile saves configuration to a YAML file
func SaveConfigFile(cfg *Config, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
