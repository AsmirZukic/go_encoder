package config

import (
	"fmt"
	"os"
	"strings"
)

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	var errors []string

	// Required fields
	if c.Input == "" {
		errors = append(errors, "input file is required")
	} else {
		// Check if input file exists
		if _, err := os.Stat(c.Input); os.IsNotExist(err) {
			errors = append(errors, fmt.Sprintf("input file does not exist: %s", c.Input))
		}
	}

	if c.Output == "" {
		errors = append(errors, "output file is required")
	}

	// Validate mode
	if !IsValidMode(c.Mode) {
		errors = append(errors, fmt.Sprintf("invalid mode '%s', must be one of: %s",
			c.Mode, strings.Join(ModeValues(), ", ")))
	}

	// Validate chunk duration
	if c.ChunkDuration <= 0 {
		errors = append(errors, "chunk duration must be positive")
	}

	// Validate workers (0 is valid, means auto-detect)
	if c.Workers < 0 {
		errors = append(errors, "workers cannot be negative (use 0 for auto-detect)")
	}

	// Validate audio config
	if err := c.Audio.Validate(); err != nil {
		errors = append(errors, fmt.Sprintf("audio config: %v", err))
	}

	// Validate video config
	if err := c.Video.Validate(); err != nil {
		errors = append(errors, fmt.Sprintf("video config: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// Validate checks if audio configuration is valid
func (ac *AudioConfig) Validate() error {
	var errors []string

	if ac.Codec == "" {
		errors = append(errors, "codec is required")
	}

	if ac.Bitrate == "" {
		errors = append(errors, "bitrate is required")
	}

	if ac.SampleRate <= 0 {
		errors = append(errors, "sample rate must be positive")
	}

	if ac.Channels <= 0 {
		errors = append(errors, "channels must be positive")
	} else if ac.Channels > 8 {
		errors = append(errors, "channels cannot exceed 8")
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ", "))
	}

	return nil
}

// Validate checks if video configuration is valid
func (vc *VideoConfig) Validate() error {
	var errors []string

	if vc.Codec == "" {
		errors = append(errors, "codec is required")
	}

	// CRF validation (if using CRF mode)
	if vc.CRF < 0 || vc.CRF > 51 {
		errors = append(errors, "CRF must be between 0 and 51")
	}

	if vc.Preset == "" {
		errors = append(errors, "preset is required")
	}

	// Frame rate validation
	if vc.FrameRate < 0 {
		errors = append(errors, "frame rate cannot be negative (use 0 for original)")
	}

	// Resolution validation (if specified)
	if vc.Resolution != "" {
		if !isValidResolution(vc.Resolution) {
			errors = append(errors, "resolution must be in format WIDTHxHEIGHT (e.g., 1920x1080)")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ", "))
	}

	return nil
}

// isValidResolution checks if resolution string is valid (e.g., "1920x1080")
func isValidResolution(res string) bool {
	if res == "" {
		return true // Empty is valid (means keep original)
	}

	parts := strings.Split(res, "x")
	if len(parts) != 2 {
		return false
	}

	// Check if both parts are numeric
	var width, height int
	_, err1 := fmt.Sscanf(parts[0], "%d", &width)
	_, err2 := fmt.Sscanf(parts[1], "%d", &height)

	return err1 == nil && err2 == nil && width > 0 && height > 0
}
