package chunker

import (
	"encoder/models"
	"fmt"
	"testing"
)

// mockMediaInfo is a simple mock implementation of MediaInfo for testing
type mockMediaInfo struct {
	duration float64
	chapters []ChapterInfo
}

func (m *mockMediaInfo) GetDuration() (float64, error) {
	return m.duration, nil
}

func (m *mockMediaInfo) HasChapters() bool {
	return len(m.chapters) > 0
}

func (m *mockMediaInfo) GetChapters() []ChapterInfo {
	return m.chapters
}

// newMockMediaInfo creates a mock media info with the specified duration and no chapters
func newMockMediaInfo(duration float64) *mockMediaInfo {
	return &mockMediaInfo{duration: duration}
}

// newMockMediaInfoWithChapters creates a mock media info with chapters
func newMockMediaInfoWithChapters(duration float64, chapters []ChapterInfo) *mockMediaInfo {
	return &mockMediaInfo{
		duration: duration,
		chapters: chapters,
	}
}

// TestNewChunker tests the constructor
func TestNewChunker(t *testing.T) {
	sourcePath := "/path/to/file.mp4"
	chunker := NewChunker(sourcePath)

	if chunker.sourcePath != sourcePath {
		t.Errorf("Expected sourcePath %s, got %s", sourcePath, chunker.sourcePath)
	}

	if chunker.chunkDuration != DefaultChunkDuration {
		t.Errorf("Expected chunkDuration %.1f, got %.1f", float64(DefaultChunkDuration), chunker.chunkDuration)
	}

	if !chunker.useChapters {
		t.Error("Expected useChapters to be true by default")
	}
}

// TestChunker_SetChunkDuration tests the SetChunkDuration method
func TestChunker_SetChunkDuration(t *testing.T) {
	chunker := NewChunker("/path/to/file.mp4")
	duration := float64(300)

	result := chunker.SetChunkDuration(duration)

	if chunker.chunkDuration != duration {
		t.Errorf("Expected chunkDuration %.1f, got %.1f", duration, chunker.chunkDuration)
	}

	// Test fluent API
	if result != chunker {
		t.Error("SetChunkDuration should return the chunker for method chaining")
	}
}

// TestChunker_SetUseChapters tests the SetUseChapters method
func TestChunker_SetUseChapters(t *testing.T) {
	chunker := NewChunker("/path/to/file.mp4")

	result := chunker.SetUseChapters(false)

	if chunker.useChapters {
		t.Error("Expected useChapters to be false")
	}

	// Test fluent API
	if result != chunker {
		t.Error("SetUseChapters should return the chunker for method chaining")
	}
}

