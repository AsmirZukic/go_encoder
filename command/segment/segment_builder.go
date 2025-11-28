package segment

import (
	"encoder/chunker"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// SegmentBuilder builds FFmpeg commands to split input files into segments.
type SegmentBuilder struct {
	sourcePath string
	outputDir  string
	chapters   []chunker.ChapterInfo
}

// NewSegmentBuilder creates a new SegmentBuilder.
func NewSegmentBuilder(sourcePath string, outputDir string, chapters []chunker.ChapterInfo) *SegmentBuilder {
	return &SegmentBuilder{
		sourcePath: sourcePath,
		outputDir:  outputDir,
		chapters:   chapters,
	}
}

// BuildArgs constructs the FFmpeg command arguments for segment splitting.
// Uses -c copy for fast stream copying without re-encoding.
// Outputs Matroska format (.mkv) for better AV1 codec compatibility.
func (s *SegmentBuilder) BuildArgs() []string {
	args := []string{
		"-i", s.sourcePath,
		"-c", "copy", // Copy streams without re-encoding (very fast)
		"-map", "0", // Map all streams
		"-f", "segment", // Segment muxer
		"-segment_format", "matroska", // Use Matroska format (better AV1 compatibility)
		"-segment_times", s.buildSegmentTimes(),
		"-reset_timestamps", "1", // Reset timestamps for each segment
	}

	// Output pattern: tmp/segment_%03d.mkv
	outputPattern := filepath.Join(s.outputDir, "segment_%03d.mkv")
	args = append(args, outputPattern)

	return args
}

// buildSegmentTimes creates a comma-separated list of chapter start times.
// FFmpeg will split at these times: "141.64,282.07,423.72,..."
func (s *SegmentBuilder) buildSegmentTimes() string {
	if len(s.chapters) <= 1 {
		return ""
	}

	times := make([]string, 0, len(s.chapters)-1)
	for i := 1; i < len(s.chapters); i++ {
		// StartTime is already a string in decimal format
		times = append(times, s.chapters[i].StartTime)
	}

	return strings.Join(times, ",")
}

// Run executes the segment splitting command.
func (s *SegmentBuilder) Run() error {
	args := s.BuildArgs()
	cmd := exec.Command("ffmpeg", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("segment split failed: %w (output: %s)", err, string(output))
	}

	return nil
}

// DryRun returns the command string without executing.
func (s *SegmentBuilder) DryRun() string {
	args := s.BuildArgs()
	return fmt.Sprintf("ffmpeg %s", strings.Join(args, " "))
}

// GetSegmentPath returns the path for a segment at the given index.
func (s *SegmentBuilder) GetSegmentPath(index int) string {
	return filepath.Join(s.outputDir, fmt.Sprintf("segment_%03d.mkv", index))
}
