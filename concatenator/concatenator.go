package concatenator

import (
	"encoder/models"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Concatenator handles merging encoded chunks into a final output file
type Concatenator struct {
	strictMode bool // If true, fail if any chunks are missing. If false, skip missing chunks.
}

// NewConcatenator creates a new concatenator
func NewConcatenator(strictMode bool) *Concatenator {
	return &Concatenator{
		strictMode: strictMode,
	}
}

// Concatenate merges encoded chunks into a final output file using ffmpeg's concat demuxer
func (c *Concatenator) Concatenate(results []*models.EncoderResult, finalOutputPath string) error {
	// Validate results
	successful, failed, err := c.validateResults(results)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(failed) > 0 {
		if c.strictMode {
			return fmt.Errorf("strict mode: %d chunks failed encoding", len(failed))
		}
		fmt.Printf("Warning: %d chunks failed, proceeding with %d successful chunks\n", len(failed), len(successful))
	}

	if len(successful) == 0 {
		return fmt.Errorf("no successful chunks to concatenate")
	}

	// Check for gaps in chunk sequence
	if err := c.checkForGaps(successful); err != nil {
		if c.strictMode {
			return fmt.Errorf("strict mode: %w", err)
		}
		fmt.Printf("Warning: %v\n", err)
	}

	// Create concat file for ffmpeg
	concatFilePath, err := c.createConcatFile(successful)
	if err != nil {
		return fmt.Errorf("failed to create concat file: %w", err)
	}
	defer os.Remove(concatFilePath) // Clean up concat file after use

	// Run ffmpeg concat
	if err := c.runConcat(concatFilePath, finalOutputPath); err != nil {
		return fmt.Errorf("ffmpeg concat failed: %w", err)
	}

	return nil
}

// validateResults separates successful and failed results
func (c *Concatenator) validateResults(results []*models.EncoderResult) (successful, failed []*models.EncoderResult, err error) {
	if len(results) == 0 {
		return nil, nil, fmt.Errorf("no results provided")
	}

	for _, result := range results {
		if result.Success && result.OutputPath != "" {
			// Verify file exists
			if _, err := os.Stat(result.OutputPath); err != nil {
				failed = append(failed, result)
			} else {
				successful = append(successful, result)
			}
		} else {
			failed = append(failed, result)
		}
	}

	// Sort successful results by ChunkID to ensure correct order
	sort.Slice(successful, func(i, j int) bool {
		return successful[i].ChunkID < successful[j].ChunkID
	})

	return successful, failed, nil
}

// checkForGaps detects missing chunks in the sequence
func (c *Concatenator) checkForGaps(successful []*models.EncoderResult) error {
	if len(successful) == 0 {
		return nil
	}

	// Check for gaps in chunk IDs
	gaps := []uint{}
	for i := 0; i < len(successful)-1; i++ {
		currentID := successful[i].ChunkID
		nextID := successful[i+1].ChunkID

		if nextID != currentID+1 {
			// Found a gap
			for id := currentID + 1; id < nextID; id++ {
				gaps = append(gaps, id)
			}
		}
	}

	if len(gaps) > 0 {
		return fmt.Errorf("missing chunks: %v", gaps)
	}

	return nil
}

// createConcatFile creates a text file listing all chunk paths for ffmpeg concat demuxer
// Format: file '/path/to/chunk1.mp4'
//
//	file '/path/to/chunk2.mp4'
func (c *Concatenator) createConcatFile(successful []*models.EncoderResult) (string, error) {
	// Create temporary concat file
	tmpFile, err := os.CreateTemp("", "concat-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	// Write file list
	for _, result := range successful {
		// Use absolute path and escape single quotes
		absPath, err := filepath.Abs(result.OutputPath)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path for %s: %w", result.OutputPath, err)
		}

		// Escape single quotes in path (replace ' with '\''  for shell)
		escapedPath := strings.ReplaceAll(absPath, "'", "'\\''")

		line := fmt.Sprintf("file '%s'\n", escapedPath)
		if _, err := tmpFile.WriteString(line); err != nil {
			return "", fmt.Errorf("failed to write to concat file: %w", err)
		}
	}

	return tmpFile.Name(), nil
}

// runConcat executes ffmpeg concat operation
func (c *Concatenator) runConcat(concatFilePath, outputPath string) error {
	args := []string{
		"-f", "concat",
		"-safe", "0",
		"-i", concatFilePath,
		"-c", "copy", // Copy without re-encoding
		"-y", // Overwrite output file
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)

	// Capture output for error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %w\nOutput: %s", err, string(output))
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); err != nil {
		return fmt.Errorf("output file not created: %w", err)
	}

	return nil
}

// ConcatenateSimple is a convenience function for basic concatenation
func ConcatenateSimple(chunkPaths []string, outputPath string) error {
	// Convert paths to encoder results
	results := make([]*models.EncoderResult, len(chunkPaths))
	for i, path := range chunkPaths {
		results[i] = &models.EncoderResult{
			ChunkID:    uint(i + 1),
			OutputPath: path,
			Success:    true,
			Error:      nil,
		}
	}

	concat := NewConcatenator(true) // strict mode
	return concat.Concatenate(results, outputPath)
}
