package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Check defaults
	if cfg.ChunkDuration != 5 {
		t.Errorf("Expected chunk duration 5, got %d", cfg.ChunkDuration)
	}
	if cfg.Workers != 0 {
		t.Errorf("Expected workers 0 (auto-detect), got %d", cfg.Workers)
	}
	if cfg.Mode != "mixed" {
		t.Errorf("Expected mode 'mixed', got %s", cfg.Mode)
	}
	if cfg.Audio.Codec != "libopus" {
		t.Errorf("Expected audio codec 'libopus', got %s", cfg.Audio.Codec)
	}
	if cfg.Video.Codec != "libx264" {
		t.Errorf("Expected video codec 'libx264', got %s", cfg.Video.Codec)
	}
	if !cfg.StrictMode {
		t.Error("Expected strict mode to be true")
	}
	if !cfg.CleanupChunks {
		t.Error("Expected cleanup chunks to be true")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      func() *Config
		expectError bool
		errorText   string
	}{
		{
			name: "valid config",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Input = createTempFile(t)
				cfg.Output = "/tmp/output.mp4"
				return cfg
			},
			expectError: false,
		},
		{
			name: "missing input",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Output = "/tmp/output.mp4"
				return cfg
			},
			expectError: true,
			errorText:   "input file is required",
		},
		{
			name: "missing output",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Input = createTempFile(t)
				return cfg
			},
			expectError: true,
			errorText:   "output file is required",
		},
		{
			name: "invalid mode",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Input = createTempFile(t)
				cfg.Output = "/tmp/output.mp4"
				cfg.Mode = "invalid"
				return cfg
			},
			expectError: true,
			errorText:   "invalid mode",
		},
		{
			name: "negative chunk duration",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Input = createTempFile(t)
				cfg.Output = "/tmp/output.mp4"
				cfg.ChunkDuration = -1
				return cfg
			},
			expectError: true,
			errorText:   "chunk duration must be positive",
		},
		{
			name: "negative workers",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Input = createTempFile(t)
				cfg.Output = "/tmp/output.mp4"
				cfg.Workers = -1
				return cfg
			},
			expectError: true,
			errorText:   "workers cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.config()
			err := cfg.Validate()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.expectError && err != nil && tt.errorText != "" {
				if !contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorText, err.Error())
				}
			}
		})
	}
}

func TestAudioConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      AudioConfig
		expectError bool
	}{
		{
			name: "valid",
			config: AudioConfig{
				Codec:      "libopus",
				Bitrate:    "128k",
				SampleRate: 48000,
				Channels:   2,
			},
			expectError: false,
		},
		{
			name: "missing codec",
			config: AudioConfig{
				Bitrate:    "128k",
				SampleRate: 48000,
				Channels:   2,
			},
			expectError: true,
		},
		{
			name: "invalid channels",
			config: AudioConfig{
				Codec:      "libopus",
				Bitrate:    "128k",
				SampleRate: 48000,
				Channels:   0,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestVideoConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      VideoConfig
		expectError bool
	}{
		{
			name: "valid",
			config: VideoConfig{
				Codec:  "libx264",
				CRF:    23,
				Preset: "medium",
			},
			expectError: false,
		},
		{
			name: "invalid CRF",
			config: VideoConfig{
				Codec:  "libx264",
				CRF:    60,
				Preset: "medium",
			},
			expectError: true,
		},
		{
			name: "valid resolution",
			config: VideoConfig{
				Codec:      "libx264",
				CRF:        23,
				Preset:     "medium",
				Resolution: "1920x1080",
			},
			expectError: false,
		},
		{
			name: "invalid resolution",
			config: VideoConfig{
				Codec:      "libx264",
				CRF:        23,
				Preset:     "medium",
				Resolution: "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestIsValidMode(t *testing.T) {
	validModes := []string{"cpu-only", "gpu-only", "mixed"}
	for _, mode := range validModes {
		if !IsValidMode(mode) {
			t.Errorf("Mode '%s' should be valid", mode)
		}
	}

	invalidModes := []string{"invalid", "CPU", "gpu", ""}
	for _, mode := range invalidModes {
		if IsValidMode(mode) {
			t.Errorf("Mode '%s' should be invalid", mode)
		}
	}
}

func TestConfigCopy(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Input = "input.mp4"
	cfg.Workers = 8

	copy := cfg.Copy()

	// Modify original
	cfg.Input = "modified.mp4"
	cfg.Workers = 16

	// Copy should be unchanged
	if copy.Input != "input.mp4" {
		t.Errorf("Copy input was modified: expected 'input.mp4', got '%s'", copy.Input)
	}
	if copy.Workers != 8 {
		t.Errorf("Copy workers was modified: expected 8, got %d", copy.Workers)
	}
}

// Helper functions

func createTempFile(t *testing.T) string {
	f, err := os.CreateTemp("", "test-*.mp4")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
