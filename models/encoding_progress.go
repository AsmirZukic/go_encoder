package models

import (
	"fmt"
	"time"
)

// EncodingProgress represents real-time encoding metrics from ffmpeg
type EncodingProgress struct {
	// Current position in the file
	Frame       int64   // Current frame number
	FPS         float64 // Frames per second being processed
	CurrentTime string  // Current timestamp (HH:MM:SS.MS)

	// Performance metrics
	Bitrate string  // Current bitrate (e.g., "128.0kbits/s")
	Speed   float64 // Encoding speed multiplier (e.g., 2.34 means 2.34x realtime)

	// Size information
	Size string // Current output file size (e.g., "1024kB")

	// Progress calculation
	TotalDuration float64 // Total duration in seconds (for percentage calculation)
	Progress      float64 // Percentage complete (0-100)

	// Metadata
	State     ProgressState // Current state of encoding
	StartTime time.Time     // When encoding started
	UpdatedAt time.Time     // Last update timestamp
}

// ProgressState represents the current state of an encoding task
type ProgressState string

const (
	ProgressStateQueued    ProgressState = "queued"    // Waiting in queue
	ProgressStateStarting  ProgressState = "starting"  // Initializing
	ProgressStateEncoding  ProgressState = "encoding"  // Actively encoding
	ProgressStateCompleted ProgressState = "completed" // Successfully finished
	ProgressStateFailed    ProgressState = "failed"    // Encountered an error
	ProgressStateCancelled ProgressState = "cancelled" // User cancelled
)

// ProgressCallback is a function that receives progress updates during encoding
type ProgressCallback func(progress *EncodingProgress)

// NewEncodingProgress creates a new progress tracker
func NewEncodingProgress(totalDuration float64) *EncodingProgress {
	return &EncodingProgress{
		TotalDuration: totalDuration,
		State:         ProgressStateQueued,
		StartTime:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// CalculateProgress updates the progress percentage based on current time
func (ep *EncodingProgress) CalculateProgress(currentSeconds float64) {
	if ep.TotalDuration > 0 {
		ep.Progress = (currentSeconds / ep.TotalDuration) * 100
		if ep.Progress > 100 {
			ep.Progress = 100
		}
	}
	ep.UpdatedAt = time.Now()
}

// EstimatedTimeRemaining calculates ETA based on current speed
func (ep *EncodingProgress) EstimatedTimeRemaining() time.Duration {
	if ep.Speed <= 0 || ep.Progress <= 0 {
		return 0
	}

	elapsed := time.Since(ep.StartTime)
	totalEstimated := time.Duration(float64(elapsed) / (ep.Progress / 100))
	remaining := totalEstimated - elapsed

	if remaining < 0 {
		return 0
	}
	return remaining
}

// FormatSummary returns a human-readable summary of the progress
func (ep *EncodingProgress) FormatSummary() string {
	eta := ep.EstimatedTimeRemaining()
	return fmt.Sprintf(
		"Progress: %.1f%% | Speed: %.2fx | Bitrate: %s | Size: %s | ETA: %s",
		ep.Progress,
		ep.Speed,
		ep.Bitrate,
		ep.Size,
		formatDuration(eta),
	)
}

// formatDuration converts a duration to a human-readable string
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "calculating..."
	}

	seconds := int(d.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}

	minutes := seconds / 60
	seconds = seconds % 60

	if minutes < 60 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}

	hours := minutes / 60
	minutes = minutes % 60
	return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
}
