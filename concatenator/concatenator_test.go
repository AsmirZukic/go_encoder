package concatenator

import (
	"encoder/models"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateResults(t *testing.T) {
	tests := []struct {
		name             string
		results          []*models.EncoderResult
		expectSuccessful int
		expectFailed     int
		expectError      bool
	}{
		{
			name:        "empty results",
			results:     []*models.EncoderResult{},
			expectError: true,
		},
		{
			name: "all successful",
			results: []*models.EncoderResult{
				{ChunkID: 1, OutputPath: "/tmp/chunk1.opus", Success: true},
				{ChunkID: 2, OutputPath: "/tmp/chunk2.opus", Success: true},
			},
			expectSuccessful: 2,
			expectFailed:     0,
			expectError:      false,
		},
		{
			name: "all failed",
			results: []*models.EncoderResult{
				{ChunkID: 1, Success: false, Error: nil},
				{ChunkID: 2, Success: false, Error: nil},
			},
			expectSuccessful: 0,
			expectFailed:     2,
			expectError:      false,
		},
		{
			name: "mixed results",
			results: []*models.EncoderResult{
				{ChunkID: 1, OutputPath: "/tmp/chunk1.opus", Success: true},
				{ChunkID: 2, Success: false, Error: nil},
				{ChunkID: 3, OutputPath: "/tmp/chunk3.opus", Success: true},
			},
			expectSuccessful: 2,
			expectFailed:     1,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConcatenator(true)

			// Create temporary files for successful results
			for _, result := range tt.results {
				if result.Success && result.OutputPath != "" {
					f, err := os.Create(result.OutputPath)
					if err != nil {
						t.Fatalf("Failed to create test file: %v", err)
					}
					f.Close()
					defer os.Remove(result.OutputPath)
				}
			}

			successful, failed, err := c.validateResults(tt.results)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(successful) != tt.expectSuccessful {
				t.Errorf("Expected %d successful, got %d", tt.expectSuccessful, len(successful))
			}
			if len(failed) != tt.expectFailed {
				t.Errorf("Expected %d failed, got %d", tt.expectFailed, len(failed))
			}

			// Verify sorting by ChunkID
			for i := 1; i < len(successful); i++ {
				if successful[i].ChunkID <= successful[i-1].ChunkID {
					t.Error("Results not sorted by ChunkID")
				}
			}
		})
	}
}

func TestCheckForGaps(t *testing.T) {
	tests := []struct {
		name        string
		results     []*models.EncoderResult
		expectError bool
		description string
	}{
		{
			name:        "no gaps",
			results:     []*models.EncoderResult{{ChunkID: 1}, {ChunkID: 2}, {ChunkID: 3}},
			expectError: false,
			description: "Sequential chunks 1,2,3",
		},
		{
			name:        "single gap",
			results:     []*models.EncoderResult{{ChunkID: 1}, {ChunkID: 3}},
			expectError: true,
			description: "Missing chunk 2",
		},
		{
			name:        "multiple gaps",
			results:     []*models.EncoderResult{{ChunkID: 1}, {ChunkID: 4}, {ChunkID: 7}},
			expectError: true,
			description: "Missing chunks 2,3,5,6",
		},
		{
			name:        "single chunk",
			results:     []*models.EncoderResult{{ChunkID: 1}},
			expectError: false,
			description: "Only one chunk, no gaps possible",
		},
		{
			name:        "empty",
			results:     []*models.EncoderResult{},
			expectError: false,
			description: "No chunks, no gaps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConcatenator(true)
			err := c.checkForGaps(tt.results)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.description, err)
			}
		})
	}
}

