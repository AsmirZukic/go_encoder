package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_AllLayersPriority(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "encoder.yaml")

	// Create temporary input file for validation
	inputPath := filepath.Join(tmpDir, "test.mp4")
	if err := os.WriteFile(inputPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp input file: %v", err)
	}

	// Config file should set mode to "mixed" and workers to 4
	configContent := `mode: mixed
workers: 4
chunk_duration: 10
audio:
  codec: aac
  bitrate: 128k
video:
  codec: libx264
  crf: 23
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}

	// Set CLI flags to override mode and workers
	os.Args = []string{
		"encoder",
		"-input", inputPath,
		"-output", "out.mp4",
		"-mode", "cpu-only",
		"-workers", "8",
		"-audio-bitrate", "192k",
		"-config", configPath,
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify priority: CLI > File > Defaults
	// Mode: CLI flag should win (cpu-only, not mixed from file)
	if cfg.Mode != "cpu-only" {
		t.Errorf("Expected mode 'cpu-only' (from CLI), got '%s'", cfg.Mode)
	}

	// Workers: CLI should win over file (8, not 4)
	if cfg.Workers != 8 {
		t.Errorf("Expected workers 8 (from CLI), got %d", cfg.Workers)
	}

	// ChunkDuration: File should win over defaults (10, not 5)
	if cfg.ChunkDuration != 10 {
		t.Errorf("Expected chunk duration 10 (from file), got %d", cfg.ChunkDuration)
	}

	// Audio bitrate: CLI should win over file (192k, not 128k)
	if cfg.Audio.Bitrate != "192k" {
		t.Errorf("Expected audio bitrate '192k' (from CLI), got '%s'", cfg.Audio.Bitrate)
	}

	// Audio codec: File should win over defaults (aac, not opus)
	if cfg.Audio.Codec != "aac" {
		t.Errorf("Expected audio codec 'aac' (from file), got '%s'", cfg.Audio.Codec)
	}

	// Video CRF: File should win over defaults (23, not 18)
	if cfg.Video.CRF != 23 {
		t.Errorf("Expected video CRF 23 (from file), got %d", cfg.Video.CRF)
	}
}

func TestLoadConfig_DefaultsOnly(t *testing.T) {
	// Create temporary input file for validation
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.mp4")
	if err := os.WriteFile(inputPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp input file: %v", err)
	}

	// Don't create config file, just provide required flags
	os.Args = []string{
		"encoder",
		"-input", inputPath,
		"-output", "out.mp4",
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Should have defaults for everything except required flags
	defaults := DefaultConfig()
	if cfg.Input != inputPath {
		t.Errorf("Expected input '%s', got '%s'", inputPath, cfg.Input)
	}
	if cfg.Mode != defaults.Mode {
		t.Errorf("Expected default mode '%s', got '%s'", defaults.Mode, cfg.Mode)
	}
	if cfg.ChunkDuration != defaults.ChunkDuration {
		t.Errorf("Expected default chunk duration %d, got %d", defaults.ChunkDuration, cfg.ChunkDuration)
	}
	if cfg.Audio.Codec != defaults.Audio.Codec {
		t.Errorf("Expected default audio codec '%s', got '%s'", defaults.Audio.Codec, cfg.Audio.Codec)
	}
}

func TestLoadConfig_ConfigFileOnly(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "encoder.yaml")

	// Create temporary input file for validation
	inputPath := filepath.Join(tmpDir, "test.mp4")
	if err := os.WriteFile(inputPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp input file: %v", err)
	}

	configContent := `mode: gpu-only
workers: 16
chunk_duration: 15
strict_mode: false
cleanup_chunks: false
verbose: true
audio:
  codec: aac
  bitrate: 256k
  sample_rate: 48000
  channels: 2
