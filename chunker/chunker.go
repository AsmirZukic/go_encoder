package chunker

import (
	"encoder/models"
	"fmt"
)

const (
	// DefaultChunkDuration is the default duration for fixed-size chunks in seconds
	DefaultChunkDuration = 600 // 10 minutes

	// MinChunkDuration is the minimum allowed chunk duration in seconds
	MinChunkDuration = 1

	// MaxChunkDuration is the maximum allowed chunk duration in seconds (24 hours)
	MaxChunkDuration = 86400
)

// Chunker handles splitting media files into chunks for parallel processing
type Chunker struct {
	sourcePath    string
	chunkDuration uint32
	useChapters   bool
}

// NewChunker creates a new Chunker with default settings
func NewChunker(sourcePath string) *Chunker {
	return &Chunker{
		sourcePath:    sourcePath,
		chunkDuration: DefaultChunkDuration,
		useChapters:   true,
	}
}

// SetChunkDuration sets the duration for fixed-size chunks
func (c *Chunker) SetChunkDuration(duration uint32) *Chunker {
	c.chunkDuration = duration
	return c
}

// SetUseChapters sets whether to use chapter markers if available
func (c *Chunker) SetUseChapters(use bool) *Chunker {
	c.useChapters = use
	return c
}

// CreateChunks creates chunks for parallel processing based on the provided media info.
//
// If chapters are available and useChapters is true, it creates chunks based on chapters.
// Otherwise, it creates fixed-duration chunks.
//
// The mediaInfo parameter should be obtained from a probing tool (e.g., ffprobe.Probe()).
//
// Example:
//
//	probeResult, _ := ffprobe.Probe("/path/to/video.mp4")
//	chunker := NewChunker("/path/to/video.mp4")
//	chunks, err := chunker.CreateChunks(probeResult)
func (c *Chunker) CreateChunks(mediaInfo MediaInfo) ([]*models.Chunk, error) {
	// Validate inputs
	if c.sourcePath == "" {
		return nil, fmt.Errorf("source path cannot be empty")
	}

	if c.chunkDuration < MinChunkDuration {
		return nil, fmt.Errorf("chunk duration must be at least %d seconds", MinChunkDuration)
	}

	if c.chunkDuration > MaxChunkDuration {
		return nil, fmt.Errorf("chunk duration cannot exceed %d seconds", MaxChunkDuration)
	}

	if mediaInfo == nil {
		return nil, fmt.Errorf("media info cannot be nil")
	}

	// Get duration
	duration, err := mediaInfo.GetDuration()
	if err != nil {
		return nil, fmt.Errorf("failed to get duration: %w", err)
	}

	if duration <= 0 {
		return nil, fmt.Errorf("invalid duration: %.2f seconds", duration)
	}

	// Try to create chunks from chapters if available and enabled
	if c.useChapters && mediaInfo.HasChapters() {
		chunks, err := c.createChunksFromChapters(mediaInfo)
		if err == nil && len(chunks) > 0 {
			return chunks, nil
		}
		// Fall through to fixed-duration chunks if chapter-based chunking fails
	}

	// Create fixed-duration chunks
	return c.createFixedDurationChunks(duration)
}

// createChunksFromChapters creates chunks based on chapter markers
func (c *Chunker) createChunksFromChapters(mediaInfo MediaInfo) ([]*models.Chunk, error) {
	chapters := mediaInfo.GetChapters()
	if len(chapters) == 0 {
		return nil, fmt.Errorf("no chapters available")
	}

	chunks := make([]*models.Chunk, 0, len(chapters))

	for i, chapter := range chapters {
		// Parse start and end times from strings
		var startTime, endTime float64
		if _, err := fmt.Sscanf(chapter.StartTime, "%f", &startTime); err != nil {
			return nil, fmt.Errorf("failed to parse start_time for chapter %d: %w", i+1, err)
		}
		if _, err := fmt.Sscanf(chapter.EndTime, "%f", &endTime); err != nil {
			return nil, fmt.Errorf("failed to parse end_time for chapter %d: %w", i+1, err)
		}

		chunk := &models.Chunk{
			ChunkID:    uint(i + 1),
			StartTime:  startTime,
			EndTime:    endTime,
			SourcePath: c.sourcePath,
		}

		// Validate the chunk
		if err := chunk.Validate(); err != nil {
			return nil, fmt.Errorf("invalid chunk %d: %w", i+1, err)
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// createFixedDurationChunks creates chunks of fixed duration
func (c *Chunker) createFixedDurationChunks(duration float64) ([]*models.Chunk, error) {
	if duration <= 0 {
		return nil, fmt.Errorf("invalid duration: %.2f seconds", duration)
	}

	// Use float64 throughout to preserve fractional seconds
	chunkDurationFloat := float64(c.chunkDuration)

	// Calculate number of chunks (ceiling division)
	chunkCount := int(duration / chunkDurationFloat)
	if duration > float64(chunkCount)*chunkDurationFloat {
		chunkCount++
	}

	if chunkCount == 0 {
		chunkCount = 1
	}

	chunks := make([]*models.Chunk, 0, chunkCount)

	for i := 0; i < chunkCount; i++ {
		startTime := float64(i) * chunkDurationFloat
		endTime := startTime + chunkDurationFloat

		// Last chunk should end at the actual duration (preserving fractional seconds)
		if endTime > duration {
			endTime = duration
		}

		chunk := &models.Chunk{
			ChunkID:    uint(i + 1),
			StartTime:  startTime,
			EndTime:    endTime,
			SourcePath: c.sourcePath,
		}

		// Validate the chunk
		if err := chunk.Validate(); err != nil {
			return nil, fmt.Errorf("invalid chunk %d: %w", i+1, err)
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// ValidateChunks validates a sequence of chunks for completeness and correctness
func ValidateChunks(chunks []*models.Chunk) error {
	if len(chunks) == 0 {
		return fmt.Errorf("chunk list is empty")
	}

	// Validate each chunk individually
	for i, chunk := range chunks {
		if err := chunk.Validate(); err != nil {
			return fmt.Errorf("chunk %d is invalid: %w", i, err)
		}
	}

	// Check for consistent source path
	firstSource := chunks[0].SourcePath
	for i, chunk := range chunks {
		if chunk.SourcePath != firstSource {
			return fmt.Errorf("chunk %d has different source path: expected %s, got %s",
				i, firstSource, chunk.SourcePath)
		}
	}

	// Check for sequential chunk IDs
	for i, chunk := range chunks {
		expectedID := uint(i + 1)
		if chunk.ChunkID != expectedID {
			return fmt.Errorf("chunk %d has incorrect ID: expected %d, got %d",
				i, expectedID, chunk.ChunkID)
		}
	}

	// Check for gaps and overlaps
	for i := 0; i < len(chunks)-1; i++ {
		currentEnd := chunks[i].EndTime
		nextStart := chunks[i+1].StartTime

		if currentEnd > nextStart {
			return fmt.Errorf("chunks %d and %d overlap: chunk %d ends at %.2f, chunk %d starts at %.2f",
				i+1, i+2, i+1, currentEnd, i+2, nextStart)
		}

		// Allow small gaps (up to 1 second) for rounding errors
		if nextStart > currentEnd+1 {
			return fmt.Errorf("gap between chunks %d and %d: chunk %d ends at %.2f, chunk %d starts at %.2f",
				i+1, i+2, i+1, currentEnd, i+2, nextStart)
		}
	}

	return nil
}
