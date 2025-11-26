// Package models provides core data structures for the encoder system.
package models

import (
	"fmt"
	"strings"
)

// Chunk represents a segment of media to be encoded.
//
// Chunks are created by splitting source media files into smaller segments
// based on chapter markers or fixed durations. Each chunk is processed
// independently and can be encoded in parallel.
//
// Use NewChunk to create a validated Chunk instance.
//
// Note: StartTime and EndTime use float64 to preserve fractional seconds,
// which is critical for precise timing, chapter markers, and audio sync.
type Chunk struct {
	ChunkID    uint    `json:"chunk_id"`
	StartTime  float64 `json:"start_time"`
	EndTime    float64 `json:"end_time"`
	SourcePath string  `json:"source_path"`
}

// NewChunk creates a new Chunk with validation.
//
// Returns an error if the chunk parameters are invalid:
//   - SourcePath cannot be empty or whitespace-only
//   - EndTime must be greater than 0
//   - StartTime must be less than EndTime
//
// StartTime and EndTime accept float64 to support fractional seconds
// (e.g., 30.53 seconds) for precise timing.
//
// Example:
//
//	chunk, err := models.NewChunk(1, 0.0, 30.53, "/path/to/video.mp4")
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewChunk(id uint, startTime, endTime float64, sourcePath string) (*Chunk, error) {
	c := &Chunk{
		ChunkID:    id,
		StartTime:  startTime,
		EndTime:    endTime,
		SourcePath: sourcePath,
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("invalid chunk: %w", err)
	}
	return c, nil
}

// Validate checks if the Chunk has valid data.
//
// Returns an error if:
//   - SourcePath is empty or whitespace-only
//   - EndTime is zero
//   - StartTime >= EndTime (invalid time range)
func (c *Chunk) Validate() error {
	// Check source path first
	if strings.TrimSpace(c.SourcePath) == "" {
		return fmt.Errorf("source_path cannot be empty")
	}

	// Check time ranges
	if c.EndTime == 0 {
		return fmt.Errorf("end_time must be greater than 0")
	}

	if c.StartTime >= c.EndTime {
		return fmt.Errorf("start_time must be less than end_time")
	}

	return nil
}