video:
  codec: libx265
  crf: 28
  preset: slow
  resolution: 1920x1080
  frame_rate: 60
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}

	os.Args = []string{
		"encoder",
		"-input", inputPath,
		"-output", "out.mp4",
		"-config", configPath,
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify all config file values were loaded
	if cfg.Mode != "gpu-only" {
		t.Errorf("Expected mode 'gpu-only', got '%s'", cfg.Mode)
	}
	if cfg.Workers != 16 {
		t.Errorf("Expected workers 16, got %d", cfg.Workers)
	}
	if cfg.ChunkDuration != 15 {
		t.Errorf("Expected chunk duration 15, got %d", cfg.ChunkDuration)
	}
	if cfg.StrictMode {
		t.Error("Expected strict mode false, got true")
	}
	if cfg.CleanupChunks {
		t.Error("Expected cleanup chunks false, got true")
	}
	if !cfg.Verbose {
		t.Error("Expected verbose true, got false")
	}
	if cfg.Audio.Codec != "aac" {
		t.Errorf("Expected audio codec 'aac', got '%s'", cfg.Audio.Codec)
	}
	if cfg.Audio.Bitrate != "256k" {
		t.Errorf("Expected audio bitrate '256k', got '%s'", cfg.Audio.Bitrate)
	}
	if cfg.Audio.SampleRate != 48000 {
		t.Errorf("Expected audio sample rate 48000, got %d", cfg.Audio.SampleRate)
	}
	if cfg.Video.Codec != "libx265" {
		t.Errorf("Expected video codec 'libx265', got '%s'", cfg.Video.Codec)
	}
	if cfg.Video.CRF != 28 {
		t.Errorf("Expected video CRF 28, got %d", cfg.Video.CRF)
	}
	if cfg.Video.Resolution != "1920x1080" {
		t.Errorf("Expected video resolution '1920x1080', got '%s'", cfg.Video.Resolution)
	}
}

func TestLoadConfig_WorkersAutoDetect(t *testing.T) {
	// Create temporary input file for validation
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.mp4")
	if err := os.WriteFile(inputPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp input file: %v", err)
	}

	os.Args = []string{
		"encoder",
		"-input", inputPath,
		"-output", "out.mp4",
		"-workers", "0", // Explicitly set to 0 to trigger auto-detect
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Workers should be auto-detected (> 0)
	if cfg.Workers <= 0 {
		t.Errorf("Expected workers > 0 (auto-detected), got %d", cfg.Workers)
	}
}

func TestLoadConfig_InvalidConfig(t *testing.T) {
	// Test with invalid mode
	os.Args = []string{
		"encoder",
		"-input", "test.mp4",
		"-output", "out.mp4",
		"-mode", "invalid-mode",
	}

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected validation error for invalid mode, got nil")
	}
}

func TestLoadConfig_InvalidConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "encoder.yaml")

	// Invalid YAML
	configContent := `mode: gpu-only
workers: not-a-number
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}

	os.Args = []string{
		"encoder",
		"-input", "test.mp4",
		"-output", "out.mp4",
		"-config", configPath,
	}

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
}

func TestLoadConfig_MissingConfigFile(t *testing.T) {
	// Point to non-existent config file
	os.Args = []string{
		"encoder",
		"-input", "test.mp4",
		"-output", "out.mp4",
		"-config", "/nonexistent/config.yaml",
	}

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error for missing config file, got nil")
	}
}

func TestLoadConfig_NoConfigSpecified(t *testing.T) {
	// Create temporary input file for validation
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.mp4")
	if err := os.WriteFile(inputPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp input file: %v", err)
	}

	// Don't specify -config flag, LoadConfig should try to find one
	// and gracefully continue if not found
	os.Args = []string{
		"encoder",
		"-input", inputPath,
		"-output", "out.mp4",
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig should not fail when no config file is found: %v", err)
	}

	// Should have defaults
	if cfg.Input != inputPath {
		t.Errorf("Expected input '%s', got '%s'", inputPath, cfg.Input)
	}
	if cfg.Output != "out.mp4" {
		t.Errorf("Expected output 'out.mp4', got '%s'", cfg.Output)
	}
}