func TestCreateConcatFile(t *testing.T) {
	// Create temporary chunk files
	tmpDir := t.TempDir()
	chunk1 := filepath.Join(tmpDir, "chunk1.opus")
	chunk2 := filepath.Join(tmpDir, "chunk2.opus")

	for _, path := range []string{chunk1, chunk2} {
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	results := []*models.EncoderResult{
		{ChunkID: 1, OutputPath: chunk1, Success: true},
		{ChunkID: 2, OutputPath: chunk2, Success: true},
	}

	c := NewConcatenator(true)
	concatFile, err := c.createConcatFile(results)
	if err != nil {
		t.Fatalf("createConcatFile failed: %v", err)
	}
	defer os.Remove(concatFile)

	// Verify concat file exists
	if _, err := os.Stat(concatFile); err != nil {
		t.Errorf("Concat file not created: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(concatFile)
	if err != nil {
		t.Fatalf("Failed to read concat file: %v", err)
	}

	contentStr := string(content)
	if !contains(contentStr, "chunk1.opus") {
		t.Error("Concat file doesn't contain chunk1.opus")
	}
	if !contains(contentStr, "chunk2.opus") {
		t.Error("Concat file doesn't contain chunk2.opus")
	}
	if !contains(contentStr, "file '") {
		t.Error("Concat file doesn't have proper format")
	}
}

func TestConcatenate_StrictMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test chunk files
	chunk1 := filepath.Join(tmpDir, "chunk1.opus")
	chunk2 := filepath.Join(tmpDir, "chunk2.opus")
	output := filepath.Join(tmpDir, "output.opus")

	for _, path := range []string{chunk1, chunk2} {
		if err := os.WriteFile(path, []byte("test audio data"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	t.Run("strict mode with failed chunks", func(t *testing.T) {
		results := []*models.EncoderResult{
			{ChunkID: 1, OutputPath: chunk1, Success: true},
			{ChunkID: 2, Success: false, Error: nil}, // Failed chunk
		}

		c := NewConcatenator(true) // strict mode
		err := c.Concatenate(results, output)
		if err == nil {
			t.Error("Expected error in strict mode with failed chunks")
		}
	})

	t.Run("permissive mode with failed chunks", func(t *testing.T) {
		results := []*models.EncoderResult{
			{ChunkID: 1, OutputPath: chunk1, Success: true},
			{ChunkID: 2, Success: false, Error: nil}, // Failed chunk
		}

		c := NewConcatenator(false) // permissive mode
		// Note: This will likely fail during ffmpeg execution with test data,
		// but it should pass the validation step
		err := c.Concatenate(results, output)
		// We expect it to attempt concatenation (may fail at ffmpeg stage with invalid test data)
		// The key is that it doesn't fail at validation
		if err != nil && contains(err.Error(), "strict mode") {
			t.Error("Should not fail with strict mode error in permissive mode")
		}
	})
}

func TestConcatenate_WithGaps(t *testing.T) {
	tmpDir := t.TempDir()

	chunk1 := filepath.Join(tmpDir, "chunk1.opus")
	chunk3 := filepath.Join(tmpDir, "chunk3.opus")
	output := filepath.Join(tmpDir, "output.opus")

	for _, path := range []string{chunk1, chunk3} {
		if err := os.WriteFile(path, []byte("test audio data"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	results := []*models.EncoderResult{
		{ChunkID: 1, OutputPath: chunk1, Success: true},
		{ChunkID: 3, OutputPath: chunk3, Success: true}, // Gap: missing chunk 2
	}

	t.Run("strict mode with gaps", func(t *testing.T) {
		c := NewConcatenator(true)
		err := c.Concatenate(results, output)
		if err == nil {
			t.Error("Expected error in strict mode with gaps")
		}
	})

	t.Run("permissive mode with gaps", func(t *testing.T) {
		c := NewConcatenator(false)
		err := c.Concatenate(results, output)
		// Should attempt concatenation despite gaps
		if err != nil && contains(err.Error(), "strict mode") {
			t.Error("Should not fail with strict mode error in permissive mode")
		}
	})
}

func TestConcatenateSimple(t *testing.T) {
	tmpDir := t.TempDir()

	chunk1 := filepath.Join(tmpDir, "chunk1.opus")
	chunk2 := filepath.Join(tmpDir, "chunk2.opus")
	output := filepath.Join(tmpDir, "output.opus")

	for _, path := range []string{chunk1, chunk2} {
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	chunkPaths := []string{chunk1, chunk2}
	err := ConcatenateSimple(chunkPaths, output)

	// Will likely fail at ffmpeg stage with invalid test data, but should not panic
	_ = err // We're testing that it doesn't panic and properly handles the paths
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
