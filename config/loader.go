package config

import (
	"fmt"
	"os"
	"runtime"
)

// LoadConfig loads configuration with priority: CLI flags > Config file > Defaults
func LoadConfig() (*Config, error) {
	// 1. Start with defaults
	cfg := DefaultConfig()

	// 2. Check if -config flag was provided (quick parse to extract it)
	configPath := ""
	for i, arg := range os.Args {
		if arg == "-config" && i+1 < len(os.Args) {
			configPath = os.Args[i+1]
			break
		}
	}

	// If no config flag, try to find config file in standard locations
	if configPath == "" {
		configPath = FindConfigFile()
	}

	// Load config file if found
	if configPath != "" {
		fileCfg, err := LoadConfigFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", configPath, err)
		}
		// Merge file config (overwrites defaults)
		cfg = fileCfg
	}

	// 3. Merge CLI flags (highest priority, overwrites everything)
	if err := cfg.MergeFromFlags(); err != nil {
		return nil, err
	}

	// Auto-detect workers if set to 0
	if cfg.Workers == 0 {
		cfg.Workers = runtime.NumCPU()
	}

	// Validate final configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