// TestChunker_CreateChunks_EmptySourcePath tests error handling for empty source path
func TestChunker_CreateChunks_EmptySourcePath(t *testing.T) {
	chunker := NewChunker("")
	mediaInfo := newMockMediaInfo(30.0)

	_, err := chunker.CreateChunks(mediaInfo)

	if err == nil {
		t.Error("Expected error for empty source path")
	}

	expectedMsg := "source path cannot be empty"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestChunker_CreateChunks_InvalidChunkDuration tests error handling for invalid chunk duration
func TestChunker_CreateChunks_InvalidChunkDuration(t *testing.T) {
	tests := []struct {
		name          string
		duration      float64
		expectedError string
	}{
		{
			name:          "duration too small",
			duration:      0,
			expectedError: fmt.Sprintf("chunk duration must be at least %d seconds", MinChunkDuration),
		},
		{
			name:          "duration exceeds maximum",
			duration:      MaxChunkDuration + 1,
			expectedError: fmt.Sprintf("chunk duration cannot exceed %d seconds", MaxChunkDuration),
		},
	}

	// Use the test file that exists
	testFile := "/home/asmir/file_example_MP4_480_1_5MG.mp4"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunker := NewChunker(testFile)
			chunker.SetChunkDuration(tt.duration)
			mediaInfo := newMockMediaInfo(30.0)

			_, err := chunker.CreateChunks(mediaInfo)

			if err == nil {
				t.Error("Expected error for invalid chunk duration")
			}

			if err.Error() != tt.expectedError {
				t.Errorf("Expected error '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

// TestChunker_CreateChunks_NilMediaInfo tests error handling for nil media info
func TestChunker_CreateChunks_NilMediaInfo(t *testing.T) {
	chunker := NewChunker("/path/to/file.mp4")

	_, err := chunker.CreateChunks(nil)

	if err == nil {
		t.Error("Expected error for nil media info")
	}

	expectedMsg := "media info cannot be nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestChunker_CreateChunks_WithMockMediaInfo tests creating chunks with mock media info
func TestChunker_CreateChunks_WithMockMediaInfo(t *testing.T) {
	testFile := "/home/asmir/file_example_MP4_480_1_5MG.mp4"

	t.Run("default settings (10 minute chunks, no chapters)", func(t *testing.T) {
		chunker := NewChunker(testFile)
		chunker.SetUseChapters(false) // No chapters

		// Mock a 30 second file
		mediaInfo := newMockMediaInfo(30.0)
		chunks, err := chunker.CreateChunks(mediaInfo)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// The file is about 30 seconds, so we should get 1 chunk
		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk for 30 second file with 10 minute chunks, got %d", len(chunks))
		}

		// Verify first chunk
		if len(chunks) > 0 {
			chunk := chunks[0]
			if chunk.ChunkID != 1 {
				t.Errorf("Expected chunk ID 1, got %d", chunk.ChunkID)
			}
			if chunk.StartTime != 0 {
				t.Errorf("Expected start time 0, got %.2f", chunk.StartTime)
			}
			if chunk.EndTime != 30 {
				t.Errorf("Expected end time 30, got %.2f", chunk.EndTime)
			}
			if chunk.SourcePath != testFile {
				t.Errorf("Expected source path %s, got %s", testFile, chunk.SourcePath)
			}
		}
	})

	t.Run("small chunk duration", func(t *testing.T) {
		chunker := NewChunker(testFile)
		chunker.SetChunkDuration(10).SetUseChapters(false)

		// Mock a 30 second file
		mediaInfo := newMockMediaInfo(30.0)
		chunks, err := chunker.CreateChunks(mediaInfo)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// The file is about 30 seconds, so we should get 3 chunks (10s each)
		if len(chunks) != 3 {
			t.Errorf("Expected 3 chunks for 30 second file with 10 second chunks, got %d", len(chunks))
		}

		// Verify chunk sequence
		for i, chunk := range chunks {
			expectedID := uint(i + 1)
			if chunk.ChunkID != expectedID {
				t.Errorf("Chunk %d: expected ID %d, got %d", i, expectedID, chunk.ChunkID)
			}

			expectedStart := float64(i * 10)
			if chunk.StartTime != expectedStart {
				t.Errorf("Chunk %d: expected start time %.2f, got %.2f", i, expectedStart, chunk.StartTime)
			}

			expectedEnd := expectedStart + 10.0
			if i == len(chunks)-1 {
				expectedEnd = 30.0 // Last chunk ends at actual duration
			}
			if chunk.EndTime != expectedEnd {
				t.Errorf("Chunk %d: expected end time %.2f, got %.2f", i, expectedEnd, chunk.EndTime)
			}
		}
	})
}

// TestChunker_CreateFixedDurationChunks tests the fixed-duration chunking logic
func TestChunker_CreateFixedDurationChunks(t *testing.T) {
	tests := []struct {
		name           string
		duration       float64
		chunkDuration  float64
		expectedChunks int
	}{
		{
			name:           "exact division",
			duration:       60.0,
			chunkDuration:  30,
			expectedChunks: 2,
		},
		{
			name:           "with remainder",
			duration:       65.0,
			chunkDuration:  30,
			expectedChunks: 3,
		},
		{
			name:           "single chunk",
			duration:       15.0,
			chunkDuration:  30,
			expectedChunks: 1,
		},
		{
			name:           "very short duration",
			duration:       1.5,
			chunkDuration:  10,
			expectedChunks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunker := &Chunker{
				sourcePath:    "/test/file.mp4",
				chunkDuration: tt.chunkDuration,
			}

			chunks, err := chunker.createFixedDurationChunks(tt.duration)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(chunks) != tt.expectedChunks {
				t.Errorf("Expected %d chunks, got %d", tt.expectedChunks, len(chunks))
			}

			// Verify chunk properties
			for i, chunk := range chunks {
				// Check ID
				expectedID := uint(i + 1)
				if chunk.ChunkID != expectedID {
					t.Errorf("Chunk %d: expected ID %d, got %d", i, expectedID, chunk.ChunkID)
				}

				// Check start time
				expectedStart := float64(i) * float64(tt.chunkDuration)
				if chunk.StartTime != expectedStart {
					t.Errorf("Chunk %d: expected start time %.2f, got %.2f", i, expectedStart, chunk.StartTime)
				}

				// Check end time
				if i == len(chunks)-1 {
					// Last chunk should end at actual duration
					expectedEnd := tt.duration
					if chunk.EndTime != expectedEnd {
						t.Errorf("Last chunk: expected end time %.2f, got %.2f", expectedEnd, chunk.EndTime)
					}
				} else {
					// Other chunks should have full duration
					expectedEnd := expectedStart + float64(tt.chunkDuration)
					if chunk.EndTime != expectedEnd {
						t.Errorf("Chunk %d: expected end time %.2f, got %.2f", i, expectedEnd, chunk.EndTime)
					}
				}

				// Validate each chunk
				if err := chunk.Validate(); err != nil {
					t.Errorf("Chunk %d failed validation: %v", i, err)
				}
			}
		})
	}
}

// TestValidateChunks tests the chunk validation function
func TestValidateChunks(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		err := ValidateChunks([]*models.Chunk{})
		if err == nil {
			t.Error("Expected error for empty chunk list")
		}
	})

	t.Run("valid sequence", func(t *testing.T) {
		chunks := []*models.Chunk{
			{ChunkID: 1, StartTime: 0, EndTime: 10, SourcePath: "/test/file.mp4"},
			{ChunkID: 2, StartTime: 10, EndTime: 20, SourcePath: "/test/file.mp4"},
			{ChunkID: 3, StartTime: 20, EndTime: 30, SourcePath: "/test/file.mp4"},
		}

		err := ValidateChunks(chunks)
		if err != nil {
			t.Errorf("Unexpected error for valid chunks: %v", err)
		}
	})

	t.Run("inconsistent source paths", func(t *testing.T) {
		chunks := []*models.Chunk{
			{ChunkID: 1, StartTime: 0, EndTime: 10, SourcePath: "/test/file1.mp4"},
			{ChunkID: 2, StartTime: 10, EndTime: 20, SourcePath: "/test/file2.mp4"},
		}

		err := ValidateChunks(chunks)
		if err == nil {
			t.Error("Expected error for inconsistent source paths")
		}
	})

	t.Run("non-sequential IDs", func(t *testing.T) {
		chunks := []*models.Chunk{
			{ChunkID: 1, StartTime: 0, EndTime: 10, SourcePath: "/test/file.mp4"},
			{ChunkID: 3, StartTime: 10, EndTime: 20, SourcePath: "/test/file.mp4"},
		}

		err := ValidateChunks(chunks)
		if err == nil {
			t.Error("Expected error for non-sequential chunk IDs")
		}
	})

	t.Run("overlapping chunks", func(t *testing.T) {
		chunks := []*models.Chunk{
			{ChunkID: 1, StartTime: 0, EndTime: 15, SourcePath: "/test/file.mp4"},
			{ChunkID: 2, StartTime: 10, EndTime: 20, SourcePath: "/test/file.mp4"},
		}

		err := ValidateChunks(chunks)
		if err == nil {
			t.Error("Expected error for overlapping chunks")
		}
	})

	t.Run("gap between chunks", func(t *testing.T) {
		chunks := []*models.Chunk{
			{ChunkID: 1, StartTime: 0, EndTime: 10, SourcePath: "/test/file.mp4"},
			{ChunkID: 2, StartTime: 15, EndTime: 25, SourcePath: "/test/file.mp4"},
		}

		err := ValidateChunks(chunks)
		if err == nil {
			t.Error("Expected error for gap between chunks")
		}
	})

	t.Run("small gap allowed (rounding)", func(t *testing.T) {
		chunks := []*models.Chunk{
			{ChunkID: 1, StartTime: 0, EndTime: 10, SourcePath: "/test/file.mp4"},
			{ChunkID: 2, StartTime: 11, EndTime: 21, SourcePath: "/test/file.mp4"}, // 1 second gap is allowed
		}

		err := ValidateChunks(chunks)
		if err != nil {
			t.Errorf("Unexpected error for small gap: %v", err)
		}
	})

	t.Run("invalid chunk in sequence", func(t *testing.T) {
		chunks := []*models.Chunk{
			{ChunkID: 1, StartTime: 0, EndTime: 10, SourcePath: "/test/file.mp4"},
			{ChunkID: 2, StartTime: 15, EndTime: 10, SourcePath: "/test/file.mp4"}, // Invalid: end < start
		}

		err := ValidateChunks(chunks)
		if err == nil {
			t.Error("Expected error for invalid chunk in sequence")
		}
	})
}

// TestChunker_FluentAPI tests method chaining
func TestChunker_FluentAPI(t *testing.T) {
	testFile := "/home/asmir/file_example_MP4_480_1_5MG.mp4"

	// Test fluent API with method chaining using mock media info
	mediaInfo := newMockMediaInfo(30.0) // 30 second file

	chunks, err := NewChunker(testFile).
		SetChunkDuration(15).
		SetUseChapters(false).
		CreateChunks(mediaInfo)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// The file is about 30 seconds, with 15 second chunks we should get 2 chunks
	if len(chunks) != 2 {
		t.Errorf("Expected 2 chunks, got %d", len(chunks))
	}
}

// TestConstants tests the package constants
func TestConstants(t *testing.T) {
	if DefaultChunkDuration != 600 {
		t.Errorf("Expected DefaultChunkDuration to be 600, got %d", DefaultChunkDuration)
	}

	if MinChunkDuration != 1 {
		t.Errorf("Expected MinChunkDuration to be 1, got %d", MinChunkDuration)
	}

	if MaxChunkDuration != 86400 {
		t.Errorf("Expected MaxChunkDuration to be 86400, got %d", MaxChunkDuration)
	}
}

// TestChunker_CreateFixedDurationChunks_EdgeCases tests edge cases for fixed-duration chunking
func TestChunker_CreateFixedDurationChunks_EdgeCases(t *testing.T) {
	t.Run("zero duration", func(t *testing.T) {
		chunker := &Chunker{
			sourcePath:    "/test/file.mp4",
			chunkDuration: 10,
		}

		_, err := chunker.createFixedDurationChunks(0.0)

		if err == nil {
			t.Error("Expected error for zero duration")
		}
	})

	t.Run("negative duration", func(t *testing.T) {
		chunker := &Chunker{
			sourcePath:    "/test/file.mp4",
			chunkDuration: 10,
		}

		_, err := chunker.createFixedDurationChunks(-10.0)

		if err == nil {
			t.Error("Expected error for negative duration")
		}
	})

	t.Run("large duration", func(t *testing.T) {
		chunker := &Chunker{
			sourcePath:    "/test/file.mp4",
			chunkDuration: 600,
		}

		chunks, err := chunker.createFixedDurationChunks(3600.0) // 1 hour

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		expectedChunks := 6 // 1 hour / 10 minutes
		if len(chunks) != expectedChunks {
			t.Errorf("Expected %d chunks, got %d", expectedChunks, len(chunks))
		}
	})

	t.Run("fractional seconds", func(t *testing.T) {
		chunker := &Chunker{
			sourcePath:    "/test/file.mp4",
			chunkDuration: 10,
		}

		chunks, err := chunker.createFixedDurationChunks(25.7)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(chunks) != 3 {
			t.Errorf("Expected 3 chunks, got %d", len(chunks))
		}

		// Last chunk should end at 25.7 (preserving fractional seconds)
		if len(chunks) > 0 {
			lastChunk := chunks[len(chunks)-1]
			if lastChunk.EndTime != 25.7 {
				t.Errorf("Expected last chunk end time 25.7, got %.2f", lastChunk.EndTime)
			}
		}
	})
}

// TestValidateChunks_EdgeCases tests additional edge cases for chunk validation
func TestValidateChunks_EdgeCases(t *testing.T) {
	t.Run("single chunk", func(t *testing.T) {
		chunks := []*models.Chunk{
			{ChunkID: 1, StartTime: 0, EndTime: 60, SourcePath: "/test/file.mp4"},
		}

		err := ValidateChunks(chunks)
		if err != nil {
			t.Errorf("Unexpected error for single chunk: %v", err)
		}
	})
}

// TestChunker_CreateChunksFromChapters tests chapter-based chunking
func TestChunker_CreateChunksFromChapters(t *testing.T) {
	t.Run("valid chapters", func(t *testing.T) {
		chunker := &Chunker{
			sourcePath:  "/test/file.mp4",
			useChapters: true,
		}

		mediaInfo := newMockMediaInfoWithChapters(360, []ChapterInfo{
			{StartTime: "0.000000", EndTime: "120.000000"},
			{StartTime: "120.000000", EndTime: "240.000000"},
			{StartTime: "240.000000", EndTime: "360.000000"},
		})

		chunks, err := chunker.createChunksFromChapters(mediaInfo)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(chunks) != 3 {
			t.Errorf("Expected 3 chunks, got %d", len(chunks))
		}

		// Verify first chunk
		if chunks[0].ChunkID != 1 {
			t.Errorf("Expected chunk ID 1, got %d", chunks[0].ChunkID)
		}
		if chunks[0].StartTime != 0 {
			t.Errorf("Expected start time 0, got %.2f", chunks[0].StartTime)
		}
		if chunks[0].EndTime != 120 {
			t.Errorf("Expected end time 120, got %.2f", chunks[0].EndTime)
		}

		// Verify last chunk
		if chunks[2].ChunkID != 3 {
			t.Errorf("Expected chunk ID 3, got %d", chunks[2].ChunkID)
		}
		if chunks[2].StartTime != 240 {
			t.Errorf("Expected start time 240, got %.2f", chunks[2].StartTime)
		}
		if chunks[2].EndTime != 360 {
			t.Errorf("Expected end time 360, got %.2f", chunks[2].EndTime)
		}
	})

	t.Run("empty chapters", func(t *testing.T) {
		chunker := &Chunker{
			sourcePath:  "/test/file.mp4",
			useChapters: true,
		}

		mediaInfo := newMockMediaInfoWithChapters(0, []ChapterInfo{})

		_, err := chunker.createChunksFromChapters(mediaInfo)

		if err == nil {
			t.Error("Expected error for empty chapters")
		}
	})

	t.Run("invalid start time format", func(t *testing.T) {
		chunker := &Chunker{
			sourcePath:  "/test/file.mp4",
			useChapters: true,
		}

		mediaInfo := newMockMediaInfoWithChapters(120, []ChapterInfo{
			{StartTime: "invalid", EndTime: "120.000000"},
		})

		_, err := chunker.createChunksFromChapters(mediaInfo)

		if err == nil {
			t.Error("Expected error for invalid start time")
		}
	})

	t.Run("invalid end time format", func(t *testing.T) {
		chunker := &Chunker{
			sourcePath:  "/test/file.mp4",
			useChapters: true,
		}

		mediaInfo := newMockMediaInfoWithChapters(120, []ChapterInfo{
			{StartTime: "0.000000", EndTime: "invalid"},
		})

		_, err := chunker.createChunksFromChapters(mediaInfo)

		if err == nil {
			t.Error("Expected error for invalid end time")
		}
	})

	t.Run("invalid chunk (end before start)", func(t *testing.T) {
		chunker := &Chunker{
			sourcePath:  "/test/file.mp4",
			useChapters: true,
		}

		mediaInfo := newMockMediaInfoWithChapters(100, []ChapterInfo{
			{StartTime: "100.000000", EndTime: "50.000000"},
		})

		_, err := chunker.createChunksFromChapters(mediaInfo)

		if err == nil {
			t.Error("Expected error for invalid chunk (end before start)")
		}
	})

	t.Run("fractional times", func(t *testing.T) {
		chunker := &Chunker{
			sourcePath:  "/test/file.mp4",
			useChapters: true,
		}

		mediaInfo := newMockMediaInfoWithChapters(240, []ChapterInfo{
			{StartTime: "0.500000", EndTime: "120.750000"},
			{StartTime: "120.750000", EndTime: "240.999999"},
		})

		chunks, err := chunker.createChunksFromChapters(mediaInfo)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(chunks) != 2 {
			t.Errorf("Expected 2 chunks, got %d", len(chunks))
		}

		// Verify fractional times are preserved (no longer truncated)
		if chunks[0].StartTime != 0.5 {
			t.Errorf("Expected start time 0.5 (fractional preserved), got %.2f", chunks[0].StartTime)
		}
		if chunks[0].EndTime != 120.75 {
			t.Errorf("Expected end time 120.75 (fractional preserved), got %.2f", chunks[0].EndTime)
		}
	})

	t.Run("chapter validation failure", func(t *testing.T) {
		chunker := &Chunker{
			sourcePath:  "", // Empty source path will cause validation failure
			useChapters: true,
		}

		mediaInfo := newMockMediaInfoWithChapters(100, []ChapterInfo{
			{StartTime: "0.0", EndTime: "100.0"},
		})

		_, err := chunker.createChunksFromChapters(mediaInfo)

		if err == nil {
			t.Error("Expected error when chunk validation fails")
		}
	})

	t.Run("chapters with zero end time causes validation error", func(t *testing.T) {
		chunker := &Chunker{
			sourcePath:  "/test/file.mp4",
			useChapters: true,
		}

		mediaInfo := newMockMediaInfoWithChapters(0, []ChapterInfo{
			{StartTime: "0.0", EndTime: "0.0"}, // EndTime 0 will fail validation
		})

		_, err := chunker.createChunksFromChapters(mediaInfo)

		if err == nil {
			t.Error("Expected error for chapter with EndTime 0")
		}
	})
}

// TestChunker_CreateChunks_WithChapters tests the full CreateChunks flow with chapters
func TestChunker_CreateChunks_WithChapters(t *testing.T) {
	t.Run("use chapters when available", func(t *testing.T) {
		chunker := NewChunker("/test/file.mp4")
		chunker.SetUseChapters(true)

		mediaInfo := newMockMediaInfoWithChapters(600, []ChapterInfo{
			{StartTime: "0", EndTime: "300"},
			{StartTime: "300", EndTime: "600"},
		})

		chunks, err := chunker.CreateChunks(mediaInfo)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should use chapters, not fixed duration
		if len(chunks) != 2 {
			t.Errorf("Expected 2 chunks from chapters, got %d", len(chunks))
		}

		if chunks[0].EndTime != 300 {
			t.Errorf("Expected first chunk to end at 300 (from chapter), got %.2f", chunks[0].EndTime)
		}
	})

	t.Run("fallback to fixed duration when no chapters", func(t *testing.T) {
		chunker := NewChunker("/test/file.mp4")
		chunker.SetChunkDuration(15).SetUseChapters(true) // Try chapters first

		mediaInfo := newMockMediaInfo(30.0) // No chapters

		chunks, err := chunker.CreateChunks(mediaInfo)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// File has no chapters, so should fall back to fixed duration
		// 30 second file / 15 second chunks = 2 chunks
		if len(chunks) != 2 {
			t.Errorf("Expected 2 chunks (30s / 15s), got %d", len(chunks))
		}
	})

	t.Run("chapters disabled uses fixed duration", func(t *testing.T) {
		chunker := NewChunker("/test/file.mp4")
		chunker.SetChunkDuration(10).SetUseChapters(false) // Explicitly disable chapters

		// Even though we have chapters, they should be ignored
		mediaInfo := newMockMediaInfoWithChapters(30, []ChapterInfo{
			{StartTime: "0", EndTime: "15"},
			{StartTime: "15", EndTime: "30"},
		})

		chunks, err := chunker.CreateChunks(mediaInfo)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should use fixed duration (30s / 10s = 3), not chapters (which would be 2)
		if len(chunks) != 3 {
			t.Errorf("Expected 3 chunks (30s / 10s), got %d", len(chunks))
		}
	})
}
