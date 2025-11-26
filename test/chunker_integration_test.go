package encoder_test

import (
	"encoder/chunker"
	"encoder/ffprobe"
	"os"
	"testing"
)

// Integration tests that use both chunker and ffprobe packages.
// These are in a separate test package to avoid import cycles.

// TestChunker_WithRealProbe tests the integration between chunker and ffprobe
func TestChunker_WithRealProbe(t *testing.T) {
	testFile := "/home/asmir/file_example_MP4_480_1_5MG.mp4"

	// Skip test if file doesn't exist
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", testFile)
	}

	t.Run("create chunks from real file probe", func(t *testing.T) {
		probeResult, err := ffprobe.Probe(testFile)
		if err != nil {
			t.Fatalf("Failed to probe file: %v", err)
		}

		chunkerObj := chunker.NewChunker(testFile)
		chunkerObj.SetUseChapters(false).SetChunkDuration(10)

		chunks, err := chunkerObj.CreateChunks(probeResult)
		if err != nil {
			t.Fatalf("Failed to create chunks: %v", err)
		}

		// The test file is 30.53 seconds, so with 10s chunks we expect 4 chunks
		// (0-10, 10-20, 20-30, 30-30.53)
		if len(chunks) != 4 {
			t.Errorf("Expected 4 chunks for 30.53s file with 10s duration, got %d", len(chunks))
		}

		// Verify last chunk has fractional end time
		if len(chunks) > 0 {
			lastChunk := chunks[len(chunks)-1]
			if lastChunk.EndTime < 30.5 || lastChunk.EndTime > 30.6 {
				t.Errorf("Expected last chunk to end around 30.53s, got %.2f", lastChunk.EndTime)
			}
		}
	})

	t.Run("fallback to fixed duration when no chapters", func(t *testing.T) {
		probeResult, err := ffprobe.Probe(testFile)
		if err != nil {
			t.Fatalf("Failed to probe file: %v", err)
		}

		chunkerObj := chunker.NewChunker(testFile)
		chunkerObj.SetChunkDuration(15).SetUseChapters(true) // Try chapters first

		chunks, err := chunkerObj.CreateChunks(probeResult)
		if err != nil {
			t.Fatalf("Failed to create chunks: %v", err)
		}

		// File has no chapters, so should fall back to fixed duration
		// 30.53 second file / 15 second chunks = 3 chunks (0-15, 15-30, 30-30.53)
		if len(chunks) != 3 {
			t.Errorf("Expected 3 chunks (30.53s / 15s), got %d", len(chunks))
		}

		// Verify fractional seconds are preserved
		if len(chunks) == 3 {
			if chunks[0].EndTime != 15.0 {
				t.Errorf("First chunk should end at 15.0, got %.2f", chunks[0].EndTime)
			}
			if chunks[1].EndTime != 30.0 {
				t.Errorf("Second chunk should end at 30.0, got %.2f", chunks[1].EndTime)
			}
			if chunks[2].EndTime < 30.5 || chunks[2].EndTime > 30.6 {
				t.Errorf("Third chunk should end around 30.53, got %.2f", chunks[2].EndTime)
			}
		}
	})
}
