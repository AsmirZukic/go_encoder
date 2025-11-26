package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	yamlContent := `
input: "test.mp4"
output: "output.mp4"
chunk_duration: 10
workers: 4
mode: "cpu-only"
audio:
  codec: "aac"
  bitrate: "192k"
  sample_rate: 44100
  channels: 2
video:
  codec: "libx265"
  crf: 28
  preset: "fast"
strict_mode: false
cleanup_chunks: false
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfigFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values
	if cfg.Input != "test.mp4" {
		t.Errorf("Expected input 'test.mp4', got '%s'", cfg.Input)
	}
	if cfg.ChunkDuration != 10 {
		t.Errorf("Expected chunk duration 10, got %d", cfg.ChunkDuration)
	}
	if cfg.Workers != 4 {
		t.Errorf("Expected workers 4, got %d", cfg.Workers)
	}
	if cfg.Mode != "cpu-only" {
		t.Errorf("Expected mode 'cpu-only', got '%s'", cfg.Mode)
	}
	if cfg.Audio.Codec != "aac" {
		t.Errorf("Expected audio codec 'aac', got '%s'", cfg.Audio.Codec)
	}
	if cfg.Video.Codec != "libx265" {
		t.Errorf("Expected video codec 'libx265', got '%s'", cfg.Video.Codec)
	}
	if cfg.StrictMode {
		t.Error("Expected strict mode false, got true")
	}
}

func TestLoadConfigFile_NotFound(t *testing.T) {
	_, err := LoadConfigFile("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestLoadConfigFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidYAML := `
input: test.mp4
invalid yaml syntax here ][{
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfigFile(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestSaveConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	cfg := DefaultConfig()
	cfg.Input = "input.mp4"
	cfg.Output = "output.mp4"
	cfg.Workers = 8

	if err := SaveConfigFile(cfg, configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load it back and verify
	loaded, err := LoadConfigFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loaded.Input != cfg.Input {
		t.Errorf("Input mismatch: expected '%s', got '%s'", cfg.Input, loaded.Input)
	}
	if loaded.Workers != cfg.Workers {
		t.Errorf("Workers mismatch: expected %d, got %d", cfg.Workers, loaded.Workers)
	}
}

func TestFindConfigFile(t *testing.T) {
	// This test depends on system state, so we'll just test it doesn't panic
	path := FindConfigFile()
	// Path can be empty if no config file exists (non-fatal)
	_ = path
}
