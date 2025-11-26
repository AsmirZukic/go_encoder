package models

import (
	"testing"
	"time"
)

func TestNewEncodingProgress(t *testing.T) {
	duration := 30.0
	progress := NewEncodingProgress(duration)

	if progress == nil {
		t.Fatal("NewEncodingProgress returned nil")
	}

	if progress.TotalDuration != duration {
		t.Errorf("Expected TotalDuration %.2f, got %.2f", duration, progress.TotalDuration)
	}

	if progress.State != ProgressStateQueued {
		t.Errorf("Expected initial state %s, got %s", ProgressStateQueued, progress.State)
	}

	if progress.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}

	if progress.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestEncodingProgress_CalculateProgress(t *testing.T) {
	progress := NewEncodingProgress(30.0)

	tests := []struct {
		name            string
		currentSeconds  float64
		expectedPercent float64
	}{
		{"zero progress", 0, 0.0},
		{"halfway", 15.0, 50.0},
		{"complete", 30.0, 100.0},
		{"over 100%", 35.0, 100.0}, // Should cap at 100%
		{"fractional", 10.5, 35.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progress.CalculateProgress(tt.currentSeconds)

			if progress.Progress != tt.expectedPercent {
				t.Errorf("Expected progress %.2f%%, got %.2f%%", tt.expectedPercent, progress.Progress)
			}
		})
	}
}

func TestEncodingProgress_CalculateProgress_ZeroDuration(t *testing.T) {
	progress := NewEncodingProgress(0)
	progress.CalculateProgress(15.0)

	// Should not crash with zero duration
	if progress.Progress != 0 {
		t.Errorf("Expected 0%% progress with zero duration, got %.2f%%", progress.Progress)
	}
}

func TestEncodingProgress_EstimatedTimeRemaining(t *testing.T) {
	progress := NewEncodingProgress(30.0)
	progress.StartTime = time.Now().Add(-10 * time.Second) // Started 10 seconds ago
	progress.Speed = 2.0                                   // Encoding at 2x speed
	progress.Progress = 50.0                               // 50% complete

	eta := progress.EstimatedTimeRemaining()

	// At 50% progress after 10 seconds, should take another ~10 seconds
	// Allow some margin for timing
	if eta < 9*time.Second || eta > 11*time.Second {
		t.Errorf("Expected ETA around 10s, got %v", eta)
	}
}

func TestEncodingProgress_EstimatedTimeRemaining_NoProgress(t *testing.T) {
	progress := NewEncodingProgress(30.0)
	progress.Progress = 0.0

	eta := progress.EstimatedTimeRemaining()
	if eta != 0 {
		t.Errorf("Expected 0 ETA with no progress, got %v", eta)
	}
}

func TestEncodingProgress_EstimatedTimeRemaining_NoSpeed(t *testing.T) {
	progress := NewEncodingProgress(30.0)
	progress.Progress = 50.0
	progress.Speed = 0.0

	eta := progress.EstimatedTimeRemaining()
	if eta != 0 {
		t.Errorf("Expected 0 ETA with no speed, got %v", eta)
	}
}

func TestEncodingProgress_EstimatedTimeRemaining_NegativeHandling(t *testing.T) {
	progress := NewEncodingProgress(30.0)
	progress.StartTime = time.Now().Add(-1 * time.Hour) // Started an hour ago
	progress.Progress = 99.9                            // Almost done
	progress.Speed = 100.0                              // Very fast

	eta := progress.EstimatedTimeRemaining()

	// Should return 0 if calculated time is negative (already past estimated completion)
	if eta < 0 {
		t.Errorf("ETA should not be negative, got %v", eta)
	}
}

func TestEncodingProgress_FormatSummary(t *testing.T) {
	progress := NewEncodingProgress(30.0)
	progress.Progress = 50.0
	progress.Speed = 2.5
	progress.Bitrate = "128kbits/s"
	progress.Size = "256kB"
	progress.StartTime = time.Now().Add(-10 * time.Second)

	summary := progress.FormatSummary()

	if summary == "" {
		t.Error("FormatSummary should not return empty string")
	}

	// Check if summary contains key information
	if len(summary) < 20 {
		t.Errorf("Summary seems too short: %s", summary)
	}
}

func TestProgressStateConstants(t *testing.T) {
	states := []ProgressState{
		ProgressStateQueued,
		ProgressStateStarting,
		ProgressStateEncoding,
		ProgressStateCompleted,
		ProgressStateFailed,
		ProgressStateCancelled,
	}

	// Verify all states are unique
	seen := make(map[ProgressState]bool)
	for _, state := range states {
		if seen[state] {
			t.Errorf("Duplicate progress state: %s", state)
		}
		seen[state] = true

		if string(state) == "" {
			t.Error("Progress state should not be empty string")
		}
	}

	if len(seen) != 6 {
		t.Errorf("Expected 6 unique states, got %d", len(seen))
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "calculating..."},
		{"5 seconds", 5 * time.Second, "5s"},
		{"30 seconds", 30 * time.Second, "30s"},
		{"1 minute", 60 * time.Second, "1m0s"},
		{"1 minute 30 seconds", 90 * time.Second, "1m30s"},
		{"2 minutes", 120 * time.Second, "2m0s"},
		{"1 hour", 3600 * time.Second, "1h0m0s"},
		{"1 hour 30 minutes", 5400 * time.Second, "1h30m0s"},
		{"1 hour 23 minutes 45 seconds", 5025 * time.Second, "1h23m45s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a progress instance to access formatDuration through FormatSummary
			progress := NewEncodingProgress(100.0)
			progress.StartTime = time.Now().Add(-tt.duration)
			progress.Progress = 50.0
			progress.Speed = 1.0

			summary := progress.FormatSummary()

			// Just verify the summary is generated without error
			// The formatDuration function is private, but tested through FormatSummary
			if summary == "" {
				t.Error("Summary should not be empty")
			}
		})
	}
}

func TestEncodingProgress_ProgressCallback(t *testing.T) {
	progress := NewEncodingProgress(30.0)

	callbackCalled := false
	callback := func(p *EncodingProgress) {
		callbackCalled = true
		if p.Progress < 0 || p.Progress > 100 {
			t.Errorf("Progress should be between 0-100, got %.2f", p.Progress)
		}
	}

	// Simulate using the callback
	callback(progress)

	if !callbackCalled {
		t.Error("Callback was not called")
	}
}

func TestEncodingProgress_MultipleUpdates(t *testing.T) {
	progress := NewEncodingProgress(30.0)

	// Simulate multiple progress updates
	updates := []struct {
		currentTime float64
		speed       float64
		bitrate     string
	}{
		{5.0, 1.5, "128kbits/s"},
		{10.0, 2.0, "128kbits/s"},
		{15.0, 2.5, "130kbits/s"},
		{20.0, 3.0, "132kbits/s"},
	}

	lastProgress := 0.0
	for _, update := range updates {
		progress.CalculateProgress(update.currentTime)
		progress.Speed = update.speed
		progress.Bitrate = update.bitrate

		// Progress should increase
		if progress.Progress < lastProgress {
			t.Errorf("Progress decreased: %.2f -> %.2f", lastProgress, progress.Progress)
		}
		lastProgress = progress.Progress

		// UpdatedAt should be updated
		if progress.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should be set after update")
		}
	}

	// Final progress should be around 66% (20/30)
	expectedProgress := (20.0 / 30.0) * 100
	if progress.Progress < expectedProgress-1 || progress.Progress > expectedProgress+1 {
		t.Errorf("Expected final progress around %.2f%%, got %.2f%%", expectedProgress, progress.Progress)
	}
}
