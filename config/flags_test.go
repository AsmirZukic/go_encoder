package config

import (
	"os"
	"testing"
)

func TestMergeFromFlags_RequiredFlags(t *testing.T) {
	// Test with required flags
	os.Args = []string{"encoder", "-input", "test.mp4", "-output", "out.mp4"}

	cfg := DefaultConfig()
	if err := cfg.MergeFromFlags(); err != nil {
		t.Fatalf("Expected no error with required flags, got: %v", err)
	}

	if cfg.Input != "test.mp4" {
		t.Errorf("Expected input 'test.mp4', got '%s'", cfg.Input)
	}
	if cfg.Output != "out.mp4" {
		t.Errorf("Expected output 'out.mp4', got '%s'", cfg.Output)
	}
}

func TestMergeFromFlags_MissingInput(t *testing.T) {
	// Test missing input file - MergeFromFlags doesn't validate, but input should remain empty
	os.Args = []string{"encoder", "-output", "out.mp4"}

	cfg := DefaultConfig()
	if err := cfg.MergeFromFlags(); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Validation should fail
	if err := cfg.Validate(); err == nil {
		t.Fatal("Expected validation error for missing input, got nil")
	}
}

func TestMergeFromFlags_MissingOutput(t *testing.T) {
	// Test missing output file - MergeFromFlags doesn't validate, but output should remain empty
	os.Args = []string{"encoder", "-input", "test.mp4"}

	cfg := DefaultConfig()
	if err := cfg.MergeFromFlags(); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Validation should fail
	if err := cfg.Validate(); err == nil {
		t.Fatal("Expected validation error for missing output, got nil")
	}
}

func TestMergeFromFlags_AllFlags(t *testing.T) {
	os.Args = []string{
		"encoder",
		"-input", "flag_input.mp4",
		"-output", "flag_output.mp4",
		"-mode", "gpu-only",
		"-workers", "12",
		"-chunk-duration", "15",
		"-audio-codec", "aac",
		"-audio-bitrate", "192k",
		"-audio-sample-rate", "44100",
		"-audio-channels", "1",
		"-video-codec", "libx265",
		"-video-crf", "28",
		"-video-preset", "slow",
		"-video-bitrate", "10M",
		"-video-resolution", "1920x1080",
		"-video-frame-rate", "60",
		"-no-strict",
		"-no-cleanup",
		"-verbose",
	}

	cfg := DefaultConfig()
	if err := cfg.MergeFromFlags(); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify all flags were parsed
	if cfg.Input != "flag_input.mp4" {
		t.Errorf("Expected input 'flag_input.mp4', got '%s'", cfg.Input)
	}
	if cfg.Output != "flag_output.mp4" {
		t.Errorf("Expected output 'flag_output.mp4', got '%s'", cfg.Output)
	}
	if cfg.Mode != "gpu-only" {
		t.Errorf("Expected mode 'gpu-only', got '%s'", cfg.Mode)
	}
	if cfg.Workers != 12 {
		t.Errorf("Expected workers 12, got %d", cfg.Workers)
	}
	if cfg.ChunkDuration != 15 {
		t.Errorf("Expected chunk duration 15, got %d", cfg.ChunkDuration)
	}
	if cfg.StrictMode {
		t.Error("Expected strict mode false, got true")
	}
	if !cfg.Verbose {
		t.Error("Expected verbose true, got false")
	}
	if cfg.Audio.Codec != "aac" {
		t.Errorf("Expected audio codec 'aac', got '%s'", cfg.Audio.Codec)
	}
	if cfg.Audio.Bitrate != "192k" {
		t.Errorf("Expected audio bitrate '192k', got '%s'", cfg.Audio.Bitrate)
	}
	if cfg.Audio.SampleRate != 44100 {
		t.Errorf("Expected audio sample rate 44100, got %d", cfg.Audio.SampleRate)
	}
	if cfg.Audio.Channels != 1 {
		t.Errorf("Expected audio channels 1, got %d", cfg.Audio.Channels)
	}
	if cfg.Video.Codec != "libx265" {
		t.Errorf("Expected video codec 'libx265', got '%s'", cfg.Video.Codec)
	}
	if cfg.Video.CRF != 28 {
		t.Errorf("Expected video CRF 28, got %d", cfg.Video.CRF)
	}
	if cfg.Video.Preset != "slow" {
		t.Errorf("Expected video preset 'slow', got '%s'", cfg.Video.Preset)
	}
	if cfg.Video.Resolution != "1920x1080" {
		t.Errorf("Expected video resolution '1920x1080', got '%s'", cfg.Video.Resolution)
	}
	if cfg.Video.FrameRate != 60 {
		t.Errorf("Expected video frame rate 60, got %d", cfg.Video.FrameRate)
	}
}

func TestMergeFromFlags_ModeShortcuts(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "CPU Only",
			args:     []string{"encoder", "-input", "test.mp4", "-output", "out.mp4", "-cpu-only"},
			expected: "cpu-only",
		},
		{
			name:     "GPU Only",
			args:     []string{"encoder", "-input", "test.mp4", "-output", "out.mp4", "-gpu-only"},
			expected: "gpu-only",
		},
		{
			name:     "Mixed",
			args:     []string{"encoder", "-input", "test.mp4", "-output", "out.mp4", "-mixed"},
			expected: "mixed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			cfg := DefaultConfig()
			if err := cfg.MergeFromFlags(); err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if cfg.Mode != tt.expected {
				t.Errorf("Expected mode '%s', got '%s'", tt.expected, cfg.Mode)
			}
		})
	}
}

func TestMergeFromFlags_DryRun(t *testing.T) {
	os.Args = []string{
		"encoder",
		"-input", "test.mp4",
		"-output", "out.mp4",
		"-dry-run",
	}

	cfg := DefaultConfig()
	if err := cfg.MergeFromFlags(); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !cfg.DryRun {
		t.Error("Expected dry-run true, got false")
	}
}

func TestMergeFromFlags_PartialOverride(t *testing.T) {
	// Only set required flags plus a few overrides
	os.Args = []string{
		"encoder",
		"-input", "test.mp4",
		"-output", "out.mp4",
		"-mode", "cpu-only",
		"-workers", "6",
	}

	cfg := DefaultConfig()
	originalCodec := cfg.Audio.Codec // Should remain unchanged

	if err := cfg.MergeFromFlags(); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify overridden values
	if cfg.Mode != "cpu-only" {
		t.Errorf("Expected mode 'cpu-only', got '%s'", cfg.Mode)
	}
	if cfg.Workers != 6 {
		t.Errorf("Expected workers 6, got %d", cfg.Workers)
	}

	// Verify unchanged values
	if cfg.Audio.Codec != originalCodec {
		t.Errorf("Audio codec should not have changed, expected '%s', got '%s'", originalCodec, cfg.Audio.Codec)
	}
}
